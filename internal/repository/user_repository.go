package repository

import (
	"database/sql"
	"websocket_try3/internal/domain"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Save(user *domain.User) error {
	query := `
		INSERT INTO users (username, created_at, updated_at) 
		VALUES ($1, $2, $3)
		ON CONFLICT (username) DO UPDATE 
		SET updated_at = $3
	`
	_, err := r.db.Exec(
		query,
		user.Username,
		user.CreatedAt,
		user.UpdatedAt,
	)
	return err
}

func (r *UserRepository) FindByUsername(username string) (*domain.User, error) {
	query := `
		SELECT username, created_at, updated_at 
		FROM users 
		WHERE username = $1
	`
	row := r.db.QueryRow(query, username)

	var user domain.User
	err := row.Scan(
		&user.Username,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) FindAll() ([]domain.User, error) {
	query := `
		SELECT username, created_at, updated_at 
		FROM users
		ORDER BY username
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var user domain.User
		err := rows.Scan(
			&user.Username,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}
