package event

import (
	"context"
	"database/sql"
	"time"

	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/permissions"
	"github.com/GGP1/groove/storage/dgraph"
	"github.com/GGP1/groove/storage/postgres"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/dgraph-io/dgo/v210"
	"github.com/dgraph-io/dgo/v210/protos/api"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

// TODO: create GetProduct and GetMedia methods

// Service represents the event service.
type Service interface {
	AddEdge(ctx context.Context, eventID string, predicate predicate, userID string) error
	CanInvite(ctx context.Context, tx *sql.Tx, userID, invitedID string) (bool, error)
	Create(ctx context.Context, eventID string, event CreateEvent) error
	CreateMedia(ctx context.Context, eventID string, media Media) error
	CreatePermission(ctx context.Context, eventID string, permission Permission) error
	CreateProduct(ctx context.Context, eventID string, product Product) error
	CreateReport(ctx context.Context, eventID string, report Report) error
	CreateRole(ctx context.Context, eventID string, role Role) error
	Delete(ctx context.Context, eventID string) error
	GetByID(ctx context.Context, eventID string) (Event, error)
	GetBanned(ctx context.Context, eventID string, params params.Query) ([]User, error)
	GetBannedCount(ctx context.Context, eventID string) (*uint64, error)
	GetBannedFollowing(ctx context.Context, eventID, userID string, params params.Query) ([]User, error)
	GetConfirmed(ctx context.Context, eventID string, params params.Query) ([]User, error)
	GetConfirmedCount(ctx context.Context, eventID string) (*uint64, error)
	GetConfirmedFollowing(ctx context.Context, eventID, userID string, params params.Query) ([]User, error)
	GetHosts(ctx context.Context, eventID string, params params.Query) ([]User, error)
	GetInvited(ctx context.Context, eventID string, params params.Query) ([]User, error)
	GetInvitedCount(ctx context.Context, eventID string) (*uint64, error)
	GetInvitedFollowing(ctx context.Context, eventID, userID string, params params.Query) ([]User, error)
	GetLikedBy(ctx context.Context, eventID string, params params.Query) ([]User, error)
	GetLikedByCount(ctx context.Context, eventID string) (*uint64, error)
	GetLikedByFollowing(ctx context.Context, eventID, userID string, params params.Query) ([]User, error)
	GetNode(ctx context.Context, eventID string) (Node, error)
	GetPermissions(ctx context.Context, eventID string) ([]Permission, error)
	GetRole(ctx context.Context, tx *sql.Tx, eventID, name string) (Role, error)
	GetRoles(ctx context.Context, eventID string) ([]Role, error)
	GetReports(ctx context.Context, eventID string) ([]Report, error)
	GetUserRole(ctx context.Context, tx *sql.Tx, eventID, userID string) (Role, error)
	IsPublic(ctx context.Context, tx *sql.Tx, eventID string) (bool, error)
	RemoveEdge(ctx context.Context, eventID string, predicate predicate, userID string) error
	Search(ctx context.Context, query string) ([]Event, error)
	SetRole(ctx context.Context, eventID, userID, roleName string) error
	PqTx(ctx context.Context, readOnly bool) (*sql.Tx, error)
	Update(ctx context.Context, eventID string, event UpdateEvent) error
	UpdateMedia(ctx context.Context, eventID string, media Media) error
	UpdateProduct(ctx context.Context, eventID string, product Product) error
}

type service struct {
	db *sqlx.DB
	dc *dgo.Dgraph
	mc *memcache.Client

	metrics metrics
}

// NewService returns a new event service.
func NewService(db *sqlx.DB, dc *dgo.Dgraph, mc *memcache.Client) Service {
	return &service{
		db:      db,
		dc:      dc,
		mc:      mc,
		metrics: initMetrics(),
	}
}

// AddEdge creates an edge between the event and the user.
func (s *service) AddEdge(ctx context.Context, eventID string, predicate predicate, userID string) error {
	s.metrics.incMethodCalls("AddEdge")

	return s.addEdgeTx(ctx, s.dc.NewTxn(), eventID, predicate, userID)
}

func (s *service) CanInvite(ctx context.Context, tx *sql.Tx, userID, invitedID string) (bool, error) {
	s.metrics.incMethodCalls("CanInvite")

	q := "SELECT invitations FROM users WHERE id=$1"
	invitations, err := postgres.ScanString(ctx, tx, q, userID)
	if err != nil {
		return false, err
	}

	switch invitations {
	case "anyone":
		return true, nil
	case "mutual_follow":
		q := `query q($user_id: string, $target_user_id: string) {
			user as var(func: eq(user_id, $user_id))
			target as var(func: eq(user_id, $target_user_id))
			
			q(func: uid(user)) {
				following @filter(uid_in(following, uid(user)) AND uid(target)) {
					user_id
				}
			}
		}`
		vars := map[string]string{"$user_id": userID, "$target_user_id": invitedID}
		res, err := s.dc.NewReadOnlyTxn().QueryRDFWithVars(ctx, q, vars)
		if err != nil {
			return false, errors.Wrap(err, "")
		}

		ids := dgraph.ParseRDFUUIDs(res.Rdf)
		return len(ids) != 0, nil
	case "nobody":
		return false, nil
	}

	return true, nil
}

// Create creates a new event.
func (s *service) Create(ctx context.Context, eventID string, event CreateEvent) error {
	s.metrics.incMethodCalls("Create")

	psqlTx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "starting transaction")
	}
	defer psqlTx.Rollback()

	exists, err := postgres.ScanBool(ctx, psqlTx, "SELECT EXISTS(SELECT 1 FROM users WHERE id=$1)", event.CreatorID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.Errorf("user with id %q does not exist", event.CreatorID)
	}

	q1 := `INSERT INTO events 
	(id, name, type, public, virtual, start_time, end_time, slots, min_age, ticket_cost, updated_at)
	VALUES 
	($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
	_, err = psqlTx.ExecContext(ctx, q1, eventID, event.Name, event.Type, event.Public, event.Virtual,
		event.StartTime, event.EndTime, event.Slots, event.MinAge, event.TicketCost, time.Time{})
	if err != nil {
		return errors.Wrap(err, "creating event")
	}

	q2 := "INSERT INTO events_roles (event_id, name, permissions_keys) VALUES ($1, $2, $3)"
	if _, err := psqlTx.ExecContext(ctx, q2, eventID, permissions.Host, permissions.All); err != nil {
		return errors.Wrap(err, "setting role")
	}

	q3 := "INSERT INTO events_users_roles (event_id, user_id, role_name) VALUES ($1, $2, $3)"
	if _, err := psqlTx.ExecContext(ctx, q3, eventID, event.CreatorID, permissions.Host); err != nil {
		return errors.Wrap(err, "setting user role")
	}

	q4 := `INSERT INTO events_locations 
	(event_id, country, state, city, address, virtual, platform, invite_url)
	VALUES
	($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err = psqlTx.ExecContext(ctx, q4, eventID, event.Location.Country, event.Location.State,
		event.Location.City, event.Location.Address, event.Location.Virtual, event.Location.Platform,
		event.Location.InviteURL)
	if err != nil {
		return errors.Wrap(err, "storing event location")
	}

	err = dgraph.Mutation(ctx, s.dc, func(dgraphTx *dgo.Txn) error {
		if err := dgraph.CreateNode(ctx, dgraphTx, dgraph.Event, eventID); err != nil {
			return err
		}
		// Add the host as confirmed to the event
		return s.addEdgeTx(ctx, dgraphTx, eventID, Confirmed, event.CreatorID)
	})
	if err != nil {
		return err
	}

	if err := psqlTx.Commit(); err != nil {
		return err
	}

	s.metrics.registeredEvents.Inc()
	return nil
}

// CreateMedia adds a photo or video to the event.
func (s *service) CreateMedia(ctx context.Context, eventID string, media Media) error {
	s.metrics.incMethodCalls("CreateMedia")

	q := "INSERT INTO events_media (id, event_id, url) VALUES ($1, $2, $3)"
	_, err := s.db.ExecContext(ctx, q, uuid.NewString(), media.EventID, media.URL)
	if err != nil {
		return errors.Wrap(err, "creating media")
	}

	return nil
}

// CreatePermission creates a permission inside the event.
func (s *service) CreatePermission(ctx context.Context, eventID string, permission Permission) error {
	s.metrics.incMethodCalls("CreatePermission")

	q := "INSERT INTO events_permissions (event_id, key, name, description) VALUES ($1, $2, $3, $4)"
	_, err := s.db.ExecContext(ctx, q, eventID, permission.Key, permission.Name, permission.Description)
	if err != nil {
		return errors.Wrap(err, "creating permission")
	}

	return nil
}

// CreateProduct adds a product to the event.
func (s *service) CreateProduct(ctx context.Context, eventID string, product Product) error {
	s.metrics.incMethodCalls("CreateProduct")

	q := `INSERT INTO events_products 
	(id, event_id, stock, brand, type, description, discount, taxes, subtotal, total) 
	VALUES 
	($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := s.db.ExecContext(ctx, q, uuid.NewString(), product.EventID, product.Stock,
		product.Brand, product.Type, product.Description, product.Discount, product.Taxes,
		product.Subtotal, product.Total)
	if err != nil {
		return errors.Wrap(err, "creating product")
	}

	return nil
}

// CreateReport adds a report to the event.
func (s *service) CreateReport(ctx context.Context, eventID string, report Report) error {
	s.metrics.incMethodCalls("CreateReport")

	q := `INSERT INTO events_reports
	(event_id, user_id, type, details)
	VALUES
	($1, $2, $3, $4)`
	_, err := s.db.ExecContext(ctx, q, report.EventID, report.UserID, report.Type, report.Details)
	if err != nil {
		return errors.Wrap(err, "creating report")
	}

	return nil
}

// CreateRole creates a new role inside an event.
func (s *service) CreateRole(ctx context.Context, eventID string, role Role) error {
	s.metrics.incMethodCalls("CreateRole")

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "starting transaction")
	}

	// Check for the existence of the keys used for the role
	for _, pk := range role.PermissionKeys {
		q1 := "SELECT EXISTS(SELECT 1 FROM events_permissions WHERE event_id=$1 AND key=$2)"
		exists, err := postgres.ScanBool(ctx, tx, q1, eventID, pk)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		if !exists {
			_ = tx.Rollback()
			return errors.Errorf("permission with key %q does not exist", pk)
		}
	}

	parsedKeys := permissions.ParseKeys(role.PermissionKeys)
	q2 := "INSERT INTO events_roles (event_id, name, permissions_keys) VALUES ($1, $2, $3)"
	if _, err := tx.ExecContext(ctx, q2, eventID, role.Name, parsedKeys); err != nil {
		_ = tx.Rollback()
		return errors.Wrap(err, "creating role")
	}

	return tx.Commit()
}

// Delete removes an event and all its edges.
func (s *service) Delete(ctx context.Context, eventID string) error {
	s.metrics.incMethodCalls("Delete")

	psqlTx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "starting transaction")
	}

	if _, err := psqlTx.ExecContext(ctx, "DELETE FROM events WHERE id=$1", eventID); err != nil {
		_ = psqlTx.Rollback()
		return errors.Wrap(err, "postgres: deleting event")
	}

	vars := map[string]string{"$id": eventID}
	q := `query q($id: string) {
		event as var(func: eq(event_id, $id))
	}`
	mu := &api.Mutation{
		DelNquads: []byte(`uid(event) * * .`),
	}
	req := &api.Request{
		Vars:      vars,
		Query:     q,
		Mutations: []*api.Mutation{mu},
		CommitNow: true,
	}
	if _, err := s.dc.NewTxn().Do(ctx, req); err != nil {
		_ = psqlTx.Rollback()
		return errors.Wrap(err, "dgraph: deleting event")
	}

	if err := psqlTx.Commit(); err != nil {
		return err
	}

	if err := s.mc.Delete(eventID); err != nil && err != memcache.ErrCacheMiss {
		return errors.Wrap(err, "memcached: deleting event")
	}

	s.metrics.registeredEvents.Dec()
	return nil
}

// GetBanned returns event's banned guests.
func (s *service) GetBanned(ctx context.Context, eventID string, params params.Query) ([]User, error) {
	s.metrics.incMethodCalls("GetBanned")

	query := banned
	if params.LookupID != "" {
		query = bannedLookup
	}

	vars := dgraph.QueryVars(eventID, params)
	return s.queryUsers(ctx, getQuery[query], vars, params)
}

// GetBannedCount returns event's banned guests count.
func (s *service) GetBannedCount(ctx context.Context, eventID string) (*uint64, error) {
	s.metrics.incMethodCalls("GetBannedCount")

	return dgraph.GetCount(ctx, s.dc, getQuery[bannedCount], eventID)
}

// GetBannedFollowing returns event likes users that are following the user passed.
func (s *service) GetBannedFollowing(ctx context.Context, eventID, userID string, params params.Query) ([]User, error) {
	s.metrics.incMethodCalls("GetBannedFollowing")

	vars := map[string]string{
		"$event_id": eventID,
		"$user_id":  userID,
		"$cursor":   params.Cursor,
		"$limit":    params.Limit,
	}

	return s.queryUsers(ctx, getMixedQuery[bannedFollowing], vars, params)
}

// GetByID returns the event with the id passed.
func (s *service) GetByID(ctx context.Context, eventID string) (Event, error) {
	s.metrics.incMethodCalls("GetByID")

	psqlTx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return Event{}, errors.Wrap(err, "starting transaction")
	}

	var event Event
	q := `SELECT 
	id, name, type, public, virtual, start_time, end_time, 
	slots, min_age, ticket_cost, created_at, updated_at 
	FROM events 
	WHERE id=$1`
	row := s.db.QueryRowContext(ctx, q, eventID)
	err = row.Scan(&event.ID, &event.Name, &event.Type, &event.Public,
		&event.Virtual, &event.StartTime, &event.EndTime, &event.Slots,
		&event.MinAge, &event.TicketCost, &event.CreatedAt, &event.UpdatedAt)
	if err != nil {
		_ = psqlTx.Rollback()
		return Event{}, errors.Wrap(err, "fetching event")
	}

	if err := s.getCounts(ctx, eventID, &event); err != nil {
		_ = psqlTx.Rollback()
		return Event{}, err
	}

	if err := psqlTx.Commit(); err != nil {
		return Event{}, err
	}

	return event, nil
}

// GetConfirmed returns event's confirmed guests.
func (s *service) GetConfirmed(ctx context.Context, eventID string, params params.Query) ([]User, error) {
	s.metrics.incMethodCalls("GetConfirmed")

	query := confirmed
	if params.LookupID != "" {
		query = confirmedLookup
	}

	vars := dgraph.QueryVars(eventID, params)
	return s.queryUsers(ctx, getQuery[query], vars, params)
}

// GetConfirmed returns event's confirmed guests count.
func (s *service) GetConfirmedCount(ctx context.Context, eventID string) (*uint64, error) {
	s.metrics.incMethodCalls("GetConfirmedCount")

	return dgraph.GetCount(ctx, s.dc, getQuery[confirmedCount], eventID)
}

// GetConfirmedFollowing returns event confirmed users that are following the user passed.
func (s *service) GetConfirmedFollowing(ctx context.Context, eventID, userID string, params params.Query) ([]User, error) {
	s.metrics.incMethodCalls("GetConfirmedFollowing")

	vars := map[string]string{
		"$event_id": eventID,
		"$user_id":  userID,
		"$cursor":   params.Cursor,
		"$limit":    params.Limit,
	}

	return s.queryUsers(ctx, getMixedQuery[confirmedFollowing], vars, params)
}

// GetHosts returns event's hosts.
func (s *service) GetHosts(ctx context.Context, eventID string, params params.Query) ([]User, error) {
	s.metrics.incMethodCalls("GetHostedEvents")

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrap(err, "starting transaction")
	}
	defer tx.Rollback()

	q1 := "SELECT user_id FROM events_users_roles WHERE event_id=$1 AND role_name='host'"
	rows, err := tx.QueryContext(ctx, q1, eventID)
	if err != nil {
		return nil, errors.Wrap(err, "fetching users ids")
	}

	var usersIds []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, errors.Wrap(err, "scanning rows")
		}
		usersIds = append(usersIds, userID)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	q2 := postgres.SelectInID(postgres.Users, usersIds, params.Fields)
	rows2, err := tx.QueryContext(ctx, q2)
	if err != nil {
		return nil, errors.Wrap(err, "fetching users")
	}

	var users []User
	for rows2.Next() {
		user, err := scanUser(rows2)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	if err := rows2.Err(); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return users, nil
}

// GetInvited returns event's invited users.
func (s *service) GetInvited(ctx context.Context, eventID string, params params.Query) ([]User, error) {
	s.metrics.incMethodCalls("GetInvited")

	query := invited
	if params.LookupID != "" {
		query = invitedLookup
	}

	vars := dgraph.QueryVars(eventID, params)
	return s.queryUsers(ctx, getQuery[query], vars, params)
}

// GetInvited returns event's invited users count.
func (s *service) GetInvitedCount(ctx context.Context, eventID string) (*uint64, error) {
	s.metrics.incMethodCalls("GetInvitedCount")

	return dgraph.GetCount(ctx, s.dc, getQuery[invitedCount], eventID)
}

// GetInvitedFollowing returns event invited users that are following the user passed.
func (s *service) GetInvitedFollowing(ctx context.Context, eventID, userID string, params params.Query) ([]User, error) {
	s.metrics.incMethodCalls("GetInvitedFollowing")

	vars := map[string]string{
		"$event_id": eventID,
		"$user_id":  userID,
		"$cursor":   params.Cursor,
		"$limit":    params.Limit,
	}

	return s.queryUsers(ctx, getMixedQuery[invitedFollowing], vars, params)
}

// GetLikedBy returns users liking the event.
func (s *service) GetLikedBy(ctx context.Context, eventID string, params params.Query) ([]User, error) {
	s.metrics.incMethodCalls("GetLikedBy")

	query := likedBy
	if params.LookupID != "" {
		query = likedByLookup
	}

	vars := dgraph.QueryVars(eventID, params)
	return s.queryUsers(ctx, getQuery[query], vars, params)
}

// GetLikedByCount returns the number of users liking the event.
func (s *service) GetLikedByCount(ctx context.Context, eventID string) (*uint64, error) {
	s.metrics.incMethodCalls("GetLikedByCount")

	return dgraph.GetCount(ctx, s.dc, getQuery[likedByCount], eventID)
}

// GetLikedByFollowing returns event likes users that are following the user passed.
func (s *service) GetLikedByFollowing(ctx context.Context, eventID, userID string, params params.Query) ([]User, error) {
	s.metrics.incMethodCalls("GetLikedByFollowing")

	vars := map[string]string{
		"$event_id": eventID,
		"$user_id":  userID,
		"$cursor":   params.Cursor,
		"$limit":    params.Limit,
	}

	return s.queryUsers(ctx, getMixedQuery[likedByFollowing], vars, params)
}

// GetNode returns a dgraph node representing an event and all its relationships.
func (s *service) GetNode(ctx context.Context, eventID string) (Node, error) {
	s.metrics.incMethodCalls("GetNode")

	vars := map[string]string{"$uuid": eventID}
	q := `
	query q($uuid: string) {
		q(func: eq(event_id, $uuid)) {
			banned {
				user_id
			}
			confirmed {
				user_id
			}
			liked_by {
				user_id
			}
			invited {
				user_id
			}
		}		
	}`

	res, err := s.dc.NewReadOnlyTxn().QueryRDFWithVars(ctx, q, vars)
	if err != nil {
		return Node{}, errors.Wrap(err, "dgraph: fetching node")
	}

	mp, err := dgraph.ParseRDFWithMap(res.Rdf)
	if err != nil {
		return Node{}, err
	}
	node := Node{
		Bans:      mp["banned"],
		Confirmed: mp["confirmed"],
		LikedBy:   mp["liked_by"],
		Invited:   mp["invited"],
	}

	return node, nil
}

// GetPermissions returns all event's permissions.
func (s *service) GetPermissions(ctx context.Context, eventID string) ([]Permission, error) {
	s.metrics.incMethodCalls("GetPermissions")

	q := "SELECT key, name, description FROM events_permissions WHERE event_id=$1"
	rows, err := s.db.QueryContext(ctx, q, eventID)
	if err != nil {
		return nil, err
	}

	var permissions []Permission
	for rows.Next() {
		var key, name, description string
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
func (s *service) GetRole(ctx context.Context, tx *sql.Tx, eventID, name string) (Role, error) {
	q := "SELECT permissions_keys FROM events_roles WHERE event_id=$1 AND name=$2"
	permissionKeys, err := postgres.ScanString(ctx, tx, q, eventID, name)
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
func (s *service) GetRoles(ctx context.Context, eventID string) ([]Role, error) {
	s.metrics.incMethodCalls("GetRoles")

	q := "SELECT name, permissions_keys FROM events_roles WHERE event_id=$1"
	rows, err := s.db.QueryContext(ctx, q, eventID)
	if err != nil {
		return nil, errors.Wrap(err, "fetching roles")
	}

	var parsedRoles []Role
	for rows.Next() {
		var name, permissionKeys string

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

// GetReports returns event's reports.
func (s *service) GetReports(ctx context.Context, eventID string) ([]Report, error) {
	s.metrics.incMethodCalls("GetReports")

	q := "SELECT * FROM reports WHERE event_id=$1"
	rows, err := s.db.QueryContext(ctx, q, eventID)
	if err != nil {
		return nil, err
	}

	var reports []Report
	for rows.Next() {
		var reportedID, userID, details string
		if err := rows.Scan(&reportedID, &userID, &details); err != nil {
			return nil, errors.Wrap(err, "scanning rows")
		}
		reports = append(reports, Report{
			EventID: reportedID,
			UserID:  userID,
			Details: details,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return reports, nil
}

// GetUserRole returns user's role inside the event.
func (s *service) GetUserRole(ctx context.Context, tx *sql.Tx, eventID, userID string) (Role, error) {
	q1 := "SELECT role_name FROM events_users_roles WHERE event_id=$1 AND user_id=$2"
	roleName, err := postgres.ScanString(ctx, tx, q1, eventID, userID)
	if err != nil {
		_ = tx.Rollback()
		return Role{}, errors.Errorf("user %q has no role in event %q", userID, eventID)
	}

	q2 := "SELECT permissions_keys FROM events_roles WHERE event_id=$1 AND name=$2"
	permissionsKeys, err := postgres.ScanString(ctx, tx, q2, eventID, roleName)
	if err != nil {
		_ = tx.Rollback()
		return Role{}, errors.Wrap(err, "fetching permission keys")
	}

	role := Role{
		Name:           roleName,
		PermissionKeys: permissions.UnparseKeys(permissionsKeys),
	}
	return role, nil
}

// IsPublic returns if the event is public or not.
func (s *service) IsPublic(ctx context.Context, tx *sql.Tx, eventID string) (bool, error) {
	s.metrics.incMethodCalls("IsPublic")

	row := tx.QueryRowContext(ctx, "SELECT public FROM events WHERE id=$1", eventID)
	var public bool
	if err := row.Scan(&public); err != nil {
		_ = tx.Rollback()
		if err == sql.ErrNoRows {
			return false, errors.Errorf("event with id %q does not exists", eventID)
		}
		return false, err
	}
	return public, nil
}

// PqTx starts and returns a new postgres transaction.
func (s *service) PqTx(ctx context.Context, readOnly bool) (*sql.Tx, error) {
	return s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: readOnly})
}

func (s *service) RemoveEdge(ctx context.Context, eventID string, predicate predicate, userID string) error {
	s.metrics.incMethodCalls("RemoveEdge")

	req := dgraph.EventEdgeRequest(eventID, string(predicate), userID, false)
	if _, err := s.dc.NewTxn().Do(ctx, req); err != nil {
		return errors.Wrapf(err, "adding %s edge", predicate)
	}

	if err := s.mc.Delete(eventID); err != nil && err != memcache.ErrCacheMiss {
		return errors.Wrap(err, "memcached: deleting event")
	}

	return nil
}

// Search looks for events given the query passed.
func (s *service) Search(ctx context.Context, query string) ([]Event, error) {
	s.metrics.incMethodCalls("Search")
	return nil, nil
}

// SetRole assigns a role to a user inside an event.
func (s *service) SetRole(ctx context.Context, eventID, userID, roleName string) error {
	s.metrics.incMethodCalls("SetRole")

	insert := "INSERT INTO events_users_roles (event_id, user_id, role_name) VALUES ($1, $2, $3)"
	if _, err := s.db.ExecContext(ctx, insert, eventID, userID, roleName); err != nil {
		return errors.Wrap(err, "setting role")
	}

	return nil
}

// Update updates an event.
func (s *service) Update(ctx context.Context, eventID string, event UpdateEvent) error {
	s.metrics.incMethodCalls("Update")

	q := `UPDATE events SET 
	name=$2 type=$3 start_time=$4 end_time=$5 slots=$6 ticket_cost=$7 min_age=$8
	WHERE id=$1`
	_, err := s.db.ExecContext(ctx, eventID, q, event.Name, event.Type,
		event.StartTime, event.EndTime, event.Slots, event.TicketCost, event.MinAge)
	if err != nil {
		return errors.Wrap(err, "scanning updated event")
	}

	if err := s.mc.Delete(eventID); err != nil && err != memcache.ErrCacheMiss {
		return errors.Wrap(err, "memcached: deleting event")
	}

	return nil
}

// UpdateMedia updates event's media.
func (s *service) UpdateMedia(ctx context.Context, eventID string, media Media) error {
	s.metrics.incMethodCalls("UpdateMedia")

	q := "UPDATE events_media SET url=$2 WHERE id=$1 AND event_id=$2"
	_, err := s.db.ExecContext(ctx, q, media.ID, eventID, media.URL)
	if err != nil {
		return errors.Wrap(err, "updating media")
	}

	if err := s.mc.Delete(eventID); err != nil && err != memcache.ErrCacheMiss {
		return errors.Wrap(err, "memcached: deleting event")
	}

	return nil
}

// UpdateProduct updates an event product.
func (s *service) UpdateProduct(ctx context.Context, eventID string, product Product) error {
	s.metrics.incMethodCalls("UpdateProduct")

	q := `UPDATE events_products SET 
	stock=$3 brand=$4 type=$5 description=$6 discount=$7 taxes=$8 subtotal=$9 total=$10 
	WHERE id=$1 AND event_id=$2`
	_, err := s.db.ExecContext(ctx, q, product.ID, eventID, product.Stock, product.Brand, product.Type,
		product.Description, product.Discount, product.Taxes, product.Subtotal, product.Total)
	if err != nil {
		return errors.Wrap(err, "updating products")
	}

	if err := s.mc.Delete(eventID); err != nil && err != memcache.ErrCacheMiss {
		return errors.Wrap(err, "memcached: deleting event")
	}

	return nil
}

// addEdgeTx is like addEdge but receives a transaction instead of creating a new one.
func (s *service) addEdgeTx(ctx context.Context, tx *dgo.Txn, eventID string, predicate predicate, userID string) error {
	req := dgraph.EventEdgeRequest(eventID, string(predicate), userID, true)
	if _, err := tx.Do(ctx, req); err != nil {
		return errors.Wrapf(err, "adding %s edge", predicate)
	}

	if err := s.mc.Delete(eventID); err != nil && err != memcache.ErrCacheMiss {
		return errors.Wrap(err, "memcached: deleting event")
	}

	return nil
}

// getCounts puts the number of edges of the event in the struct passed.
func (s *service) getCounts(ctx context.Context, eventID string, event *Event) error {
	q := `query q($id: string) {
		q(func: eq(event_id, $id)) {
			count(banned)
			count(confirmed)
			count(invited)
			count(liked_by)
		}
	}`
	vars := map[string]string{"$id": eventID}

	res, err := s.dc.NewReadOnlyTxn().QueryRDFWithVars(ctx, q, vars)
	if err != nil {
		return errors.Wrap(err, "querying count")
	}

	mp, err := dgraph.ParseCountWithMap(res.Rdf)
	if err != nil {
		return err
	}
	event.BannedCount = mp["banned"]
	event.ConfirmedCount = mp["confirmed"]
	event.InvitedCount = mp["invited"]
	event.LikesCount = mp["liked_by"]

	return nil
}

// queryUsers returns the users found in the dgraph query passed.
func (s *service) queryUsers(ctx context.Context, query string, vars map[string]string, params params.Query) ([]User, error) {
	res, err := s.dc.NewReadOnlyTxn().QueryRDFWithVars(ctx, query, vars)
	if err != nil {
		return nil, errors.Wrap(err, "dgraph: fetching users ids")
	}

	usersIds := dgraph.ParseRDFUUIDs(res.Rdf)
	if len(usersIds) == 0 {
		return nil, nil
	}

	q := postgres.SelectInID(postgres.Users, usersIds, params.Fields)
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, errors.Wrap(err, "postgres: fetching users")
	}

	var users []User
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}
