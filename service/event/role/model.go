package role

import (
	"strings"
	"time"

	"github.com/GGP1/groove/internal/permissions"
	"github.com/GGP1/groove/internal/romap"

	"github.com/pkg/errors"
)

// ReservedRoles is a read-only map that contains reserved roles
// and its permission keys applying to all events.
var ReservedRoles = romap.New(map[string]interface{}{
	Host: map[string]struct{}{
		permissions.All: {},
	},
	Attendant: map[string]struct{}{
		permissions.Access: {},
	},
	Moderator: map[string]struct{}{
		permissions.Access:   {},
		permissions.BanUsers: {},
	},
})

const (
	// Host is the default role used when creating an event
	Host = "host"
	// Attendant is the default role used when a user is attending an event
	Attendant = "attendant"
	// Moderator is in charge of banning problematic users
	Moderator = "moderator"
)

// Role represents a set of permissions inside the event.
type Role struct {
	Name           string              `json:"name,omitempty"`
	PermissionKeys map[string]struct{} `json:"permission_keys,omitempty"`
}

// Validate returns an error if the role is invalid.
func (r Role) Validate() error {
	if r.Name == "" {
		return errors.New("name required")
	}
	if len(r.Name) > 20 {
		return errors.New("invalid name length, maximum is 20")
	}
	if ReservedRoles.Exists(r.Name) {
		return errors.New("reserved name")
	}
	if len(r.PermissionKeys) == 0 {
		return errors.New("permissions_keys required")
	}
	for pk := range r.PermissionKeys {
		if strings.Contains(pk, permissions.Separator) {
			return errors.Errorf("permission key [%q] cannot contain character %q", pk, permissions.Separator)
		}
	}
	return nil
}

// UpdateRole is the structure used to update roles.
type UpdateRole struct {
	Name           *string              `json:"name,omitempty"`
	PermissionKeys *map[string]struct{} `json:"permission_keys,omitempty"`
}

// Validate ..
func (r UpdateRole) Validate() error {
	if r.Name != nil {
		if *r.Name == "" {
			return errors.New("name required")
		}
		if len(*r.Name) > 20 {
			return errors.New("invalid name length, maximum is 20")
		}
		if ReservedRoles.Exists(*r.Name) {
			return errors.New("reserved name")
		}
	}
	if r.PermissionKeys != nil {
		if len(*r.PermissionKeys) == 0 {
			return errors.New("permissions_keys required")
		}
		for pk := range *r.PermissionKeys {
			if strings.Contains(pk, permissions.Separator) {
				return errors.Errorf("permission key [%q] cannot contain character %q", pk, permissions.Separator)
			}
		}
	}

	return nil
}

// Permission represents a privilege inside an event.
type Permission struct {
	Name        string    `json:"name,omitempty"`
	Key         string    `json:"key,omitempty"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty" db:"created_at"`
}

// Validate returns an error if the permission is invalid.
func (p Permission) Validate() error {
	if p.Key == "" {
		return errors.New("key required")
	}
	if permissions.ReservedKeys.Exists(p.Key) {
		return errors.New("reserved key")
	}
	if len(p.Key) > 20 {
		return errors.New("invalid key length, maximum is 20")
	}
	if strings.Contains(p.Key, permissions.Separator) {
		return errors.Errorf("permission key cannot contain character %q", permissions.Separator)
	}
	if p.Name == "" {
		return errors.New("name required")
	}
	if len(p.Name) > 20 {
		return errors.New("invalid name length, maximum is 20")
	}
	if len(p.Description) > 20 {
		return errors.New("invalid description length, maximum is 50")
	}
	return nil
}

// UpdatePermission is the structure used to update permissions.
type UpdatePermission struct {
	Name        *string `json:"name,omitempty"`
	Key         *string `json:"key,omitempty"`
	Description *string `json:"description,omitempty"`
}

// Validate ..
func (p UpdatePermission) Validate() error {
	if p.Key != nil {
		if *p.Key == "" {
			return errors.New("key required")
		}
		if permissions.ReservedKeys.Exists(*p.Key) {
			return errors.New("reserved key")
		}
		if len(*p.Key) > 20 {
			return errors.New("invalid key length, maximum is 20")
		}
		if strings.Contains(*p.Key, permissions.Separator) {
			return errors.Errorf("permission key cannot contain character %q", permissions.Separator)
		}
	}
	if p.Name != nil {
		if *p.Name == "" {
			return errors.New("name required")
		}
		if len(*p.Name) > 20 {
			return errors.New("invalid name length, maximum is 20")
		}
	}
	if p.Description != nil {
		if len(*p.Description) > 20 {
			return errors.New("invalid description length, maximum is 50")
		}
	}
	return nil
}
