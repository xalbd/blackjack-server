package game

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	// TODO: add origin check
	CheckOrigin: func(r *http.Request) bool { return true },
}

type server struct {
	rooms     map[string]room
	app       *firebase.App
	firestore *firestore.Client
	ctx       context.Context
}

type info struct {
	Rooms []string `json:"rooms"`
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

	server := server{rooms: make(map[string]room), app: app, firestore: firestore, ctx: ctx}

	i := room{
		make(map[*websocket.Conn]string),
		make(chan wsCommand),
		make(chan playersUpdate),
		make(chan moneyUpdate),
		make(chan []byte),
		app,
		firestore,
		ctx,
	}

	j := room{
		make(map[*websocket.Conn]string),
		make(chan wsCommand),
		make(chan playersUpdate),
		make(chan moneyUpdate),
		make(chan []byte),
		app,
		firestore,
		ctx,
	}

	server.addRoom("roomy", i)
	server.addRoom("another", j)

	http.HandleFunc("/room/{room}/ws", server.handleWebsocketConnections)
	http.HandleFunc("/room/{room}", server.handleRoomRequest)
	http.HandleFunc("/info", server.handleInfoRequest)
	http.ListenAndServe(":8080", nil)
}

func (server *server) addRoom(roomCode string, r room) {
	server.rooms[roomCode] = r
	go r.startTable()
	go r.broadcastMessages()
	go r.updateMoney()
}

func (server *server) handleInfoRequest(w http.ResponseWriter, r *http.Request) {
	info := info{make([]string, len(server.rooms))}

	i := 0
	for k := range server.rooms {
		info.Rooms[i] = k
		i++
	}

	out, err := json.Marshal(info)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// also fix this CORS issue
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	w.Write(out)
}

func (server *server) handleRoomRequest(w http.ResponseWriter, r *http.Request) {
	roomCode := r.PathValue("room")

	switch r.Method {
	case http.MethodGet:
		// status 404 if room doesn't exist; status 200 if it does to let client know they can establish websocket connection
		_, ok := server.rooms[roomCode]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusOK)
		}

	case http.MethodPost:
		// temporary room creation
		server.addRoom(roomCode, room{
			make(map[*websocket.Conn]string),
			make(chan wsCommand),
			make(chan playersUpdate),
			make(chan moneyUpdate),
			make(chan []byte),
			server.app,
			server.firestore,
			server.ctx,
		})
	}
}

func (server *server) handleWebsocketConnections(w http.ResponseWriter, r *http.Request) {
	// grab requested room from path
	roomCode := r.PathValue("room")
	room, ok := server.rooms[roomCode]
	if !ok {
		return
	}

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
	room.playersUpdates <- playersUpdate{room.clients[c], player.Data()["money"].(int64), true}
	defer room.removePlayer(c)

	for {
		c.SetReadDeadline(time.Now().Add(5 * time.Minute))
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Println("error reading from websocket:", err)
			break
		}
		log.Printf("recv: %s", message)

		room.wsCommands <- wsCommand{message, room.clients[c]}
	}
}
