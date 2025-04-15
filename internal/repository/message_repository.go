// repository/postgres/message_repository.go
package repository

import (
	"database/sql"
	"websocket_try3/internal/domain"
)

type MessageRepository struct {
	db *sql.DB
}

func NewMessageRepository(db *sql.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) SavePrivateMessage(msg *domain.Message) error {

	query := `
		INSERT INTO messages (
			from_user, to_user, content, type, created_at
		) VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.Exec(
		query,
		msg.From,
		msg.To,
		msg.Content,
		"private",
		msg.CreatedAt,
	)

	if err != nil {
		return err
	}

	return nil
}

func (r *MessageRepository) SaveGroupMessage(msg *domain.Message) error {

	query := `
		INSERT INTO messages (
			from_user, content, type, group_id, created_at
		) VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.Exec(
		query,
		msg.From,
		msg.Content,
		"group",
		msg.GroupID,
		msg.CreatedAt,
	)
	if err != nil {
		return err
	}
	return nil
}

func (r *MessageRepository) GetPrivateMessages(from, to string, limit int) ([]domain.Message, error) {
	query := `
		SELECT id, from_user, to_user, content, type, created_at
		FROM messages
		WHERE type = 'private' AND (
			(from_user = $1 AND to_user = $2) OR 
			(from_user = $2 AND to_user = $1)
		ORDER BY created_at DESC
		LIMIT $3
	`
	rows, err := r.db.Query(query, from, to, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []domain.Message
	for rows.Next() {
		var msg domain.Message
		err := rows.Scan(
			&msg.ID,
			&msg.From,
			&msg.To,
			&msg.Content,
			&msg.Type,
			&msg.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	// Reverse to get chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

func (r *MessageRepository) GetGroupMessages(roomID int, limit int) ([]domain.Message, error) {
	query := `
		SELECT id, from_user, content, type, group_id, created_at
		FROM messages
		WHERE type = 'group' AND group_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	rows, err := r.db.Query(query, roomID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []domain.Message
	for rows.Next() {
		var msg domain.Message
		err := rows.Scan(
			&msg.ID,
			&msg.From,
			&msg.Content,
			&msg.Type,
			&msg.GroupID,
			&msg.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	// Reverse to get chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}
