package server

import (
	"database/sql"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/langgexyz/open-im-hub-server/internal/config"
	"github.com/langgexyz/open-im-hub-server/internal/handler"
	"github.com/langgexyz/open-im-hub-server/internal/store"
)

func NewHTTPServer(cfg *config.Config, db *sql.DB) (*gin.Engine, error) {
	s, err := store.New(db)
	if err != nil {
		return nil, fmt.Errorf("init store: %w", err)
	}

	deviceTokenH := handler.NewDeviceTokenHandler(s.DeviceTokens, cfg.HubPublicKey)
	directoryH := handler.NewDirectoryHandler(s.Nodes)
	activateH := handler.NewActivateHandler(s.Nodes, cfg.HubPrivateKey, cfg.HubGRPCAddr, cfg.HubWebOrigin)

	r := gin.New()
	r.Use(gin.Recovery())
	r.POST("/user/device-token", deviceTokenH.Register)
	r.GET("/nodes", directoryH.List)
	r.POST("/node/activate", activateH.Activate)
	return r, nil
}
