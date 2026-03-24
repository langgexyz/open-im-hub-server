package server

import (
	"database/sql"
	"fmt"
	"net"

	_ "github.com/go-sql-driver/mysql"
	"google.golang.org/grpc"
	"github.com/langgexyz/open-im-hub-server/internal/config"
	grpcserver "github.com/langgexyz/open-im-hub-server/internal/grpc"
	"github.com/langgexyz/open-im-hub-server/internal/store"
)

type GRPCServer struct {
	srv *grpc.Server
	lis net.Listener
}

func NewGRPCServer(cfg *config.Config, db *sql.DB) (*GRPCServer, error) {
	s, err := store.New(db)
	if err != nil {
		return nil, fmt.Errorf("init store: %w", err)
	}

	ns := &nodeStoreAdapter{s: s}
	srv := grpcserver.New(ns, cfg.HubPrivateKeyObj, cfg.HubPublicKey)

	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		return nil, fmt.Errorf("listen grpc %s: %w", cfg.GRPCAddr, err)
	}
	return &GRPCServer{srv: srv, lis: lis}, nil
}

func (g *GRPCServer) Serve() error { return g.srv.Serve(g.lis) }
func (g *GRPCServer) Stop()        { g.srv.GracefulStop() }
func (g *GRPCServer) Addr() string { return g.lis.Addr().String() }

// nodeStoreAdapter 将 *store.Store 适配为 grpcserver.NodeStore 接口
type nodeStoreAdapter struct{ s *store.Store }

func (a *nodeStoreAdapter) GetByPublicKey(k string) (*store.Node, error) { return a.s.Nodes.GetByPublicKey(k) }
func (a *nodeStoreAdapter) Upsert(n *store.Node) error                   { return a.s.Nodes.Upsert(n) }
func (a *nodeStoreAdapter) UpdateHeartbeat(k string) error               { return a.s.Nodes.UpdateHeartbeat(k) }
func (a *nodeStoreAdapter) List() ([]*store.Node, error)                 { return a.s.Nodes.List() }
func (a *nodeStoreAdapter) GetDeviceTokens(appUIDs []string) (map[string][]store.DeviceToken, error) {
	return a.s.DeviceTokens.GetByUIDs(appUIDs)
}
func (a *nodeStoreAdapter) UpdateProfile(appID, name, avatar, description string) error {
	return a.s.Nodes.UpdateProfile(appID, name, avatar, description)
}
