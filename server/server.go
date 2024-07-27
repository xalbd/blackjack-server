package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/xalbd/blackjack-server/game"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type command struct {
	message string
	player  int
}

type broadcast struct {
	Id      int
	Players []game.Player
}

type Room struct {
	clients   map[*websocket.Conn]int
	commands  chan command
	broadcast chan broadcast
}

func StartServer() {
	server := Room{
		make(map[*websocket.Conn]int),
		make(chan command),
		make(chan broadcast),
	}

	go server.startTable()
	go server.broadcastMessages()

	http.HandleFunc("/", server.handleConnections)
	http.ListenAndServe(":8080", nil)
}

func (room *Room) handleConnections(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer func() {
		c.Close()
		delete(room.clients, c)
	}()

	room.clients[c] = len(room.clients)
	room.commands <- command{"new", room.clients[c]}

	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Println("error:", err)
			break
		}
		log.Printf("recv: %s", message)

		room.commands <- command{string(message), room.clients[c]}
	}
}

func (room *Room) startTable() {
	table := game.NewTable()
	table.ResetHands()

	for {
		command := <-room.commands
		if command.message == "new" {
			for len(table.Players) <= command.player {
				table.Players = append(table.Players, game.Player{Money: 1000})
			}
			room.broadcast <- broadcast{Players: table.Players}
			continue
		}
		table.HandleCommand(command.player, command.message)
		room.broadcast <- broadcast{Players: table.Players}
	}
}

func (room *Room) broadcastMessages() {
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
