package role

import (
	"time"

	"github.com/GGP1/groove/internal/permissions"
	"github.com/GGP1/groove/internal/romap"

	"github.com/lib/pq"
	"github.com/pkg/errors"
)

// ReservedRoles is a read-only map that contains reserved roles
// and its permission keys applying to all events.
var ReservedRoles = romap.New(map[string]interface{}{
	Host:      []string{permissions.All},
	Attendant: []string{permissions.Access},
	Moderator: []string{permissions.Access, permissions.BanUsers},
	Viewer:    []string{permissions.ViewEvent},
})

const (
	// Host is the default role used when creating an event
	Host = "host"
	// Attendant is the default role used when a user is attending an event
	Attendant = "attendant"
	// Moderator is in charge of banning problematic users
	Moderator = "moderator"
	// Viewer can see the content's of an event's page
	Viewer = "viewer"
)

// Role represents a set of permissions inside the event.
type Role struct {
	Name           string         `json:"name,omitempty"`
	PermissionKeys pq.StringArray `json:"permission_keys,omitempty"`
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
		return errors.New("permission_keys required")
	}
	return nil
}

// UpdateRole is the structure used to update roles.
type UpdateRole struct {
	PermissionKeys *pq.StringArray `json:"permission_keys,omitempty"`
}

// Validate validates update roles fields.
func (r UpdateRole) Validate() error {
	if r.PermissionKeys != nil {
		if len(*r.PermissionKeys) == 0 {
			return errors.New("permission_keys required")
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
	Description *string `json:"description,omitempty"`
}

// Validate ..
func (p UpdatePermission) Validate() error {
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
