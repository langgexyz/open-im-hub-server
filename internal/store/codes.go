package store

import (
	"database/sql"
	"errors"
	"time"
)

var (
	ErrCodeNotFound = errors.New("activation code not found")
	ErrCodeUsed     = errors.New("activation code already used")
	ErrCodeExpired  = errors.New("activation code expired")
)

type CodeStore struct{ db *sql.DB }

func (s *CodeStore) Insert(code string, expiresAt time.Time) error {
	_, err := s.db.Exec(`INSERT INTO activation_codes (code, expires_at) VALUES (?, ?)`, code, expiresAt)
	return err
}

func (s *CodeStore) Consume(code string) error {
	var used bool
	var expiresAt time.Time
	err := s.db.QueryRow(`SELECT used, expires_at FROM activation_codes WHERE code = ?`, code).Scan(&used, &expiresAt)
	if err == sql.ErrNoRows {
		return ErrCodeNotFound
	}
	if err != nil {
		return err
	}
	if used {
		return ErrCodeUsed
	}
	if time.Now().After(expiresAt) {
		return ErrCodeExpired
	}
	res, err := s.db.Exec(`UPDATE activation_codes SET used = TRUE WHERE code = ? AND used = FALSE`, code)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrCodeUsed // 并发场景：另一请求先消费了
	}
	return nil
}
