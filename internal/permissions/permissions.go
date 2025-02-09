package permissions

import (
	"github.com/GGP1/groove/internal/romap"

	"github.com/pkg/errors"
)

// Reserved permission key.
const (
	All               = "*"
	Access            = "access"
	BanUsers          = "ban_users"
	InviteUsers       = "invite_users"
	ModifyPermissions = "modify_permissions"
	ModifyPosts       = "modify_posts"
	ModifyProducts    = "modify_products"
	ModifyRoles       = "modify_roles"
	ModifyTickets     = "modify_tickets"
	ModifyZones       = "modify_zones"
	SetUserRole       = "set_user_role"
	UpdateEvent       = "update_event"
	ViewEvent         = "view_event"
)

// Reserved is a read-only map containing all reserved permissions keys.
var Reserved = romap.New(map[string]struct{}{
	All:               {},
	Access:            {},
	BanUsers:          {},
	ModifyPermissions: {},
	ModifyPosts:       {},
	ModifyProducts:    {},
	ModifyRoles:       {},
	ModifyTickets:     {},
	ModifyZones:       {},
	InviteUsers:       {},
	SetUserRole:       {},
	UpdateEvent:       {},
	ViewEvent:         {},
})

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
