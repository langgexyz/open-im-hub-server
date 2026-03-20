package store

import (
	"database/sql"
	"fmt"
)

type Store struct {
	DB           *sql.DB
	Nodes        *NodeStore
	Codes        *CodeStore
	DeviceTokens *DeviceTokenStore
}

func New(db *sql.DB) (*Store, error) {
	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &Store{
		DB:           db,
		Nodes:        &NodeStore{db: db},
		Codes:        &CodeStore{db: db},
		DeviceTokens: &DeviceTokenStore{db: db},
	}, nil
}

func migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS nodes (
			id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
			app_id          VARCHAR(64) NOT NULL UNIQUE,
			node_public_key VARCHAR(42) NOT NULL UNIQUE,
			name            VARCHAR(128) NOT NULL,
			avatar          VARCHAR(512),
			description     TEXT,
			ws_addr         VARCHAR(512) NOT NULL,
			status          TINYINT DEFAULT 1,
			expires_at      TIMESTAMP NOT NULL,
			last_heartbeat  TIMESTAMP NULL,
			created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS activation_codes (
			id         BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
			code       VARCHAR(64) NOT NULL UNIQUE,
			used       BOOLEAN DEFAULT FALSE,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS device_tokens (
			id         BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
			app_uid    VARCHAR(64) NOT NULL,
			platform   TINYINT NOT NULL,
			token      VARCHAR(256) NOT NULL,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			UNIQUE KEY uk_uid_platform (app_uid, platform)
		)`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}
