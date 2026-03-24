package grpcserver

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"log"
	"strconv"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	hubauth "github.com/langgexyz/open-im-hub-server/internal/auth"
	hubcrypto "github.com/langgexyz/open-im-hub-server/internal/crypto"
	"github.com/langgexyz/open-im-hub-server/internal/push"
	"github.com/langgexyz/open-im-hub-server/internal/store"
	hubv1 "github.com/langgexyz/open-im-hub-proto/hub/v1"
)

type hubService struct {
	hubv1.UnimplementedHubServiceServer
	store         NodeStore
	hubPrivKey    *ecdsa.PrivateKey
	hubPublicKey  string
	iosPusher     push.Pusher
	androidPusher push.Pusher
}

// Heartbeat 节点心跳（需通过 interceptor 节点签名验证）
func (s *hubService) Heartbeat(ctx context.Context, req *hubv1.HeartbeatRequest) (*hubv1.HeartbeatResponse, error) {
	node := ctx.Value(nodeKey{}).(*store.Node)
	if err := s.store.UpdateHeartbeat(node.AppPublicKey); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &hubv1.HeartbeatResponse{Ok: true}, nil
}

// SignSession 验证 user_credential，签发 session_sig
func (s *hubService) SignSession(ctx context.Context, req *hubv1.SignSessionRequest) (*hubv1.SignSessionResponse, error) {
	node := ctx.Value(nodeKey{}).(*store.Node)

	credStr := strings.TrimPrefix(req.UserCredential, "Bearer ")

	uid, appID, err := hubauth.VerifyCredential(credStr, s.hubPublicKey)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid credential: "+err.Error())
	}
	// 验证 credential 绑定的 app_id 与本节点匹配（防跨节点重放）
	if !strings.EqualFold(appID, node.AppID) {
		return nil, status.Error(codes.Unauthenticated, "credential app_id mismatch")
	}

	msg := buildSessionMsg(node.AppPublicKey, uid, req.Expiry)
	sig, err := hubcrypto.Sign(msg, s.hubPrivKey)
	if err != nil {
		return nil, status.Error(codes.Internal, "sign failed")
	}
	return &hubv1.SignSessionResponse{
		SessionSig: "0x" + hex.EncodeToString(sig),
		AppUid:     uid,
	}, nil
}

// UpdateNodeProfile — Node Server calls this after /node/init to update Hub directory profile
func (s *hubService) UpdateNodeProfile(ctx context.Context, req *hubv1.UpdateNodeProfileRequest) (*hubv1.UpdateNodeProfileResponse, error) {
	if err := s.store.UpdateProfile(req.AppId, req.Name, req.Avatar, req.Description); err != nil {
		return nil, status.Error(codes.Internal, "update profile: "+err.Error())
	}
	return &hubv1.UpdateNodeProfileResponse{Ok: true}, nil
}

// PushNotify 转发离线推送（APNs/FCM）
func (s *hubService) PushNotify(ctx context.Context, req *hubv1.PushNotifyRequest) (*hubv1.PushNotifyResponse, error) {
	if len(req.AppUids) == 0 {
		return &hubv1.PushNotifyResponse{Ok: true}, nil
	}

	tokenMap, err := s.store.GetDeviceTokens(req.AppUids)
	if err != nil {
		log.Printf("PushNotify: get device tokens error: %v", err)
		return nil, status.Error(codes.Internal, "get device tokens")
	}

	var dataMap map[string]any
	if req.DataJson != "" {
		_ = json.Unmarshal([]byte(req.DataJson), &dataMap)
	}

	sent, failed, skipped := 0, 0, 0
	for _, tokens := range tokenMap {
		for _, dt := range tokens {
			var p push.Pusher
			switch dt.Platform {
			case push.PlatformIOS:
				p = s.iosPusher
			case push.PlatformAndroid:
				p = s.androidPusher
			default:
				log.Printf("PushNotify: unknown platform %d uid=%s", dt.Platform, dt.AppUID)
				skipped++
				continue
			}
			msg := push.Message{
				Token:    dt.Token,
				Platform: dt.Platform,
				Title:    req.Title,
				Body:     req.Body,
				Data:     dataMap,
			}
			if err := p.Send(ctx, msg); err != nil {
				log.Printf("PushNotify: send failed uid=%s platform=%d token=%.12s... err=%v",
					dt.AppUID, dt.Platform, dt.Token, err)
				failed++
			} else {
				log.Printf("PushNotify: sent uid=%s platform=%d token=%.12s...",
					dt.AppUID, dt.Platform, dt.Token)
				sent++
			}
		}
	}
	log.Printf("PushNotify: requested=%d tokens_found=%d sent=%d failed=%d skipped=%d",
		len(req.AppUids), len(tokenMap), sent, failed, skipped)

	return &hubv1.PushNotifyResponse{Ok: true}, nil
}

// --- 内部工具 ---


func buildSessionMsg(nodePublicKey, appUID string, expiry int64) []byte {
	var msg []byte
	msg = append(msg, []byte(nodePublicKey)...)
	msg = append(msg, 0x00)
	msg = append(msg, []byte(appUID)...)
	msg = append(msg, 0x00)
	msg = append(msg, []byte(strconv.FormatInt(expiry, 10))...)
	return msg
}
