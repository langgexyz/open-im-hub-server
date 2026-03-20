package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/langgexyz/open-im-hub-server/internal/config"
	"github.com/langgexyz/open-im-hub-server/internal/server"
	"github.com/langgexyz/open-im-hub-server/internal/store"
)

func main() {
	genCode := flag.Bool("gen-code", false, "生成一个激活码并打印")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db, err := sql.Open("mysql", cfg.MySQLDSN)
	if err != nil {
		log.Fatalf("open mysql: %v", err)
	}

	if *genCode {
		s, err := store.New(db)
		if err != nil {
			log.Fatalf("init store: %v", err)
		}
		code := uuid.NewString()
		if err := s.Codes.Insert(code, time.Now().Add(30*24*time.Hour)); err != nil {
			log.Fatalf("insert code: %v", err)
		}
		fmt.Printf("激活码：%s（30天有效）\n", code)
		return
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
