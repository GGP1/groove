package main

import (
	"context"

	"github.com/GGP1/groove/config"
	"github.com/GGP1/groove/http/rest/router"
	"github.com/GGP1/groove/internal/log"
	"github.com/GGP1/groove/server"
	"github.com/GGP1/groove/storage/dgraph"
	"github.com/GGP1/groove/storage/memcached"
	"github.com/GGP1/groove/storage/postgres"
	"github.com/GGP1/groove/storage/redis"

	_ "github.com/lib/pq"
)

var (
	version = "development"
	commit  = ""
	branch  = ""
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.New()
	if err != nil {
		log.Sugar().Fatalf("Failed creating the configuration: %v", err)
	}
	defer log.Sync() // Flush buffered entries

	db, err := postgres.Connect(ctx, cfg.Postgres)
	if err != nil {
		log.Sugar().Fatalf("Failed connecting to postgres: %v", err)
	}
	defer db.Close()

	dc, closeConn, err := dgraph.Connect(ctx, cfg.Dgraph)
	if err != nil {
		log.Sugar().Fatalf("Failed connecting to Dgraph: %v", err)
	}
	defer closeConn()

	cache, err := memcached.NewClient(cfg.Memcached)
	if err != nil {
		log.Sugar().Fatalf("Failed connecting to memcached: %v", err)
	}

	rdb, err := redis.Connect(ctx, cfg.Redis)
	if err != nil {
		log.Sugar().Fatalf("Failed connecting to redis: %v", err)
	}
	defer rdb.Close()

	router := router.New(cfg, db, dc, rdb, cache)
	server := server.New(cfg.Server, router)

	log.Sugar().Infof("Server started: version %q, branch %q, commit %q", version, branch, commit)
	if err := server.Run(ctx); err != nil {
		log.Sugar().Fatalf("Failed running server: %v", err)
	}
}
