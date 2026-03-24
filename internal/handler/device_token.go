package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	hubauth "github.com/langgexyz/open-im-hub-server/internal/auth"
	"github.com/langgexyz/open-im-hub-server/internal/store"
)

type DeviceTokenHandler struct {
	deviceTokens *store.DeviceTokenStore
	hubPublicKey string
}

func NewDeviceTokenHandler(dt *store.DeviceTokenStore, hubPublicKey string) *DeviceTokenHandler {
	return &DeviceTokenHandler{deviceTokens: dt, hubPublicKey: hubPublicKey}
}

// Register POST /user/device-token
// Authorization: Bearer <user_credential>
func (h *DeviceTokenHandler) Register(c *gin.Context) {
	credStr := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
	if credStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing credential"})
		return
	}
	appUID, _, err := hubauth.VerifyCredential(credStr, h.hubPublicKey)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credential: " + err.Error()})
		return
	}
	var body struct {
		Platform int8   `json:"platform" binding:"required"`
		Token    string `json:"token"    binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.deviceTokens.Upsert(appUID, body.Platform, body.Token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
