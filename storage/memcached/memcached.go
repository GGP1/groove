package memcached

import (
	"strings"

	"github.com/GGP1/groove/config"
	"github.com/GGP1/groove/internal/log"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/pkg/errors"
)

// Connect establishes a connection with memcached clients.
func Connect(config config.Memcached) (*memcache.Client, error) {
	// Consider forking memcache repository to remove unnecessary elements from
	// memcache.Item (Expiration and caseid). That would save 12 bytes per item.
	mc := memcache.New(config.Servers...)
	if err := mc.Ping(); err != nil {
		return nil, errors.Wrap(err, "ping error")
	}

	log.Sugar().Infof("Connected to memcached on %s", strings.Join(config.Servers, ", "))
	return mc, nil
}
