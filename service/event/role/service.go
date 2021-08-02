package role

import (
	"context"
	"database/sql"

	"github.com/GGP1/groove/internal/bufferpool"
	"github.com/GGP1/groove/internal/cache"
	"github.com/GGP1/groove/internal/permissions"
	"github.com/GGP1/groove/storage/postgres"

	"github.com/pkg/errors"
)

// Service interface for the roles service.
type Service interface {
	ClonePermissions(ctx context.Context, sqlTx *sql.Tx, exporterEventID, importerEventID string) error
	CloneRoles(ctx context.Context, sqlTx *sql.Tx, exporterEventID, importerEventID string) error
	CreatePermission(ctx context.Context, sqlTx *sql.Tx, eventID string, permission Permission) error
	CreateRole(ctx context.Context, sqlTx *sql.Tx, eventID string, role Role) error
	DeletePermission(ctx context.Context, sqlTx *sql.Tx, eventID, key string) error
	DeleteRole(ctx context.Context, sqlTx *sql.Tx, eventID, name string) error
	GetPermission(ctx context.Context, sqlTx *sql.Tx, eventID, key string) (Permission, error)
	GetPermissions(ctx context.Context, sqlTx *sql.Tx, eventID string) ([]Permission, error)
	GetRole(ctx context.Context, sqlTx *sql.Tx, eventID, name string) (Role, error)
	GetRoles(ctx context.Context, sqlTx *sql.Tx, eventID string) ([]Role, error)
	GetUserRole(ctx context.Context, sqlTx *sql.Tx, eventID, userID string) (Role, error)
	IsHost(ctx context.Context, sqlTx *sql.Tx, userID string, eventIDs ...string) (bool, error)
	SetRoles(ctx context.Context, sqlTx *sql.Tx, eventID, roleName string, userIDs ...string) error
	SetViewerRole(ctx context.Context, sqlTx *sql.Tx, eventID, userID string) error
	UpdatePermission(ctx context.Context, sqlTx *sql.Tx, eventID, key string, permission UpdatePermission) error
	UpdateRole(ctx context.Context, sqlTx *sql.Tx, eventID, name string, role UpdateRole) error
	UserHasRole(ctx context.Context, sqlTx *sql.Tx, eventID, userID string) (bool, error)
}

type service struct {
	db    *sql.DB
	cache cache.Client
}

// NewService returns a new role service.
func NewService(db *sql.DB, cache cache.Client) Service {
	return service{
		db:    db,
		cache: cache,
	}
}

// ClonePermissions takes the permissions from the exporter event and creates them in the importer event.
func (s service) ClonePermissions(ctx context.Context, sqlTx *sql.Tx, exporterEventID, importerEventID string) error {
	create := "CREATE TEMPORARY TABLE perm_temp AS (SELECT * FROM events_permissions WHERE event_id=$1)"
	if _, err := sqlTx.ExecContext(ctx, create, exporterEventID); err != nil {
		return errors.Wrap(err, "cloning permissions")
	}

	update := "UPDATE perm_temp SET event_id=$1"
	if _, err := sqlTx.ExecContext(ctx, update, importerEventID); err != nil {
		return errors.Wrap(err, "cloning permissions")
	}

	insert := "INSERT INTO events_permissions SELECT * FROM perm_temp ON CONFLICT DO NOTHING"
	if _, err := sqlTx.ExecContext(ctx, insert); err != nil {
		return errors.Wrap(err, "cloning permissions")
	}

	return nil
}

// CloneRoles takes the roles from the exporter event and creates them in the importer event.
//
// It also takes care of cloning permissions.
func (s service) CloneRoles(ctx context.Context, sqlTx *sql.Tx, exporterEventID, importerEventID string) error {
	// Clone permissions as they are required to create roles.
	if err := s.ClonePermissions(ctx, sqlTx, exporterEventID, importerEventID); err != nil {
		return err
	}

	create := "CREATE TEMPORARY TABLE roles_temp AS (SELECT * FROM events_roles WHERE event_id=$1)"
	if _, err := sqlTx.ExecContext(ctx, create, exporterEventID); err != nil {
		return errors.Wrap(err, "cloning permissions")
	}

	update := "UPDATE roles_temp SET event_id=$1"
	if _, err := sqlTx.ExecContext(ctx, update, importerEventID); err != nil {
		return errors.Wrap(err, "cloning permissions")
	}

	insert := "INSERT INTO events_roles SELECT * FROM roles_temp ON CONFLICT DO NOTHING"
	if _, err := sqlTx.ExecContext(ctx, insert); err != nil {
		return errors.Wrap(err, "cloning permissions")
	}

	return nil
}

// CreatePermission creates a permission inside the event.
func (s service) CreatePermission(ctx context.Context, sqlTx *sql.Tx, eventID string, permission Permission) error {
	q := "INSERT INTO events_permissions (event_id, key, name, description) VALUES ($1, $2, $3, $4)"
	_, err := sqlTx.ExecContext(ctx, q, eventID, permission.Key, permission.Name, permission.Description)
	if err != nil {
		return errors.Wrap(err, "creating permission")
	}

	if err := s.cache.Delete(cache.PermissionsKey(eventID)); err != nil {
		return errors.Wrap(err, "deleting permission")
	}

	return nil
}

// CreateRole creates a new role inside an event.
func (s service) CreateRole(ctx context.Context, sqlTx *sql.Tx, eventID string, role Role) error {
	q1 := "SELECT EXISTS(SELECT 1 FROM events_permissions WHERE event_id=$1 AND key=$2)"
	exists := false

	stmt, err := sqlTx.PrepareContext(ctx, q1)
	if err != nil {
		return errors.Wrap(err, "preparing statement")
	}

	// Check for the existence of the keys used for the role
	for _, key := range role.PermissionKeys {
		if permissions.ReservedKeys.Exists(key) {
			continue
		}
		row := stmt.QueryRowContext(ctx, eventID, key)
		if err := row.Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return errors.Errorf("permission with key %q does not exist", key)
		}
	}

	q2 := "INSERT INTO events_roles (event_id, name, permission_keys) VALUES ($1, $2, $3)"
	if _, err := sqlTx.ExecContext(ctx, q2, eventID, role.Name, role.PermissionKeys); err != nil {
		return errors.Wrap(err, "creating role")
	}

	if err := s.cache.Delete(cache.RolesKey(eventID)); err != nil {
		return errors.Wrap(err, "deleting roles")
	}

	return nil
}

// DeletePermission removes a permission from the event.
func (s service) DeletePermission(ctx context.Context, sqlTx *sql.Tx, eventID, key string) error {
	q := "DELETE FROM events_permissions WHERE event_id=$1 AND key=$2"
	if _, err := sqlTx.ExecContext(ctx, q, eventID, key); err != nil {
		return errors.Wrap(err, "deleting permission")
	}
	return nil
}

// DeleteRole removes a role from the event.
func (s service) DeleteRole(ctx context.Context, sqlTx *sql.Tx, eventID, name string) error {
	q := "DELETE FROM events_roles WHERE event_id=$1 AND name=$2"
	if _, err := sqlTx.ExecContext(ctx, q, eventID, name); err != nil {
		return errors.Wrap(err, "deleting role")
	}
	return nil
}

// GetPermission returns a permission from an event with the given key.
func (s service) GetPermission(ctx context.Context, sqlTx *sql.Tx, eventID, key string) (Permission, error) {
	q := "SELECT name, description, created_at FROM events_permissions WHERE event_id=$1 AND key=$2"
	row := sqlTx.QueryRowContext(ctx, q, eventID, key)

	permission := Permission{Key: key}
	if err := row.Scan(&permission.Name, &permission.Description, &permission.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return Permission{}, errors.Errorf("permission with key %q in event %q does not exists", key, eventID)
		}
		return Permission{}, errors.Wrap(err, "scanning permission")
	}

	return permission, nil
}

// GetPermissions returns all event's permissions.
func (s service) GetPermissions(ctx context.Context, sqlTx *sql.Tx, eventID string) ([]Permission, error) {
	q := "SELECT key, name, description FROM events_permissions WHERE event_id=$1"
	rows, err := sqlTx.QueryContext(ctx, q, eventID)
	if err != nil {
		return nil, errors.Wrap(err, "querying permissions")
	}

	var (
		permissions            []Permission
		key, name, description string
	)
	for rows.Next() {
		if err := rows.Scan(&key, &name, &description); err != nil {
			return nil, errors.Wrap(err, "scanning rows")
		}
		permissions = append(permissions, Permission{
			Key:         key,
			Name:        name,
			Description: description,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return permissions, nil
}

// GetRole returns a role in a given event.
func (s service) GetRole(ctx context.Context, sqlTx *sql.Tx, eventID, name string) (Role, error) {
	if keys, ok := ReservedRoles.GetStringSlice(name); ok {
		return Role{Name: name, PermissionKeys: keys}, nil
	}

	q := "SELECT permission_keys FROM events_roles WHERE event_id=$1 AND name=$2"
	row := sqlTx.QueryRowContext(ctx, q, eventID, name)

	role := Role{Name: name}
	if err := row.Scan(&role.PermissionKeys); err != nil {
		return Role{}, errors.Wrap(err, "scanning role permission keys")
	}

	return role, nil
}

// GetRoles returns all event's roles.
func (s service) GetRoles(ctx context.Context, sqlTx *sql.Tx, eventID string) ([]Role, error) {
	q := "SELECT name, permission_keys FROM events_roles WHERE event_id=$1"
	rows, err := sqlTx.QueryContext(ctx, q, eventID)
	if err != nil {
		return nil, errors.Wrap(err, "fetching roles")
	}

	var (
		role  Role
		roles []Role
	)
	for rows.Next() {
		if err := rows.Scan(&role.Name, &role.PermissionKeys); err != nil {
			return nil, errors.Wrap(err, "scanning fields")
		}

		roles = append(roles, role)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return roles, nil
}

// GetUserRole returns user's role inside the event.
func (s service) GetUserRole(ctx context.Context, sqlTx *sql.Tx, eventID, userID string) (Role, error) {
	q1 := "SELECT role_name FROM events_users_roles WHERE event_id=$1 AND user_id=$2"
	roleName, err := postgres.QueryString(ctx, sqlTx, q1, eventID, userID)
	if err != nil {
		return Role{}, errors.Errorf("user %q has no role in event %q", userID, eventID)
	}

	if keys, ok := ReservedRoles.GetStringSlice(roleName); ok {
		return Role{Name: roleName, PermissionKeys: keys}, nil
	}

	role, err := s.GetRole(ctx, sqlTx, eventID, roleName)
	if err != nil {
		return Role{}, err
	}

	return role, nil
}

// IsHost returns if the user's role in the events passed is host.
func (s service) IsHost(ctx context.Context, sqlTx *sql.Tx, userID string, eventIDs ...string) (bool, error) {
	q := "SELECT EXISTS(SELECT 1 FROM events_users_roles WHERE event_id=$1 AND user_id=$2 AND role_name='host')"
	var isHost bool

	stmt, err := sqlTx.PrepareContext(ctx, q)
	if err != nil {
		return false, errors.Wrap(err, "prepraring statement")
	}

	for _, eventID := range eventIDs {
		row := stmt.QueryRowContext(ctx, eventID, userID)
		if err := row.Scan(&isHost); err != nil {
			return false, err
		}
		if !isHost {
			return false, nil
		}
	}

	return true, nil
}

// SetRoles assigns a role to n users inside an event.
func (s service) SetRoles(ctx context.Context, sqlTx *sql.Tx, eventID, roleName string, userIDs ...string) error {
	public, err := postgres.QueryBool(ctx, sqlTx, "SELECT public FROM events WHERE id=$1", eventID)
	if err != nil {
		return err
	}

	if !public {
		// Verify that the users already have a role before assigning them other (ensuring they already take part in the event).
		// The only exception is the creator's host role or when the event is public.
		q1 := "SELECT EXISTS(SELECT 1 FROM events_users_roles WHERE event_id=$1 AND user_id=$2)"
		stmt, err := sqlTx.PrepareContext(ctx, q1)
		if err != nil {
			return errors.Wrap(err, "prepraring statement")
		}

		var hasRole bool // Will be overwritten on each iteration
		for _, userID := range userIDs {
			if err := stmt.QueryRowContext(ctx, eventID, userID).Scan(&hasRole); err != nil {
				return err
			}
			if !hasRole {
				return errors.Errorf("user %q is not part of the event %q", userID, eventID)
			}
		}
	}

	// TODO: use arguments or validate the role name to avoid sql injection.
	q2 := "INSERT INTO events_users_roles (event_id, user_id, role_name) VALUES"
	insert := postgres.BulkInsertRoles(q2, eventID, roleName, userIDs)
	if _, err := sqlTx.ExecContext(ctx, insert); err != nil {
		return errors.Wrap(err, "setting roles")
	}

	return nil
}

// SetViewerRole assigns the viewer role to a user.
func (s service) SetViewerRole(ctx context.Context, sqlTx *sql.Tx, eventID, userID string) error {
	q := "INSERT INTO events_users_roles (event_id, user_id, role_name) VALUES ($1, $2, $3)"
	if _, err := sqlTx.ExecContext(ctx, q, eventID, userID, Viewer); err != nil {
		return errors.Wrap(err, "setting viewer role")
	}

	return nil
}

// UpdatePemission sets new values for a permission.
func (s service) UpdatePermission(ctx context.Context, sqlTx *sql.Tx, eventID, key string, permission UpdatePermission) error {
	buf := bufferpool.Get()
	buf.WriteString("UPDATE events_roles SET")

	if permission.Name != nil {
		buf.WriteString(" name='")
		buf.WriteString(*permission.Name)
		buf.WriteString("',")
	}
	if permission.Description != nil {
		buf.WriteString(" description='")
		buf.WriteString(*permission.Description)
		buf.WriteByte('\'')
	}

	buf.WriteString(" WHERE event_id='")
	buf.WriteString(eventID)
	buf.WriteByte('\'')
	buf.WriteString("AND key='")
	buf.WriteString(key)
	buf.WriteByte('\'')

	if _, err := sqlTx.ExecContext(ctx, buf.String()); err != nil {
		return errors.Wrap(err, "updating permission")
	}

	bufferpool.Put(buf)
	return nil
}

// UpdateRole sets new values for a role.
func (s service) UpdateRole(ctx context.Context, sqlTx *sql.Tx, eventID, name string, role UpdateRole) error {
	buf := bufferpool.Get()
	buf.WriteString("UPDATE events_roles SET")

	if role.PermissionKeys != nil {
		buf.WriteString(" permission_keys='")
		buf.WriteString("'{")
		for i, key := range *role.PermissionKeys {
			if i != 0 {
				buf.WriteByte(',')
			}
			buf.WriteByte('"')
			buf.WriteString(key)
			buf.WriteByte('"')
		}
		buf.WriteString("}'")
	}

	buf.WriteString(" WHERE event_id='")
	buf.WriteString(eventID)
	buf.WriteByte('\'')
	buf.WriteString("AND name='")
	buf.WriteString(name)
	buf.WriteByte('\'')

	if _, err := sqlTx.ExecContext(ctx, buf.String()); err != nil {
		return errors.Wrap(err, "updating role")
	}

	bufferpool.Put(buf)
	return nil
}

// UserHasRole returns if the user has a role inside the event or not.
func (s service) UserHasRole(ctx context.Context, sqlTx *sql.Tx, eventID, userID string) (bool, error) {
	q := "SELECT EXISTS(SELECT 1 FROM events_users_roles WHERE event_id=$1 AND user_id=$2)"
	hasRole, err := postgres.QueryBool(ctx, sqlTx, q, eventID, userID)
	if err != nil {
		return false, err
	}

	return hasRole, nil
}
