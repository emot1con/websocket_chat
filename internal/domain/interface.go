package domain

type UserRepository interface {
	Save(user *User) error
	FindByUsername(username string) (*User, error)
	FindAll() ([]User, error)
}

type MessageRepository interface {
	SavePrivateMessage(msg *Message) error
	SaveGroupMessage(msg *Message) error
	GetPrivateMessages(from, to string, limit int) ([]Message, error)
	GetGroupMessages(roomID int, limit int) ([]Message, error)
}

type RoomRepository interface {
	SaveRoom(room *Room) error
	FindRoomByID(id int) (*Room, error)
	AddMember(member *RoomMember) error
	GetAllRooms() ([]*Room, error)
	GetRoomMembers(roomID int) ([]RoomMember, error)
	GetUserRooms(username string) ([]Room, error)
}
