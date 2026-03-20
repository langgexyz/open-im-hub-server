package store_test

import (
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
	"github.com/langgexyz/open-im-hub-server/internal/store"
)

func testStore(t *testing.T) *store.Store {
	t.Helper()
	dsn := os.Getenv("TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("TEST_MYSQL_DSN not set")
	}
	db, err := sql.Open("mysql", dsn)
	require.NoError(t, err)
	s, err := store.New(db)
	require.NoError(t, err)
	t.Cleanup(func() {
		db.Exec("TRUNCATE TABLE nodes")
		db.Exec("TRUNCATE TABLE activation_codes")
		db.Exec("TRUNCATE TABLE device_tokens")
		db.Close()
	})
	return s
}

func TestActivationCode(t *testing.T) {
	s := testStore(t)

	err := s.Codes.Insert("CODE123", time.Now().Add(time.Hour))
	require.NoError(t, err)

	err = s.Codes.Consume("CODE123")
	require.NoError(t, err)

	err = s.Codes.Consume("CODE123")
	require.ErrorIs(t, err, store.ErrCodeUsed)

	err = s.Codes.Consume("NOTEXIST")
	require.ErrorIs(t, err, store.ErrCodeNotFound)
}

func TestNodeCRUD(t *testing.T) {
	s := testStore(t)

	node := &store.Node{
		NodeID:        "app-001",
		NodePublicKey: "0xabc",
		Name:          "Test Node",
		WSAddr:        "wss://test.example.com",
		Status:        1,
		ExpiresAt:     time.Now().Add(365 * 24 * time.Hour),
	}
	require.NoError(t, s.Nodes.Insert(node))

	found, err := s.Nodes.GetByPublicKey("0xabc")
	require.NoError(t, err)
	require.Equal(t, "app-001", found.NodeID)

	require.NoError(t, s.Nodes.UpdateHeartbeat("0xabc"))

	nodes, err := s.Nodes.List()
	require.NoError(t, err)
	require.Len(t, nodes, 1)
}

func TestDeviceTokens(t *testing.T) {
	s := testStore(t)

	require.NoError(t, s.DeviceTokens.Upsert("uid_aaa", 1, "token_ios_aaa"))
	require.NoError(t, s.DeviceTokens.Upsert("uid_aaa", 1, "token_ios_aaa_v2"))

	tokens, err := s.DeviceTokens.GetByUIDs([]string{"uid_aaa", "uid_bbb"})
	require.NoError(t, err)
	require.Len(t, tokens, 1)
	require.Equal(t, "token_ios_aaa_v2", tokens["uid_aaa"][0].Token)
}
