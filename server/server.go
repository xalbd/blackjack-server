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

type broadcast struct {
	Id         uuid.UUID
	ActiveHand int
	Players    []game.Player
	Hands      []game.Hand
}

type room struct {
	clients   map[*websocket.Conn]uuid.UUID
	commands  chan command
	broadcast chan broadcast
}

func StartServer() {
	server := room{
		make(map[*websocket.Conn]uuid.UUID),
		make(chan command),
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
	defer func() {
		c.Close()
		delete(room.clients, c)
	}()

	room.clients[c] = uuid.New()

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

func (room *room) startTable() {
	table := game.NewTable()
	table.ResetHands()

	for {
		command := <-room.commands
		table.HandleCommand(command.playerId, command.message)
		room.broadcast <- broadcast{Players: table.Players, Hands: table.Hands, ActiveHand: table.ActiveHand}
	}
}

func (room *room) broadcastMessages() {
	for {
		message := <-room.broadcast
		for client := range room.clients {
			message.Id = room.clients[client]
			out, _ := json.Marshal(message)
			err := client.WriteMessage(websocket.TextMessage, []byte(out))
			if err != nil {
				log.Printf("error: %v", err)
				client.Close()
				delete(room.clients, client)
			}
		}
	}
}
