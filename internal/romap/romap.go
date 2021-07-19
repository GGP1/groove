// Package romap implements a read-only map.
package romap

// ReadOnlyMap is a map that cannot be modified.
type ReadOnlyMap struct {
	// caseSensitive bool
	keys []string
	mp   map[string]interface{}
}

// New returns a new read-only map.
func New(mp map[string]interface{}) ReadOnlyMap {
	keys := make([]string, 0, len(mp))
	for key := range mp {
		keys = append(keys, key)
	}

	return ReadOnlyMap{
		keys: keys,
		mp:   mp,
	}
}

// Exists returns if the key is inside the map or not.
func (r ReadOnlyMap) Exists(key string) bool {
	_, ok := r.mp[key]
	return ok
}

// Get returns the value corresponding to the key passed.
func (r ReadOnlyMap) Get(key string) (interface{}, bool) {
	v, ok := r.mp[key]
	return v, ok
}

// GetStringMapStruct is like Get but it casts the value to a map[string]struct{}.
//
// This should be used for the reserved roles map only.
// Panic is not avoided on purpose.
func (r ReadOnlyMap) GetStringMapStruct(key string) (map[string]struct{}, bool) {
	v, ok := r.Get(key)
	return v.(map[string]struct{}), ok
}

// Keys returns a slice with all the map's keys.
func (r ReadOnlyMap) Keys() []string {
	return r.keys
}

// Map returns a modifiable copy of the underlying map.
func (r ReadOnlyMap) Map() map[string]interface{} {
	return r.mp
}
