package memcached

import (
	"time"

	"github.com/GGP1/groove/config"
	"github.com/GGP1/groove/internal/cache"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/pkg/errors"
)

// client represents a client used to communicate with memcache.
type client struct {
	mc              *memcache.Client
	itemsExpiration int32
}

// NewClient returns a new memcached client.
func NewClient(config config.Memcached) (cache.Client, error) {
	mc, err := Connect(config)
	if err != nil {
		return nil, err
	}
	mc.MaxIdleConns = config.MaxIdleConns
	mc.Timeout = config.Timeout * time.Millisecond

	client := client{
		mc:              mc,
		itemsExpiration: config.ItemsExpiration,
	}

	return client, nil
}

// Delete removes an item from the cache, returns an error if it's not nil nor a miss.
func (c client) Delete(key string) error {
	if err := c.mc.Delete(key); err != nil && !c.Miss(err) {
		return errors.Wrap(err, "memcached deletion")
	}
	return nil
}

// Get returns an item from the cache.
func (c client) Get(key string) ([]byte, error) {
	item, err := c.mc.Get(key)
	if err != nil {
		return nil, errors.Wrap(err, "memcached lookup")
	}
	return item.Value, nil
}

// Miss returns if the error is a cache miss or not.
func (c client) Miss(err error) bool {
	return errors.Is(err, memcache.ErrCacheMiss)
}

// Set saves an item into the cache.
func (c client) Set(key string, value []byte) error {
	item := &memcache.Item{
		Key:        key,
		Value:      value,
		Expiration: c.itemsExpiration,
	}
	if err := c.mc.Set(item); err != nil {
		return errors.Wrap(err, "memcached write")
	}
	return nil
}
