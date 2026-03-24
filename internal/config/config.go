package config

import (
	"crypto/ecdsa"
	"fmt"
	"os"
	"strconv"

	hubcrypto "github.com/langgexyz/open-im-hub-server/internal/crypto"
)

type Config struct {
	// Hub Server EVM 私钥，用于签发 session_sig 和验证 user_credential
	HubPrivateKey    string
	HubPrivateKeyObj *ecdsa.PrivateKey
	HubPublicKey     string // 从私钥派生，激活时返回给节点

	MySQLDSN     string
	HTTPAddr     string // App 客户端 HTTP 服务，默认 ":8080"
	GRPCAddr     string // 节点 gRPC 服务，默认 ":50051"
	HubGRPCAddr  string // 下发给 Node 的外部 gRPC 地址，如 "hub.example.com:50051"
	HubWebOrigin string // 下发给 Node 的 Web origin，如 "https://hub.example.com"

	// APNs 配置（可选）
	APNsKeyFile  string
	APNsKeyID    string
	APNsTeamID   string
	APNsBundleID string
	APNsSandbox  bool

	// FCM 配置（可选）
	FCMServerKey string
}

func Load() (*Config, error) {
	privateKeyHex := requireEnv("HUB_PRIVATE_KEY")
	privKey, err := hubcrypto.PrivKeyFromHex(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid HUB_PRIVATE_KEY: %w", err)
	}
	pubKey, err := hubcrypto.PrivKeyToAddress(privKey)
	if err != nil {
		return nil, fmt.Errorf("derive public key: %w", err)
	}

	httpAddr := os.Getenv("HUB_HTTP_ADDR")
	if httpAddr == "" {
		httpAddr = ":8080"
	}
	grpcAddr := os.Getenv("HUB_GRPC_ADDR")
	if grpcAddr == "" {
		grpcAddr = ":50051"
	}
	hubGRPCAddr := os.Getenv("HUB_GRPC_EXTERNAL_ADDR")
	hubWebOrigin := os.Getenv("HUB_WEB_ORIGIN")

	sandbox, _ := strconv.ParseBool(os.Getenv("APNS_SANDBOX"))

	return &Config{
		HubPrivateKey:    privateKeyHex,
		HubPrivateKeyObj: privKey,
		HubPublicKey:     pubKey,
		MySQLDSN:         requireEnv("MYSQL_DSN"),
		HTTPAddr:         httpAddr,
		GRPCAddr:         grpcAddr,
		HubGRPCAddr:      hubGRPCAddr,
		HubWebOrigin:     hubWebOrigin,
		APNsKeyFile:      os.Getenv("APNS_KEY_FILE"),
		APNsKeyID:        os.Getenv("APNS_KEY_ID"),
		APNsTeamID:       os.Getenv("APNS_TEAM_ID"),
		APNsBundleID:     os.Getenv("APNS_BUNDLE_ID"),
		APNsSandbox:      sandbox,
		FCMServerKey:     os.Getenv("FCM_SERVER_KEY"),
	}, nil
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required environment variable %s is not set", key))
	}
	return v
}
