package cache

import "github.com/bradfitz/gomemcache/memcache"

const (
	eventsSuffix      = "_events"
	permissionsSuffix = "_permissions"
	rolesSuffix       = "_roles"
	usersSuffix       = "_users"
	zonesSuffix       = "_zones"
)

// Client is the interface for a cache client.
type Client interface {
	Delete(key string) error
	Get(key string) (*memcache.Item, error)
	Miss(err error) bool
	Set(key string, value []byte) error
}

// EventsKey returns eventID + events' key suffix.
func EventsKey(eventID string) string {
	return eventsSuffix + eventID
}

// PermissionsKey eventID + permissions' key suffix.
func PermissionsKey(eventID string) string {
	return eventID + permissionsSuffix
}

// RolesKey eventID + roles' key suffix.
func RolesKey(eventID string) string {
	return eventID + rolesSuffix
}

// UsersKey userID + users' key suffix.
func UsersKey(userID string) string {
	return userID + usersSuffix
}

// ZonesKey eventID + zones' key suffix.
func ZonesKey(eventID string) string {
	return eventID + zonesSuffix
}
