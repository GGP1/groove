package cache

const (
	permissionsSuffix = "_permissions"
	rolesSuffix       = "_roles"
	zonesSuffix       = "_zones"
)

// PermissionsKey returns eventID + permissions' key suffix.
func PermissionsKey(eventID string) string {
	return eventID + permissionsSuffix
}

// RolesKey returns eventID + roles' key suffix.
func RolesKey(eventID string) string {
	return eventID + rolesSuffix
}

// ZonesKey returns eventID + zones' key suffix.
func ZonesKey(eventID string) string {
	return eventID + zonesSuffix
}
