package zone

import (
	"errors"

	"github.com/lib/pq"
)

// Zone represents a zone inside an event.
type Zone struct {
	Name                   string         `json:"name,omitempty"`
	RequiredPermissionKeys pq.StringArray `json:"required_permission_keys,omitempty" db:"required_permission_keys"`
}

// Validate validates zone values.
func (z Zone) Validate() error {
	if z.Name == "" {
		return errors.New("name required")
	}
	return nil
}

// UpdateZone is used to update a zone.
type UpdateZone struct {
	RequiredPermissionKeys *pq.StringArray `json:"required_permission_keys,omitempty"`
}

// Validate validates zone values.
func (z UpdateZone) Validate() error {
	if z.RequiredPermissionKeys == nil {
		return errors.New("required_permission_keys required")
	}
	return nil
}
