package repository

import (
	"database/sql"
	"websocket_try3/internal/domain"
)

type RoomRepository struct {
	db *sql.DB
}

func NewRoomRepository(db *sql.DB) *RoomRepository {
	return &RoomRepository{db: db}
}

func (r *RoomRepository) SaveRoom(room *domain.Room) error {
	query := `
		INSERT INTO rooms (name, created_by, created_at)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	err := r.db.QueryRow(
		query,
		room.Name,
		room.CreatedBy,
		room.CreatedAt,
	).Scan(&room.ID)
	return err
}

func (r *RoomRepository) FindRoomByID(id int) (*domain.Room, error) {
	query := `
		SELECT id, name, created_by, created_at
		FROM rooms
		WHERE id = $1
	`
	row := r.db.QueryRow(query, id)

	var room domain.Room
	err := row.Scan(
		&room.ID,
		&room.Name,
		&room.CreatedBy,
		&room.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &room, nil
}

func (r *RoomRepository) AddMember(member *domain.RoomMember) error {
	query := `
		INSERT INTO room_members (room_id, username, joined_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (room_id, username) DO NOTHING
	`
	_, err := r.db.Exec(
		query,
		member.RoomID,
		member.Username,
		member.JoinedAt,
	)
	return err
}

func (r *RoomRepository) GetAllRooms() ([]*domain.Room, error) {
	SQL := "SELECT id, name, created_by, created_at FROM rooms ORDER BY name"
	rows, err := r.db.Query(SQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rooms []*domain.Room
	for rows.Next() {
		room := new(domain.Room)
		err := rows.Scan(
			&room.ID,
			&room.Name,
			&room.CreatedBy,
			&room.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		rooms = append(rooms, room)
	}
	return rooms, nil
}

func (r *RoomRepository) GetRoomMembers(roomID int) ([]domain.RoomMember, error) {
	query := `
		SELECT room_id, username, joined_at
		FROM room_members
		WHERE room_id = $1
		ORDER BY joined_at
	`
	rows, err := r.db.Query(query, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []domain.RoomMember
	for rows.Next() {
		var member domain.RoomMember
		err := rows.Scan(
			&member.RoomID,
			&member.Username,
			&member.JoinedAt,
		)
		if err != nil {
			return nil, err
		}
		members = append(members, member)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return members, nil
}

func (r *RoomRepository) GetUserRooms(username string) ([]domain.Room, error) {
	query := `
		SELECT r.id, r.name, r.created_by, r.created_at
		FROM rooms r
		JOIN room_members rm ON r.id = rm.room_id
		WHERE rm.username = $1
		ORDER BY r.name
	`
	rows, err := r.db.Query(query, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rooms []domain.Room
	for rows.Next() {
		var room domain.Room
		err := rows.Scan(
			&room.ID,
			&room.Name,
			&room.CreatedBy,
			&room.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		rooms = append(rooms, room)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return rooms, nil
}
