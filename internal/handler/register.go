package handler

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/langgexyz/open-im-hub-server/internal/crypto"
	"github.com/langgexyz/open-im-hub-server/internal/store"
)

type RegisterHandler struct {
	store      *store.Store
	hubPrivKey string // hex，无 0x 前缀
}

func NewRegisterHandler(s *store.Store, hubPrivKeyHex string) *RegisterHandler {
	return &RegisterHandler{store: s, hubPrivKey: hubPrivKeyHex}
}

type nodeActivatePayload struct {
	NodeID         string `json:"node_id"`
	NodePrivateKey string `json:"node_private_key"`
	NodePublicKey  string `json:"node_public_key"`
	HubPublicKey   string `json:"hub_public_key"`
}

// Register 处理 GET /register?node=<encoded_activate_url>
func (h *RegisterHandler) Register(c *gin.Context) {
	nodeParam := c.Query("node")
	if nodeParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing node parameter"})
		return
	}

	nodeURL, err := url.QueryUnescape(nodeParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid node parameter encoding"})
		return
	}
	parsed, err := url.ParseRequestURI(nodeURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid node URL"})
		return
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported scheme"})
		return
	}

	code := parsed.Query().Get("code")
	if len(code) < 16 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing or too short code in node URL"})
		return
	}

	// 探活：向 /node/info 发 GET，避免触发 POST-only 的 /node/activate
	probeURL := fmt.Sprintf("%s://%s/node/info", parsed.Scheme, parsed.Host)
	resp, err := http.Get(probeURL) //nolint:noctx
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "node unreachable"})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		c.JSON(http.StatusBadGateway, gin.H{"error": "node unreachable"})
		return
	}

	// 生成节点密钥对
	privKey, pubKey, err := crypto.GenerateKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "key generation failed"})
		return
	}
	privKeyHex := crypto.PrivKeyToHex(privKey)

	// 分配 node_id
	nodeID := uuid.New().String()

	// Hub Server 公钥（由 hub_private_key 推导）
	hubPubKey, err := crypto.PubKeyFromPrivHex(h.hubPrivKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hub key error"})
		return
	}

	// 写入 nodes 表
	if err := h.store.Nodes.Upsert(&store.Node{
		AppID:        nodeID,
		AppPublicKey: pubKey,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	// 构造加密 payload
	payload := nodeActivatePayload{
		NodeID:         nodeID,
		NodePrivateKey: privKeyHex,
		NodePublicKey:  pubKey,
		HubPublicKey:   hubPubKey,
	}
	plaintext, err := json.Marshal(payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "payload encoding failed"})
		return
	}

	aesKey := makeAESKey(code)
	ciphertext, err := crypto.AESEncrypt(aesKey, plaintext)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encryption failed"})
		return
	}

	// POST 加密数据到节点激活端点
	activateURL := fmt.Sprintf("%s://%s/node/activate?code=%s", parsed.Scheme, parsed.Host, code)
	httpResp, err := http.Post(activateURL, "application/octet-stream", bytes.NewReader(ciphertext)) //nolint:noctx
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to deliver activation data to node"})
		return
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode != http.StatusOK {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to deliver activation data to node"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"node_id": nodeID, "message": "node activated successfully"})
}

// makeAESKey 对 code 做 SHA-256 派生 32 字节 AES key
func makeAESKey(code string) []byte {
	sum := sha256.Sum256([]byte(code))
	return sum[:]
}
