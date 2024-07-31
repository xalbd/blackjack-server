package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/xalbd/blackjack-server/game"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type command struct {
	message  []byte
	playerId uuid.UUID
}

type playerUpdate struct {
	playerId uuid.UUID
	connect  bool
}

type broadcast struct {
	PlayerId   uuid.UUID     `json:"playerId"`
	ActiveHand int           `json:"activeHand"`
	Players    []game.Player `json:"players"`
	Hands      []game.Hand   `json:"hands"`
}

type room struct {
	clients       map[*websocket.Conn]uuid.UUID
	commands      chan command
	playerUpdates chan playerUpdate
	broadcast     chan broadcast
}

func StartServer() {
	server := room{
		make(map[*websocket.Conn]uuid.UUID),
		make(chan command),
		make(chan playerUpdate),
		make(chan broadcast),
	}

	go server.startTable()
	go server.broadcastMessages()

	http.HandleFunc("/", server.handleConnections)
	http.ListenAndServe(":8080", nil)
}

func (room *room) handleConnections(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer room.closeConnection(c)

	room.clients[c] = uuid.New()
	room.playerUpdates <- playerUpdate{room.clients[c], true}

	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Println("error:", err)
			break
		}
		log.Printf("recv: %s", message)

		room.commands <- command{message, room.clients[c]}
	}
}

func (room *room) closeConnection(c *websocket.Conn) {
	c.Close()
	room.playerUpdates <- playerUpdate{room.clients[c], false}
	delete(room.clients, c)
}

func (room *room) startTable() {
	table := game.NewTable()
	table.ResetHands()

	for {
		select {
		case command := <-room.commands:
			table.HandleCommand(command.playerId, command.message)
		case playerUpdate := <-room.playerUpdates:
			table.HandlePlayerUpdate(playerUpdate.playerId, playerUpdate.connect)
		}

		room.broadcast <- broadcast{Players: table.Players, Hands: table.Hands, ActiveHand: table.ActiveHand}
	}
}

func (room *room) broadcastMessages() {
	for {
		message := <-room.broadcast
		for c := range room.clients {
			message.PlayerId = room.clients[c]
			out, _ := json.Marshal(message)
			err := c.WriteMessage(websocket.TextMessage, []byte(out))
			if err != nil {
				log.Printf("error: %v", err)
			}
		}
	}
}
