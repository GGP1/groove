// Package romap implements a read-only map.
package romap

// ReadOnlyMap is a map that cannot be modified.
type ReadOnlyMap[T any] struct {
	mp   map[string]T
	keys []string
}

// New returns a new read-only map.
func New[T any](mp map[string]T) ReadOnlyMap[T] {
	keys := make([]string, 0, len(mp))
	for key := range mp {
		keys = append(keys, key)
	}

	return ReadOnlyMap[T]{
		keys: keys,
		mp:   mp,
	}
}

// Exists returns if the key is inside the map or not.
func (r ReadOnlyMap[T]) Exists(key string) bool {
	_, ok := r.mp[key]
	return ok
}

// Get returns the value corresponding to the key passed.
func (r ReadOnlyMap[T]) Get(key string) (T, bool) {
	v, ok := r.mp[key]
	return v, ok
}

// Keys returns a slice with all the map's keys.
func (r ReadOnlyMap[T]) Keys() []string {
	return r.keys
}

// Map returns a modifiable copy of the underlying map.
func (r ReadOnlyMap[T]) Map() map[string]T {
	return r.mp
}
