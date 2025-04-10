package hub

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

var (
	writeWait      = 10 * time.Second
	maxMessageSize = 1024
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
)

type Hub struct {
	Clients          map[*Client]bool
	Room             map[int]*Room
	NewRoom          chan *CreateRoomRequest
	Registered       chan *Client
	Unregistered     chan *Client
	PrivateMessage   chan *PrivateMessage
	BroadcastMessage chan []byte
	GroupMessage     chan []byte
	Shutdown         chan struct{}
}

type Client struct {
	Username string
	Conn     *websocket.Conn
	Send     chan []byte
	Hub      *Hub
}

type PrivateMessage struct {
	From    *Client
	To      string
	Message []byte
}

type Room struct {
	ID      int
	Name    string
	Clients map[*Client]bool
}

type CreateRoomRequest struct {
	Creator *Client
	Name    string
}

type Message struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Type    string `json:"type"`
	Content string `json:"content"`
}

func NewHub() *Hub {
	return &Hub{
		Clients:          make(map[*Client]bool),
		NewRoom:          make(chan *CreateRoomRequest),
		Room:             make(map[int]*Room),
		Registered:       make(chan *Client),
		Unregistered:     make(chan *Client),
		PrivateMessage:   make(chan *PrivateMessage),
		BroadcastMessage: make(chan []byte),
		GroupMessage:     make(chan []byte),
		Shutdown:         make(chan struct{}),
	}
}

func (u *Hub) Run() {
	for {
		select {
		case client := <-u.Registered:
			u.Clients[client] = true
			log.Printf("%s Is Connected", client.Username)
			log.Printf("Total Connected Users: %d", len(u.Clients))
			u.broadcastOnlineUsers()

		case client := <-u.Unregistered:
			if _, ok := u.Clients[client]; ok {
				delete(u.Clients, client)
				close(client.Send)
				log.Printf("%s Is Disconnected", client.Username)
				u.broadcastOnlineUsers()
			}

		case msg := <-u.BroadcastMessage:
			var fromMessage Message
			if err := json.Unmarshal(msg, &fromMessage); err != nil {
				log.Printf("Error unmarshalling message when from message: %v", err)
				continue
			}
			for client := range u.Clients {
				if client.Username == fromMessage.From {
					continue
				}

				select {
				case client.Send <- msg:
				default:
					delete(u.Clients, client)
					close(client.Send)
				}
			}

		case msg := <-u.PrivateMessage:
			found := false
			for client := range u.Clients {
				if msg.To == client.Username {
					found = true
					select {
					case client.Send <- msg.Message:
						if msg.From != nil {
							receipt := []byte(`{"type":"status","content":"Message delivered to ` + msg.To + `"}`)
							msg.From.Send <- receipt
						}
					default:
						delete(u.Clients, client)
						close(client.Send)
					}
					break
				}
			}
			if !found || msg.To == "" {
				receipt := []byte(`{"type":"status","content":"User ` + msg.To + ` is not found or not connected"}`)
				msg.From.Send <- receipt
			}

		case req := <-u.NewRoom:
			ID := len(u.Room) + 1

			room := &Room{
				ID:      ID,
				Name:    req.Name,
				Clients: make(map[*Client]bool),
			}
			room.Clients[req.Creator] = true
			u.Room[ID] = room

			log.Printf("Room %s created by %s with ID %d", req.Name, req.Creator.Username, ID)

			receipt := []byte(`{"type":"status","content":"Room ` + req.Name + ` created"}`)
			req.Creator.Send <- receipt

		case <-u.Shutdown:
			for client := range u.Clients {
				close(client.Send)
				delete(u.Clients, client)
			}
			return
		}

	}
}

func (u *Hub) broadcastOnlineUsers() {
	var clients []string

	for client := range u.Clients {
		clients = append(clients, client.Username)
	}

	userList, _ := json.Marshal(map[string]any{
		"type":         "userList",
		"online_users": clients,
	})

	log.Println("Broadcasting online users:", string(userList))

	for client := range u.Clients {
		select {
		case client.Send <- userList:
		default:
			close(client.Send)
			delete(u.Clients, client)
		}
	}
}

func (u *Client) ReadPump() {
	defer func() {
		u.Hub.Unregistered <- u
		u.Conn.Close()
	}()

	u.Conn.SetReadLimit(int64(maxMessageSize))
	u.Conn.SetReadDeadline(time.Now().Add(pongWait))
	u.Conn.SetPongHandler(func(appData string) error {
		u.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, msg, err := u.Conn.ReadMessage()
		if err != nil {
			log.Println(err.Error())
			return
		}

		message := new(Message)
		if err := json.Unmarshal(msg, message); err != nil {
			log.Printf("Error unmarshalling message: %v", err)
			continue
		}

		message.From = u.Username

		if message.To == "" && message.Type == "group" {
			message.To = "general"
		}

		if message.Content == "" {
			continue
		}

		msg, err = json.Marshal(message)
		if err != nil {
			log.Printf("Error marshalling message: %v", err)
			continue
		}

		if message.Type == "group" {
			u.Hub.BroadcastMessage <- msg
		} else if message.Type == "private" {
			if message.To == "" {
				continue
			}
			u.Hub.PrivateMessage <- &PrivateMessage{
				From:    u,
				To:      message.To,
				Message: msg,
			}
		} else if message.Type == "create_room" {
			if message.Content == "" {
				continue
			}
			newRoom := &CreateRoomRequest{
				Creator: u,
				Name:    message.Content,
			}

			u.Hub.NewRoom <- newRoom
		}
	}
}

func (u *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		u.Conn.Close()
	}()

	for {
		select {
		case msg, ok := <-u.Send:
			u.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				u.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			writer, err := u.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				log.Printf("Error getting writer: %v", err)
				return
			}

			if _, err := writer.Write(msg); err != nil {
				log.Printf("Error writing message: %v", err)
				return
			}

			n := len(u.Send)
			for i := 0; i < n; i++ {
				if _, err := writer.Write(<-u.Send); err != nil {
					log.Printf("Error writing message: %v", err)
					return
				}
			}
			if err := writer.Close(); err != nil {
				return
			}

		case <-ticker.C:
			u.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := u.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Error writing ping message: %v", err)
				return
			}
		}
	}
}
