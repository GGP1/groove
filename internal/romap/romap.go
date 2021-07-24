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

// GetStringSlice is like Get but it casts the value to a []string.
//
// Panic is not avoided on purpose.
func (r ReadOnlyMap) GetStringSlice(key string) ([]string, bool) {
	v, ok := r.Get(key)
	if v == nil {
		return nil, false
	}
	return v.([]string), ok
}

// Keys returns a slice with all the map's keys.
func (r ReadOnlyMap) Keys() []string {
	return r.keys
}

// Map returns a modifiable copy of the underlying map.
func (r ReadOnlyMap) Map() map[string]interface{} {
	return r.mp
}
