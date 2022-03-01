package cache

const (
	eventPrivacySuffix = ":privacy"
	permissionsSuffix  = ":permissions"
	rolesSuffix        = ":roles"
	userTypeSuffix     = ":type"
	zonesSuffix        = ":zones"
)

// EventPrivacy returns eventID + privacy's key suffix.
func EventPrivacy(eventID string) string {
	return eventID + eventPrivacySuffix
}

// PermissionsKey returns eventID + permissions' key suffix.
func PermissionsKey(eventID string) string {
	return eventID + permissionsSuffix
}

// RolesKey returns eventID + roles' key suffix.
func RolesKey(eventID string) string {
	return eventID + rolesSuffix
}

// UserTypeKey returns userID + type's key suffix.
func UserTypeKey(userID string) string {
	return userID + userTypeSuffix
}

// ZonesKey returns eventID + zones' key suffix.
func ZonesKey(eventID string) string {
	return eventID + zonesSuffix
}
