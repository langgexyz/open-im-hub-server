package grpcserver

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	hubcrypto "github.com/langgexyz/open-im-hub-server/internal/crypto"
	"github.com/langgexyz/open-im-hub-server/internal/store"
	hubv1 "github.com/langgexyz/open-im-hub-proto/hub/v1"
)

type nodeKey struct{}

// NodeStore 是 server 依赖的 store 接口（便于测试 mock）
type NodeStore interface {
	GetByPublicKey(pubKey string) (*store.Node, error)
	Insert(n *store.Node) error
	UpdateHeartbeat(pubKey string) error
	List() ([]*store.Node, error)
}

// New 创建 gRPC server，包含节点签名验证 interceptor
func New(s NodeStore, hubPrivKey *ecdsa.PrivateKey, hubPublicKey string) *grpc.Server {
	srv := grpc.NewServer(
		grpc.UnaryInterceptor(nodeAuthInterceptor(s)),
	)
	hubv1.RegisterHubServiceServer(srv, &hubService{
		store:        s,
		hubPrivKey:   hubPrivKey,
		hubPublicKey: hubPublicKey,
	})
	return srv
}

func nodeAuthInterceptor(nodes NodeStore) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}
		get := func(key string) string {
			if vals := md.Get(key); len(vals) > 0 {
				return vals[0]
			}
			return ""
		}

		nodePublicKey := get("x-node-public-key")
		timestamp := get("x-node-timestamp")
		bodyHashHex := get("x-node-body-hash")
		sigHex := get("x-node-sig")

		if nodePublicKey == "" || timestamp == "" || bodyHashHex == "" || sigHex == "" {
			return nil, status.Error(codes.Unauthenticated, "missing auth metadata")
		}

		ts, err := strconv.ParseInt(timestamp, 10, 64)
		if err != nil || absInt(time.Now().Unix()-ts) > 60 {
			return nil, status.Error(codes.Unauthenticated, "stale timestamp")
		}

		sig, err := hex.DecodeString(sigHex)
		if err != nil || len(sig) != 65 {
			return nil, status.Error(codes.Unauthenticated, "invalid sig format")
		}
		bodyHash, err := hex.DecodeString(bodyHashHex)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid body hash")
		}

		msg := buildMsg([]byte(info.FullMethod), bodyHash, []byte(timestamp))
		recovered, err := hubcrypto.Ecrecover(msg, sig)
		if err != nil || !strings.EqualFold(recovered, nodePublicKey) {
			return nil, status.Error(codes.Unauthenticated, "invalid signature")
		}

		node, err := nodes.GetByPublicKey(nodePublicKey)
		if err != nil {
			return nil, status.Error(codes.PermissionDenied, "node not found")
		}
		if node.Status != 1 || time.Now().After(node.ExpiresAt) {
			return nil, status.Error(codes.PermissionDenied, "node not authorized")
		}

		ctx = context.WithValue(ctx, nodeKey{}, node)
		return handler(ctx, req)
	}
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

func absInt(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
