package role

import (
	"time"

	"github.com/GGP1/groove/internal/permissions"
	"github.com/GGP1/groove/internal/roles"
	"github.com/GGP1/groove/internal/validate"

	"github.com/lib/pq"
	"github.com/pkg/errors"
)

// reservedRoles should be kept in sync with the roles.Reserved map.
var reservedRoles = []Role{
	{
		Name:           string(roles.Host),
		PermissionKeys: []string{permissions.All},
	},
	{
		Name:           string(roles.Attendant),
		PermissionKeys: []string{permissions.Access},
	},
	{
		Name:           string(roles.Moderator),
		PermissionKeys: []string{permissions.Access, permissions.BanUsers},
	},
	{
		Name:           string(roles.Viewer),
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

// Permission represents a privilege inside an event.
type Permission struct {
	CreatedAt   *time.Time `json:"created_at,omitempty" db:"created_at"`
	Name        string     `json:"name,omitempty"`
	Key         string     `json:"key,omitempty"`
	Description string     `json:"description,omitempty"`
}

// Validate returns an error if the permission is invalid.
func (p Permission) Validate() error {
	if p.Key == "" {
		return errors.New("key required")
	}
	if permissions.Reserved.Exists(p.Key) {
		return errors.New("reserved key")
	}
	if err := validate.Key(p.Key); err != nil {
		return err
	}
	if p.Name == "" {
		return errors.New("name required")
	}
	if len(p.Name) > 40 {
		return errors.New("invalid name length, maximum is 40")
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
}

// Validate validates the correctness of struct fields.
func (p UpdatePermission) Validate() error {
	if p.Name != nil {
		if *p.Name == "" {
			return errors.New("name required")
		}
		if len(*p.Name) > 40 {
			return errors.New("invalid name length, maximum is 40")
		}
	}
	if p.Description != nil {
		if len(*p.Description) > 200 {
			return errors.New("invalid description length, maximum is 200")
		}
	}
	return nil
}
