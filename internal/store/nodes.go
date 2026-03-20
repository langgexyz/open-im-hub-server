package store

import (
	"database/sql"
	"fmt"
	"time"
)

type Node struct {
	ID            uint64
	NodeID        string
	NodePublicKey string
	Name          string
	Avatar        string
	Description   string
	WSAddr        string
	Status        int8
	ExpiresAt     time.Time
	LastHeartbeat *time.Time
	CreatedAt     time.Time
}

type NodeStore struct{ db *sql.DB }

func (s *NodeStore) Insert(n *Node) error {
	_, err := s.db.Exec(
		`INSERT INTO nodes (node_id, node_public_key, name, ws_addr, status, expires_at) VALUES (?,?,?,?,?,?)`,
		n.NodeID, n.NodePublicKey, n.Name, n.WSAddr, n.Status, n.ExpiresAt,
	)
	return err
}

func (s *NodeStore) GetByPublicKey(pubKey string) (*Node, error) {
	var n Node
	var lastHB sql.NullTime
	err := s.db.QueryRow(
		`SELECT id, node_id, node_public_key, name, ws_addr, status, expires_at, last_heartbeat FROM nodes WHERE node_public_key = ?`,
		pubKey,
	).Scan(&n.ID, &n.NodeID, &n.NodePublicKey, &n.Name, &n.WSAddr, &n.Status, &n.ExpiresAt, &lastHB)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("node not found")
	}
	if err != nil {
		return nil, err
	}
	if lastHB.Valid {
		n.LastHeartbeat = &lastHB.Time
	}
	return &n, nil
}

func (s *NodeStore) UpdateHeartbeat(pubKey string) error {
	_, err := s.db.Exec(`UPDATE nodes SET last_heartbeat = NOW() WHERE node_public_key = ?`, pubKey)
	return err
}

func (s *NodeStore) List() ([]*Node, error) {
	rows, err := s.db.Query(
		`SELECT id, node_id, node_public_key, name, avatar, description, ws_addr, status, expires_at, last_heartbeat FROM nodes WHERE status = 1`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var nodes []*Node
	for rows.Next() {
		var n Node
		var avatar, description sql.NullString
		var lastHB sql.NullTime
		if err := rows.Scan(&n.ID, &n.NodeID, &n.NodePublicKey, &n.Name, &avatar, &description, &n.WSAddr, &n.Status, &n.ExpiresAt, &lastHB); err != nil {
			return nil, err
		}
		n.Avatar = avatar.String
		n.Description = description.String
		if lastHB.Valid {
			n.LastHeartbeat = &lastHB.Time
		}
		nodes = append(nodes, &n)
	}
	return nodes, rows.Err()
}
