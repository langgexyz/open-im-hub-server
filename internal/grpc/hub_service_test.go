package grpcserver_test

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	hubcrypto "github.com/langgexyz/open-im-hub-server/internal/crypto"
	grpcserver "github.com/langgexyz/open-im-hub-server/internal/grpc"
	"github.com/langgexyz/open-im-hub-server/internal/store"
	hubv1 "github.com/langgexyz/open-im-hub-proto/hub/v1"
)

// 确认新 proto 方法存在
var _ = hubv1.UpdateNodeProfileRequest{}

// mockStore 最小化 store mock，供测试使用
type mockStore struct {
	nodes map[string]*store.Node
}

func newMockStore() *mockStore {
	return &mockStore{
		nodes: map[string]*store.Node{},
	}
}

func (m *mockStore) GetByPublicKey(k string) (*store.Node, error) {
	n, ok := m.nodes[k]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return n, nil
}
func (m *mockStore) Upsert(n *store.Node) error         { m.nodes[n.AppPublicKey] = n; return nil }
func (m *mockStore) UpdateHeartbeat(k string) error     { return nil }
func (m *mockStore) List() ([]*store.Node, error)       { return nil, nil }
func (m *mockStore) GetDeviceTokens(appUIDs []string) (map[string][]store.DeviceToken, error) {
	return nil, nil
}
func buildMsg(parts ...[]byte) []byte {
	var msg []byte
	for i, p := range parts {
		msg = append(msg, p...)
		if i < len(parts)-1 {
			msg = append(msg, 0x00)
		}
	}
	return msg
}

func TestUpdateNodeProfile(t *testing.T) {
	t.Skip("TODO: implement TestUpdateNodeProfile once UpdateNodeProfile is wired in hub_service.go")
	ms := newMockStore()
	nodePriv, nodePub, err := hubcrypto.GenerateKey()
	require.NoError(t, err)
	_ = nodePriv
	ms.nodes[nodePub] = &store.Node{
		AppPublicKey: nodePub, Status: 1, ExpiresAt: time.Now().Add(time.Hour),
	}

	hubPriv, hubPub, _ := hubcrypto.GenerateKey()
	srv := grpcserver.New(ms, hubPriv, hubPub)
	lis, _ := net.Listen("tcp", "127.0.0.1:0")
	go srv.Serve(lis)
	defer srv.Stop()

	conn, _ := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	defer conn.Close()
	client := hubv1.NewHubServiceClient(conn)

	// TODO: build signed metadata and call svc.UpdateNodeProfile(...)
	_ = client
	req := &hubv1.UpdateNodeProfileRequest{}
	_ = req
}

func TestHeartbeat(t *testing.T) {
	ms := newMockStore()
	nodePriv, nodePub, err := hubcrypto.GenerateKey()
	require.NoError(t, err)
	ms.nodes[nodePub] = &store.Node{
		AppPublicKey: nodePub, Status: 1, ExpiresAt: time.Now().Add(time.Hour),
	}

	hubPriv, hubPub, _ := hubcrypto.GenerateKey()
	srv := grpcserver.New(ms, hubPriv, hubPub)
	lis, _ := net.Listen("tcp", "127.0.0.1:0")
	go srv.Serve(lis)
	defer srv.Stop()

	conn, _ := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	defer conn.Close()
	client := hubv1.NewHubServiceClient(conn)

	req := &hubv1.HeartbeatRequest{NodePublicKey: nodePub, WsAddr: "wss://test.example.com"}
	reqBytes, _ := proto.Marshal(req)

	method := "/hub.v1.HubService/Heartbeat"
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	bodyHash := hubcrypto.Keccak256(reqBytes)
	msg := buildMsg([]byte(method), bodyHash, []byte(timestamp))
	sig, _ := hubcrypto.Sign(msg, nodePriv)

	md := metadata.Pairs(
		"x-node-public-key", nodePub,
		"x-node-timestamp", timestamp,
		"x-node-body-hash", hex.EncodeToString(bodyHash),
		"x-node-sig", hex.EncodeToString(sig),
	)
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	resp, err := client.Heartbeat(ctx, req)
	require.NoError(t, err)
	require.True(t, resp.Ok)
}

func TestSignSession(t *testing.T) {
	ms := newMockStore()
	nodePriv, nodePub, _ := hubcrypto.GenerateKey()
	ms.nodes[nodePub] = &store.Node{
		AppPublicKey: nodePub, Status: 1, ExpiresAt: time.Now().Add(time.Hour),
	}

	hubPriv, hubPub, _ := hubcrypto.GenerateKey()
	srv := grpcserver.New(ms, hubPriv, hubPub)
	lis, _ := net.Listen("tcp", "127.0.0.1:0")
	go srv.Serve(lis)
	defer srv.Stop()

	conn, _ := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	defer conn.Close()
	client := hubv1.NewHubServiceClient(conn)

	// 构造有效 user_credential（由 hub 私钥签名）
	expiry := time.Now().Add(time.Hour).Unix()
	payload := fmt.Sprintf(`{"app_uid":"user_abc","exp":%d}`, expiry)
	b64 := base64.RawURLEncoding.EncodeToString([]byte(payload))
	credSig, _ := hubcrypto.Sign([]byte(b64), hubPriv)
	credential := b64 + "." + hex.EncodeToString(credSig)

	req := &hubv1.SignSessionRequest{UserCredential: "Bearer " + credential, Expiry: expiry}
	reqBytes, _ := proto.Marshal(req)

	method := "/hub.v1.HubService/SignSession"
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	bodyHash := hubcrypto.Keccak256(reqBytes)
	msg := buildMsg([]byte(method), bodyHash, []byte(timestamp))
	sig, _ := hubcrypto.Sign(msg, nodePriv)

	md := metadata.Pairs(
		"x-node-public-key", nodePub,
		"x-node-timestamp", timestamp,
		"x-node-body-hash", hex.EncodeToString(bodyHash),
		"x-node-sig", hex.EncodeToString(sig),
	)
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	resp, err := client.SignSession(ctx, req)
	require.NoError(t, err)
	require.Equal(t, "user_abc", resp.AppUid)
	require.NotEmpty(t, resp.SessionSig)
}
