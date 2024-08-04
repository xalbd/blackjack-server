package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"github.com/gorilla/websocket"
	"github.com/xalbd/blackjack-server/game"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type command struct {
	message  []byte
	playerId string
}

type playerUpdate struct {
	playerId string
	money    int64
	connect  bool
}

type broadcast struct {
	PlayerId   string        `json:"playerId"`
	ActiveHand int           `json:"activeHand"`
	Players    []game.Player `json:"players"`
	Hands      []game.Hand   `json:"hands"`
}

type room struct {
	clients       map[*websocket.Conn]string
	commands      chan command
	playerUpdates chan playerUpdate
	moneyUpdates  chan game.MoneyUpdate
	broadcast     chan broadcast
	firebase      *firebase.App
	firestore     *firestore.Client
	ctx           context.Context
}

func StartServer() {
	ctx := context.Background()
	app, err := firebase.NewApp(ctx, nil)
	if err != nil {
		log.Fatalf("error initializing app: %v\n", err)
	}

	firestore, err := app.Firestore(ctx)
	if err != nil {
		log.Fatalln(err)
	}
	defer firestore.Close()

	server := room{
		make(map[*websocket.Conn]string),
		make(chan command),
		make(chan playerUpdate),
		make(chan game.MoneyUpdate),
		make(chan broadcast),
		app,
		firestore,
		ctx,
	}

	go server.startTable()
	go server.broadcastMessages()
	go server.updateMoney()

	http.HandleFunc("/", server.handleConnections)
	http.ListenAndServe(":8080", nil)
}

func (room *room) handleConnections(w http.ResponseWriter, r *http.Request) {
	// upgrade any connections to a websocket
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("websocket upgrade error:", err)
		return
	}
	defer c.Close()

	// require jwt token within 5 seconds of connection to authorize user
	c.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, message, err := c.ReadMessage()
	if err != nil {
		log.Println("did not recv firebase auth jwt:", err)
		return
	}

	// authorize user with firebase
	client, err := room.firebase.Auth(room.ctx)
	if err != nil {
		log.Println("error fetching firebase auth client:", err)
		return
	}

	token, err := client.VerifyIDToken(room.ctx, string(message))
	if err != nil {
		log.Println("error verifying auth token:", err)
		return
	}

	room.firestore.Collection("users").Doc(token.UID).Create(room.ctx,
		map[string]interface{}{
			"money": 1000,
		})

	player, err := room.firestore.Collection("users").Doc(token.UID).Get(room.ctx)
	if err != nil {
		log.Println("error fetching user money:", err)
		return
	}

	// track authorized user and notify other players
	room.clients[c] = token.UID
	room.playerUpdates <- playerUpdate{room.clients[c], player.Data()["money"].(int64), true}
	defer room.removePlayer(c)

	for {
		c.SetReadDeadline(time.Now().Add(5 * time.Minute))
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Println("error reading from websocket:", err)
			break
		}
		log.Printf("recv: %s", message)

		room.commands <- command{message, room.clients[c]}
	}
}

func (room *room) removePlayer(c *websocket.Conn) {
	room.playerUpdates <- playerUpdate{room.clients[c], 0, false}
	delete(room.clients, c)
}

func (room *room) startTable() {
	table := game.NewTable(room.moneyUpdates)
	table.ResetHands()

	for {
		select {
		case command := <-room.commands:
			table.HandleCommand(command.playerId, command.message)
		case playerUpdate := <-room.playerUpdates:
			table.HandlePlayerUpdate(playerUpdate.playerId, playerUpdate.money, playerUpdate.connect)
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
