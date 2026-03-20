package grpcserver

import (
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"github.com/google/uuid"
	hubcrypto "github.com/langgexyz/open-im-hub-server/internal/crypto"
	"github.com/langgexyz/open-im-hub-server/internal/store"
	hubv1 "github.com/langgexyz/open-im-hub-proto/hub/v1"
)

type hubService struct {
	hubv1.UnimplementedHubServiceServer
	store        NodeStore
	hubPrivKey   *ecdsa.PrivateKey
	hubPublicKey string
}

// Activate 节点注册：激活码鉴权（metadata: x-activation-code）
func (s *hubService) Activate(ctx context.Context, req *hubv1.ActivateRequest) (*hubv1.ActivateResponse, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	codesList := md.Get("x-activation-code")
	if len(codesList) == 0 {
		return nil, status.Error(codes.Unauthenticated, "missing x-activation-code")
	}
	if err := s.store.ConsumeCode(codesList[0]); err != nil {
		if errors.Is(err, store.ErrCodeNotFound) || errors.Is(err, store.ErrCodeUsed) {
			return nil, status.Error(codes.Unauthenticated, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	appID := uuid.NewString()
	node := &store.Node{
		AppID:         appID,
		NodePublicKey: req.NodePublicKey,
		Name:          appID,
		WSAddr:        req.NodeWsAddr,
		Status:        1,
		ExpiresAt:     time.Now().Add(365 * 24 * time.Hour),
	}
	if err := s.store.Insert(node); err != nil {
		return nil, status.Error(codes.Internal, "register node: "+err.Error())
	}
	return &hubv1.ActivateResponse{
		AppId:        appID,
		HubPublicKey: s.hubPublicKey,
	}, nil
}

// Heartbeat 节点心跳（需通过 interceptor 节点签名验证）
func (s *hubService) Heartbeat(ctx context.Context, req *hubv1.HeartbeatRequest) (*hubv1.HeartbeatResponse, error) {
	node := ctx.Value(nodeKey{}).(*store.Node)
	if err := s.store.UpdateHeartbeat(node.NodePublicKey); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &hubv1.HeartbeatResponse{Ok: true}, nil
}

// SignSession 验证 user_credential，签发 session_sig
func (s *hubService) SignSession(ctx context.Context, req *hubv1.SignSessionRequest) (*hubv1.SignSessionResponse, error) {
	node := ctx.Value(nodeKey{}).(*store.Node)

	credStr := strings.TrimPrefix(req.UserCredential, "Bearer ")
	appUID, err := verifyCredential(credStr, s.hubPublicKey)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid credential: "+err.Error())
	}

	// session_sig = Sign(keccak256(node_public_key || 0x00 || app_uid || 0x00 || expiry), hub_private_key)
	msg := buildSessionMsg(node.NodePublicKey, appUID, req.Expiry)
	sig, err := hubcrypto.Sign(msg, s.hubPrivKey)
	if err != nil {
		return nil, status.Error(codes.Internal, "sign failed")
	}
	return &hubv1.SignSessionResponse{
		SessionSig: "0x" + hex.EncodeToString(sig),
		AppUid:     appUID,
	}, nil
}

// PushNotify 转发离线推送
func (s *hubService) PushNotify(ctx context.Context, req *hubv1.PushNotifyRequest) (*hubv1.PushNotifyResponse, error) {
	// push 逻辑通过注入的 Pusher 完成，在 Task 6 集成后补充
	return &hubv1.PushNotifyResponse{Ok: true}, nil
}

// --- 内部工具 ---

func verifyCredential(tokenStr, hubPublicKey string) (string, error) {
	parts := strings.SplitN(tokenStr, ".", 2)
	if len(parts) != 2 {
		return "", errors.New("malformed credential")
	}
	payloadB64, sigHex := parts[0], parts[1]
	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return "", errors.New("invalid payload encoding")
	}
	var payload struct {
		AppUID string `json:"app_uid"`
		Exp    int64  `json:"exp"`
	}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return "", errors.New("invalid payload json")
	}
	if time.Now().Unix() > payload.Exp {
		return "", errors.New("credential expired")
	}
	sig, err := hex.DecodeString(sigHex)
	if err != nil || len(sig) != 65 {
		return "", errors.New("invalid signature format")
	}
	recovered, err := hubcrypto.Ecrecover([]byte(payloadB64), sig)
	if err != nil || !strings.EqualFold(recovered, hubPublicKey) {
		return "", errors.New("signature verification failed")
	}
	return payload.AppUID, nil
}

func buildSessionMsg(nodePublicKey, appUID string, expiry int64) []byte {
	var msg []byte
	msg = append(msg, []byte(nodePublicKey)...)
	msg = append(msg, 0x00)
	msg = append(msg, []byte(appUID)...)
	msg = append(msg, 0x00)
	msg = append(msg, []byte(strconv.FormatInt(expiry, 10))...)
	return msg
}
