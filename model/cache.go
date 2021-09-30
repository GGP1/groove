package model

const (
	permissionsSuffix = "_permissions"
	rolesSuffix       = "_roles"
	zonesSuffix       = "_zones"
)

// PermissionsCacheKey returns eventID + permissions' key suffix.
func PermissionsCacheKey(eventID string) string {
	return eventID + permissionsSuffix
}

// RolesCacheKey returns eventID + roles' key suffix.
func RolesCacheKey(eventID string) string {
	return eventID + rolesSuffix
}

// ZonesCacheKey returns eventID + zones' key suffix.
func ZonesCacheKey(eventID string) string {
	return eventID + zonesSuffix
}
