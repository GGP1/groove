package model

import (
	"github.com/GGP1/groove/internal/validate"
	"github.com/lib/pq"
	"github.com/pkg/errors"
)

// Zone represents a zone inside an event.
type Zone struct {
	Name                   string         `json:"name,omitempty"`
	RequiredPermissionKeys pq.StringArray `json:"required_permission_keys,omitempty" db:"required_permission_keys"`
}

// Validate validates zone values.
func (z Zone) Validate() error {
	if err := validate.Name(z.Name); err != nil {
		return err
	}
	for _, key := range z.RequiredPermissionKeys {
		if err := validate.Key(key); err != nil {
			return errors.Wrapf(err, "%q is invalid", key)
		}
	}
	return nil
}

// UpdateZone is used to update a zone.
type UpdateZone struct {
	Name                   *string         `json:"name,omitempty"`
	RequiredPermissionKeys *pq.StringArray `json:"required_permission_keys,omitempty"`
}

// Validate validates zone values.
func (z UpdateZone) Validate() error {
	if z.Name != nil {
		if err := validate.Name(*z.Name); err != nil {
			return err
		}
	}
	if z.RequiredPermissionKeys != nil {
		for _, key := range *z.RequiredPermissionKeys {
			if err := validate.Key(key); err != nil {
				return errors.Wrapf(err, "%q is invalid", key)
			}
		}
	}
	return nil
}
