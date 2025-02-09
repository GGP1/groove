package role

import (
	"context"
	"database/sql"

	"github.com/GGP1/groove/internal/cache"
	"github.com/GGP1/groove/internal/httperr"
	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/permissions"
	"github.com/GGP1/groove/internal/roles"
	"github.com/GGP1/groove/internal/txgroup"
	"github.com/GGP1/groove/model"
	"github.com/GGP1/groove/service/auth"
	"github.com/GGP1/groove/storage/postgres"
	"github.com/GGP1/sqan"

	"github.com/go-redis/redis/v8"
	"github.com/lib/pq"
	"github.com/pkg/errors"
)

var errAccessDenied = httperr.Forbidden("Access denied")

// Service interface for the roles service.
type Service interface {
	ClonePermissions(ctx context.Context, exporterEventID, importerEventID string) error
	CloneRoles(ctx context.Context, exporterEventID, importerEventID string) error
	CreatePermission(ctx context.Context, eventID string, permission model.Permission) error
	CreateRole(ctx context.Context, eventID string, role model.Role) error
	DeletePermission(ctx context.Context, eventID, key string) error
	DeleteRole(ctx context.Context, eventID, name string) error
	GetMembers(ctx context.Context, eventID string, params params.Query) ([]model.User, error)
	GetMembersCount(ctx context.Context, eventID string) (int64, error)
	GetMembersFriends(ctx context.Context, eventID, userID string, params params.Query) ([]model.User, error)
	GetMembersFriendsCount(ctx context.Context, eventID, userID string) (int64, error)
	GetPermission(ctx context.Context, eventID, key string) (model.Permission, error)
	GetPermissions(ctx context.Context, eventID string) ([]model.Permission, error)
	GetRole(ctx context.Context, eventID, name string) (model.Role, error)
	GetRoles(ctx context.Context, eventID string) ([]model.Role, error)
	GetUserEventsCount(ctx context.Context, userID, roleName string) (int64, error)
	GetUserRole(ctx context.Context, eventID, userID string) (model.Role, error)
	GetUsersByRole(ctx context.Context, eventID, roleName string, params params.Query) ([]model.User, error)
	GetUsersCountByRole(ctx context.Context, eventID, roleName string) (int64, error)
	GetUserFriendsByRole(ctx context.Context, eventID, userID, roleName string, params params.Query) ([]model.User, error)
	GetUserFriendsCountByRole(ctx context.Context, eventID, userID, roleName string) (int64, error)
	HasRole(ctx context.Context, eventID, userID string) (bool, error)
	IsHost(ctx context.Context, userID string, eventIDs ...string) (bool, error)
	RequirePermissions(ctx context.Context, session auth.Session, eventID string, permKeys ...string) error
	SetRole(ctx context.Context, eventID, roleName string, userIDs ...string) error
	UnsetRole(ctx context.Context, eventID, userID string) error
	UpdatePermission(ctx context.Context, eventID, key string, permission model.UpdatePermission) error
	UpdateRole(ctx context.Context, eventID, name string, role model.UpdateRole) error
}

type service struct {
	db  *sql.DB
	rdb *redis.Client
}

// NewService returns a new role service.
func NewService(db *sql.DB, rdb *redis.Client) Service {
	return &service{
		db:  db,
		rdb: rdb,
	}
}

// ClonePermissions takes the permissions from the exporter event and creates them in the importer event.
func (s *service) ClonePermissions(ctx context.Context, exporterEventID, importerEventID string) error {
	sqlTx := txgroup.SQLTx(ctx)

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
func (s *service) CloneRoles(ctx context.Context, exporterEventID, importerEventID string) error {
	sqlTx := txgroup.SQLTx(ctx)

	// Clone permissions as they are required to create roles.
	if err := s.ClonePermissions(ctx, exporterEventID, importerEventID); err != nil {
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
func (s *service) CreatePermission(ctx context.Context, eventID string, permission model.Permission) error {
	sqlTx := txgroup.SQLTx(ctx)

	q := "INSERT INTO events_permissions (event_id, key, name, description) VALUES ($1, $2, $3, $4)"
	_, err := sqlTx.ExecContext(ctx, q, eventID, permission.Key, permission.Name, permission.Description)
	if err != nil {
		return errors.Wrap(err, "creating permission")
	}

	if err := s.rdb.Del(ctx, cache.PermissionsKey(eventID)).Err(); err != nil {
		return errors.Wrap(err, "deleting permission")
	}

	return nil
}

// CreateRole creates a new role inside an event.
func (s *service) CreateRole(ctx context.Context, eventID string, role model.Role) error {
	sqlTx := txgroup.SQLTx(ctx)

	if err := s.permissionKeysExist(ctx, eventID, role.PermissionKeys); err != nil {
		return err
	}

	q := "INSERT INTO events_roles (event_id, name, permission_keys) VALUES ($1, $2, $3)"
	if _, err := sqlTx.ExecContext(ctx, q, eventID, role.Name, role.PermissionKeys); err != nil {
		return errors.Wrap(err, "creating role")
	}

	if err := s.rdb.Del(ctx, cache.RolesKey(eventID)).Err(); err != nil {
		return errors.Wrap(err, "deleting roles")
	}

	return nil
}

// DeletePermission removes a permission from the event.
func (s *service) DeletePermission(ctx context.Context, eventID, key string) error {
	sqlTx := txgroup.SQLTx(ctx)

	q := "DELETE FROM events_permissions WHERE event_id=$1 AND key=$2"
	if _, err := sqlTx.ExecContext(ctx, q, eventID, key); err != nil {
		return errors.Wrap(err, "deleting permission")
	}
	return nil
}

// DeleteRole removes a role from the event.
func (s *service) DeleteRole(ctx context.Context, eventID, name string) error {
	sqlTx := txgroup.SQLTx(ctx)

	q := "DELETE FROM events_roles WHERE event_id=$1 AND name=$2"
	if _, err := sqlTx.ExecContext(ctx, q, eventID, name); err != nil {
		return errors.Wrap(err, "deleting role")
	}
	return nil
}

// GetMembers returns a list with all the members (non-viewers) of an event. TODO: add their roles to the struct returned.
func (s *service) GetMembers(ctx context.Context, eventID string, params params.Query) ([]model.User, error) {
	q := `SELECT {fields} FROM {table} WHERE id IN 
	(SELECT user_id FROM events_users_roles WHERE event_id=$1 AND role_name != $2)
	{pag}`
	query := postgres.Select(model.T.User, q, params)
	rows, err := s.db.QueryContext(ctx, query, eventID, roles.Viewer)
	if err != nil {
		return nil, err
	}

	var users []model.User
	if err := sqan.Rows(&users, rows); err != nil {
		return nil, err
	}

	return users, nil
}

// GetMembersCount returns the number of members (non-viewers) of an event.
func (s *service) GetMembersCount(ctx context.Context, eventID string) (int64, error) {
	q := "SELECT COUNT(*) FROM events_users_roles WHERE event_id=$1 AND role_name != $2"
	return postgres.Query[int64](ctx, s.db, q, eventID, roles.Viewer)
}

// GetMembersFriends returns the members of an event that are friends of userID.
func (s *service) GetMembersFriends(ctx context.Context, eventID, userID string, params params.Query) ([]model.User, error) {
	q := `SELECT {fields} FROM {table} WHERE id IN (
		SELECT user_id FROM events_users_roles WHERE event_id=$1 AND role_name != $2
		INTERSECT
		(
			SELECT friend_id FROM users_friends WHERE user_id=$3
			UNION
			SELECT user_id FROM users_friends WHERE friend_id=$3
		)
	) {pag}`
	query := postgres.Select(model.T.User, q, params)

	rows, err := s.db.QueryContext(ctx, query, eventID, roles.Viewer, userID)
	if err != nil {
		return nil, err
	}

	var users []model.User
	if err := sqan.Rows(&users, rows); err != nil {
		return nil, err
	}

	return users, nil
}

// GetMembersFriendsCount returns the count of the members of an event that are friends of userID.
func (s *service) GetMembersFriendsCount(ctx context.Context, eventID, userID string) (int64, error) {
	q := `SELECT COUNT(*) FROM events_users_roles WHERE event_id=$1 AND user_id IN (
		SELECT user_id FROM events_users_roles WHERE event_id=$1 AND role_name != $2
		INTERSECT
		(
			SELECT friend_id FROM users_friends WHERE user_id=$3
			UNION
			SELECT user_id FROM users_friends WHERE friend_id=$3
		)
	)`
	return postgres.Query[int64](ctx, s.db, q, eventID, roles.Viewer, userID)
}

// GetPermission returns a permission from an event with the given key.
func (s *service) GetPermission(ctx context.Context, eventID, key string) (model.Permission, error) {
	q := "SELECT name, description, created_at FROM events_permissions WHERE event_id=$1 AND key=$2"
	rows, err := s.db.QueryContext(ctx, q, eventID, key)
	if err != nil {
		return model.Permission{}, err
	}

	permission := model.Permission{Key: key}
	if err := sqan.Row(&permission, rows); err != nil {
		return model.Permission{}, errors.Wrap(err, "scanning permission")
	}

	return permission, nil
}

// GetPermissions returns all event's permissions.
func (s *service) GetPermissions(ctx context.Context, eventID string) ([]model.Permission, error) {
	q := "SELECT key, name, description FROM events_permissions WHERE event_id=$1"
	rows, err := s.db.QueryContext(ctx, q, eventID)
	if err != nil {
		return nil, errors.Wrap(err, "querying permissions")
	}

	var permissions []model.Permission
	if err := sqan.Rows(&permissions, rows); err != nil {
		return nil, err
	}

	return permissions, nil
}

// GetRole returns a role in a given event.
func (s *service) GetRole(ctx context.Context, eventID, name string) (model.Role, error) {
	if keys, ok := roles.Reserved.Get(name); ok {
		return model.Role{Name: name, PermissionKeys: keys}, nil
	}

	q := "SELECT permission_keys FROM events_roles WHERE event_id=$1 AND name=$2"
	row := s.db.QueryRowContext(ctx, q, eventID, name)

	role := model.Role{Name: name}
	if err := row.Scan(&role.PermissionKeys); err != nil {
		return model.Role{}, errors.Wrap(err, "scanning role permission keys")
	}

	return role, nil
}

// GetRoles returns all event's roles.
func (s *service) GetRoles(ctx context.Context, eventID string) ([]model.Role, error) {
	q := "SELECT name, permission_keys FROM events_roles WHERE event_id=$1"
	rows, err := s.db.QueryContext(ctx, q, eventID)
	if err != nil {
		return nil, errors.Wrap(err, "querying roles")
	}

	var roles []model.Role
	if err := sqan.Rows(&roles, rows); err != nil {
		return nil, errors.Wrap(err, "scanning roles")
	}
	// Prepend reserved roles
	roles = append(model.ReservedRoles, roles...)

	return roles, nil
}

// GetUserEventsCount returns the number of events the user has a role in.
func (s *service) GetUserEventsCount(ctx context.Context, userID, roleName string) (int64, error) {
	q := "SELECT COUNT(*) FROM events_users_roles WHERE user_id=$1 AND role_name=$2"
	return postgres.Query[int64](ctx, s.db, q, userID, roleName)
}

// GetUserRole returns user's role inside the event.
func (s *service) GetUserRole(ctx context.Context, eventID, userID string) (model.Role, error) {
	q1 := "SELECT role_name FROM events_users_roles WHERE event_id=$1 AND user_id=$2"
	roleName, err := postgres.Query[string](ctx, s.db, q1, eventID, userID)
	if err != nil {
		return model.Role{}, errors.Errorf("user %q has no role in event %q", userID, eventID)
	}

	role, err := s.GetRole(ctx, eventID, roleName)
	if err != nil {
		return model.Role{}, err
	}

	return role, nil
}

// GetUsersByRole returns the users with the specified role in the event.
func (s *service) GetUsersByRole(ctx context.Context, eventID, roleName string, params params.Query) ([]model.User, error) {
	q := `SELECT {fields} FROM {table} WHERE id IN 
	(SELECT user_id FROM events_users_roles WHERE event_id=$1 AND role_name=$2)
	{pag}`
	query := postgres.Select(model.T.User, q, params)
	rows, err := s.db.QueryContext(ctx, query, eventID, roleName)
	if err != nil {
		return nil, err
	}

	var users []model.User
	if err := sqan.Rows(&users, rows); err != nil {
		return nil, err
	}

	return users, nil
}

// GetUsersCountByRole returns the number of users with the specified role in the event.
func (s *service) GetUsersCountByRole(ctx context.Context, eventID, roleName string) (int64, error) {
	q := "SELECT COUNT(*) FROM events_users_roles WHERE event_id=$1 AND role_name=$2"
	return postgres.Query[int64](ctx, s.db, q, eventID, roleName)
}

// GetUserFriendsByRole returns a user's friends with a determined role in an event.
func (s *service) GetUserFriendsByRole(ctx context.Context, eventID, userID, roleName string, params params.Query) ([]model.User, error) {
	sqlTx := txgroup.SQLTx(ctx)

	q := `SELECT {fields} FROM {table} WHERE id IN (
		SELECT user_id FROM events_users_roles WHERE event_id=$1 AND role_name=$2
		INTERSECT
		(
			SELECT friend_id FROM users_friends WHERE user_id=$3
			UNION
			SELECT user_id FROM users_friends WHERE friend_id=$3
		)
	) {pag}`
	query := postgres.Select(model.T.User, q, params)

	rows, err := sqlTx.QueryContext(ctx, query, eventID, roleName, userID)
	if err != nil {
		return nil, err
	}

	var users []model.User
	if err := sqan.Rows(&users, rows); err != nil {
		return nil, err
	}

	return users, nil
}

// GetUserFriendsCountByRole returns the count of the users with a role in an event that are friends of user.
func (s *service) GetUserFriendsCountByRole(ctx context.Context, eventID, userID, roleName string) (int64, error) {
	q := `SELECT COUNT(*) FROM events_users_roles WHERE 
	event_id=$1 AND 
	role_name=$2 AND 
	user_id IN (
		SELECT friend_id FROM users_friends WHERE user_id=$3
		UNION
		SELECT user_id FROM users_friends WHERE friend_id=$3
	)`
	return postgres.Query[int64](ctx, s.db, q, eventID, roleName, userID)
}

// HasRole returns if the user has a role inside the event or not (if the user is a member).
func (s *service) HasRole(ctx context.Context, eventID, userID string) (bool, error) {
	q := "SELECT EXISTS(SELECT 1 FROM events_users_roles WHERE event_id=$1 AND user_id=$2)"
	return postgres.Query[bool](ctx, s.db, q, eventID, userID)
}

// IsHost returns if the user's role in the events passed is host.
func (s *service) IsHost(ctx context.Context, userID string, eventIDs ...string) (bool, error) {
	sqlTx := txgroup.SQLTx(ctx)

	q := "SELECT EXISTS(SELECT 1 FROM events_users_roles WHERE event_id=$1 AND user_id=$2 AND role_name='host')"
	stmt, err := sqlTx.PrepareContext(ctx, q)
	if err != nil {
		return false, errors.Wrap(err, "prepraring statement")
	}
	defer stmt.Close()

	var isHost bool
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

// RequirePermission returns an error if the user does not have the required permissions to proceed.
func (s *service) RequirePermissions(ctx context.Context, session auth.Session, eventID string, permKeys ...string) error {
	if len(permKeys) == 0 {
		return nil
	}

	role, err := s.GetUserRole(ctx, eventID, session.ID)
	if err != nil {
		return errors.Wrap(err, "require permissions: querying user role")
	}

	userPermKeys := sliceToMap(role.PermissionKeys)
	if err := permissions.Require(userPermKeys, permKeys...); err != nil {
		return errAccessDenied
	}

	return nil
}

// SetRole assigns a role to n users inside an event. TODO: improve readability.
func (s *service) SetRole(ctx context.Context, eventID, roleName string, userIDs ...string) error {
	sqlTx := txgroup.SQLTx(ctx)

	q := "SELECT EXISTS(SELECT 1 FROM users WHERE type=$1 AND id = ANY($2))"
	row := sqlTx.QueryRowContext(ctx, q, model.Business, pq.Array(userIDs))
	var exists bool
	if err := row.Scan(&exists); err != nil {
		return errors.Wrap(err, "scanning user types")
	}
	if exists {
		return httperr.Forbidden("a bussiness can't take part in third party events")
	}

	row2 := sqlTx.QueryRowContext(ctx, "SELECT public FROM events WHERE id=$1", eventID)
	var public bool
	if err := row2.Scan(&public); err != nil {
		return errors.Wrap(err, "scanning public")
	}

	if !public {
		// In a private event, the users first need to have a role ("viewer" if they are invited).
		q := "SELECT EXISTS(SELECT 1 FROM events_users_roles WHERE event_id=$1 AND user_id=$2)"
		stmt, err := sqlTx.PrepareContext(ctx, q)
		if err != nil {
			return errors.Wrap(err, "preparing role statement")
		}
		defer stmt.Close()

		var hasRole bool // Reuse
		for _, userID := range userIDs {
			if err := stmt.QueryRowContext(ctx, eventID, userID).Scan(&hasRole); err != nil {
				return err
			}
			if !hasRole {
				return errors.Errorf("user %q is not part of the event %q", userID, eventID)
			}
		}

		if err := stmt.Close(); err != nil {
			return err
		}
	}

	stmt2, err := postgres.BulkInsert(ctx, sqlTx, "events_users_roles", "event_id", "user_id", "role_name")
	if err != nil {
		return err
	}
	defer stmt2.Close()

	for _, userID := range userIDs {
		if _, err := stmt2.ExecContext(ctx, eventID, userID, roleName); err != nil {
			return errors.Wrap(err, "setting role")
		}
	}

	// Flush buffered data
	if _, err := stmt2.ExecContext(ctx); err != nil {
		return errors.Wrap(err, "flushing buffered data")
	}

	return nil
}

// UnsetRole removes the role a user had in the event.
func (s *service) UnsetRole(ctx context.Context, eventID, userID string) error {
	sqlTx := txgroup.SQLTx(ctx)

	q := "DELETE FROM events_users_roles WHERE event_id=$1 AND user_id=$2"
	if _, err := sqlTx.ExecContext(ctx, q, eventID, userID); err != nil {
		return errors.Wrap(err, "unsetting role")
	}

	return nil
}

// UpdatePemission sets new values for a permission.
func (s *service) UpdatePermission(ctx context.Context, eventID, key string, permission model.UpdatePermission) error {
	sqlTx := txgroup.SQLTx(ctx)

	q := `UPDATE events_permissions SET 
	name = COALESCE($3,name),
	description = COALESCE($4,description),
	key = COALESCE($5,key)
	WHERE event_id=$1 AND key=$2`
	_, err := sqlTx.ExecContext(ctx, q, eventID, key, permission.Name, permission.Description, permission.Key)
	if err != nil {
		return errors.Wrap(err, "updating permission")
	}

	return nil
}

// UpdateRole sets new values for a role.
func (s *service) UpdateRole(ctx context.Context, eventID, name string, role model.UpdateRole) error {
	sqlTx := txgroup.SQLTx(ctx)

	if err := s.permissionKeysExist(ctx, eventID, *role.PermissionKeys); err != nil {
		return err
	}

	q := `UPDATE events_roles SET
	name = COALESCE($3,name),
	permission_keys = COALESCE($4,permission_keys)
	WHERE event_id=$1 AND name=$2`
	if _, err := sqlTx.ExecContext(ctx, q, eventID, name, role.Name, role.PermissionKeys); err != nil {
		return errors.Wrap(err, "updating role")
	}

	return nil
}

func (s *service) permissionKeysExist(ctx context.Context, eventID string, permissionKeys pq.StringArray) error {
	if len(permissionKeys) == 0 {
		return nil
	}

	q := "SELECT EXISTS(SELECT 1 FROM events_permissions WHERE event_id=$1 AND key=$2)"
	stmt, err := s.db.PrepareContext(ctx, q)
	if err != nil {
		return errors.Wrap(err, "preparing statement")
	}
	defer stmt.Close()

	var exists bool // Reuse
	for _, key := range permissionKeys {
		if permissions.Reserved.Exists(key) {
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

	return nil
}

func sliceToMap(slice []string) map[string]struct{} {
	mp := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		mp[s] = struct{}{}
	}
	return mp
}
