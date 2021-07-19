package event

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/permissions"
	"github.com/GGP1/groove/service/event/media"
	"github.com/GGP1/groove/service/event/product"
	"github.com/GGP1/groove/service/event/role"
	"github.com/GGP1/groove/service/event/zone"
	"github.com/GGP1/groove/storage/dgraph"
	"github.com/GGP1/groove/storage/postgres"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/dgraph-io/dgo/v210"
	"github.com/dgraph-io/dgo/v210/protos/api"
	"github.com/pkg/errors"
)

// Service represents the event service.
type Service interface {
	AddEdge(ctx context.Context, eventID string, predicate predicate, userID string) error
	BeginSQLTx(ctx context.Context, readOnly bool) *sql.Tx
	CanInvite(ctx context.Context, sqlTx *sql.Tx, userID, invitedID string) (bool, error)
	Create(ctx context.Context, eventID string, event CreateEvent) error
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
	IsPublic(ctx context.Context, sqlTx *sql.Tx, eventID string) (bool, error)
	RemoveEdge(ctx context.Context, eventID string, predicate predicate, userID string) error
	Search(ctx context.Context, query string, params params.Query) ([]Event, error)
	SQLTx(ctx context.Context, readOnly bool, f func(tx *sql.Tx) (int, error)) (int, error)
	Update(ctx context.Context, sqlTx *sql.Tx, eventID string, event UpdateEvent) error

	media.Service
	product.Service
	role.Service
	zone.Service
}

type service struct {
	db *sql.DB
	dc *dgo.Dgraph
	mc *memcache.Client

	mediaService   media.Service
	productService product.Service
	roleService    role.Service
	zoneService    zone.Service

	metrics metrics
}

// NewService returns a new event service.
func NewService(db *sql.DB, dc *dgo.Dgraph, mc *memcache.Client) Service {
	return &service{
		db:             db,
		dc:             dc,
		mc:             mc,
		mediaService:   media.NewService(db, mc),
		productService: product.NewService(db, mc),
		roleService:    role.NewService(db, mc),
		zoneService:    zone.NewService(db, mc),
		metrics:        initMetrics(),
	}
}

// AddEdge creates an edge between the event and the user.
func (s *service) AddEdge(ctx context.Context, eventID string, predicate predicate, userID string) error {
	s.metrics.incMethodCalls("AddEdge")

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

		ids := dgraph.ParseRDFULIDs(res.Rdf)
		return len(ids) != 0, nil
	case "nobody":
		return false, nil
	}

	return true, nil
}

// ClonePermissions takes the permissions from the exporter event and creates them in the importer event.
func (s *service) ClonePermissions(ctx context.Context, sqlTx *sql.Tx, exporterEventID, importerEventID string) error {
	s.metrics.incMethodCalls("ClonePermissions")
	return s.roleService.ClonePermissions(ctx, sqlTx, exporterEventID, importerEventID)
}

// CloneRoles takes the role from the exporter event and creates them in the importer event.
func (s *service) CloneRoles(ctx context.Context, sqlTx *sql.Tx, exporterEventID, importerEventID string) error {
	s.metrics.incMethodCalls("CloneRoles")
	return s.roleService.CloneRoles(ctx, sqlTx, exporterEventID, importerEventID)
}

// Create creates a new event.
func (s *service) Create(ctx context.Context, eventID string, event CreateEvent) error {
	s.metrics.incMethodCalls("Create")

	sqlTx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "starting transaction")
	}
	defer sqlTx.Rollback()

	exists, err := postgres.QueryBool(ctx, sqlTx, "SELECT EXISTS(SELECT 1 FROM users WHERE id=$1)", event.HostID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.Errorf("user with id %q does not exist", event.HostID)
	}

	q1 := `INSERT INTO events 
	(id, name, type, public, start_time, end_time, slots, min_age, ticket_cost, updated_at)
	VALUES 
	($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err = sqlTx.ExecContext(ctx, q1, eventID, event.Name, event.Type, event.Public,
		event.StartTime, event.EndTime, event.Slots, event.MinAge, event.TicketCost, time.Time{})
	if err != nil {
		return errors.Wrap(err, "creating event")
	}

	// Create host and attendant roles
	q2 := `INSERT INTO events_roles 
	(event_id, name, permissions_keys) 
	VALUES 
	($1, $2, $3), ($1, $4, $5)`
	_, err = sqlTx.ExecContext(ctx, q2, eventID, role.Host, permissions.All,
		role.Attendant, permissions.Access)
	if err != nil {
		return errors.Wrap(err, "setting role")
	}

	q3 := "INSERT INTO events_users_roles (event_id, user_id, role_name) VALUES ($1, $2, $3)"
	if _, err := sqlTx.ExecContext(ctx, q3, eventID, event.HostID, role.Host); err != nil {
		return errors.Wrap(err, "setting user role")
	}

	q4 := `INSERT INTO events_locations 
	(event_id, virtual, country, state, city, address, platform, url)
	VALUES
	($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err = sqlTx.ExecContext(ctx, q4, eventID, event.Location.Virtual, event.Location.Country,
		event.Location.State, event.Location.City, event.Location.Address, event.Location.Platform,
		event.Location.URL)
	if err != nil {
		return errors.Wrap(err, "storing event location")
	}

	err = dgraph.Mutation(ctx, s.dc, func(dgraphTx *dgo.Txn) error {
		if err := dgraph.CreateNode(ctx, dgraphTx, dgraph.Event, eventID); err != nil {
			return err
		}
		// Add the host as confirmed to the event
		req := dgraph.EventEdgeRequest(eventID, string(Confirmed), event.HostID, true)
		if _, err := dgraphTx.Do(ctx, req); err != nil {
			return errors.Wrapf(err, "adding %s edge", Confirmed)
		}
		return nil
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
func (s *service) CreateMedia(ctx context.Context, sqlTx *sql.Tx, eventID string, media media.CreateMedia) error {
	s.metrics.incMethodCalls("CreateMedia")
	return s.mediaService.CreateMedia(ctx, sqlTx, eventID, media)
}

// CreatePermission creates a permission inside the event.
func (s *service) CreatePermission(ctx context.Context, sqlTx *sql.Tx, eventID string, permission role.Permission) error {
	s.metrics.incMethodCalls("CreatePermission")
	return s.roleService.CreatePermission(ctx, sqlTx, eventID, permission)
}

// CreateProduct adds a product to the event.
func (s *service) CreateProduct(ctx context.Context, sqlTx *sql.Tx, eventID string, product product.Product) error {
	s.metrics.incMethodCalls("CreateProduct")
	return s.productService.CreateProduct(ctx, sqlTx, eventID, product)
}

// CreateRole creates a new role inside an event.
func (s *service) CreateRole(ctx context.Context, sqlTx *sql.Tx, eventID string, role role.Role) error {
	s.metrics.incMethodCalls("CreateRole")
	return s.roleService.CreateRole(ctx, sqlTx, eventID, role)
}

// CreateZone creates a new zone inside an event.
func (s *service) CreateZone(ctx context.Context, sqlTx *sql.Tx, eventID string, zone zone.Zone) error {
	s.metrics.incMethodCalls("CreateZone")
	return s.zoneService.CreateZone(ctx, sqlTx, eventID, zone)
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
	eventQ := `SELECT 
	id, name, type, public, start_time, end_time, slots, 
	min_age, ticket_cost, created_at, updated_at 
	FROM events 
	WHERE id=$1`
	row := sqlTx.QueryRowContext(ctx, eventQ, eventID)
	err := row.Scan(&event.ID, &event.Name, &event.Type, &event.Public,
		&event.StartTime, &event.EndTime, &event.Slots, &event.MinAge,
		&event.TicketCost, &event.CreatedAt, &event.UpdatedAt)
	if err != nil {
		return Event{}, errors.Wrap(err, "fetching event")
	}

	countsQ := `query q($id: string) {
		q(func: eq(event_id, $id)) {
			count(banned)
			count(confirmed)
			count(invited)
			count(liked_by)
		}
	}`
	vars := map[string]string{"$id": eventID}

	res, err := s.dc.NewReadOnlyTxn().QueryRDFWithVars(ctx, countsQ, vars)
	if err != nil {
		return Event{}, errors.Wrap(err, "querying count")
	}

	mp, err := dgraph.ParseCountWithMap(res.Rdf)
	if err != nil {
		return Event{}, err
	}
	event.BannedCount = mp["banned"]
	event.ConfirmedCount = mp["confirmed"]
	event.InvitedCount = mp["invited"]
	event.LikesCount = mp["liked_by"]

	locationQ := `SELECT 
	virtual, country, state, zip_code, city, address, platform, url
	FROM events_locations WHERE event_id=$1`
	locRow := sqlTx.QueryRowContext(ctx, locationQ, eventID)
	event.Location, err = scanEventLocation(locRow)
	if err != nil {
		return Event{}, nil
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
	query := "SELECT user_id FROM events_users_roles WHERE event_id=$1 AND role_name='host'"
	q := postgres.AddPagination(query, "user_id", params)
	rows, err := sqlTx.QueryContext(ctx, q, eventID)
	if err != nil {
		return nil, errors.Wrap(err, "fetching users")
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

func (s *service) GetMedia(ctx context.Context, sqlTx *sql.Tx, eventID string, params params.Query) ([]media.Media, error) {
	s.metrics.incMethodCalls("GetMedia")
	return s.mediaService.GetMedia(ctx, sqlTx, eventID, params)
}

// GetPermissions returns all event's permissions.
func (s *service) GetPermissions(ctx context.Context, sqlTx *sql.Tx, eventID string) ([]role.Permission, error) {
	s.metrics.incMethodCalls("GetPermissions")
	return s.roleService.GetPermissions(ctx, sqlTx, eventID)
}

// GetProducts returns the product from an event.
func (s *service) GetProducts(ctx context.Context, sqlTx *sql.Tx, eventID string, params params.Query) ([]product.Product, error) {
	s.metrics.incMethodCalls("GetProducts")
	return s.productService.GetProducts(ctx, sqlTx, eventID, params)
}

// GetRole returns a role in a given event.
func (s *service) GetRole(ctx context.Context, sqlTx *sql.Tx, eventID, name string) (role.Role, error) {
	s.metrics.incMethodCalls("GetRole")
	return s.roleService.GetRole(ctx, sqlTx, eventID, name)
}

// GetRoles returns all event's role.
func (s *service) GetRoles(ctx context.Context, sqlTx *sql.Tx, eventID string) ([]role.Role, error) {
	s.metrics.incMethodCalls("GetRoles")
	return s.roleService.GetRoles(ctx, sqlTx, eventID)
}

// GetUserRole returns user's role inside the event.
func (s *service) GetUserRole(ctx context.Context, sqlTx *sql.Tx, eventID, userID string) (role.Role, error) {
	s.metrics.incMethodCalls("GetUserRole")
	return s.roleService.GetUserRole(ctx, sqlTx, eventID, userID)
}

// GetZoneByName returns the permission keys required to enter a zone.
func (s *service) GetZoneByName(ctx context.Context, sqlTx *sql.Tx, eventID, name string) (zone.Zone, error) {
	s.metrics.incMethodCalls("GetZoneByName")
	return s.zoneService.GetZoneByName(ctx, sqlTx, eventID, name)
}

// GetZones gets an event's zones.
func (s *service) GetZones(ctx context.Context, sqlTx *sql.Tx, eventID string) ([]zone.Zone, error) {
	s.metrics.incMethodCalls("GetZones")
	return s.zoneService.GetZones(ctx, sqlTx, eventID)
}

// IsHost returns if the user's role in the events passed is host.
func (s *service) IsHost(ctx context.Context, sqlTx *sql.Tx, userID string, eventIDs ...string) (bool, error) {
	s.metrics.incMethodCalls("IsHost")
	return s.roleService.IsHost(ctx, sqlTx, userID, eventIDs...)
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

// Search returns events matching the given query.
func (s *service) Search(ctx context.Context, query string, params params.Query) ([]Event, error) {
	s.metrics.incMethodCalls("Search")

	searchFields := []string{"id", "name"}
	q := postgres.FullTextSearch(postgres.Users, searchFields, query, params)
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, errors.Wrap(err, "events searching")
	}

	events, err := scanEvents(rows)
	if err != nil {
		return nil, err
	}

	return events, nil
}

// SetRoles assigns a role to n users inside an event.
func (s *service) SetRoles(ctx context.Context, sqlTx *sql.Tx, eventID, roleName string, userIDs ...string) error {
	s.metrics.incMethodCalls("SetRoles")
	return s.roleService.SetRoles(ctx, sqlTx, eventID, roleName, userIDs...)
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

	// The query includes two positional parameters: id and updated_at
	q := updateEventQuery(event)
	_, err := sqlTx.ExecContext(ctx, eventID, q, eventID, time.Now())
	if err != nil {
		return errors.Wrap(err, "updating event")
	}

	if err := s.mc.Delete(eventID); err != nil && err != memcache.ErrCacheMiss {
		return errors.Wrap(err, "memcached: deleting event")
	}

	return nil
}

// UpdateMedia updates event's media.
func (s *service) UpdateMedia(ctx context.Context, sqlTx *sql.Tx, eventID string, media media.Media) error {
	s.metrics.incMethodCalls("UpdateMedia")
	return s.mediaService.UpdateMedia(ctx, sqlTx, eventID, media)
}

// UpdatePermission ..
func (s *service) UpdatePermission(ctx context.Context, sqlTx *sql.Tx, eventID string, permission role.Permission) error {
	s.metrics.incMethodCalls("UpdatePermission")
	return s.roleService.UpdatePermission(ctx, sqlTx, eventID, permission)
}

// UpdateRole ..
func (s *service) UpdateRole(ctx context.Context, sqlTx *sql.Tx, eventID string, role role.Role) error {
	s.metrics.incMethodCalls("UpdateRole")
	return s.roleService.UpdateRole(ctx, sqlTx, eventID, role)
}

// UpdateProduct updates an event product.
func (s *service) UpdateProduct(ctx context.Context, sqlTx *sql.Tx, eventID string, product product.Product) error {
	s.metrics.incMethodCalls("UpdateProduct")
	return s.productService.UpdateProduct(ctx, sqlTx, eventID, product)
}

// UserHasRole returns if the user has a role inside the event or not.
func (s *service) UserHasRole(ctx context.Context, sqlTx *sql.Tx, eventID, userID string) (bool, error) {
	s.metrics.incMethodCalls("UserHasRole")
	return s.roleService.UserHasRole(ctx, sqlTx, eventID, userID)
}

// queryUsers returns the users found in the dgraph query passed.
func (s *service) queryUsers(ctx context.Context, sqlTx *sql.Tx, query string, vars map[string]string, params params.Query) ([]User, error) {
	res, err := s.dc.NewReadOnlyTxn().QueryRDFWithVars(ctx, query, vars)
	if err != nil {
		return nil, errors.Wrap(err, "dgraph: fetching users ids")
	}

	usersIds := dgraph.ParseRDFULIDs(res.Rdf)
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
