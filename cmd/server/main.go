package main

import (
	"database/sql"
	"log"

	_ "github.com/go-sql-driver/mysql"
	"github.com/langgexyz/open-im-hub-server/internal/config"
	"github.com/langgexyz/open-im-hub-server/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db, err := sql.Open("mysql", cfg.MySQLDSN)
	if err != nil {
		log.Fatalf("open mysql: %v", err)
	}

	grpcSrv, err := server.NewGRPCServer(cfg, db)
	if err != nil {
		log.Fatalf("init grpc server: %v", err)
	}
	httpSrv, err := server.NewHTTPServer(cfg, db)
	if err != nil {
		log.Fatalf("init http server: %v", err)
	}

	log.Printf("Hub Server gRPC: %s  HTTP: %s  公钥: %s", grpcSrv.Addr(), cfg.HTTPAddr, cfg.HubPublicKey)

	go func() {
		if err := grpcSrv.Serve(); err != nil {
			log.Fatalf("gRPC server: %v", err)
		}
	}()

	if err := httpSrv.Run(cfg.HTTPAddr); err != nil {
		log.Fatalf("HTTP server: %v", err)
	}
}
