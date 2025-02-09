package redis

import (
	"context"
	"net"
	"time"

	"github.com/GGP1/groove/config"
	"github.com/GGP1/groove/internal/log"

	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
)

// ItemExpiration represents the time until an object is considered expired.
var ItemExpiration = 10 * time.Minute

// Connect establishes a connection with the redis client.
func Connect(ctx context.Context, config config.Redis) (*redis.Client, error) {
	addr := net.JoinHostPort(config.Host, config.Port)
	rdb := redis.NewClient(&redis.Options{
		Network:      "tcp",
		Addr:         addr,
		Password:     config.Password,
		DB:           0,
		PoolSize:     config.PoolSize,
		MinIdleConns: config.MinIdleConns,
		// TLSConfig: &tls.Config{
		// 	MinVersion:   tls.VersionTLS12,
		// 	Certificates: config.TLSCertificates,
		// },
	}).WithContext(ctx)

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, errors.Wrap(err, "ping error")
	}

	ItemExpiration = config.DefaultExpiration * time.Minute

	runMetrics(rdb, config)

	log.Sugar().Infof("Connected to redis on %s", addr)
	return rdb, nil
}
