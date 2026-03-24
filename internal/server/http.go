package server

import (
	"database/sql"
	"fmt"

	"github.com/gin-gonic/gin"
	hubauth "github.com/langgexyz/open-im-hub-server/internal/auth"
	"github.com/langgexyz/open-im-hub-server/internal/config"
	"github.com/langgexyz/open-im-hub-server/internal/handler"
	"github.com/langgexyz/open-im-hub-server/internal/store"
)

func NewHTTPServer(cfg *config.Config, db *sql.DB) (*gin.Engine, error) {
	s, err := store.New(db)
	if err != nil {
		return nil, fmt.Errorf("init store: %w", err)
	}

	userH        := handler.NewUserHandler(s.Users, cfg.HubPrivateKey)
	credH        := handler.NewCredentialHandler(cfg.HubPrivateKey)
	activateH    := handler.NewActivateHandler(s.Nodes, cfg.HubPrivateKey, cfg.HubGRPCAddr, cfg.HubWebOrigin)
	directoryH   := handler.NewDirectoryHandler(s.Nodes)
	deviceTokenH := handler.NewDeviceTokenHandler(s.DeviceTokens, cfg.HubPublicKey)

	r := gin.New()
	r.Use(gin.Recovery())

	// 公开接口
	r.POST("/user/register", userH.Register)
	r.POST("/user/login", userH.Login)
	r.GET("/nodes", directoryH.List)
	r.GET("/nodes/:app_id", directoryH.Get)
	r.POST("/user/device-token", deviceTokenH.Register)

	// 需要登录（JWT）
	auth := r.Group("/", hubauth.JWTMiddleware(cfg.HubPrivateKey))
	auth.POST("/user/credential", credH.Issue)
	auth.POST("/node/activate", activateH.Activate)

	return r, nil
}
