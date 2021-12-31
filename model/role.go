package model

import (
	"github.com/GGP1/groove/internal/permissions"
	"github.com/GGP1/groove/internal/roles"
	"github.com/GGP1/groove/internal/validate"

	"github.com/lib/pq"
	"github.com/pkg/errors"
)

// ReservedRoles is an slice with reserved roles and it should be kept in sync with the roles.Reserved map.
var ReservedRoles = []Role{
	{
		Name:           roles.Host,
		PermissionKeys: []string{permissions.All},
	},
	{
		Name:           roles.Attendant,
		PermissionKeys: []string{permissions.Access},
	},
	{
		Name:           roles.Moderator,
		PermissionKeys: []string{permissions.Access, permissions.BanUsers},
	},
	{
		Name:           roles.Viewer,
		PermissionKeys: []string{permissions.ViewEvent},
	},
}

// Role represents a set of permissions inside the event.
type Role struct {
	Name           string         `json:"name,omitempty"`
	PermissionKeys pq.StringArray `json:"permission_keys,omitempty" db:"permission_keys"`
}

// Validate returns an error if the role is invalid.
func (r Role) Validate() error {
	if r.Name == "" {
		return errors.New("name required")
	}
	if err := validate.RoleName(r.Name); err != nil {
		return err
	}
	if roles.Reserved.Exists(r.Name) {
		return errors.New("reserved name")
	}
	if len(r.PermissionKeys) == 0 {
		return errors.New("permission_keys required")
	}
	for i, k := range r.PermissionKeys {
		if err := validate.Key(k); err != nil {
			return errors.Wrapf(err, "permission_keys [%d]", i)
		}
	}
	return nil
}

// SetRole is the struct used to assign roles to multiple users.
type SetRole struct {
	RoleName string   `json:"role_name,omitempty"`
	UserIDs  []string `json:"user_ids,omitempty"`
}

// Validate verifies the ids and the role name passed is correct.
func (sr SetRole) Validate() error {
	if err := validate.RoleName(sr.RoleName); err != nil {
		return err
	}
	return validate.ULIDs(sr.UserIDs...)
}

// UpdateRole is the structure used to update roles.
type UpdateRole struct {
	PermissionKeys *pq.StringArray `json:"permission_keys,omitempty" db:"permission_keys"`
}

// Validate validates update roles fields.
func (r UpdateRole) Validate() error {
	if r.PermissionKeys == nil || len(*r.PermissionKeys) == 0 {
		return errors.New("permission_keys required")
	}
	for i, k := range *r.PermissionKeys {
		if err := validate.Key(k); err != nil {
			return errors.Wrapf(err, "permission_keys [%d]", i)
		}
	}
	return nil
}
