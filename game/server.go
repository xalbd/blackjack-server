package game

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return r.Header.Get("Origin") == os.Getenv("FRONTEND")
	},
}

type server struct {
	rooms      map[string]room
	players    map[string]int64
	playerLock sync.RWMutex
	firebase   *firebase.App
	firestore  *firestore.Client
	ctx        context.Context // TODO: still no idea what context actually is but keeping it here seems fine (?)
}

type roomInfo struct {
	Code       string `json:"code"`
	Seats      int    `json:"seats"`
	TakenSeats int    `json:"takenSeats"`
}

type infoResponse struct {
	Rooms []roomInfo `json:"rooms"`
}

type createRoomRequest struct {
	Seats int
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

	server := server{
		rooms:     make(map[string]room),
		players:   make(map[string]int64),
		firebase:  app,
		firestore: firestore,
		ctx:       ctx,
	}

	server.addRoom("roomy", 6)
	server.addRoom("another", 4)

	mux := http.NewServeMux()

	mux.HandleFunc("/room/{room}/ws", server.handleWebsocketConnections)
	mux.HandleFunc("/room/{room}", server.handleRoomRequest)
	mux.HandleFunc("/create", server.handleCreateRequest)
	mux.HandleFunc("/info", server.handleInfoRequest)
	http.ListenAndServe(":8080", checkCORS(mux))
}

func checkCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == os.Getenv("FRONTEND") {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
		}
		next.ServeHTTP(w, r)
	})
}

func (server *server) addRoom(roomCode string, seats int) {
	broadcastChannel := make(chan []byte)

	r := room{
		newTable(broadcastChannel, seats, server.getMoney, server.deltaMoney),
		make(map[*websocket.Conn]string),
		make(chan wsCommand),
		make(chan playersUpdate),
		broadcastChannel,
	}

	server.rooms[roomCode] = r
	go r.startTable()
	go r.broadcastMessages()
}

func (server *server) handleInfoRequest(w http.ResponseWriter, r *http.Request) {
	info := infoResponse{make([]roomInfo, len(server.rooms))}

	i := 0
	for k, v := range server.rooms {
		info.Rooms[i].Code = k
		info.Rooms[i].Seats = v.table.seats
		info.Rooms[i].TakenSeats = v.table.seatsTaken()
		i++
	}

	out, err := json.Marshal(info)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

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
	}
}

func (server *server) generateNewRoomCode() string {
	characters := "abcdefghjkmnpqrstuvwxyz23456789"
	code := make([]byte, 4)
	for {
		for i := 0; i < 4; i++ {
			code[i] = characters[rand.Intn(len(characters))]
		}

		if _, ok := server.rooms[string(code)]; !ok {
			break
		}
	}

	return string(code)
}

func (server *server) setMoney(uid string, amount int64) {
	server.playerLock.Lock()
	// prevent race condition by allowing initial money to be set only once
	if _, ok := server.players[uid]; !ok {
		server.players[uid] = amount
	}
	server.playerLock.Unlock()
}

func (server *server) deltaMoney(uid string, delta int64) {
	server.playerLock.Lock()
	server.players[uid] += delta
	go server.firestore.Collection("users").Doc(uid).Set(server.ctx,
		map[string]interface{}{
			"money": server.players[uid],
		})
	server.playerLock.Unlock()
}

func (server *server) getMoney(uid string) int64 {
	server.playerLock.RLock()
	defer server.playerLock.RUnlock()
	return server.players[uid]
}

func (server *server) handleCreateRequest(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req createRoomRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if req.Seats < 2 || req.Seats > 8 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		roomCode := server.generateNewRoomCode()
		server.addRoom(roomCode, req.Seats)

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(roomCode))
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

	// authorize user with firebase & create/grab firestore document
	client, err := server.firebase.Auth(server.ctx)
	if err != nil {
		log.Println("error fetching firebase auth client:", err)
		return
	}

	token, err := client.VerifyIDToken(server.ctx, string(message))
	if err != nil {
		log.Println("error verifying auth token:", err)
		return
	}

	server.firestore.Collection("users").Doc(token.UID).Create(server.ctx,
		map[string]interface{}{
			"money": 1000,
		})

	player, err := server.firestore.Collection("users").Doc(token.UID).Get(server.ctx)
	if err != nil {
		log.Println("error fetching user money:", err)
		return
	}

	// grab display name
	ur, err := client.GetUser(server.ctx, token.UID)
	if err != nil {
		log.Println("error fetching user information:", err)
		return
	}

	// track authorized user and notify other players
	room.clients[c] = token.UID
	room.playersUpdates <- playersUpdate{room.clients[c], ur.DisplayName, true}
	server.setMoney(token.UID, player.Data()["money"].(int64))
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
