package handler

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	hubauth "github.com/langgexyz/open-im-hub-server/internal/auth"
	hubcrypto "github.com/langgexyz/open-im-hub-server/internal/crypto"
	"github.com/langgexyz/open-im-hub-server/internal/store"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

// ActivateHandler handles idempotent node activation via POST /node/activate.
type ActivateHandler struct {
	nodes        *store.NodeStore
	hubPrivKey   *ecdsa.PrivateKey
	hubGRPCAddr  string // delivered to Node, e.g. "hub.example.com:50051"
	hubWebOrigin string // delivered to Node, e.g. "https://hub.example.com"
	hubPublicKey string
}

type activatePayload struct {
	AppID         string `json:"app_id"`
	AppPrivateKey string `json:"app_private_key"`
	AppPublicKey  string `json:"app_public_key"`
	HubGRPCAddr   string `json:"hub_grpc_addr"`
	HubPublicKey  string `json:"hub_public_key"`
	HubWebOrigin  string `json:"hub_web_origin"`
}

func NewActivateHandler(nodes *store.NodeStore, hubPrivKeyHex, hubGRPCAddr, hubWebOrigin string) *ActivateHandler {
	priv, err := hubcrypto.PrivKeyFromHex(hubPrivKeyHex)
	if err != nil {
		panic("invalid hub private key: " + err.Error())
	}
	pub, err := hubcrypto.PubKeyFromPrivHex(hubPrivKeyHex)
	if err != nil {
		panic("cannot derive hub public key: " + err.Error())
	}
	return &ActivateHandler{
		nodes:        nodes,
		hubPrivKey:   priv,
		hubGRPCAddr:  hubGRPCAddr,
		hubWebOrigin: hubWebOrigin,
		hubPublicKey: pub,
	}
}

// Activate handles POST /node/activate { code, node_server_addr, node_web_addr }.
// Requires JWT middleware to have injected uid (admin_uid) into context.
// The operation is idempotent: calling it again with the same code returns 200.
func (h *ActivateHandler) Activate(c *gin.Context) {
	var req struct {
		Code           string `json:"code"             binding:"required,len=64"`
		NodeServerAddr string `json:"node_server_addr" binding:"required"`
		NodeWebAddr    string `json:"node_web_addr"    binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	adminUID := c.GetString(hubauth.ContextUID)
	if adminUID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing uid"})
		return
	}

	// 1. Probe node liveness via GET /node/info
	infoURL := req.NodeServerAddr + "/node/info"
	resp, err := httpClient.Get(infoURL) //nolint:noctx
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "node unreachable"})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		c.JSON(http.StatusBadGateway, gin.H{"error": "node unreachable"})
		return
	}

	// 1b. Check if already activated (idempotent return)
	existing, err := h.nodes.GetByAppID(req.Code)
	if err == nil && existing.Status == 1 {
		c.JSON(http.StatusOK, gin.H{"app_id": req.Code, "message": "node already activated"})
		return
	}

	// 2. Generate a fresh node key pair (address used as AppPublicKey per existing convention)
	nodePriv, nodePub, err := hubcrypto.GenerateKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "key generation failed"})
		return
	}
	nodePrivHex := hubcrypto.PrivKeyToHex(nodePriv)

	// 3. UPSERT into nodes table (idempotent; code == AppID)
	node := &store.Node{
		AppID:          req.Code,
		AppPublicKey:   nodePub,
		NodeServerAddr: req.NodeServerAddr,
		NodeWebAddr:    req.NodeWebAddr,
		AdminUID:       adminUID,
		Status:         0,
	}
	if err := h.nodes.Upsert(node); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error: " + err.Error()})
		return
	}

	// 4. Build AES-encrypted activation payload and POST it to the Node Server
	payload := activatePayload{
		AppID:         req.Code,
		AppPrivateKey: nodePrivHex,
		AppPublicKey:  nodePub,
		HubGRPCAddr:   h.hubGRPCAddr,
		HubPublicKey:  h.hubPublicKey,
		HubWebOrigin:  h.hubWebOrigin,
	}
	plaintext, err := json.Marshal(payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "marshal payload failed"})
		return
	}
	aesKey := deriveAESKey(req.Code)
	ciphertext, err := hubcrypto.AESEncrypt(aesKey, plaintext)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encryption failed"})
		return
	}

	activateURL := fmt.Sprintf("%s/node/activate?code=%s", req.NodeServerAddr, req.Code)
	httpResp, err := httpClient.Post(activateURL, "application/octet-stream", bytes.NewReader(ciphertext)) //nolint:noctx
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to activate node"})
		return
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode != http.StatusOK {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to activate node"})
		return
	}

	// 5. Mark node as active (status=1). Ignore ErrNodeNotFound — means it was
	// already activated (idempotent second call where status was already 1).
	if err := h.nodes.Activate(req.Code); err != nil && !errors.Is(err, store.ErrNodeNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "activate status update failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"app_id": req.Code, "message": "node activated successfully"})
}

// deriveAESKey derives a 32-byte AES key from the activation code via SHA-256.
func deriveAESKey(code string) []byte {
	sum := sha256.Sum256([]byte(code))
	return sum[:]
}
