package cache

import "github.com/bradfitz/gomemcache/memcache"

// Client is the interface for a cache client.
type Client interface {
	Delete(key string) error
	Get(key string) (*memcache.Item, error)
	Miss(err error) bool
	Set(key string, value []byte) error
}
