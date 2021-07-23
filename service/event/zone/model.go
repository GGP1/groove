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

// Validate ..
func (z Zone) Validate() error {
	if z.Name == "" {
		return errors.New("name required")
	}
	return nil
}
