package game

import (
	"log"

	"github.com/gorilla/websocket"
)

type room struct {
	table          table
	clients        map[*websocket.Conn]string
	wsCommands     chan wsCommand
	playersUpdates chan playersUpdate
	broadcast      chan []byte
}

type wsCommand struct {
	message  []byte
	playerId string
}

type playersUpdate struct {
	playerId    string
	displayName string
	connect     bool
}

func (room *room) removePlayer(c *websocket.Conn) {
	allGone := true
	for k, v := range room.clients {
		if k != c && v == room.clients[c] {
			allGone = false
			break
		}
	}

	if allGone {
		room.playersUpdates <- playersUpdate{playerId: room.clients[c], connect: false}
	}
	delete(room.clients, c)
}

func (room *room) startTable() {
	room.table.resetHands()

	for {
		select {
		case command := <-room.wsCommands:
			room.table.handleCommand(command)
		case playerUpdate := <-room.playersUpdates:
			room.table.handlePlayerUpdate(playerUpdate)
		}
	}
}

func (room *room) broadcastMessages() {
	for {
		message := <-room.broadcast
		for c := range room.clients {
			err := c.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				log.Println("error sending to websocket:", err)
			}
		}
	}
}
