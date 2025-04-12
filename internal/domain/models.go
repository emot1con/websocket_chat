package domain

import "time"

type User struct {
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Message struct {
	ID        int       `json:"id"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Content   string    `json:"content"`
	Type      string    `json:"type"`
	GroupID   int       `json:"group_id"`
	CreatedAt time.Time `json:"created_at"`
}

type Room struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type RoomMember struct {
	RoomID   int       `json:"room_id"`
	Username string    `json:"username"`
	JoinedAt time.Time `json:"joined_at"`
}
