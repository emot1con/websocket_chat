package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"
	"websocket_try3/internal/usecase"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

var (
	writeWait      = 10 * time.Second
	maxMessageSize = 1024
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
)

type Hub struct {
	Clients        map[*Client]bool
	ClientsByUser  map[string]*Client
	Room           map[int]*Room
	NewRoom        chan *CreateRoomRequest
	JoinRoom       chan *JoinRoomRequest
	Registered     chan *Client
	Unregistered   chan *Client
	PrivateMessage chan *PrivateMessage
	GroupMessage   chan *GroupMessage
	UseCase        *usecase.WebSocketUsecase
	Shutdown       chan struct{}
	Mutex          *sync.Mutex
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
	Content []byte
}

type GroupMessage struct {
	From    *Client
	Room    *Room
	Content []byte
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

type JoinRoomRequest struct {
	Client  *Client
	GroupID int
}

type Message struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Type    string `json:"type"`
	Content string `json:"content"`
	GroupID int    `json:"group_id"`
}

func NewHub() *Hub {
	return &Hub{
		Clients:        make(map[*Client]bool),
		ClientsByUser:  make(map[string]*Client),
		NewRoom:        make(chan *CreateRoomRequest),
		JoinRoom:       make(chan *JoinRoomRequest),
		Room:           make(map[int]*Room),
		Registered:     make(chan *Client),
		Unregistered:   make(chan *Client),
		PrivateMessage: make(chan *PrivateMessage),
		GroupMessage:   make(chan *GroupMessage),
		UseCase:        &usecase.WebSocketUsecase{},
		Shutdown:       make(chan struct{}),
		Mutex:          &sync.Mutex{},
	}
}

func (u *Hub) Run(r *redis.Client) {
	for {
		select {
		case client := <-u.Registered:
			u.Mutex.Lock()
			u.Clients[client] = true
			u.ClientsByUser[client.Username] = client

			log.Printf("%s Is Connected", client.Username)
			log.Printf("Total Connected Users: %d", len(u.Clients))

			rooms, err := u.UseCase.ListUserRooms(client.Username)
			if err != nil {
				log.Printf("Failed to get user rooms: %v", err)
				u.Mutex.Unlock()
				continue
			}
			for _, room := range rooms {
				existingRoom, exists := u.Room[room.ID]
				if !exists {
					existingRoom = &Room{
						ID:      room.ID,
						Name:    room.Name,
						Clients: make(map[*Client]bool),
					}
					u.Room[room.ID] = existingRoom

					members, err := u.UseCase.GetRoomMembers(room.ID)
					if err != nil {
						log.Printf("Failed to get room members: %v", err)
						continue
					}

					for _, member := range members {
						for c := range u.Clients {
							if c.Username == member.Username {
								existingRoom.Clients[c] = true
								break
							}
						}
					}
					log.Printf("total %s member: %d", room.Name, len(members))
				}
				existingRoom.Clients[client] = true
			}
			u.Mutex.Unlock()

			u.broadcastOnlineUsers()

		case client := <-u.Unregistered:
			u.Mutex.Lock()
			if _, ok := u.Clients[client]; ok {
				delete(u.Clients, client)
				delete(u.ClientsByUser, client.Username)
				close(client.Send)
				log.Printf("%s Is Disconnected", client.Username)

				rooms, err := u.UseCase.ListUserRooms(client.Username)
				if err != nil {
					log.Printf("Failed to get user rooms: %v", err)
					u.Mutex.Unlock()
					continue
				}

				for _, room := range rooms {
					existingRoom := u.Room[room.ID]
					existingRoom.Clients[client] = false
					delete(existingRoom.Clients, client)
					log.Printf("Logged Out user %s from room %s", client.Username, room.Name)
				}
				u.broadcastOnlineUsers()
			}
			u.Mutex.Unlock()

		case msg := <-u.GroupMessage:
			if err := u.UseCase.SendGroupMessage(msg.From.Username, msg.Room.ID, string(msg.Content)); err != nil {
				log.Printf("Error sending group message: %v", err)
			}
			for client := range msg.Room.Clients {
				if client == msg.From {
					continue
				}
				select {
				case client.Send <- msg.Content:

				default:
					delete(u.Clients, client)
					delete(u.ClientsByUser, client.Username)
					close(client.Send)
				}
			}

		case msg := <-u.PrivateMessage:
			found := false

			cacheClient := new(Client)
			clientCache, err := r.Get(context.Background(), fmt.Sprintf("cache:user:%s", msg.To)).Result()
			if err == nil {
				if err := json.Unmarshal([]byte(clientCache), cacheClient); err != nil {
					log.Printf("Error unmarshalling cache client: %v", err)
				}
				if msg.To == cacheClient.Username {
					found = true
					select {
					case cacheClient.Send <- msg.Content:
						if msg.From != nil || msg.To != "" {
							log.Println("Sending private message to cache client")
							u.UseCase.SendPrivateMessage(msg.From.Username, msg.To, string(msg.Content))
							receipt := []byte(`{"type":"status","content":"Message delivered to ` + msg.To + `"}`)
							msg.From.Send <- receipt
						}
					default:
						delete(u.Clients, cacheClient)
						close(cacheClient.Send)
					}
				}
			} else {
				for client := range u.Clients {
					if msg.To == client.Username {
						found = true
						select {
						case client.Send <- msg.Content:
							if msg.From != nil || msg.To != "" {
								u.UseCase.SendPrivateMessage(msg.From.Username, msg.To, string(msg.Content))
								receipt := []byte(`{"type":"status","content":"Message delivered to ` + msg.To + `"}`)
								msg.From.Send <- receipt

								parsedClient, err := json.Marshal(client)
								if err != nil {
									log.Printf("Error marshalling client: %v", err)
									continue
								}

								if err := r.Set(context.Background(), fmt.Sprintf("cache:user:%s", client.Username), parsedClient, 15*time.Minute).Err(); err != nil {
									log.Printf("Error setting cache: %v", err)
								}
							}
						default:
							delete(u.Clients, client)
							delete(u.ClientsByUser, client.Username)
							close(client.Send)
						}
						break
					}
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

			if _, err := u.UseCase.CreateRoom(req.Name, req.Creator.Username); err != nil {
				log.Printf("Error creating room: %v", err)
				return
			}

			log.Printf("Room %s created by %s with ID %d", req.Name, req.Creator.Username, ID)

			receipt := []byte(`{"type":"status","content":"Room ` + req.Name + ` created"}`)
			req.Creator.Send <- receipt

		case req := <-u.JoinRoom:
			room, ok := u.Room[req.GroupID]
			if !ok {
				receipt := []byte(`{"type":"status","content":"Room not found"}`)
				req.Client.Send <- receipt
				continue
			}

			if _, ok := room.Clients[req.Client]; ok {
				receipt := []byte(`{"type":"status","content":"You are already in this room"}`)
				req.Client.Send <- receipt
				continue
			}

			room.Clients[req.Client] = true

			u.UseCase.AddRoomMember(room.ID, req.Client.Username)

			receipt := []byte(`{"type":"status","content":"You're joining ` + room.Name + `"}`)
			req.Client.Send <- receipt

			member := len(room.Clients)
			broadcastMsg := []byte(`{"type":"status","content":"` + req.Client.Username + ` Connected. Total Members: ` + strconv.Itoa(member) + `"}`)

			for client := range room.Clients {
				if client == req.Client {
					continue
				}
				select {
				case client.Send <- broadcastMsg:
				default:
					delete(u.Clients, client)
					close(client.Send)
				}
			}

		case <-u.Shutdown:
			for client := range u.Clients {
				close(client.Send)
				delete(u.Clients, client)
				delete(u.ClientsByUser, client.Username)
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

		if message.To == "" && message.Type == "group_chat" {
			message.To = "general"
		}

		if message.Content == "" {
			receipt := []byte(`{"type":"status","content":"Message content is required"}`)
			u.Send <- receipt
			continue
		}

		msg, err = json.Marshal(message)
		if err != nil {
			log.Printf("Error marshalling message: %v", err)
			continue
		}

		if message.Type == "group_chat" {
			if message.GroupID == 0 {
				continue
			}

			room, ok := u.Hub.Room[message.GroupID]
			if !ok || !room.Clients[u] {
				receipt := []byte(`{"type":"status","content":"Group not found"}`)
				u.Send <- receipt
				continue
			}

			u.Hub.GroupMessage <- &GroupMessage{
				From:    u,
				Room:    room,
				Content: msg,
			}
		} else if message.Type == "private_chat" {
			if message.To == "" {
				continue
			}
			u.Hub.PrivateMessage <- &PrivateMessage{
				From:    u,
				To:      message.To,
				Content: msg,
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
		} else if message.Type == "join_room" {
			if message.GroupID == 0 {
				receipt := []byte(`{"type":"status","content":"Group ID is required"}`)
				u.Send <- receipt
				continue
			}
			u.Hub.JoinRoom <- &JoinRoomRequest{
				Client:  u,
				GroupID: message.GroupID,
			}
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
