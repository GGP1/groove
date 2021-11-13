package cache

// Client is the interface for a cache client.
type Client interface {
	Delete(key string) error
	Get(key string) ([]byte, error)
	Miss(err error) bool
	Set(key string, value []byte) error
}
