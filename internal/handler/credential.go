package handler

import (
	"crypto/ecdsa"
	"net/http"

	"github.com/gin-gonic/gin"
	hubauth "github.com/langgexyz/open-im-hub-server/internal/auth"
	hubcrypto "github.com/langgexyz/open-im-hub-server/internal/crypto"
)

const credentialTTL = 300 // 5 分钟，足够完成订阅流程

type CredentialHandler struct {
	hubPrivKey *ecdsa.PrivateKey
}

func NewCredentialHandler(hubPrivKeyHex string) *CredentialHandler {
	priv, err := hubcrypto.PrivKeyFromHex(hubPrivKeyHex)
	if err != nil {
		panic("invalid hub private key: " + err.Error())
	}
	return &CredentialHandler{hubPrivKey: priv}
}

// Issue POST /user/credential { target_app_id }
// 需要 JWT 中间件注入 uid
func (h *CredentialHandler) Issue(c *gin.Context) {
	var req struct {
		TargetAppID string `json:"target_app_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	uid := c.GetString(hubauth.ContextUID)
	if uid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing uid"})
		return
	}

	cred, err := hubauth.IssueCredential(uid, req.TargetAppID, h.hubPrivKey, credentialTTL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "issue credential failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"credential": cred})
}
