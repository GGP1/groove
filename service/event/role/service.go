package role

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	"github.com/GGP1/groove/internal/bufferpool"
	"github.com/GGP1/groove/internal/cache"
	"github.com/GGP1/groove/internal/httperr"
	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/permissions"
	"github.com/GGP1/groove/internal/roles"
	"github.com/GGP1/groove/internal/sanitize"
	"github.com/GGP1/groove/internal/txgroup"
	"github.com/GGP1/groove/model"
	"github.com/GGP1/groove/service/auth"
	"github.com/GGP1/groove/storage/dgraph"
	"github.com/GGP1/groove/storage/postgres"
	"github.com/GGP1/sqan"

	"github.com/dgraph-io/dgo/v210"
	"github.com/pkg/errors"
)

var errAccessDenied = httperr.Forbidden("Access denied")

// Service interface for the roles service.
type Service interface {
	ClonePermissions(ctx context.Context, exporterEventID, importerEventID string) error
	CloneRoles(ctx context.Context, exporterEventID, importerEventID string) error
	CreatePermission(ctx context.Context, eventID string, permission Permission) error
	CreateRole(ctx context.Context, eventID string, role Role) error
	DeletePermission(ctx context.Context, eventID, key string) error
	DeleteRole(ctx context.Context, eventID, name string) error
	GetMembers(ctx context.Context, eventID string, params params.Query) ([]model.ListUser, error)
	GetMembersCount(ctx context.Context, eventID string) (int64, error)
	GetMembersFriends(ctx context.Context, eventID, userID string, params params.Query) ([]model.ListUser, error)
	GetMembersFriendsCount(ctx context.Context, eventID, userID string) (int64, error)
	GetPermission(ctx context.Context, eventID, key string) (Permission, error)
	GetPermissions(ctx context.Context, eventID string) ([]Permission, error)
	GetRole(ctx context.Context, eventID, name string) (Role, error)
	GetRoles(ctx context.Context, eventID string) ([]Role, error)
	GetUserEventsCount(ctx context.Context, userID, roleName string) (int64, error)
	GetUserRole(ctx context.Context, eventID, userID string) (Role, error)
	GetUsersByRole(ctx context.Context, eventID, roleName string, params params.Query) ([]model.ListUser, error)
	GetUsersCountByRole(ctx context.Context, eventID, roleName string) (int64, error)
	GetUserFriendsByRole(ctx context.Context, eventID, userID, roleName string, params params.Query) ([]model.ListUser, error)
	GetUserFriendsCountByRole(ctx context.Context, eventID, userID, roleName string) (int64, error)
	HasRole(ctx context.Context, eventID, userID string) (bool, error)
	IsHost(ctx context.Context, userID string, eventIDs ...string) (bool, error)
	PrivacyFilter(ctx context.Context, r *http.Request, eventID string) error
	RequirePermissions(ctx context.Context, r *http.Request, eventID string, permKeys ...string) error
	SetRole(ctx context.Context, eventID string, setRole SetRole) error
	SetReservedRole(ctx context.Context, eventID, userID string, roleName roles.Name) error
	UnsetRole(ctx context.Context, eventID, userID string) error
	UpdatePermission(ctx context.Context, eventID, key string, permission UpdatePermission) error
	UpdateRole(ctx context.Context, eventID, name string, role UpdateRole) error
}

type service struct {
	db    *sql.DB
	dc    *dgo.Dgraph
	cache cache.Client
}

// NewService returns a new role service.
func NewService(db *sql.DB, dc *dgo.Dgraph, cache cache.Client) Service {
	return service{
		db:    db,
		dc:    dc,
		cache: cache,
	}
}

// ClonePermissions takes the permissions from the exporter event and creates them in the importer event.
func (s service) ClonePermissions(ctx context.Context, exporterEventID, importerEventID string) error {
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
func (s service) CloneRoles(ctx context.Context, exporterEventID, importerEventID string) error {
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
func (s service) CreatePermission(ctx context.Context, eventID string, permission Permission) error {
	sanitize.Strings(&permission.Name)
	permission.Key = strings.ToLower(permission.Key)
	if err := permission.Validate(); err != nil {
		return httperr.BadRequest(err.Error())
	}

	q := "INSERT INTO events_permissions (event_id, key, name, description) VALUES ($1, $2, $3, $4)"
	sqlTx := txgroup.SQLTx(ctx)
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
func (s service) CreateRole(ctx context.Context, eventID string, role Role) error {
	role.Name = strings.ToLower(role.Name)
	if err := role.Validate(); err != nil {
		return httperr.BadRequest(err.Error())
	}

	q1 := "SELECT EXISTS(SELECT 1 FROM events_permissions WHERE event_id=$1 AND key=$2)"
	exists := false

	sqlTx := txgroup.SQLTx(ctx)
	stmt, err := sqlTx.PrepareContext(ctx, q1)
	if err != nil {
		return errors.Wrap(err, "preparing statement")
	}
	defer stmt.Close()

	// Check for the existence of the keys used for the role
	for _, key := range role.PermissionKeys {
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
func (s service) DeletePermission(ctx context.Context, eventID, key string) error {
	sqlTx := txgroup.SQLTx(ctx)

	q := "DELETE FROM events_permissions WHERE event_id=$1 AND key=$2"
	if _, err := sqlTx.ExecContext(ctx, q, eventID, key); err != nil {
		return errors.Wrap(err, "deleting permission")
	}
	return nil
}

// DeleteRole removes a role from the event.
func (s service) DeleteRole(ctx context.Context, eventID, name string) error {
	sqlTx := txgroup.SQLTx(ctx)

	q := "DELETE FROM events_roles WHERE event_id=$1 AND name=$2"
	if _, err := sqlTx.ExecContext(ctx, q, eventID, name); err != nil {
		return errors.Wrap(err, "deleting role")
	}
	return nil
}

// GetMembers returns a list with all the members (non-viewers) of an event.
func (s service) GetMembers(ctx context.Context, eventID string, params params.Query) ([]model.ListUser, error) {
	whereCond := "id IN (SELECT user_id FROM events_users_roles WHERE event_id=$1 AND role_name != $2)"
	q := postgres.SelectWhere(model.User, whereCond, "id", params)
	rows, err := s.db.QueryContext(ctx, q, eventID, roles.Viewer)
	if err != nil {
		return nil, err
	}

	var users []model.ListUser
	if err := sqan.Rows(&users, rows); err != nil {
		return nil, err
	}

	return users, nil
}

// GetMembersCount returns the number of members (non-viewers) of an event.
func (s service) GetMembersCount(ctx context.Context, eventID string) (int64, error) {
	q := "SELECT COUNT(*) FROM events_users_roles WHERE event_id=$1 AND role_name != $2"
	return postgres.QueryInt(ctx, s.db, q, eventID, roles.Viewer)
}

// GetMembersFriends returns the members of an event that are friends of userID.
func (s service) GetMembersFriends(ctx context.Context, eventID, userID string, params params.Query) ([]model.ListUser, error) {
	friendsIDs, err := s.getFriendsIDsWithParams(ctx, userID, params)
	if err != nil {
		return nil, err
	}

	q := selectFriendsQuery(friendsIDs, params.Fields, false)
	rows, err := s.db.QueryContext(ctx, q, eventID, roles.Viewer)
	if err != nil {
		return nil, err
	}

	var users []model.ListUser
	if err := sqan.Rows(&users, rows); err != nil {
		return nil, err
	}

	return users, nil
}

// GetMembersFriendsCount returns the count of the members of an event that are friends of userID.
func (s service) GetMembersFriendsCount(ctx context.Context, eventID, userID string) (int64, error) {
	friendsIDs, err := s.getFriendsIDs(ctx, userID)
	if err != nil {
		return 0, err
	}

	query := "SELECT COUNT(*) FROM events_users_roles WHERE event_id=$1 AND role_name != $2 AND user_id"
	q := postgres.AppendInIDs(query, friendsIDs)
	return postgres.QueryInt(ctx, s.db, q, eventID, roles.Viewer)
}

// GetPermission returns a permission from an event with the given key.
func (s service) GetPermission(ctx context.Context, eventID, key string) (Permission, error) {
	q := "SELECT name, description, created_at FROM events_permissions WHERE event_id=$1 AND key=$2"
	rows, err := s.db.QueryContext(ctx, q, eventID, key)
	if err != nil {
		return Permission{}, err
	}

	permission := Permission{Key: key}
	if err := sqan.Row(&permission, rows); err != nil {
		return Permission{}, errors.Wrap(err, "scanning permission")
	}

	return permission, nil
}

// GetPermissions returns all event's permissions.
func (s service) GetPermissions(ctx context.Context, eventID string) ([]Permission, error) {
	q := "SELECT key, name, description FROM events_permissions WHERE event_id=$1"
	rows, err := s.db.QueryContext(ctx, q, eventID)
	if err != nil {
		return nil, errors.Wrap(err, "querying permissions")
	}

	var permissions []Permission
	if err := sqan.Rows(&permissions, rows); err != nil {
		return nil, err
	}

	return permissions, nil
}

// GetRole returns a role in a given event.
func (s service) GetRole(ctx context.Context, eventID, name string) (Role, error) {
	if keys, ok := roles.Reserved.GetStringSlice(name); ok {
		return Role{Name: name, PermissionKeys: keys}, nil
	}

	q := "SELECT permission_keys FROM events_roles WHERE event_id=$1 AND name=$2"
	row := s.db.QueryRowContext(ctx, q, eventID, name)

	role := Role{Name: name}
	if err := row.Scan(&role.PermissionKeys); err != nil {
		return Role{}, errors.Wrap(err, "scanning role permission keys")
	}

	return role, nil
}

// GetRoles returns all event's roles.
func (s service) GetRoles(ctx context.Context, eventID string) ([]Role, error) {
	q := "SELECT name, permission_keys FROM events_roles WHERE event_id=$1"
	rows, err := s.db.QueryContext(ctx, q, eventID)
	if err != nil {
		return nil, errors.Wrap(err, "fetching roles")
	}

	var roles []Role
	if err := sqan.Rows(&roles, rows); err != nil {
		return nil, errors.Wrap(err, "scanning roles")
	}
	// Prepend reserved roles
	roles = append(reservedRoles, roles...)

	return roles, nil
}

// GetUserEventsCount returns the number of events the user has a role in.
func (s service) GetUserEventsCount(ctx context.Context, userID, roleName string) (int64, error) {
	q := "SELECT COUNT(*) FROM events_users_roles WHERE user_id=$1 AND role_name=$2"
	return postgres.QueryInt(ctx, s.db, q, userID, roleName)
}

// GetUserRole returns user's role inside the event.
func (s service) GetUserRole(ctx context.Context, eventID, userID string) (Role, error) {
	q1 := "SELECT role_name FROM events_users_roles WHERE event_id=$1 AND user_id=$2"
	roleName, err := postgres.QueryString(ctx, s.db, q1, eventID, userID)
	if err != nil {
		return Role{}, errors.Errorf("user %q has no role in event %q", userID, eventID)
	}

	if keys, ok := roles.Reserved.GetStringSlice(roleName); ok {
		return Role{Name: roleName, PermissionKeys: keys}, nil
	}

	role, err := s.GetRole(ctx, eventID, roleName)
	if err != nil {
		return Role{}, err
	}

	return role, nil
}

// GetUsersByRole returns the users with the specified role in the event.
func (s service) GetUsersByRole(ctx context.Context, eventID, roleName string, params params.Query) ([]model.ListUser, error) {
	sqlTx := txgroup.SQLTx(ctx)

	whereCond := "id IN (SELECT user_id FROM events_users_roles WHERE event_id=$1 AND role_name=$2)"
	q := postgres.SelectWhere(model.User, whereCond, "id", params)
	rows, err := sqlTx.QueryContext(ctx, q, eventID, roleName)
	if err != nil {
		return nil, err
	}

	var users []model.ListUser
	if err := sqan.Rows(&users, rows); err != nil {
		return nil, err
	}

	return users, nil
}

// GetUsersCountByRole returns the number of users with the specified role in the event.
func (s service) GetUsersCountByRole(ctx context.Context, eventID, roleName string) (int64, error) {
	q := "SELECT COUNT(*) FROM events_users_roles WHERE event_id=$1 AND role_name=$2"
	return postgres.QueryInt(ctx, s.db, q, eventID, roleName)
}

// GetUserFriendsByRole returns a user's friends with a determined role in an event.
func (s service) GetUserFriendsByRole(ctx context.Context, eventID, userID, roleName string, params params.Query) ([]model.ListUser, error) {
	sqlTx := txgroup.SQLTx(ctx)

	friendsIDs, err := s.getFriendsIDsWithParams(ctx, userID, params)
	if err != nil {
		return nil, err
	}

	q := selectFriendsQuery(friendsIDs, params.Fields, true)
	rows, err := sqlTx.QueryContext(ctx, q, eventID, roleName)
	if err != nil {
		return nil, err
	}

	var users []model.ListUser
	if err := sqan.Rows(&users, rows); err != nil {
		return nil, err
	}

	return users, nil
}

// GetUserFriendsCountByRole returns the count of the users with a role in an event that are friends of user.
func (s service) GetUserFriendsCountByRole(ctx context.Context, eventID, userID, roleName string) (int64, error) {
	friendsIDs, err := s.getFriendsIDs(ctx, userID)
	if err != nil {
		return 0, err
	}

	query := "SELECT COUNT(*) FROM events WHERE event_id=$1 AND role_name=$2 AND id"
	q := postgres.AppendInIDs(query, friendsIDs)
	return postgres.QueryInt(ctx, s.db, q, eventID, roleName)
}

// HasRole returns if the user has a role inside the event or not (if the user is a member).
func (s service) HasRole(ctx context.Context, eventID, userID string) (bool, error) {
	q := "SELECT EXISTS(SELECT 1 FROM events_users_roles WHERE event_id=$1 AND user_id=$2)"
	return postgres.QueryBool(ctx, s.db, q, eventID, userID)
}

// IsHost returns if the user's role in the events passed is host.
func (s service) IsHost(ctx context.Context, userID string, eventIDs ...string) (bool, error) {
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

// privacyFilter lets through only users that can fetch the event data if it's private,
// if it's public it lets anyone in. TODO: try to cache
func (s service) PrivacyFilter(ctx context.Context, r *http.Request, eventID string) error {
	session, err := auth.GetSession(ctx, r)
	if err != nil {
		return err
	}

	row := s.db.QueryRowContext(ctx, "SELECT public FROM events WHERE id=$1", eventID)
	var isPublic bool
	if err := row.Scan(&isPublic); err != nil {
		return errors.Wrap(err, "privacy filter: fetching event visibility")
	}
	if isPublic {
		// Event is public, no restrictions applied
		return nil
	}

	// If the user has a role in the event, then he's able to retrieve its information
	hasRole, err := s.HasRole(ctx, eventID, session.ID)
	if err != nil {
		return errors.Wrap(err, "privacy filter: fetching user role")
	}
	if !hasRole {
		return errAccessDenied
	}

	return nil
}

// RequirePermission returns an error if the user does not have the required permissions to proceed.
func (s service) RequirePermissions(ctx context.Context, r *http.Request, eventID string, permKeys ...string) error {
	if len(permKeys) == 0 {
		return nil
	}

	session, err := auth.GetSession(ctx, r)
	if err != nil {
		return err
	}

	role, err := s.GetUserRole(ctx, eventID, session.ID)
	if err != nil {
		return errors.Wrap(err, "require permissions: fetching user role")
	}

	userPermKeys := sliceToMap(role.PermissionKeys)
	if err := permissions.Require(userPermKeys, permKeys...); err != nil {
		return errAccessDenied
	}

	return nil
}

// SetRole assigns a role to n users inside an event.
func (s service) SetRole(ctx context.Context, eventID string, role SetRole) error {
	if err := role.Validate(); err != nil {
		return httperr.BadRequest(err.Error())
	}

	sqlTx := txgroup.SQLTx(ctx)
	row := sqlTx.QueryRowContext(ctx, "SELECT public FROM events WHERE id=$1", eventID)

	var public bool
	if err := row.Scan(&public); err != nil {
		return errors.Wrap(err, "scanning public")
	}

	if !public {
		// In a private event, the users first need to have a role ("viewer" if they are invited).
		q1 := "SELECT EXISTS(SELECT 1 FROM events_users_roles WHERE event_id=$1 AND user_id=$2)"
		stmt1, err := sqlTx.PrepareContext(ctx, q1)
		if err != nil {
			return errors.Wrap(err, "prepraring statement")
		}
		defer stmt1.Close()

		var hasRole bool // Reuse
		for _, userID := range role.UserIDs {
			if err := stmt1.QueryRowContext(ctx, eventID, userID).Scan(&hasRole); err != nil {
				return err
			}
			if !hasRole {
				return errors.Errorf("user %q is not part of the event %q", userID, eventID)
			}
		}
	}

	stmt2, err := postgres.BulkInsert(ctx, sqlTx, "events_users_roles", "event_id", "user_id", "role_name")
	if err != nil {
		return err
	}
	defer stmt2.Close()

	for _, userID := range role.UserIDs {
		if _, err := stmt2.ExecContext(ctx, eventID, userID, role.RoleName); err != nil {
			return errors.Wrap(err, "setting roles")
		}
	}

	// Flush buffered data
	if _, err := stmt2.ExecContext(ctx); err != nil {
		return errors.Wrap(err, "flushing buffered data")
	}

	return nil
}

// SetReservedRole assigns a reserved role to a user.
func (s service) SetReservedRole(ctx context.Context, eventID, userID string, roleName roles.Name) error {
	// I'd prefer to use the user service here but it's not possible
	sqlTx := txgroup.SQLTx(ctx)
	row := sqlTx.QueryRowContext(ctx, "SELECT type FROM users WHERE id=$1", userID)

	var typ int64
	if err := row.Scan(&typ); err != nil {
		return errors.Wrap(err, "scanning account type")
	}
	if model.UserType(typ) == model.Business && roleName != roles.Host {
		return httperr.Forbidden("a bussiness can't take part in third party events")
	}

	q := "INSERT INTO events_users_roles (event_id, user_id, role_name) VALUES ($1, $2, $3)"
	if _, err := sqlTx.ExecContext(ctx, q, eventID, userID, roleName); err != nil {
		return errors.Wrap(err, "setting reserved role")
	}

	return nil
}

// UnsetRole removes the role a user had in the event.
func (s service) UnsetRole(ctx context.Context, eventID, userID string) error {
	sqlTx := txgroup.SQLTx(ctx)

	q := "DELETE FROM events_users_roles WHERE event_id=$1 AND user_id=$2"
	if _, err := sqlTx.ExecContext(ctx, q, eventID, userID); err != nil {
		return errors.Wrap(err, "unsetting role")
	}

	return nil
}

// UpdatePemission sets new values for a permission.
func (s service) UpdatePermission(ctx context.Context, eventID, key string, permission UpdatePermission) error {
	if err := permission.Validate(); err != nil {
		return httperr.BadRequest(err.Error())
	}

	q := `UPDATE events_permissions SET 
	name = COALESCE($3,name), 
	description = COALESCE($4,description) 
	WHERE event_id=$1 AND key=$2`
	sqlTx := txgroup.SQLTx(ctx)
	if _, err := sqlTx.ExecContext(ctx, q, eventID, key, permission.Name, permission.Description); err != nil {
		return errors.Wrap(err, "updating permission")
	}

	return nil
}

// UpdateRole sets new values for a role.
func (s service) UpdateRole(ctx context.Context, eventID, name string, role UpdateRole) error {
	if err := role.Validate(); err != nil {
		return httperr.BadRequest(err.Error())
	}

	q := `UPDATE events_roles SET
	permission_keys = COALESCE($3,permission_keys)
	WHERE event_id=$1 AND name=$2`
	sqlTx := txgroup.SQLTx(ctx)
	if _, err := sqlTx.ExecContext(ctx, q, eventID, name, role.PermissionKeys); err != nil {
		return errors.Wrap(err, "updating role")
	}

	return nil
}

func (s service) getFriendsIDs(ctx context.Context, userID string) ([]string, error) {
	q := `query q($id: string) {
		q(func: eq(user_id, $id)) {
			friend {
				user_id
			}
		}
	}`
	return queryFriendIDs(ctx, s.dc, q, map[string]string{"$id": userID})
}

func (s service) getFriendsIDsWithParams(ctx context.Context, userID string, params params.Query) ([]string, error) {
	q := `query q($id: string, $cursor: string, $limit: string) {
		q(func: eq(user_id, $id)) {
			friend (orderasc: user_id) (first: $limit, offset: $cursor) {
				user_id
			}
		}
	}`
	vars := dgraph.QueryVars(userID, params)
	return queryFriendIDs(ctx, s.dc, q, vars)
}

func queryFriendIDs(ctx context.Context, dc *dgo.Dgraph, query string, vars map[string]string) ([]string, error) {
	res, err := dc.NewReadOnlyTxn().QueryRDFWithVars(ctx, query, vars)
	if err != nil {
		return nil, errors.Wrap(err, "dgraph: fetching friends")
	}

	friendsIDs := dgraph.ParseRDFULIDs(res.Rdf)
	if len(friendsIDs) == 0 {
		return nil, nil
	}

	return friendsIDs, nil
}

func selectFriendsQuery(friendsIDs, fields []string, roleEquals bool) string {
	buf := bufferpool.Get()

	m := model.User
	buf.WriteString("SELECT ")
	postgres.WriteFields(buf, m, fields)
	buf.WriteString(" FROM ")
	buf.WriteString(m.Tablename())
	buf.WriteString(" WHERE id IN (SELECT user_id FROM events_users_roles WHERE event_id=$1 AND role_name ")
	if !roleEquals {
		buf.WriteRune('!')
	}
	buf.WriteString("= $2 AND user_id IN (")
	for i, id := range friendsIDs {
		if i != 0 {
			buf.WriteByte(',')
		}
		buf.WriteByte('\'')
		buf.WriteString(id)
		buf.WriteByte('\'')
	}
	// Order like pagination does just in case it was used in a query prior to this one, so the client
	// receives the results in the order expected
	buf.WriteString(")) ORDER BY id DESC")

	query := buf.String()
	bufferpool.Put(buf)

	return query
}

func sliceToMap(slice []string) map[string]struct{} {
	mp := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		mp[s] = struct{}{}
	}
	return mp
}
