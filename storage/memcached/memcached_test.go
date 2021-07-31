package memcached_test

import (
	"net"
	"testing"

	"github.com/GGP1/groove/config"
	"github.com/GGP1/groove/storage/memcached"
	"github.com/GGP1/groove/test"

	"github.com/stretchr/testify/assert"
)

func TestConnect(t *testing.T) {
	pool, resource := test.NewResource(t, "memcached", "1.6.9-alpine", nil)

	err := pool.Retry(func() error {
		_, err := memcached.Connect(config.Memcached{
			Servers: []string{net.JoinHostPort("localhost", resource.GetPort("11211/tcp"))},
		})
		return err
	})
	assert.NoError(t, err)
}
