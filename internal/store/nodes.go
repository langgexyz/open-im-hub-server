package store

import (
	"database/sql"
	"errors"
	"time"
)

// ErrNodeNotFound is returned when a node lookup finds no matching row.
var ErrNodeNotFound = errors.New("node not found")

type Node struct {
	ID             uint64
	AppID          string
	AppPublicKey   string
	Name           string
	Avatar         string
	Description    string
	NodeServerAddr string
	NodeWebAddr    string
	AdminUID       string
	Status         int8
	ExpiresAt      time.Time
	LastHeartbeat  *time.Time
	CreatedAt      time.Time
}

type NodeStore struct{ db *sql.DB }

// Upsert 以 app_id 为唯一键做幂等写入（INSERT ... ON DUPLICATE KEY UPDATE）
// status=0（pending），激活成功后调用 Activate 改为 status=1
// Note: expires_at is always set to now+1year regardless of n.ExpiresAt;
// this is intentional — expiry is managed server-side, not caller-supplied.
func (s *NodeStore) Upsert(n *Node) error {
	_, err := s.db.Exec(`
        INSERT INTO nodes (app_id, app_public_key, node_server_addr, node_web_addr, admin_uid, status, expires_at)
        VALUES (?, ?, ?, ?, ?, ?, ?)
        ON DUPLICATE KEY UPDATE
            app_public_key   = VALUES(app_public_key),
            node_server_addr = VALUES(node_server_addr),
            node_web_addr    = VALUES(node_web_addr),
            admin_uid        = VALUES(admin_uid)`,
		n.AppID, n.AppPublicKey, n.NodeServerAddr, n.NodeWebAddr, n.AdminUID,
		n.Status, time.Now().Add(365*24*time.Hour),
	)
	return err
}

// Activate 将节点标记为 status=1（active）
func (s *NodeStore) Activate(appID string) error {
	res, err := s.db.Exec(`UPDATE nodes SET status = 1 WHERE app_id = ?`, appID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNodeNotFound
	}
	return nil
}

// UpdateProfile 更新 Hub 目录中的节点资料（由 UpdateNodeProfile gRPC 调用）
func (s *NodeStore) UpdateProfile(appID, name, avatar, description string) error {
	_, err := s.db.Exec(
		`UPDATE nodes SET name = ?, avatar = ?, description = ? WHERE app_id = ?`,
		name, avatar, description, appID,
	)
	return err
}

// GetByAppID 按 app_id 查询节点
func (s *NodeStore) GetByAppID(appID string) (*Node, error) {
	var n Node
	var avatar, description, adminUID sql.NullString
	var lastHB sql.NullTime
	err := s.db.QueryRow(`
        SELECT id, app_id, app_public_key, name, avatar, description,
               node_server_addr, node_web_addr, admin_uid,
               status, expires_at, last_heartbeat
        FROM nodes WHERE app_id = ?`, appID,
	).Scan(&n.ID, &n.AppID, &n.AppPublicKey, &n.Name,
		&avatar, &description, &n.NodeServerAddr, &n.NodeWebAddr,
		&adminUID, &n.Status, &n.ExpiresAt, &lastHB)
	if err == sql.ErrNoRows {
		return nil, ErrNodeNotFound
	}
	if err != nil {
		return nil, err
	}
	n.Avatar = avatar.String
	n.Description = description.String
	n.AdminUID = adminUID.String
	if lastHB.Valid {
		n.LastHeartbeat = &lastHB.Time
	}
	return &n, nil
}

// GetByPublicKey 按 app_public_key 查询（gRPC 拦截器使用）
// Note: intentionally scans only the fields needed for authentication;
// profile fields (name, avatar, description) are omitted for fast-path lookup.
func (s *NodeStore) GetByPublicKey(pubKey string) (*Node, error) {
	var n Node
	var lastHB sql.NullTime
	err := s.db.QueryRow(`
        SELECT id, app_id, app_public_key, node_server_addr, status, expires_at, last_heartbeat
        FROM nodes WHERE app_public_key = ?`, pubKey,
	).Scan(&n.ID, &n.AppID, &n.AppPublicKey, &n.NodeServerAddr, &n.Status, &n.ExpiresAt, &lastHB)
	if err == sql.ErrNoRows {
		return nil, ErrNodeNotFound
	}
	if err != nil {
		return nil, err
	}
	if lastHB.Valid {
		n.LastHeartbeat = &lastHB.Time
	}
	return &n, nil
}

// UpdateHeartbeat 更新节点心跳时间
func (s *NodeStore) UpdateHeartbeat(pubKey string) error {
	_, err := s.db.Exec(`UPDATE nodes SET last_heartbeat = NOW() WHERE app_public_key = ?`, pubKey)
	return err
}

// List 返回所有 status=1 的节点（节点广场）
func (s *NodeStore) List() ([]*Node, error) {
	rows, err := s.db.Query(`
        SELECT id, app_id, app_public_key, name, avatar, description,
               node_server_addr, node_web_addr, admin_uid
        FROM nodes WHERE status = 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var nodes []*Node
	for rows.Next() {
		var n Node
		var avatar, description, adminUID sql.NullString
		if err := rows.Scan(&n.ID, &n.AppID, &n.AppPublicKey, &n.Name,
			&avatar, &description, &n.NodeServerAddr, &n.NodeWebAddr, &adminUID); err != nil {
			return nil, err
		}
		n.Avatar = avatar.String
		n.Description = description.String
		n.AdminUID = adminUID.String
		nodes = append(nodes, &n)
	}
	return nodes, rows.Err()
}
