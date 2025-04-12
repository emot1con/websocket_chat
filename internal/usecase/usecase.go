package usecase

import (
	"time"
	"websocket_try3/internal/domain"
)

type WebSocketUsecase struct {
	userRepo    domain.UserRepository
	messageRepo domain.MessageRepository
	roomRepo    domain.RoomRepository
}

func NewWebSocketUsecase(
	userRepo domain.UserRepository,
	messageRepo domain.MessageRepository,
	roomRepo domain.RoomRepository,
) *WebSocketUsecase {
	return &WebSocketUsecase{
		userRepo:    userRepo,
		messageRepo: messageRepo,
		roomRepo:    roomRepo,
	}
}

// User registration and management
func (u *WebSocketUsecase) RegisterUser(username string) error {
	existingUser, err := u.userRepo.FindByUsername(username)
	if err != nil {
		return err
	}

	now := time.Now()
	user := &domain.User{
		Username:  username,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Jika user sudah ada, update updated_at saja
	if existingUser != nil {
		user.CreatedAt = existingUser.CreatedAt
	}

	return u.userRepo.Save(user)
}

func (u *WebSocketUsecase) GetUser(username string) (*domain.User, error) {
	return u.userRepo.FindByUsername(username)
}

func (u *WebSocketUsecase) ListOnlineUsers() ([]domain.User, error) {
	return u.userRepo.FindAll()
}

// Message handling
func (u *WebSocketUsecase) SendPrivateMessage(sender, recipient, content string) error {
	// Validasi pengirim dan penerima
	if _, err := u.userRepo.FindByUsername(sender); err != nil {
		return err
	}

	if _, err := u.userRepo.FindByUsername(recipient); err != nil {
		return err
	}

	message := &domain.Message{
		From:      sender,
		To:        recipient,
		Content:   content,
		Type:      "private",
		CreatedAt: time.Now(),
	}

	return u.messageRepo.SavePrivateMessage(message)
}

func (u *WebSocketUsecase) SendGroupMessage(sender string, roomID int, content string) error {
	// Validasi pengirim dan room
	if _, err := u.userRepo.FindByUsername(sender); err != nil {
		return err
	}

	if _, err := u.roomRepo.FindRoomByID(roomID); err != nil {
		return err
	}

	message := &domain.Message{
		From:      sender,
		Content:   content,
		Type:      "group",
		GroupID:   roomID,
		CreatedAt: time.Now(),
	}

	return u.messageRepo.SaveGroupMessage(message)
}

func (u *WebSocketUsecase) GetPrivateMessageHistory(user1, user2 string, limit int) ([]domain.Message, error) {
	return u.messageRepo.GetPrivateMessages(user1, user2, limit)
}

func (u *WebSocketUsecase) GetGroupMessageHistory(roomID int, limit int) ([]domain.Message, error) {
	return u.messageRepo.GetGroupMessages(roomID, limit)
}

// Room management
func (u *WebSocketUsecase) CreateRoom(roomName, creator string) (*domain.Room, error) {
	// Validasi creator
	if _, err := u.userRepo.FindByUsername(creator); err != nil {
		return nil, err
	}

	room := &domain.Room{
		Name:      roomName,
		CreatedBy: creator,
		CreatedAt: time.Now(),
	}

	if err := u.roomRepo.SaveRoom(room); err != nil {
		return nil, err
	}

	// Otomatis tambahkan creator sebagai member
	if err := u.AddRoomMember(room.ID, creator); err != nil {
		return nil, err
	}

	return room, nil
}

func (u *WebSocketUsecase) AddRoomMember(roomID int, username string) error {
	// Validasi user dan room
	if _, err := u.userRepo.FindByUsername(username); err != nil {
		return err
	}

	if _, err := u.roomRepo.FindRoomByID(roomID); err != nil {
		return err
	}

	member := &domain.RoomMember{
		RoomID:   roomID,
		Username: username,
		JoinedAt: time.Now(),
	}

	return u.roomRepo.AddMember(member)
}

func (u *WebSocketUsecase) ListAllRooms() ([]*domain.Room, error) {
	return u.roomRepo.GetAllRooms()
}

func (u *WebSocketUsecase) GetRoomInfo(roomID int) (*domain.Room, []domain.RoomMember, error) {
	room, err := u.roomRepo.FindRoomByID(roomID)
	if err != nil {
		return nil, nil, err
	}

	members, err := u.roomRepo.GetRoomMembers(roomID)
	if err != nil {
		return nil, nil, err
	}

	return room, members, nil
}

func (u *WebSocketUsecase) ListUserRooms(username string) ([]domain.Room, error) {
	return u.roomRepo.GetUserRooms(username)
}
