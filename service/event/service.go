package event

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"

	"github.com/GGP1/groove/internal/bufferpool"
	"github.com/GGP1/groove/internal/cache"
	"github.com/GGP1/groove/internal/httperr"
	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/roles"
	"github.com/GGP1/groove/internal/sqltx"
	"github.com/GGP1/groove/model"
	"github.com/GGP1/groove/service/auth"
	"github.com/GGP1/groove/service/event/role"
	"github.com/GGP1/groove/service/notification"
	"github.com/GGP1/groove/storage/dgraph"
	"github.com/GGP1/groove/storage/postgres"
	"github.com/GGP1/sqan"

	"github.com/dgraph-io/dgo/v210"
	"github.com/dgraph-io/dgo/v210/protos/api"
	"github.com/pkg/errors"
)

// Service represents the event service.
type Service interface {
	AddEdge(ctx context.Context, eventID string, predicate dgraph.Predicate, userID string) error
	AvailableSlots(ctx context.Context, eventID string) (int64, error)
	Create(ctx context.Context, eventID string, event CreateEvent) error
	Delete(ctx context.Context, eventID string) error
	GetBanned(ctx context.Context, eventID string, params params.Query) ([]model.ListUser, error)
	GetBannedCount(ctx context.Context, eventID string) (*uint64, error)
	GetBannedFriends(ctx context.Context, eventID, userID string, params params.Query) ([]model.ListUser, error)
	GetBannedFriendsCount(ctx context.Context, eventID, userID string) (*uint64, error)
	GetByID(ctx context.Context, eventID string) (Event, error)
	GetHosts(ctx context.Context, eventID string, params params.Query) ([]model.ListUser, error)
	GetInvited(ctx context.Context, eventID string, params params.Query) ([]model.ListUser, error)
	GetInvitedCount(ctx context.Context, eventID string) (int64, error)
	GetInvitedFriends(ctx context.Context, eventID, userID string, params params.Query) ([]model.ListUser, error)
	GetInvitedFriendsCount(ctx context.Context, eventID, userID string) (int64, error)
	GetLikedBy(ctx context.Context, eventID string, params params.Query) ([]model.ListUser, error)
	GetLikedByCount(ctx context.Context, eventID string) (*uint64, error)
	GetLikedByFriends(ctx context.Context, eventID, userID string, params params.Query) ([]model.ListUser, error)
	GetLikedByFriendsCount(ctx context.Context, eventID, userID string) (*uint64, error)
	GetRecommended(ctx context.Context, session auth.Session, userCoords Coordinates, params params.Query) ([]Event, error)
	GetStatistics(ctx context.Context, eventID string) (Statistics, error)
	IsBanned(ctx context.Context, eventID, userID string) (bool, error)
	IsInvited(ctx context.Context, eventID, userID string) (bool, error)
	IsPublic(ctx context.Context, eventID string) (bool, error)
	RemoveEdge(ctx context.Context, eventID string, predicate dgraph.Predicate, userID string) error
	Search(ctx context.Context, query string, session auth.Session, params params.Query) ([]Event, error)
	SearchByLocation(ctx context.Context, session auth.Session, location LocationSearch) ([]Event, error)
	Update(ctx context.Context, eventID string, event UpdateEvent) error
}

type service struct {
	db    *sql.DB
	dc    *dgo.Dgraph
	cache cache.Client

	notificationService notification.Service
	roleService         role.Service

	metrics metrics
}

// NewService returns a new event service.
func NewService(
	db *sql.DB,
	dc *dgo.Dgraph,
	cache cache.Client,
	notificationService notification.Service,
	roleService role.Service,
) Service {
	return &service{
		db:                  db,
		dc:                  dc,
		cache:               cache,
		notificationService: notificationService,
		roleService:         roleService,
		metrics:             initMetrics(),
	}
}

// AddEdge creates an edge between the event and the user.
func (s service) AddEdge(ctx context.Context, eventID string, predicate dgraph.Predicate, userID string) error {
	s.metrics.incMethodCalls("AddEdge")

	return dgraph.AddEventEdge(ctx, s.dc, eventID, predicate, userID)
}

// AvailableSlots returns an even'ts number of slots available.
func (s service) AvailableSlots(ctx context.Context, eventID string) (int64, error) {
	s.metrics.incMethodCalls("AvailableSlots")

	q := "SELECT slots FROM events WHERE id=$1"
	slots, err := postgres.QueryInt(ctx, s.db, q, eventID)
	if err != nil {
		return 0, errors.Wrap(err, "scanning slots")
	}

	membersCount, err := s.roleService.GetMembersCount(ctx, eventID)
	if err != nil {
		return 0, err
	}

	return slots - membersCount, nil
}

// Create creates a new event.
func (s service) Create(ctx context.Context, eventID string, event CreateEvent) error {
	s.metrics.incMethodCalls("Create")

	sqlTx, ctx := postgres.BeginTx(ctx, s.db)
	defer sqlTx.Rollback()

	q1 := `INSERT INTO events 
	(id, name, description, type, ticket_type, virtual, url, logo_url, header_url, address, 
	latitude, longitude, public, cron, start_date, end_date, slots, min_age, updated_at)
	VALUES 
	($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)`
	_, err := sqlTx.ExecContext(ctx, q1, eventID, event.Name, event.Description, event.Type,
		event.TicketType, event.Virtual, event.URL, event.LogoURL, event.HeaderURL, event.Location.Address,
		event.Location.Coordinates.Latitude, event.Location.Coordinates.Longitude, event.Public,
		event.Cron, event.StartDate, event.EndDate, event.Slots, event.MinAge, time.Time{})
	if err != nil {
		return errors.Wrap(err, "creating event")
	}

	if err := s.roleService.SetReservedRole(ctx, eventID, event.HostID, roles.Host); err != nil {
		return err
	}

	err = dgraph.Mutation(ctx, s.dc, func(dgraphTx *dgo.Txn) error {
		return dgraph.CreateNode(ctx, dgraphTx, model.Event, eventID)
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

// Delete removes an event and all its edges.
func (s service) Delete(ctx context.Context, eventID string) error {
	s.metrics.incMethodCalls("Delete")

	sqlTx, _ := postgres.BeginTx(ctx, s.db)
	defer sqlTx.Rollback()

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

	if err := sqlTx.Commit(); err != nil {
		return err
	}

	if err := s.cache.Delete(model.Event.CacheKey(eventID)); err != nil {
		return errors.Wrap(err, "deleting event")
	}

	s.metrics.registeredEvents.Dec()
	return nil
}

// GetBanned returns event's banned guests.
func (s service) GetBanned(ctx context.Context, eventID string, params params.Query) ([]model.ListUser, error) {
	s.metrics.incMethodCalls("GetBanned")
	query := banned
	if params.LookupID != "" {
		query = bannedLookup
	}

	vars := dgraph.QueryVars(eventID, params)
	return s.queryUsers(ctx, getQuery[query], vars, params)
}

// GetBannedCount returns event's banned guests count.
func (s service) GetBannedCount(ctx context.Context, eventID string) (*uint64, error) {
	s.metrics.incMethodCalls("GetBannedCount")
	return dgraph.GetCount(ctx, s.dc, getQuery[bannedCount], eventID)
}

// GetBannedFriends returns event likes users that are friend of the user passed.
func (s service) GetBannedFriends(ctx context.Context, eventID, userID string, params params.Query) ([]model.ListUser, error) {
	s.metrics.incMethodCalls("GetBannedFriends")

	query := bannedFriends
	if params.LookupID != "" {
		query = bannedFriendsLookup
	}

	vars := mixedQueryVars(eventID, userID, params)
	return s.queryUsers(ctx, getMixedQuery[query], vars, params)
}

// GetBannedFriendsCount returns event's banned friends count.
func (s service) GetBannedFriendsCount(ctx context.Context, eventID, userID string) (*uint64, error) {
	s.metrics.incMethodCalls("GetBannedFriendsCount")

	vars := map[string]string{
		"$event_id": eventID,
		"$user_id":  userID,
	}
	return dgraph.GetCountWithVars(ctx, s.dc, getMixedQuery[bannedFriendsCount], vars)
}

// GetByID returns the event with the id passed.
func (s service) GetByID(ctx context.Context, eventID string) (Event, error) {
	s.metrics.incMethodCalls("GetByID")

	q := `SELECT id, name, description, virtual, url, logo_url, header_url, address, latitude, longitude, 
	type, ticket_type, public, cron, start_date, end_date, slots, min_age, created_at, updated_at 
	FROM events WHERE id=$1`
	rows, err := s.db.QueryContext(ctx, q, eventID)
	if err != nil {
		return Event{}, errors.Wrap(err, "querying event")
	}

	var event Event
	if err := sqan.Row(&event, rows); err != nil {
		return Event{}, errors.Wrap(err, "scanning event")
	}

	return event, nil
}

// GetHosts returns event's hosts.
func (s service) GetHosts(ctx context.Context, eventID string, params params.Query) ([]model.ListUser, error) {
	s.metrics.incMethodCalls("GetHosts")

	whereCond := "id IN (SELECT user_id FROM events_users_roles WHERE event_id=$1 AND role_name='host')"
	q := postgres.SelectWhere(model.User, whereCond, "id", params)
	rows, err := s.db.QueryContext(ctx, q, eventID)
	if err != nil {
		return nil, errors.Wrap(err, "fetching users")
	}

	var users []model.ListUser
	if err := sqan.Rows(&users, rows); err != nil {
		return nil, err
	}

	return users, nil
}

// GetInvited returns event's invited users.
func (s service) GetInvited(ctx context.Context, eventID string, params params.Query) ([]model.ListUser, error) {
	s.metrics.incMethodCalls("GetInvited")

	return s.roleService.GetUsersByRole(ctx, eventID, string(roles.Viewer), params)
}

// GetInvited returns event's invited users count.
func (s service) GetInvitedCount(ctx context.Context, eventID string) (int64, error) {
	s.metrics.incMethodCalls("GetInvitedCount")

	return s.roleService.GetUsersCountByRole(ctx, eventID, string(roles.Viewer))
}

// GetInvitedFriends returns event invited users that are friends of the user passed.
func (s service) GetInvitedFriends(ctx context.Context, eventID, userID string, params params.Query) ([]model.ListUser, error) {
	s.metrics.incMethodCalls("GetInvitedFriends")

	return s.roleService.GetUserFriendsByRole(ctx, eventID, userID, string(roles.Viewer), params)
}

// GetInvitedFriendsCount returns event's invited friends count.
func (s service) GetInvitedFriendsCount(ctx context.Context, eventID, userID string) (int64, error) {
	s.metrics.incMethodCalls("GetInvitedFriendsCount")

	return s.roleService.GetUserFriendsCountByRole(ctx, eventID, userID, string(roles.Viewer))
}

// GetLikedBy returns users liking the event.
func (s service) GetLikedBy(ctx context.Context, eventID string, params params.Query) ([]model.ListUser, error) {
	s.metrics.incMethodCalls("GetLikedBy")

	query := likedBy
	if params.LookupID != "" {
		query = likedByLookup
	}

	vars := dgraph.QueryVars(eventID, params)
	return s.queryUsers(ctx, getQuery[query], vars, params)
}

// GetRecommended returns a list of events that may be interesting for the logged in user, all of them must be public.
func (s service) GetRecommended(ctx context.Context, session auth.Session, userCoords Coordinates, params params.Query) ([]Event, error) {
	s.metrics.incMethodCalls("GetRecommended")

	query := `query q($user_id: string) {
		var(func: type("Event")) {
			likes as count(liked_by)
		}
		likedByTop(func: type("Event"), orderdesc: val(likes), first: 50) {
			event_id
		}
		
		friends(func: eq(user_id, $user_id)) {
			friend (first: 50) {
				user_id
			}
		}
	}`
	res, err := s.dc.NewReadOnlyTxn().QueryRDFWithVars(ctx, query, map[string]string{"$user_id": session.ID})
	if err != nil {
		return nil, errors.Wrap(err, "multi query")
	}

	mp, err := dgraph.ParseRDF(res.Rdf)
	if err != nil {
		return nil, err
	}

	eventIDs := mp["event_id"]
	friendIDs := mp["user_id"]

	buf := bufferpool.Get()
	buf.WriteString("SELECT ")
	postgres.WriteFields(buf, model.Event, params.Fields)
	buf.WriteString(" FROM events WHERE public=true AND ((latitude BETWEEN $1 AND $2) AND (longitude BETWEEN $3 AND $4)")
	if len(eventIDs) > 0 {
		buf.WriteString(" OR id IN ")
		postgres.WriteIDs(buf, eventIDs)
	}
	if len(friendIDs) > 0 {
		buf.WriteString(" OR id IN (SELECT event_id FROM events_users_roles WHERE user_id IN ")
		postgres.WriteIDs(buf, friendIDs)
		buf.WriteRune(')')
	}
	buf.WriteRune(')')

	q := postgres.AddPagination(buf.String(), "id", params)
	bufferpool.Put(buf)

	latMin := math.Mod(userCoords.Latitude-1, 90)
	latMax := math.Mod(userCoords.Latitude+1, 90)
	longMin := math.Mod(userCoords.Longitude-1, 180)
	longMax := math.Mod(userCoords.Longitude+1, 180)

	// Maybe add the events liked by friends in the future
	// TODO: consider using temporary tables instead of unions
	rows, err := s.db.QueryContext(ctx, q, latMin, latMax, longMin, longMax)
	if err != nil {
		return nil, errors.Wrap(err, "querying recommended events")
	}

	var events []Event
	if err := sqan.Rows(&events, rows); err != nil {
		return nil, errors.Wrap(err, "scanning events")
	}

	return events, nil
}

// GetLikedByCount returns the number of users liking the event.
func (s service) GetLikedByCount(ctx context.Context, eventID string) (*uint64, error) {
	s.metrics.incMethodCalls("GetLikedByCount")
	return dgraph.GetCount(ctx, s.dc, getQuery[likedByCount], eventID)
}

// GetLikedByFriends returns the users that are friends of the user passed and liked the event.
func (s service) GetLikedByFriends(ctx context.Context, eventID, userID string, params params.Query) ([]model.ListUser, error) {
	s.metrics.incMethodCalls("GetLikedByFriends")

	query := likedByFriends
	if params.LookupID != "" {
		query = likedByFriendsLookup
	}

	vars := mixedQueryVars(eventID, userID, params)
	return s.queryUsers(ctx, getMixedQuery[query], vars, params)
}

// GetLikedByFriendsCount returns event's liked by friends count.
func (s service) GetLikedByFriendsCount(ctx context.Context, eventID, userID string) (*uint64, error) {
	s.metrics.incMethodCalls("GetLikedByFriendsCount")

	vars := map[string]string{
		"$event_id": eventID,
		"$user_id":  userID,
	}
	return dgraph.GetCountWithVars(ctx, s.dc, getMixedQuery[likedByFriendsCount], vars)
}

// GetStatistics returns events' predicates statistics.
func (s service) GetStatistics(ctx context.Context, eventID string) (Statistics, error) {
	s.metrics.incMethodCalls("GetStatistics")

	q := `query q($id: string) {
		q(func: eq(event_id, $id)) {
			count(banned)
			count(invited)
			count(liked_by)
		}
	}`
	vars := map[string]string{"$id": eventID}

	res, err := s.dc.NewReadOnlyTxn().QueryRDFWithVars(ctx, q, vars)
	if err != nil {
		return Statistics{}, errors.Wrap(err, "querying count")
	}

	mp, err := dgraph.ParseCountWithMap(res.Rdf)
	if err != nil {
		return Statistics{}, err
	}

	membersCount, err := s.roleService.GetMembersCount(ctx, eventID)
	if err != nil {
		return Statistics{}, err
	}

	return Statistics{
		Banned:  mp["banned"],
		Members: membersCount,
		Invited: mp["invited"],
		Likes:   mp["liked_by"],
	}, nil
}

// IsBanned returns if the user is banned or not from the event.
func (s service) IsBanned(ctx context.Context, eventID, userID string) (bool, error) {
	s.metrics.incMethodCalls("IsBanned")

	vars := map[string]string{
		"$id":        eventID,
		"$lookup_id": userID,
	}
	res, err := s.dc.NewReadOnlyTxn().QueryRDFWithVars(ctx, getQuery[isBanned], vars)
	if err != nil {
		return false, err
	}
	count, err := dgraph.ParseCount(res.Rdf)
	if err != nil {
		return false, err
	}
	return *count == 1, nil
}

// IsInvited returns if the user is invited or not to the event.
func (s service) IsInvited(ctx context.Context, eventID, userID string) (bool, error) {
	s.metrics.incMethodCalls("IsInvited")

	role, err := s.roleService.GetUserRole(ctx, eventID, userID)
	if err != nil {
		return false, err
	}
	return role.Name == string(roles.Viewer), nil
}

// IsPublic returns if the event is public or not.
func (s service) IsPublic(ctx context.Context, eventID string) (bool, error) {
	s.metrics.incMethodCalls("IsPublic")
	sqlTx := sqltx.FromContext(ctx)

	cacheKey := eventID + "_visibility"
	if v, err := s.cache.Get(cacheKey); err == nil {
		return bytes.Equal(v, []byte("1")), nil
	}

	var public bool
	row := sqlTx.QueryRowContext(ctx, "SELECT public FROM events WHERE id=$1", eventID)
	if err := row.Scan(&public); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, httperr.BadRequest(fmt.Sprintf("event with id %q does not exists", eventID))
		}
		return false, err
	}

	if err := s.cache.Set(cacheKey, boolToByte(public)); err != nil {
		return false, errors.Wrap(err, "setting event visibility to the cache")
	}

	return public, nil
}

func (s service) RemoveEdge(ctx context.Context, eventID string, predicate dgraph.Predicate, userID string) error {
	s.metrics.incMethodCalls("RemoveEdge")

	return dgraph.Mutation(ctx, s.dc, func(tx *dgo.Txn) error {
		req := dgraph.EventEdgeRequest(eventID, predicate, userID, false)
		_, err := tx.Do(ctx, req)
		return err
	})
}

// Search returns events matching the given query.
func (s service) Search(ctx context.Context, query string, session auth.Session, params params.Query) ([]Event, error) {
	s.metrics.incMethodCalls("Search")

	q := postgres.FullTextSearch(model.Event, params)
	rows, err := s.db.QueryContext(ctx, q, postgres.ToTSQuery(query), session.ID)
	if err != nil {
		return nil, errors.Wrap(err, "events searching")
	}

	var events []Event
	if err := sqan.Rows(&events, rows); err != nil {
		return nil, err
	}

	return events, nil
}

// SearchByLocation returns the events located within the coordinates given.
func (s service) SearchByLocation(ctx context.Context, session auth.Session, location LocationSearch) ([]Event, error) {
	s.metrics.incMethodCalls("SearchByLocation")

	latMin := location.Latitude - location.LatitudeDelta
	latMax := location.Latitude + location.LatitudeDelta
	longMin := location.Longitude - location.LongitudeDelta
	longMax := location.Longitude + location.LongitudeDelta

	q := `SELECT
	id, name, description, header_url, logo_url, latitude, longitude
	FROM events WHERE
	(latitude BETWEEN $1 AND $2) AND
	(longitude BETWEEN $3 AND $4) AND
	(public=true OR id IN (SELECT event_id FROM events_users_roles WHERE user_id=$5))`
	rows, err := s.db.QueryContext(ctx, q, latMin, latMax, longMin, longMax, session.ID)
	if err != nil {
		return nil, err
	}

	var events []Event
	if err := sqan.Rows(&events, rows); err != nil {
		return nil, err
	}

	return events, nil
}

// Update updates an event.
func (s service) Update(ctx context.Context, eventID string, event UpdateEvent) error {
	s.metrics.incMethodCalls("Update")

	var endDate time.Time
	if err := s.db.QueryRowContext(ctx, "SELECT end_date FROM events WHERE id=$1", eventID).Scan(&endDate); err != nil {
		return errors.Wrap(err, "scanning end_date")
	}

	if endDate.Before(time.Now()) {
		return httperr.Forbidden("cannot modify an event that has ended")
	}

	if event.Slots != nil {
		membersCount, err := s.roleService.GetMembersCount(ctx, eventID)
		if err != nil {
			return err
		}

		if *event.Slots < uint64(membersCount) {
			return httperr.BadRequest("slots must be higher than the current number of members")
		}
	}

	q := `UPDATE events SET 
	name = COALESCE($2,name),
	description = COALESCE($3,description), 
	type = COALESCE($4,type),
	url = COALESCE($5,url),
	logo_url = COALESCE($6,logo_url),
	header_url = COALESCE($7,header_url),
	address = COALESCE($8,address),
	latitude = COALESCE($9,latitude),
	longitude = COALESCE($10,longitude),
	cron = COALESCE($11,cron),
	start_date = COALESCE($12,start_date),
	end_date = COALESCE($13,end_date),
	slots = COALESCE($14,slots),
	updated_at = $15
	WHERE id = $1`
	_, err := s.db.ExecContext(ctx, q, eventID, event.Name, event.Description, event.Type,
		event.URL, event.LogoURL, event.HeaderURL, event.Location.Address,
		event.Location.Coordinates.Latitude, event.Location.Coordinates.Longitude,
		event.Cron, event.StartDate, event.EndDate, event.Slots, time.Now())
	if err != nil {
		return errors.Wrap(err, "postgres: updating event")
	}

	if err := s.cache.Delete(model.Event.CacheKey(eventID)); err != nil {
		return errors.Wrap(err, "updating event")
	}

	return nil
}

// queryUsers returns the users found in the dgraph query passed.
func (s service) queryUsers(ctx context.Context, query string, vars map[string]string, params params.Query) ([]model.ListUser, error) {
	res, err := s.dc.NewReadOnlyTxn().QueryRDFWithVars(ctx, query, vars)
	if err != nil {
		return nil, errors.Wrap(err, "dgraph: fetching users ids")
	}

	usersIDs := dgraph.ParseRDFULIDs(res.Rdf)
	if len(usersIDs) == 0 {
		return nil, nil
	}

	q := postgres.SelectInID(model.User, params.Fields, usersIDs)
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, errors.Wrap(err, "postgres: fetching users")
	}

	var users []model.ListUser
	if err := sqan.Rows(&users, rows); err != nil {
		return nil, err
	}

	return users, nil
}

func boolToByte(b bool) []byte {
	if b {
		return []byte("1")
	}
	return []byte("0")
}
