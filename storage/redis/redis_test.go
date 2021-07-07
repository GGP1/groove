package redis_test

import (
	"context"
	"testing"

	"github.com/GGP1/groove/config"
	"github.com/GGP1/groove/storage/redis"
	"github.com/GGP1/groove/test"

	"github.com/stretchr/testify/assert"
)

func TestConnect(t *testing.T) {
	pool, resource := test.NewResource(t, "redis", "6.2.4-alpine", nil)

	err := pool.Retry(func() error {
		rdb, err := redis.Connect(context.Background(), config.Redis{
			Host: "localhost",
			Port: resource.GetPort("6379/tcp"),
		})
		assert.NoError(t, err)
		return rdb.Close()
	})
	assert.NoError(t, err)
}
