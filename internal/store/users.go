package store

import (
	"database/sql"
	"fmt"
	"time"
)

type User struct {
	ID        uint64
	Email     string
	Password  string // bcrypt hash
	CreatedAt time.Time
}

type UserStore struct{ db *sql.DB }

// Create 插入新用户，返回 auto-increment id（= UID）
func (s *UserStore) Create(email, passwordHash string) (uint64, error) {
	res, err := s.db.Exec(
		`INSERT INTO users (email, password) VALUES (?, ?)`, email, passwordHash,
	)
	if err != nil {
		return 0, fmt.Errorf("create user: %w", err)
	}
	id, _ := res.LastInsertId()
	return uint64(id), nil
}

// GetByEmail 按邮箱查询用户
func (s *UserStore) GetByEmail(email string) (*User, error) {
	var u User
	err := s.db.QueryRow(
		`SELECT id, email, password, created_at FROM users WHERE email = ?`, email,
	).Scan(&u.ID, &u.Email, &u.Password, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	return &u, err
}
