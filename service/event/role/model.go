package role

import (
	"strings"
	"time"

	"github.com/GGP1/groove/internal/permissions"

	"github.com/pkg/errors"
)

const (
	// Host is the default role used when creating an event
	Host = "host"
	// Attendant is the default role used when a user is attending an event
	Attendant = "attendant"
)

// Role represents a set of permissions inside the event.
type Role struct {
	Name           string              `json:"name,omitempty"`
	PermissionKeys map[string]struct{} `json:"permission_keys,omitempty"`
}

// Validate ..
func (r Role) Validate() error {
	if r.Name == "" {
		return errors.New("name required")
	}
	if len(r.Name) > 20 {
		return errors.New("invalid name length, maximum is 20")
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

// Permission represents a privilege inside an event.
type Permission struct {
	Name        string     `json:"name,omitempty"`
	Key         string     `json:"key,omitempty"`
	Description string     `json:"description,omitempty"`
	CreatedAt   *time.Time `json:"created_at,omitempty" db:"created_at"`
}

// Validate ..
func (p Permission) Validate() error {
	if p.Key == "" {
		return errors.New("key required")
	}
	if p.Key == permissions.All {
		return errors.New("invalid key")
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
