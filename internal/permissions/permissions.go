package permissions

import (
	"github.com/GGP1/groove/internal/romap"

	"github.com/pkg/errors"
)

// ReservedKeys is a read-only map containing all reserved permissions keys.
var ReservedKeys = romap.New(map[string]interface{}{
	All:               struct{}{},
	Access:            struct{}{},
	BanUsers:          struct{}{},
	ModifyMedia:       struct{}{},
	ModifyPermissions: struct{}{},
	ModifyProducts:    struct{}{},
	ModifyRoles:       struct{}{},
	ModifyZones:       struct{}{},
	InviteUsers:       struct{}{},
	SetUserRole:       struct{}{},
	UpdateEvent:       struct{}{},
	ViewEvent:         struct{}{},
})

// Pre-defined permission key.
const (
	All               = "*"
	Access            = "access"
	BanUsers          = "ban_users"
	InviteUsers       = "invite_users"
	ModifyMedia       = "modify_media"
	ModifyPermissions = "modify_permissions"
	ModifyProducts    = "modify_products"
	ModifyRoles       = "modify_roles"
	ModifyZones       = "modify_zones"
	SetUserRole       = "set_user_role"
	UpdateEvent       = "update_event"
	ViewEvent         = "view_event"
)

// Require makes sure the user has all the permissions required.
func Require(userPermKeys map[string]struct{}, required ...string) error {
	if _, ok := userPermKeys[All]; ok {
		return nil
	}

	if len(required) > len(userPermKeys) {
		return errors.New("permission keys missing")
	}

	for _, r := range required {
		if _, ok := userPermKeys[r]; !ok {
			return errors.Errorf("permission key %q missing", r)
		}
	}

	return nil
}
