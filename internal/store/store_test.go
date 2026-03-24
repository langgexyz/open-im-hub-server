package store_test

import (
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/langgexyz/open-im-hub-server/internal/store"
	"github.com/stretchr/testify/require"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("TEST_MYSQL_DSN not set")
	}
	db, err := sql.Open("mysql", dsn)
	require.NoError(t, err)
	return db
}

func testStore(t *testing.T) (*store.Store, *sql.DB) {
	t.Helper()
	db := openTestDB(t)
	s, err := store.New(db)
	require.NoError(t, err)
	t.Cleanup(func() {
		db.Exec("TRUNCATE TABLE nodes")
		db.Exec("TRUNCATE TABLE users")
		db.Exec("TRUNCATE TABLE device_tokens")
		db.Close()
	})
	return s, db
}

func TestNodeStoreUpsert(t *testing.T) {
	db := openTestDB(t)
	s, _ := store.New(db)
	t.Cleanup(func() {
		db.Exec("TRUNCATE TABLE nodes")
		db.Close()
	})

	node := &store.Node{
		AppID:          "code-abc123",
		AppPublicKey:   "0xDEAD",
		NodeServerAddr: "http://node:8080",
		NodeWebAddr:    "http://node.example.com",
		AdminUID:       "10001",
		Status:         0,
	}
	err := s.Nodes.Upsert(node)
	require.NoError(t, err)

	// 幂等重试
	err = s.Nodes.Upsert(node)
	require.NoError(t, err)

	// 查询
	n, err := s.Nodes.GetByAppID("code-abc123")
	require.NoError(t, err)
	require.Equal(t, "0xDEAD", n.AppPublicKey)
}

func TestNodeStoreUpdateProfile(t *testing.T) {
	db := openTestDB(t)
	s, _ := store.New(db)
	t.Cleanup(func() {
		db.Exec("TRUNCATE TABLE nodes")
		db.Close()
	})

	// 先插入
	_ = s.Nodes.Upsert(&store.Node{AppID: "n1", AppPublicKey: "0xABC", Status: 0})
	// 更新资料
	err := s.Nodes.UpdateProfile("n1", "科技快讯", "http://avatar.png", "科技资讯公众号")
	require.NoError(t, err)

	n, _ := s.Nodes.GetByAppID("n1")
	require.Equal(t, "科技快讯", n.Name)
}

func TestDeviceTokens(t *testing.T) {
	s, _ := testStore(t)

	require.NoError(t, s.DeviceTokens.Upsert("uid_aaa", 1, "token_ios_aaa"))
	require.NoError(t, s.DeviceTokens.Upsert("uid_aaa", 1, "token_ios_aaa_v2"))

	tokens, err := s.DeviceTokens.GetByUIDs([]string{"uid_aaa", "uid_bbb"})
	require.NoError(t, err)
	require.Len(t, tokens, 1)
	require.Equal(t, "token_ios_aaa_v2", tokens["uid_aaa"][0].Token)
}

func TestUsersTableExists(t *testing.T) {
	db := openTestDB(t)
	s, err := store.New(db)
	require.NoError(t, err)
	_ = s
	t.Cleanup(func() {
		db.Exec("TRUNCATE TABLE users")
		db.Close()
	})
	// 验证 users 表存在且能插入
	_, err = db.Exec(`INSERT INTO users (email, password) VALUES (?,?)`, "test@example.com", "hash")
	require.NoError(t, err)
}

func TestNodesNewSchema(t *testing.T) {
	db := openTestDB(t)
	s, err := store.New(db)
	require.NoError(t, err)
	_ = s
	t.Cleanup(func() {
		db.Exec("TRUNCATE TABLE nodes")
		db.Close()
	})
	// 验证 nodes 新字段存在
	_, err = db.Exec(`INSERT INTO nodes (app_id, app_public_key, node_server_addr, node_web_addr, admin_uid, status, expires_at)
        VALUES (?,?,?,?,?,?,?)`, "testid", "0xabc", "http://node:8080", "http://node.example.com", "uid1", 0, time.Now().Add(time.Hour))
	require.NoError(t, err)
}
