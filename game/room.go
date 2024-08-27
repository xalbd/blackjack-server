package game

import (
	"context"
	"log"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"github.com/gorilla/websocket"
)

type room struct {
	clients        map[*websocket.Conn]string
	wsCommands     chan wsCommand
	playersUpdates chan playersUpdate
	moneyUpdates   chan moneyUpdate
	broadcast      chan []byte
	firebase       *firebase.App
	firestore      *firestore.Client
	ctx            context.Context // no clue what context actually is but I just pass this around everywhere it's requested
}

type wsCommand struct {
	message  []byte
	playerId string
}

type playersUpdate struct {
	playerId string
	money    int64
	connect  bool
}

type moneyUpdate struct {
	UID   string
	Money int64
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
		room.playersUpdates <- playersUpdate{room.clients[c], 0, false}
	}
	delete(room.clients, c)
}

func (room *room) startTable() {
	table := newTable(room.moneyUpdates, room.broadcast)
	table.resetHands()

	for {
		select {
		case command := <-room.wsCommands:
			table.handleCommand(command)
		case playerUpdate := <-room.playersUpdates:
			table.handlePlayerUpdate(playerUpdate)
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

func (room *room) updateMoney() {
	for {
		moneyUpdate := <-room.moneyUpdates
		room.firestore.Collection("users").Doc(moneyUpdate.UID).Set(room.ctx,
			map[string]interface{}{
				"money": moneyUpdate.Money,
			})
	}
}
