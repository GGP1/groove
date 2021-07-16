package role

import (
	"context"
	"database/sql"

	"github.com/GGP1/groove/internal/permissions"
	"github.com/GGP1/groove/storage/postgres"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/pkg/errors"
)

// TODO: create UpdatePermission and UpdateRole
// Use users sets instead of cloning roles and permissions?

// TODO: implement events_roles_defaults and events_permissions_defaults tables
// with pre-defined roles and permissions populated.
// When checking for their details, check default tables if "name" or "key" is a default one
// also do not let to overwrite them

// Service interface for the roles service.
type Service interface {
	ClonePermissions(ctx context.Context, sqlTx *sql.Tx, exporterEventID, importerEventID string) error
	CloneRoles(ctx context.Context, sqlTx *sql.Tx, exporterEventID, importerEventID string) error
	CreatePermission(ctx context.Context, sqlTx *sql.Tx, eventID string, permission Permission) error
	CreateRole(ctx context.Context, sqlTx *sql.Tx, eventID string, role Role) error
	GetPermissions(ctx context.Context, sqlTx *sql.Tx, eventID string) ([]Permission, error)
	GetRole(ctx context.Context, sqlTx *sql.Tx, eventID, name string) (Role, error)
	GetRoles(ctx context.Context, sqlTx *sql.Tx, eventID string) ([]Role, error)
	GetUserRole(ctx context.Context, sqlTx *sql.Tx, eventID, userID string) (Role, error)
	IsHost(ctx context.Context, sqlTx *sql.Tx, userID string, eventIDs ...string) (bool, error)
	SetRoles(ctx context.Context, sqlTx *sql.Tx, eventID, roleName string, userIDs ...string) error
	UpdatePermission(ctx context.Context, sqlTx *sql.Tx, eventID string, permission Permission) error
	UpdateRole(ctx context.Context, sqlTx *sql.Tx, eventID string, role Role) error
	UserHasRole(ctx context.Context, sqlTx *sql.Tx, eventID, userID string) (bool, error)
}

type service struct {
	db *sql.DB
	mc *memcache.Client
}

// NewService returns a new role service.
func NewService(db *sql.DB, mc *memcache.Client) Service {
	return service{
		db: db,
		mc: mc,
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

	if err := s.mc.Delete(eventID + "_permissions"); err != nil && err != memcache.ErrCacheMiss {
		return errors.Wrap(err, "memcached: deleting event")
	}

	return nil
}

// CreateRole creates a new role inside an event.
func (s service) CreateRole(ctx context.Context, sqlTx *sql.Tx, eventID string, role Role) error {
	q1 := "SELECT EXISTS(SELECT 1 FROM events_permissions WHERE event_id=$1 AND key=$2)"
	exists := false

	// Check for the existence of the keys used for the role
	for pk := range role.PermissionKeys {
		row := sqlTx.QueryRowContext(ctx, q1, eventID, pk)
		if err := row.Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return errors.Errorf("permission with key %q does not exist", pk)
		}
	}

	parsedKeys := permissions.ParseKeys(role.PermissionKeys)
	q2 := "INSERT INTO events_roles (event_id, name, permissions_keys) VALUES ($1, $2, $3)"
	if _, err := sqlTx.ExecContext(ctx, q2, eventID, role.Name, parsedKeys); err != nil {
		return errors.Wrap(err, "creating role")
	}

	if err := s.mc.Delete(eventID + "_roles"); err != nil && err != memcache.ErrCacheMiss {
		return errors.Wrap(err, "memcached: deleting event")
	}

	return nil
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
	q := "SELECT permissions_keys FROM events_roles WHERE event_id=$1 AND name=$2"
	permissionKeys, err := postgres.QueryString(ctx, sqlTx, q, eventID, name)
	if err != nil {
		return Role{}, errors.Wrap(err, "fetching permissions_keys")
	}

	role := Role{
		Name:           name,
		PermissionKeys: permissions.UnparseKeys(permissionKeys),
	}
	return role, nil
}

// GetRoles returns all event's roles.
func (s service) GetRoles(ctx context.Context, sqlTx *sql.Tx, eventID string) ([]Role, error) {
	q := "SELECT name, permissions_keys FROM events_roles WHERE event_id=$1"
	rows, err := sqlTx.QueryContext(ctx, q, eventID)
	if err != nil {
		return nil, errors.Wrap(err, "fetching roles")
	}

	var (
		parsedRoles          []Role
		name, permissionKeys string
	)
	for rows.Next() {
		if err := rows.Scan(&name, &permissionKeys); err != nil {
			return nil, errors.Wrap(err, "scanning fields")
		}

		role := Role{Name: name, PermissionKeys: permissions.UnparseKeys(permissionKeys)}
		parsedRoles = append(parsedRoles, role)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return parsedRoles, nil
}

// GetUserRole returns user's role inside the event.
func (s service) GetUserRole(ctx context.Context, sqlTx *sql.Tx, eventID, userID string) (Role, error) {
	q1 := "SELECT role_name FROM events_users_roles WHERE event_id=$1 AND user_id=$2"
	roleName, err := postgres.QueryString(ctx, sqlTx, q1, eventID, userID)
	if err != nil {
		return Role{}, errors.Errorf("user %q has no role in event %q", userID, eventID)
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

	for _, eventID := range eventIDs {
		row := sqlTx.QueryRowContext(ctx, q, eventID, userID)
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
	q := "INSERT INTO events_users_roles (event_id, user_id, role_name) VALUES"
	insert := postgres.BulkInsertRoles(q, eventID, roleName, userIDs)

	if _, err := sqlTx.ExecContext(ctx, insert); err != nil {
		return errors.Wrap(err, "setting roles")
	}

	return nil
}

// UpdatePemission ..
func (s service) UpdatePermission(ctx context.Context, sqlTx *sql.Tx, eventID string, permission Permission) error {
	return nil
}

// UpdateRole ..
func (s service) UpdateRole(ctx context.Context, sqlTx *sql.Tx, eventID string, role Role) error {
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
