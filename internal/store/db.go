package store

import (
	"database/sql"
	"fmt"
)

type Store struct {
	DB           *sql.DB
	Nodes        *NodeStore
	Users        *UserStore
	DeviceTokens *DeviceTokenStore
}

func New(db *sql.DB) (*Store, error) {
	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &Store{
		DB:           db,
		Nodes:        &NodeStore{db: db},
		Users:        &UserStore{db: db},
		DeviceTokens: &DeviceTokenStore{db: db},
	}, nil
}

func migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
            id         BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
            email      VARCHAR(255) NOT NULL UNIQUE,
            password   VARCHAR(255) NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )`,
		`CREATE TABLE IF NOT EXISTS nodes (
            id               BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
            app_id           VARCHAR(64)  NOT NULL UNIQUE,
            app_public_key   VARCHAR(42)  NOT NULL UNIQUE,
            name             VARCHAR(128) NOT NULL DEFAULT '',
            avatar           VARCHAR(512),
            description      TEXT,
            node_server_addr VARCHAR(512) NOT NULL DEFAULT '',
            node_web_addr    VARCHAR(512) NOT NULL DEFAULT '',
            admin_uid        VARCHAR(64),
            status           TINYINT DEFAULT 0,
            expires_at       TIMESTAMP NOT NULL DEFAULT (NOW() + INTERVAL 1 YEAR),
            last_heartbeat   TIMESTAMP NULL,
            created_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )`,
		`CREATE TABLE IF NOT EXISTS device_tokens (
            id         BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
            uid        VARCHAR(64) NOT NULL,
            platform   TINYINT NOT NULL,
            token      VARCHAR(256) NOT NULL,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
            UNIQUE KEY uk_uid_platform (uid, platform)
        )`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}
