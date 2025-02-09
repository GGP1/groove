package cache

const (
	eventPrivacyPrefix = "event_privacy:"
	permissionsPrefix  = "permissions:"
	rolesPrefix        = "roles:"
	userTypePrefix     = "type:"
	zonesPrefix        = "zones:"
)

// EventPrivacy returns event privacy cache key .
func EventPrivacy(eventID string) string {
	return eventPrivacyPrefix + eventID
}

// PermissionsKey returns permissions cache key.
func PermissionsKey(eventID string) string {
	return permissionsPrefix + eventID
}

// RolesKey returns roles cache key.
func RolesKey(eventID string) string {
	return rolesPrefix + eventID
}

// UserTypeKey returns user type cache key.
func UserTypeKey(userID string) string {
	return userTypePrefix + userID
}

// ZonesKey returns zones cache key.
func ZonesKey(eventID string) string {
	return zonesPrefix + eventID
}
