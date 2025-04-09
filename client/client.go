package client

import (
	"log"
	"net/http"
	"websocket_try3/hub"

	"github.com/gorilla/websocket"
)

var upgrade = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func ServeWs(w http.ResponseWriter, r *http.Request, hubs *hub.Hub) {
	conn, err := upgrade.Upgrade(w, r, nil)
	if err != nil {
		log.Fatalf("Error while upgrading connection: %v", err)
		return
	}

	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	client := &hub.Client{
		Username: username,
		Conn:     conn,
		Send:     make(chan []byte, 256),
		Hub:      hubs,
	}
	hubs.Registered <- client

	go client.WritePump()
	go client.ReadPump()
}
