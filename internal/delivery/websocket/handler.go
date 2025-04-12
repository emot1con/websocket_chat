package websocket

import (
	"fmt"
	"net/http"

	"websocket_try3/internal/usecase"

	"github.com/gorilla/websocket"
)

type WebSocketHandler struct {
	upgrader *websocket.Upgrader
	usecase  *usecase.WebSocketUsecase
}

func NewWebSocketHandler(usecase *usecase.WebSocketUsecase) *WebSocketHandler {
	return &WebSocketHandler{
		upgrader: &websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		usecase: usecase,
	}
}

func (h *WebSocketHandler) ServeWS(w http.ResponseWriter, r *http.Request, hubs *Hub) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to register user: %v", err.Error()), http.StatusInternalServerError)
		return
	}

	hubs.UseCase = h.usecase

	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	// Register user to database
	if err := h.usecase.RegisterUser(username); err != nil {
		http.Error(w, fmt.Sprintf("Failed to register user: %v", err.Error()), http.StatusInternalServerError)
		conn.Close()
		return
	}

	client := &Client{
		Username: username,
		Conn:     conn,
		Send:     make(chan []byte, 256),
		Hub:      hubs,
	}
	hubs.Registered <- client

	go client.WritePump()
	go client.ReadPump()
}
