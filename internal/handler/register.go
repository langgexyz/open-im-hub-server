package handler

import (
	"bytes"
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

	code := parsed.Query().Get("code")
	if len(code) < 16 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing or too short code in node URL"})
		return
	}

	// 探活：向 /node/info 发 GET，避免触发 POST-only 的 /node/activate
	probeURL := fmt.Sprintf("%s://%s/node/info", parsed.Scheme, parsed.Host)
	resp, err := http.Get(probeURL) //nolint:noctx
	if err != nil || resp.StatusCode >= 500 {
		c.JSON(http.StatusBadGateway, gin.H{"error": "node unreachable"})
		return
	}
	resp.Body.Close()

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
	if err := h.store.Nodes.Insert(&store.Node{
		NodeID:        nodeID,
		NodePublicKey: pubKey,
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
	plaintext, _ := json.Marshal(payload)

	aesKey := makeAESKey(code)
	ciphertext, err := crypto.AESEncrypt(aesKey, plaintext)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encryption failed"})
		return
	}

	// POST 加密数据到节点激活端点
	activateURL := fmt.Sprintf("%s://%s/node/activate?code=%s", parsed.Scheme, parsed.Host, code)
	httpResp, err := http.Post(activateURL, "application/octet-stream", bytes.NewReader(ciphertext)) //nolint:noctx
	if err != nil || httpResp.StatusCode != http.StatusOK {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to deliver activation data to node"})
		return
	}
	httpResp.Body.Close()

	c.JSON(http.StatusOK, gin.H{"node_id": nodeID, "message": "node activated successfully"})
}

// makeAESKey 将 code 转换为 32 字节 AES key（截断或右填充 0）
func makeAESKey(code string) []byte {
	key := make([]byte, 32)
	copy(key, []byte(code))
	return key
}
