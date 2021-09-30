package role

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/GGP1/groove/internal/bufferpool"
	"github.com/GGP1/groove/internal/cache"
	"github.com/GGP1/groove/internal/httperr"
	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/permissions"
	"github.com/GGP1/groove/internal/roles"
	"github.com/GGP1/groove/internal/scan"
	"github.com/GGP1/groove/internal/sqltx"
	"github.com/GGP1/groove/model"
	"github.com/GGP1/groove/service/auth"
	"github.com/GGP1/groove/storage/dgraph"
	"github.com/GGP1/groove/storage/postgres"

	"github.com/dgraph-io/dgo/v210"
	"github.com/pkg/errors"
)

var errAccessDenied = httperr.New("Access denied", httperr.Forbidden)

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
	GetUserRole(ctx context.Context, eventID, userID string) (Role, error)
	HasRole(ctx context.Context, eventID, userID string) (bool, error)
	IsHost(ctx context.Context, userID string, eventIDs ...string) (bool, error)
	PrivacyFilter(ctx context.Context, r *http.Request, eventID string) error
	RequirePermissions(ctx context.Context, r *http.Request, eventID string, permKeys ...string) error
	SetRoles(ctx context.Context, eventID, roleName string, userIDs ...string) error
	SetReservedRole(ctx context.Context, eventID, userID string, roleName roles.Name) error
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
	sqlTx := sqltx.FromContext(ctx)

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
	sqlTx := sqltx.FromContext(ctx)

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
	sqlTx := sqltx.FromContext(ctx)

	q := "INSERT INTO events_permissions (event_id, key, name, description) VALUES ($1, $2, $3, $4)"
	_, err := sqlTx.ExecContext(ctx, q, eventID, permission.Key, permission.Name, permission.Description)
	if err != nil {
		return errors.Wrap(err, "creating permission")
	}

	if err := s.cache.Delete(model.PermissionsCacheKey(eventID)); err != nil {
		return errors.Wrap(err, "deleting permission")
	}

	return nil
}

// CreateRole creates a new role inside an event.
func (s service) CreateRole(ctx context.Context, eventID string, role Role) error {
	sqlTx := sqltx.FromContext(ctx)

	q1 := "SELECT EXISTS(SELECT 1 FROM events_permissions WHERE event_id=$1 AND key=$2)"
	exists := false

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

	if err := s.cache.Delete(model.RolesCacheKey(eventID)); err != nil {
		return errors.Wrap(err, "deleting roles")
	}

	return nil
}

// DeletePermission removes a permission from the event.
func (s service) DeletePermission(ctx context.Context, eventID, key string) error {
	sqlTx := sqltx.FromContext(ctx)

	q := "DELETE FROM events_permissions WHERE event_id=$1 AND key=$2"
	if _, err := sqlTx.ExecContext(ctx, q, eventID, key); err != nil {
		return errors.Wrap(err, "deleting permission")
	}
	return nil
}

// DeleteRole removes a role from the event.
func (s service) DeleteRole(ctx context.Context, eventID, name string) error {
	sqlTx := sqltx.FromContext(ctx)

	q := "DELETE FROM events_roles WHERE event_id=$1 AND name=$2"
	if _, err := sqlTx.ExecContext(ctx, q, eventID, name); err != nil {
		return errors.Wrap(err, "deleting role")
	}
	return nil
}

// GetMembers returns the member list of an event.
//
// TODO: return the user role as well?
func (s service) GetMembers(ctx context.Context, eventID string, params params.Query) ([]model.ListUser, error) {
	sqlTx := sqltx.FromContext(ctx)

	whereCond := "id IN (SELECT user_id FROM events_users_roles WHERE event_id=$1 AND role_name != 'view_event')"
	q := postgres.SelectWhere(model.User, whereCond, "id", params)
	rows, err := sqlTx.QueryContext(ctx, q, eventID)
	if err != nil {
		return nil, err
	}

	var users []model.ListUser
	if err := scan.Rows(&users, rows); err != nil {
		return nil, err
	}

	return users, nil
}

// GetMembersCount returns the number of members of an event.
func (s service) GetMembersCount(ctx context.Context, eventID string) (int64, error) {
	sqlTx := sqltx.FromContext(ctx)

	q := "SELECT members_count FROM events WHERE id=$1"
	return postgres.QueryInt(ctx, sqlTx, q, eventID)
}

// GetMembersFriends returns the members of an event that are friends of userID.
func (s service) GetMembersFriends(ctx context.Context, eventID, userID string, params params.Query) ([]model.ListUser, error) {
	sqlTx := sqltx.FromContext(ctx)

	q1 := `query q($id: string, $cursor: string, $limit: string) {
		q(func: eq(user_id, $id)) {
			friend (orderasc: user_id) (first: $limit, offset: $cursor) {
				user_id
			}
		}
	}`
	vars := dgraph.QueryVars(userID, params)
	res, err := s.dc.NewReadOnlyTxn().QueryRDFWithVars(ctx, q1, vars)
	if err != nil {
		return nil, errors.Wrap(err, "dgraph: fetching friends ids")
	}

	friendsIDs := dgraph.ParseRDFULIDs(res.Rdf)
	if len(friendsIDs) == 0 {
		return nil, nil
	}

	q2 := selectMembersFriends(model.User, friendsIDs, params.Fields)
	rows, err := sqlTx.QueryContext(ctx, q2, eventID)
	if err != nil {
		return nil, err
	}

	var users []model.ListUser
	if err := scan.Rows(&users, rows); err != nil {
		return nil, err
	}

	return users, nil
}

// GetMembersFriendsCount returns the count of the members of an event that are friends of userID.
func (s service) GetMembersFriendsCount(ctx context.Context, eventID, userID string) (int64, error) {
	sqlTx := sqltx.FromContext(ctx)

	q1 := `query q($id: string, $cursor: string, $limit: string) {
		q(func: eq(user_id, $id)) {
			friend (orderasc: user_id) {
				user_id
			}
		}
	}`

	res, err := s.dc.NewReadOnlyTxn().QueryRDFWithVars(ctx, q1, map[string]string{"$id": userID})
	if err != nil {
		return 0, errors.Wrap(err, "dgraph: fetching friends ids")
	}

	friendsIDs := dgraph.ParseRDFULIDs(res.Rdf)
	if len(friendsIDs) == 0 {
		return 0, nil
	}

	query := "SELECT COUNT(*) FROM events WHERE event_id=$1 AND role_name != 'view_event'"
	q2 := postgres.AppendInIDs(query, friendsIDs, false)
	return postgres.QueryInt(ctx, sqlTx, q2, eventID)
}

// GetPermission returns a permission from an event with the given key.
func (s service) GetPermission(ctx context.Context, eventID, key string) (Permission, error) {
	sqlTx := sqltx.FromContext(ctx)

	q := "SELECT name, description, created_at FROM events_permissions WHERE event_id=$1 AND key=$2"
	row := sqlTx.QueryRowContext(ctx, q, eventID, key)

	permission := Permission{Key: key}
	if err := row.Scan(&permission.Name, &permission.Description, &permission.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Permission{}, errors.Errorf("permission with key %q in event %q does not exists", key, eventID)
		}
		return Permission{}, errors.Wrap(err, "scanning permission")
	}

	return permission, nil
}

// GetPermissions returns all event's permissions.
func (s service) GetPermissions(ctx context.Context, eventID string) ([]Permission, error) {
	sqlTx := sqltx.FromContext(ctx)

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
func (s service) GetRole(ctx context.Context, eventID, name string) (Role, error) {
	if keys, ok := roles.Reserved.GetStringSlice(name); ok {
		return Role{Name: name, PermissionKeys: keys}, nil
	}

	sqlTx := sqltx.FromContext(ctx)

	q := "SELECT permission_keys FROM events_roles WHERE event_id=$1 AND name=$2"
	row := sqlTx.QueryRowContext(ctx, q, eventID, name)

	role := Role{Name: name}
	if err := row.Scan(&role.PermissionKeys); err != nil {
		return Role{}, errors.Wrap(err, "scanning role permission keys")
	}

	return role, nil
}

// GetRoles returns all event's roles.
func (s service) GetRoles(ctx context.Context, eventID string) ([]Role, error) {
	sqlTx := sqltx.FromContext(ctx)

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
	// Prepend reserved roles
	roles = append(reservedRoles, roles...)

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return roles, nil
}

// GetUserRole returns user's role inside the event.
func (s service) GetUserRole(ctx context.Context, eventID, userID string) (Role, error) {
	sqlTx := sqltx.FromContext(ctx)

	// TODO: cache?
	q1 := "SELECT role_name FROM events_users_roles WHERE event_id=$1 AND user_id=$2"
	roleName, err := postgres.QueryString(ctx, sqlTx, q1, eventID, userID)
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

// IncMembersCount increments the members count field by one.
func (s service) IncMembersCount(ctx context.Context, eventID, roleName string) error {
	if roleName != string(roles.Viewer) {
		sqlTx := sqltx.FromContext(ctx)

		if _, err := sqlTx.ExecContext(ctx, "UPDATE events SET members_count=members_count+1 WHERE id=$1", eventID); err != nil {
			return errors.Wrap(err, "incrementing members count")
		}
	}
	return nil
}

// HasRole returns if the user has a role inside the event or not (if the user is a member).
func (s service) HasRole(ctx context.Context, eventID, userID string) (bool, error) {
	sqlTx := sqltx.FromContext(ctx)

	q := "SELECT EXISTS(SELECT 1 FROM events_users_roles WHERE event_id=$1 AND user_id=$2)"
	return postgres.QueryBool(ctx, sqlTx, q, eventID, userID)
}

// IsHost returns if the user's role in the events passed is host.
func (s service) IsHost(ctx context.Context, userID string, eventIDs ...string) (bool, error) {
	sqlTx := sqltx.FromContext(ctx)

	q := "SELECT EXISTS(SELECT 1 FROM events_users_roles WHERE event_id=$1 AND user_id=$2 AND role_name='host')"
	var isHost bool

	stmt, err := sqlTx.PrepareContext(ctx, q)
	if err != nil {
		return false, errors.Wrap(err, "prepraring statement")
	}
	defer stmt.Close()

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
// if it's public it lets anyone in. TODO: cache
func (s service) PrivacyFilter(ctx context.Context, r *http.Request, eventID string) error {
	session, err := auth.GetSession(ctx, r)
	if err != nil {
		return err
	}

	sqlTx := sqltx.FromContext(ctx)

	var isPublic bool
	row := sqlTx.QueryRowContext(ctx, "SELECT public FROM events WHERE id=$1", eventID)
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

// SetRoles assigns a role to n users inside an event.
func (s service) SetRoles(ctx context.Context, eventID, roleName string, userIDs ...string) error {
	sqlTx := sqltx.FromContext(ctx)

	public, err := postgres.QueryBool(ctx, sqlTx, "SELECT public FROM events WHERE id=$1", eventID)
	if err != nil {
		return err
	}

	if !public {
		// In a private event, the users first need to have a role ("viewer" if they are invited).
		q1 := "SELECT EXISTS(SELECT 1 FROM events_users_roles WHERE event_id=$1 AND user_id=$2)"
		stmt1, err := sqlTx.PrepareContext(ctx, q1)
		if err != nil {
			return errors.Wrap(err, "prepraring statement")
		}
		defer stmt1.Close()

		var hasRole bool // Will be overwritten on each iteration
		for _, userID := range userIDs {
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

	for _, userID := range userIDs {
		if _, err := stmt2.ExecContext(ctx, eventID, userID, roleName); err != nil {
			return errors.Wrap(err, "setting roles")
		}
	}

	// Flush buffered data
	if _, err := stmt2.ExecContext(ctx); err != nil {
		return errors.Wrap(err, "flushing buffered data")
	}

	return s.IncMembersCount(ctx, eventID, roleName)
}

// SetReservedRole assigns a reserved role to a user.
func (s service) SetReservedRole(ctx context.Context, eventID, userID string, roleName roles.Name) error {
	sqlTx := sqltx.FromContext(ctx)

	// I'd prefer to use the user service here but it's not possible
	typ, err := postgres.QueryInt(ctx, sqlTx, "SELECT type FROM users WHERE id=$1", userID)
	if err != nil {
		return err
	}
	if typ == int64(model.Organization) {
		return httperr.New("an organization can't take part in third party events", httperr.Forbidden)
	}

	q := "INSERT INTO events_users_roles (event_id, user_id, role_name) VALUES ($1, $2, $3)"
	if _, err := sqlTx.ExecContext(ctx, q, eventID, userID, roleName); err != nil {
		return errors.Wrap(err, "setting reserved role")
	}

	return s.IncMembersCount(ctx, eventID, string(roleName))
}

// UpdatePemission sets new values for a permission.
func (s service) UpdatePermission(ctx context.Context, eventID, key string, permission UpdatePermission) error {
	sqlTx := sqltx.FromContext(ctx)

	q := `UPDATE events_permissions SET 
	name = COALESCE($3,name), 
	description = COALESCE($4,description) 
	WHERE event_id=$1 AND key=$2`
	if _, err := sqlTx.ExecContext(ctx, q, eventID, key, permission.Name, permission.Description); err != nil {
		return errors.Wrap(err, "updating permission")
	}

	return nil
}

// UpdateRole sets new values for a role.
func (s service) UpdateRole(ctx context.Context, eventID, name string, role UpdateRole) error {
	sqlTx := sqltx.FromContext(ctx)

	q := `UPDATE events_roles SET
	permission_keys = COALESCE($3,permission_keys)
	WHERE event_id=$1 AND name=$2`
	if _, err := sqlTx.ExecContext(ctx, q, eventID, name, role.PermissionKeys); err != nil {
		return errors.Wrap(err, "updating role")
	}

	return nil
}

func selectMembersFriends(model model.Model, friendsIDs, fields []string) string {
	buf := bufferpool.Get()

	buf.WriteString("SELECT ")
	postgres.WriteFields(buf, model, fields)
	buf.WriteString(" FROM ")
	buf.WriteString(model.Tablename())
	buf.WriteString(" WHERE id IN (SELECT user_id FROM events_users_roles WHERE event_id=$1 AND role_name != 'view_event' AND user_id IN (")
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
