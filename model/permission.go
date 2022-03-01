package model

import (
	"time"

	"github.com/GGP1/groove/internal/permissions"
	"github.com/GGP1/groove/internal/validate"

	"github.com/pkg/errors"
)

// Permission represents a privilege inside an event.
type Permission struct {
	CreatedAt   *time.Time `json:"created_at,omitempty" db:"created_at"`
	Name        string     `json:"name,omitempty"`
	Key         string     `json:"key,omitempty"`
	Description string     `json:"description,omitempty"`
}

// Validate returns an error if the permission is invalid.
func (p Permission) Validate() error {
	if err := validate.Name(p.Name); err != nil {
		return err
	}
	if p.Key == "" {
		return errors.New("key required")
	}
	if permissions.Reserved.Exists(p.Key) {
		return errors.New("reserved key")
	}
	if err := validate.Key(p.Key); err != nil {
		return err
	}
	if len(p.Description) > 200 {
		return errors.New("invalid description length, maximum is 200")
	}
	return nil
}

// UpdatePermission is the structure used to update permissions.
type UpdatePermission struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Key         *string `json:"key,omitempty"`
}

// Validate validates the correctness of struct fields.
func (p UpdatePermission) Validate() error {
	if p.Name != nil {
		if err := validate.Name(*p.Name); err != nil {
			return err
		}
	}
	if p.Description != nil {
		if len(*p.Description) > 200 {
			return errors.New("invalid description length, maximum is 200")
		}
	}
	if p.Key != nil {
		if permissions.Reserved.Exists(*p.Key) {
			return errors.New("reserved key")
		}
		if err := validate.Key(*p.Key); err != nil {
			return err
		}
	}
	return nil
}
