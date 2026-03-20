package server

import (
	"database/sql"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/langgexyz/open-im-hub-server/internal/config"
	"github.com/langgexyz/open-im-hub-server/internal/handler"
	"github.com/langgexyz/open-im-hub-server/internal/push"
	"github.com/langgexyz/open-im-hub-server/internal/store"
)

func NewHTTPServer(cfg *config.Config, db *sql.DB) (*gin.Engine, error) {
	s, err := store.New(db)
	if err != nil {
		return nil, fmt.Errorf("init store: %w", err)
	}

	var iosPusher push.Pusher = push.NoopPusher{}
	var androidPusher push.Pusher = push.NoopPusher{}
	if cfg.APNsKeyFile != "" {
		apns, err := push.NewAPNsPusher(cfg.APNsKeyFile, cfg.APNsKeyID, cfg.APNsTeamID, cfg.APNsBundleID, cfg.APNsSandbox)
		if err != nil {
			return nil, fmt.Errorf("init apns: %w", err)
		}
		iosPusher = apns
	}
	if cfg.FCMServerKey != "" {
		androidPusher = push.NewFCMPusher(cfg.FCMServerKey)
	}
	_ = iosPusher
	_ = androidPusher

	deviceTokenH := handler.NewDeviceTokenHandler(s.DeviceTokens, cfg.HubPublicKey)
	directoryH := handler.NewDirectoryHandler(s.Nodes)

	r := gin.New()
	r.Use(gin.Recovery())
	r.POST("/user/device-token", deviceTokenH.Register)
	r.GET("/nodes", directoryH.List)
	return r, nil
}
