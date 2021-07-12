package event

import (
	"context"
	"database/sql"
	"net/http"
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

// TODO: create UpdatePermission and UpdateRole

// Service represents the event service.
type Service interface {
	AddEdge(ctx context.Context, eventID string, predicate predicate, userID string) error
	BeginSQLTx(ctx context.Context, readOnly bool) *sql.Tx
	CanInvite(ctx context.Context, sqlTx *sql.Tx, userID, invitedID string) (bool, error)
	ClonePermissions(ctx context.Context, sqlTx *sql.Tx, exporterEventID, importerEventID string) error
	CloneRoles(ctx context.Context, sqlTx *sql.Tx, exporterEventID, importerEventID string) error
	Create(ctx context.Context, eventID string, event CreateEvent) error
	CreateMedia(ctx context.Context, sqlTx *sql.Tx, eventID string, media Media) error
	CreatePermission(ctx context.Context, sqlTx *sql.Tx, eventID string, permission Permission) error
	CreateProduct(ctx context.Context, sqlTx *sql.Tx, eventID string, product Product) error
	CreateReport(ctx context.Context, eventID string, report Report) error
	CreateRole(ctx context.Context, sqlTx *sql.Tx, eventID string, role Role) error
	Delete(ctx context.Context, sqlTx *sql.Tx, eventID string) error
	GetByID(ctx context.Context, sqlTx *sql.Tx, eventID string) (Event, error)
	GetBanned(ctx context.Context, sqlTx *sql.Tx, eventID string, params params.Query) ([]User, error)
	GetBannedCount(ctx context.Context, eventID string) (*uint64, error)
	GetBannedFollowing(ctx context.Context, sqlTx *sql.Tx, eventID, userID string, params params.Query) ([]User, error)
	GetConfirmed(ctx context.Context, sqlTx *sql.Tx, eventID string, params params.Query) ([]User, error)
	GetConfirmedCount(ctx context.Context, eventID string) (*uint64, error)
	GetConfirmedFollowing(ctx context.Context, sqlTx *sql.Tx, eventID, userID string, params params.Query) ([]User, error)
	GetHosts(ctx context.Context, sqlTx *sql.Tx, eventID string, params params.Query) ([]User, error)
	GetInvited(ctx context.Context, sqlTx *sql.Tx, eventID string, params params.Query) ([]User, error)
	GetInvitedCount(ctx context.Context, eventID string) (*uint64, error)
	GetInvitedFollowing(ctx context.Context, sqlTx *sql.Tx, eventID, userID string, params params.Query) ([]User, error)
	GetLikedBy(ctx context.Context, sqlTx *sql.Tx, eventID string, params params.Query) ([]User, error)
	GetLikedByCount(ctx context.Context, eventID string) (*uint64, error)
	GetLikedByFollowing(ctx context.Context, sqlTx *sql.Tx, eventID, userID string, params params.Query) ([]User, error)
	GetMedia(ctx context.Context, sqlTx *sql.Tx, eventID string, params params.Query) ([]Media, error)
	GetNode(ctx context.Context, eventID string) (Node, error)
	GetPermissions(ctx context.Context, sqlTx *sql.Tx, eventID string) ([]Permission, error)
	GetProducts(ctx context.Context, sqlTx *sql.Tx, eventID string, params params.Query) ([]Product, error)
	GetRole(ctx context.Context, sqlTx *sql.Tx, eventID, name string) (Role, error)
	GetRoles(ctx context.Context, sqlTx *sql.Tx, eventID string) ([]Role, error)
	GetReports(ctx context.Context, sqlTx *sql.Tx, eventID string) ([]Report, error)
	GetUserRole(ctx context.Context, sqlTx *sql.Tx, eventID, userID string) (Role, error)
	IsHost(ctx context.Context, sqlTx *sql.Tx, userID string, eventIDs ...string) (bool, error)
	IsPublic(ctx context.Context, sqlTx *sql.Tx, eventID string) (bool, error)
	RemoveEdge(ctx context.Context, eventID string, predicate predicate, userID string) error
	Search(ctx context.Context, query string) ([]Event, error)
	SetRole(ctx context.Context, sqlTx *sql.Tx, eventID, userID, roleName string) error
	SQLTx(ctx context.Context, readOnly bool, f func(tx *sql.Tx) (int, error)) (int, error)
	Update(ctx context.Context, sqlTx *sql.Tx, eventID string, event UpdateEvent) error
	UpdateMedia(ctx context.Context, sqlTx *sql.Tx, eventID string, media Media) error
	UpdateProduct(ctx context.Context, sqlTx *sql.Tx, eventID string, product Product) error
	UserHasRole(ctx context.Context, sqlTx *sql.Tx, eventID, userID string) (bool, error)
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

// BeginSQLTx starts and returns a new postgres transaction, if the connection fails it panics.
func (s *service) BeginSQLTx(ctx context.Context, readOnly bool) *sql.Tx {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: readOnly})
	if err != nil {
		panic(err)
	}

	return tx
}

func (s *service) CanInvite(ctx context.Context, tx *sql.Tx, userID, invitedID string) (bool, error) {
	s.metrics.incMethodCalls("CanInvite")

	q := "SELECT invitations FROM users WHERE id=$1"
	invitations, err := postgres.QueryString(ctx, tx, q, userID)
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
			return false, err
		}

		ids := dgraph.ParseRDFUUIDs(res.Rdf)
		return len(ids) != 0, nil
	case "nobody":
		return false, nil
	}

	return true, nil
}

// ClonePermissions takes the permissions from the exporter event and creates them in the importer event.
func (s *service) ClonePermissions(ctx context.Context, sqlTx *sql.Tx, exporterEventID, importerEventID string) error {
	s.metrics.incMethodCalls("ClonePermissions")

	// TODO: Do it all using one transaction
	permissions, err := s.GetPermissions(ctx, sqlTx, exporterEventID)
	if err != nil {
		return err
	}

	for _, permission := range permissions {
		if err := s.CreatePermission(ctx, sqlTx, importerEventID, permission); err != nil {
			return err
		}
	}

	return nil
}

// CloneRoles takes the roles from the exporter event and creates them in the importer event.
func (s *service) CloneRoles(ctx context.Context, sqlTx *sql.Tx, exporterEventID, importerEventID string) error {
	s.metrics.incMethodCalls("CloneRoles")

	// TODO: Do it all using one transaction
	roles, err := s.GetRoles(ctx, sqlTx, exporterEventID)
	if err != nil {
		return err
	}

	for _, role := range roles {
		if err := s.CreateRole(ctx, sqlTx, importerEventID, role); err != nil {
			return err
		}
	}

	return nil
}

// Create creates a new event.
func (s *service) Create(ctx context.Context, eventID string, event CreateEvent) error {
	s.metrics.incMethodCalls("Create")

	sqlTx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "starting transaction")
	}
	defer sqlTx.Rollback()

	exists, err := postgres.QueryBool(ctx, sqlTx, "SELECT EXISTS(SELECT 1 FROM users WHERE id=$1)", event.CreatorID)
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
	_, err = sqlTx.ExecContext(ctx, q1, eventID, event.Name, event.Type, event.Public, event.Virtual,
		event.StartTime, event.EndTime, event.Slots, event.MinAge, event.TicketCost, time.Time{})
	if err != nil {
		return errors.Wrap(err, "creating event")
	}

	q2 := "INSERT INTO events_roles (event_id, name, permissions_keys) VALUES ($1, $2, $3)"
	if _, err := sqlTx.ExecContext(ctx, q2, eventID, permissions.Host, permissions.All); err != nil {
		return errors.Wrap(err, "setting role")
	}

	q3 := "INSERT INTO events_users_roles (event_id, user_id, role_name) VALUES ($1, $2, $3)"
	if _, err := sqlTx.ExecContext(ctx, q3, eventID, event.CreatorID, permissions.Host); err != nil {
		return errors.Wrap(err, "setting user role")
	}

	q4 := `INSERT INTO events_locations 
	(event_id, country, state, city, address)
	VALUES
	($1, $2, $3, $4, $5)`
	_, err = sqlTx.ExecContext(ctx, q4, eventID, event.Location.Country, event.Location.State,
		event.Location.City, event.Location.Address)
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

	if err := sqlTx.Commit(); err != nil {
		return err
	}

	s.metrics.registeredEvents.Inc()
	return nil
}

// CreateMedia adds a photo or video to the event.
func (s *service) CreateMedia(ctx context.Context, sqlTx *sql.Tx, eventID string, media Media) error {
	s.metrics.incMethodCalls("CreateMedia")

	q := "INSERT INTO events_media (id, event_id, url) VALUES ($1, $2, $3)"
	_, err := sqlTx.ExecContext(ctx, q, uuid.New(), media.EventID, media.URL)
	if err != nil {
		return errors.Wrap(err, "creating media")
	}

	return nil
}

// CreatePermission creates a permission inside the event.
func (s *service) CreatePermission(ctx context.Context, sqlTx *sql.Tx, eventID string, permission Permission) error {
	s.metrics.incMethodCalls("CreatePermission")

	q := "INSERT INTO events_permissions (event_id, key, name, description) VALUES ($1, $2, $3, $4)"
	_, err := sqlTx.ExecContext(ctx, q, eventID, permission.Key, permission.Name, permission.Description)
	if err != nil {
		return errors.Wrap(err, "creating permission")
	}

	if err := s.mc.Delete(eventID + "_permission"); err != nil && err != memcache.ErrCacheMiss {
		return errors.Wrap(err, "memcached: deleting event")
	}

	return nil
}

// CreateProduct adds a product to the event.
func (s *service) CreateProduct(ctx context.Context, sqlTx *sql.Tx, eventID string, product Product) error {
	s.metrics.incMethodCalls("CreateProduct")

	q := `INSERT INTO events_products 
	(id, event_id, stock, brand, type, description, discount, taxes, subtotal, total) 
	VALUES 
	($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := sqlTx.ExecContext(ctx, q, uuid.New(), product.EventID, product.Stock,
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
func (s *service) CreateRole(ctx context.Context, sqlTx *sql.Tx, eventID string, role Role) error {
	s.metrics.incMethodCalls("CreateRole")

	q1 := "SELECT EXISTS(SELECT 1 FROM events_permissions WHERE event_id=$1 AND key=$2)"

	// Check for the existence of the keys used for the role
	for pk := range role.PermissionKeys {
		exists, err := postgres.QueryBool(ctx, sqlTx, q1, eventID, pk)
		if err != nil {
			return errors.Wrap(err, "checking event permissions")
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

// Delete removes an event and all its edges.
func (s *service) Delete(ctx context.Context, sqlTx *sql.Tx, eventID string) error {
	s.metrics.incMethodCalls("Delete")

	if _, err := sqlTx.ExecContext(ctx, "DELETE FROM events WHERE id=$1", eventID); err != nil {
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
		return errors.Wrap(err, "dgraph: deleting event")
	}

	if err := s.mc.Delete(eventID); err != nil && err != memcache.ErrCacheMiss {
		return errors.Wrap(err, "memcached: deleting event")
	}

	s.metrics.registeredEvents.Dec()
	return nil
}

// GetBanned returns event's banned guests.
func (s *service) GetBanned(ctx context.Context, sqlTx *sql.Tx, eventID string, params params.Query) ([]User, error) {
	s.metrics.incMethodCalls("GetBanned")

	query := banned
	if params.LookupID != "" {
		query = bannedLookup
	}

	vars := dgraph.QueryVars(eventID, params)
	return s.queryUsers(ctx, sqlTx, getQuery[query], vars, params)
}

// GetBannedCount returns event's banned guests count.
func (s *service) GetBannedCount(ctx context.Context, eventID string) (*uint64, error) {
	s.metrics.incMethodCalls("GetBannedCount")

	return dgraph.GetCount(ctx, s.dc, getQuery[bannedCount], eventID)
}

// GetBannedFollowing returns event likes users that are following the user passed.
func (s *service) GetBannedFollowing(ctx context.Context, sqlTx *sql.Tx, eventID, userID string, params params.Query) ([]User, error) {
	s.metrics.incMethodCalls("GetBannedFollowing")

	vars := map[string]string{
		"$event_id": eventID,
		"$user_id":  userID,
		"$cursor":   params.Cursor,
		"$limit":    params.Limit,
	}

	return s.queryUsers(ctx, sqlTx, getMixedQuery[bannedFollowing], vars, params)
}

// GetByID returns the event with the id passed.
func (s *service) GetByID(ctx context.Context, sqlTx *sql.Tx, eventID string) (Event, error) {
	s.metrics.incMethodCalls("GetByID")

	var event Event
	q := `SELECT 
	id, name, type, public, virtual, start_time, end_time, 
	slots, min_age, ticket_cost, created_at, updated_at 
	FROM events 
	WHERE id=$1`
	row := sqlTx.QueryRowContext(ctx, q, eventID)
	err := row.Scan(&event.ID, &event.Name, &event.Type, &event.Public,
		&event.Virtual, &event.StartTime, &event.EndTime, &event.Slots,
		&event.MinAge, &event.TicketCost, &event.CreatedAt, &event.UpdatedAt)
	if err != nil {
		return Event{}, errors.Wrap(err, "fetching event")
	}

	if err := s.getCounts(ctx, eventID, &event); err != nil {
		return Event{}, err
	}

	return event, nil
}

// GetConfirmed returns event's confirmed guests.
func (s *service) GetConfirmed(ctx context.Context, sqlTx *sql.Tx, eventID string, params params.Query) ([]User, error) {
	s.metrics.incMethodCalls("GetConfirmed")

	query := confirmed
	if params.LookupID != "" {
		query = confirmedLookup
	}

	vars := dgraph.QueryVars(eventID, params)
	return s.queryUsers(ctx, sqlTx, getQuery[query], vars, params)
}

// GetConfirmed returns event's confirmed guests count.
func (s *service) GetConfirmedCount(ctx context.Context, eventID string) (*uint64, error) {
	s.metrics.incMethodCalls("GetConfirmedCount")

	return dgraph.GetCount(ctx, s.dc, getQuery[confirmedCount], eventID)
}

// GetConfirmedFollowing returns event confirmed users that are following the user passed.
func (s *service) GetConfirmedFollowing(ctx context.Context, sqlTx *sql.Tx, eventID, userID string, params params.Query) ([]User, error) {
	s.metrics.incMethodCalls("GetConfirmedFollowing")

	vars := map[string]string{
		"$event_id": eventID,
		"$user_id":  userID,
		"$cursor":   params.Cursor,
		"$limit":    params.Limit,
	}

	return s.queryUsers(ctx, sqlTx, getMixedQuery[confirmedFollowing], vars, params)
}

// GetHosts returns event's hosts.
func (s *service) GetHosts(ctx context.Context, sqlTx *sql.Tx, eventID string, params params.Query) ([]User, error) {
	s.metrics.incMethodCalls("GetHosts")

	// TODO: add pagination
	var (
		rows *sql.Rows
		err  error
	)
	if params.LookupID != "" {
		q1 := "SELECT user_id FROM events_users_roles WHERE event_id=$1 AND role_name='host' AND user_id=$2"
		rows, err = sqlTx.QueryContext(ctx, q1, eventID, params.LookupID)
		if err != nil {
			return nil, errors.Wrapf(err, "fetching user %q", params.LookupID)
		}
	} else {
		q1 := "SELECT user_id FROM events_users_roles WHERE event_id=$1 AND role_name='host'"
		rows, err = sqlTx.QueryContext(ctx, q1, eventID)
		if err != nil {
			return nil, errors.Wrap(err, "fetching users ids")
		}
	}

	usersIds, err := postgres.ScanStringSlice(rows)
	if err != nil {
		return nil, err
	}

	if len(usersIds) == 0 {
		return nil, nil
	}

	q2 := postgres.SelectInID(postgres.Users, usersIds, params.Fields)
	rows2, err := sqlTx.QueryContext(ctx, q2)
	if err != nil {
		return nil, errors.Wrap(err, "fetching users")
	}

	users, err := scanUsers(rows2)
	if err != nil {
		return nil, err
	}

	return users, nil
}

// GetInvited returns event's invited users.
func (s *service) GetInvited(ctx context.Context, sqlTx *sql.Tx, eventID string, params params.Query) ([]User, error) {
	s.metrics.incMethodCalls("GetInvited")

	query := invited
	if params.LookupID != "" {
		query = invitedLookup
	}

	vars := dgraph.QueryVars(eventID, params)
	return s.queryUsers(ctx, sqlTx, getQuery[query], vars, params)
}

// GetInvited returns event's invited users count.
func (s *service) GetInvitedCount(ctx context.Context, eventID string) (*uint64, error) {
	s.metrics.incMethodCalls("GetInvitedCount")

	return dgraph.GetCount(ctx, s.dc, getQuery[invitedCount], eventID)
}

// GetInvitedFollowing returns event invited users that are following the user passed.
func (s *service) GetInvitedFollowing(ctx context.Context, sqlTx *sql.Tx, eventID, userID string, params params.Query) ([]User, error) {
	s.metrics.incMethodCalls("GetInvitedFollowing")

	vars := map[string]string{
		"$event_id": eventID,
		"$user_id":  userID,
		"$cursor":   params.Cursor,
		"$limit":    params.Limit,
	}

	return s.queryUsers(ctx, sqlTx, getMixedQuery[invitedFollowing], vars, params)
}

// GetLikedBy returns users liking the event.
func (s *service) GetLikedBy(ctx context.Context, sqlTx *sql.Tx, eventID string, params params.Query) ([]User, error) {
	s.metrics.incMethodCalls("GetLikedBy")

	query := likedBy
	if params.LookupID != "" {
		query = likedByLookup
	}

	vars := dgraph.QueryVars(eventID, params)
	return s.queryUsers(ctx, sqlTx, getQuery[query], vars, params)
}

// GetLikedByCount returns the number of users liking the event.
func (s *service) GetLikedByCount(ctx context.Context, eventID string) (*uint64, error) {
	s.metrics.incMethodCalls("GetLikedByCount")

	return dgraph.GetCount(ctx, s.dc, getQuery[likedByCount], eventID)
}

// GetLikedByFollowing returns event likes users that are following the user passed.
func (s *service) GetLikedByFollowing(ctx context.Context, sqlTx *sql.Tx, eventID, userID string, params params.Query) ([]User, error) {
	s.metrics.incMethodCalls("GetLikedByFollowing")

	vars := map[string]string{
		"$event_id": eventID,
		"$user_id":  userID,
		"$cursor":   params.Cursor,
		"$limit":    params.Limit,
	}

	return s.queryUsers(ctx, sqlTx, getMixedQuery[likedByFollowing], vars, params)
}

func (s *service) GetMedia(ctx context.Context, sqlTx *sql.Tx, eventID string, params params.Query) ([]Media, error) {
	s.metrics.incMethodCalls("GetMedia")

	// TODO: add pagination
	q := postgres.SelectWhereID(postgres.Media, params.Fields, "event_id", eventID)
	if params.LookupID != "" {
		q += "AND id='" + params.LookupID + "'"
	}
	rows, err := sqlTx.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}

	media, err := scanMedia(rows)
	if err != nil {
		return nil, err
	}

	return media, nil
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
func (s *service) GetPermissions(ctx context.Context, sqlTx *sql.Tx, eventID string) ([]Permission, error) {
	s.metrics.incMethodCalls("GetPermissions")

	q := "SELECT key, name, description FROM events_permissions WHERE event_id=$1"
	rows, err := sqlTx.QueryContext(ctx, q, eventID)
	if err != nil {
		return nil, errors.Wrap(err, "querying permissions")
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

// GetProducts returns the products from an event.
func (s *service) GetProducts(ctx context.Context, sqlTx *sql.Tx, eventID string, params params.Query) ([]Product, error) {
	s.metrics.incMethodCalls("GetProducts")

	// TODO: add pagination
	q := postgres.SelectWhereID(postgres.Products, params.Fields, "event_id", eventID)
	if params.LookupID != "" {
		q += "AND id='" + params.LookupID + "'"
	}
	rows, err := sqlTx.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}

	products, err := scanProducts(rows)
	if err != nil {
		return nil, err
	}

	return products, nil
}

// GetRole returns a role in a given event.
func (s *service) GetRole(ctx context.Context, tx *sql.Tx, eventID, name string) (Role, error) {
	s.metrics.incMethodCalls("GetRole")

	q := "SELECT permissions_keys FROM events_roles WHERE event_id=$1 AND name=$2"
	permissionKeys, err := postgres.QueryString(ctx, tx, q, eventID, name)
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
func (s *service) GetRoles(ctx context.Context, sqlTx *sql.Tx, eventID string) ([]Role, error) {
	s.metrics.incMethodCalls("GetRoles")

	q := "SELECT name, permissions_keys FROM events_roles WHERE event_id=$1"
	rows, err := sqlTx.QueryContext(ctx, q, eventID)
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
func (s *service) GetReports(ctx context.Context, sqlTx *sql.Tx, eventID string) ([]Report, error) {
	s.metrics.incMethodCalls("GetReports")

	q := "SELECT * FROM events_reports WHERE event_id=$1"
	rows, err := sqlTx.QueryContext(ctx, q, eventID)
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
	roleName, err := postgres.QueryString(ctx, tx, q1, eventID, userID)
	if err != nil {
		return Role{}, errors.Errorf("user %q has no role in event %q", userID, eventID)
	}

	q2 := "SELECT permissions_keys FROM events_roles WHERE event_id=$1 AND name=$2"
	permissionsKeys, err := postgres.QueryString(ctx, tx, q2, eventID, roleName)
	if err != nil {
		return Role{}, errors.Wrap(err, "fetching permission keys")
	}

	role := Role{
		Name:           roleName,
		PermissionKeys: permissions.UnparseKeys(permissionsKeys),
	}
	return role, nil
}

// IsHost returns if the user's role in the events passed is host.
func (s *service) IsHost(ctx context.Context, sqlTx *sql.Tx, userID string, eventIDs ...string) (bool, error) {
	s.metrics.incMethodCalls("IsHost")

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

// IsPublic returns if the event is public or not.
func (s *service) IsPublic(ctx context.Context, sqlTx *sql.Tx, eventID string) (bool, error) {
	s.metrics.incMethodCalls("IsPublic")

	var public bool
	row := sqlTx.QueryRowContext(ctx, "SELECT public FROM events WHERE id=$1", eventID)
	if err := row.Scan(&public); err != nil {
		if err == sql.ErrNoRows {
			return false, errors.Errorf("event with id %q does not exists", eventID)
		}
		return false, err
	}

	return public, nil
}

func (s *service) RemoveEdge(ctx context.Context, eventID string, predicate predicate, userID string) error {
	s.metrics.incMethodCalls("RemoveEdge")

	err := dgraph.Mutation(ctx, s.dc, func(tx *dgo.Txn) error {
		req := dgraph.EventEdgeRequest(eventID, string(predicate), userID, false)
		_, err := tx.Do(ctx, req)
		return err
	})
	if err != nil {
		return err
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
func (s *service) SetRole(ctx context.Context, sqlTx *sql.Tx, eventID, userID, roleName string) error {
	s.metrics.incMethodCalls("SetRole")

	insert := "INSERT INTO events_users_roles (event_id, user_id, role_name) VALUES ($1, $2, $3)"
	if _, err := sqlTx.ExecContext(ctx, insert, eventID, userID, roleName); err != nil {
		return errors.Wrap(err, "setting role")
	}

	return nil
}

// BeginSQLTx starts and returns a new postgres transaction, if the connection fails it panics.
func (s *service) SQLTx(ctx context.Context, readOnly bool, f func(tx *sql.Tx) (int, error)) (int, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: readOnly})
	if err != nil {
		return http.StatusInternalServerError, errors.Wrap(err, "starting transaction")
	}

	status, err := f(tx)
	if err != nil {
		if err := tx.Rollback(); err != nil {
			return http.StatusInternalServerError, errors.Wrap(err, "rolling back transaction")
		}

		return status, err
	}

	return 0, tx.Commit()
}

// Update updates an event.
func (s *service) Update(ctx context.Context, sqlTx *sql.Tx, eventID string, event UpdateEvent) error {
	s.metrics.incMethodCalls("Update")

	q := `UPDATE events SET 
	name=$2 type=$3 start_time=$4 end_time=$5 slots=$6 ticket_cost=$7 min_age=$8
	WHERE id=$1`
	_, err := sqlTx.ExecContext(ctx, eventID, q, event.Name, event.Type,
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
func (s *service) UpdateMedia(ctx context.Context, sqlTx *sql.Tx, eventID string, media Media) error {
	s.metrics.incMethodCalls("UpdateMedia")

	q := "UPDATE events_media SET url=$2 WHERE id=$1 AND event_id=$2"
	_, err := sqlTx.ExecContext(ctx, q, media.ID, eventID, media.URL)
	if err != nil {
		return errors.Wrap(err, "updating media")
	}

	if err := s.mc.Delete(eventID + "_media"); err != nil && err != memcache.ErrCacheMiss {
		return errors.Wrap(err, "memcached: deleting media")
	}

	return nil
}

// UpdateProduct updates an event product.
func (s *service) UpdateProduct(ctx context.Context, sqlTx *sql.Tx, eventID string, product Product) error {
	s.metrics.incMethodCalls("UpdateProduct")

	q := `UPDATE events_products SET 
	stock=$3 brand=$4 type=$5 description=$6 discount=$7 taxes=$8 subtotal=$9 total=$10 
	WHERE id=$1 AND event_id=$2`
	_, err := sqlTx.ExecContext(ctx, q, product.ID, eventID, product.Stock, product.Brand, product.Type,
		product.Description, product.Discount, product.Taxes, product.Subtotal, product.Total)
	if err != nil {
		return errors.Wrap(err, "updating products")
	}

	if err := s.mc.Delete(eventID + "_products"); err != nil && err != memcache.ErrCacheMiss {
		return errors.Wrap(err, "memcached: deleting products")
	}

	return nil
}

// UserHasRole returns if the user has a role inside the event or not.
func (s *service) UserHasRole(ctx context.Context, sqlTx *sql.Tx, eventID, userID string) (bool, error) {
	s.metrics.incMethodCalls("UserHasRole")

	q := "SELECT EXISTS(SELECT 1 FROM events_users_roles WHERE event_id=$1 AND user_id=$2)"
	hasRole, err := postgres.QueryBool(ctx, sqlTx, q, eventID, userID)
	if err != nil {
		return false, err
	}

	return hasRole, nil
}

// addEdgeTx is like addEdge but receives a transaction instead of creating a new one.
func (s *service) addEdgeTx(ctx context.Context, tx *dgo.Txn, eventID string, predicate predicate, userID string) error {
	err := dgraph.Mutation(ctx, s.dc, func(tx *dgo.Txn) error {
		req := dgraph.EventEdgeRequest(eventID, string(predicate), userID, true)
		if _, err := tx.Do(ctx, req); err != nil {
			return errors.Wrapf(err, "adding %s edge", predicate)
		}
		return nil
	})
	if err != nil {
		return err
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
func (s *service) queryUsers(ctx context.Context, sqlTx *sql.Tx, query string, vars map[string]string, params params.Query) ([]User, error) {
	res, err := s.dc.NewReadOnlyTxn().QueryRDFWithVars(ctx, query, vars)
	if err != nil {
		return nil, errors.Wrap(err, "dgraph: fetching users ids")
	}

	usersIds := dgraph.ParseRDFUUIDs(res.Rdf)
	if len(usersIds) == 0 {
		return nil, nil
	}

	q := postgres.SelectInID(postgres.Users, usersIds, params.Fields)
	rows, err := sqlTx.QueryContext(ctx, q)
	if err != nil {
		return nil, errors.Wrap(err, "postgres: fetching users")
	}

	users, err := scanUsers(rows)
	if err != nil {
		return nil, err
	}

	return users, nil
}
