package event

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"

	"github.com/GGP1/groove/internal/cache"
	"github.com/GGP1/groove/internal/httperr"
	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/roles"
	"github.com/GGP1/groove/internal/txgroup"
	"github.com/GGP1/groove/internal/ulid"
	"github.com/GGP1/groove/model"
	"github.com/GGP1/groove/service/event/role"
	"github.com/GGP1/groove/service/notification"
	"github.com/GGP1/groove/storage/postgres"
	redi "github.com/GGP1/groove/storage/redis"
	"github.com/GGP1/sqan"

	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
)

// Service represents the event service.
type Service interface {
	AvailableSlots(ctx context.Context, eventID string) (int64, error)
	Ban(ctx context.Context, eventID, userID string) error
	Create(ctx context.Context, event model.CreateEvent) (string, error)
	Delete(ctx context.Context, eventID string) error
	GetBanned(ctx context.Context, eventID string, params params.Query) ([]model.User, error)
	GetBannedCount(ctx context.Context, eventID string) (int64, error)
	GetBannedFriends(ctx context.Context, eventID, userID string, params params.Query) ([]model.User, error)
	GetBannedFriendsCount(ctx context.Context, eventID, userID string) (int64, error)
	GetByID(ctx context.Context, eventID string) (model.Event, error)
	GetHosts(ctx context.Context, eventID string, params params.Query) ([]model.User, error)
	GetInvited(ctx context.Context, eventID string, params params.Query) ([]model.User, error)
	GetInvitedCount(ctx context.Context, eventID string) (int64, error)
	GetInvitedFriends(ctx context.Context, eventID, userID string, params params.Query) ([]model.User, error)
	GetInvitedFriendsCount(ctx context.Context, eventID, userID string) (int64, error)
	GetLikes(ctx context.Context, eventID string, params params.Query) ([]model.User, error)
	GetLikesCount(ctx context.Context, eventID string) (int64, error)
	GetLikesByFriends(ctx context.Context, eventID, userID string, params params.Query) ([]model.User, error)
	GetLikesByFriendsCount(ctx context.Context, eventID, userID string) (int64, error)
	GetRecommended(ctx context.Context, userID string, userCoords model.Coordinates, params params.Query) ([]model.Event, error)
	GetStatistics(ctx context.Context, eventID string) (model.EventStatistics, error)
	IsBanned(ctx context.Context, eventID, userID string) (bool, error)
	IsInvited(ctx context.Context, eventID, userID string) (bool, error)
	IsPublic(ctx context.Context, eventID string) (bool, error)
	Like(ctx context.Context, eventID, userID string) error
	PrivacyFilter(ctx context.Context, eventID, userID string) error
	RemoveBan(ctx context.Context, eventID, userID string) error
	RemoveLike(ctx context.Context, eventID, userID string) error
	Search(ctx context.Context, query string, userID string, params params.Query) ([]model.Event, error)
	SearchByLocation(ctx context.Context, userID string, location model.LocationSearch) ([]model.Event, error)
	Update(ctx context.Context, eventID string, event model.UpdateEvent) error
}

type service struct {
	db  *sql.DB
	rdb *redis.Client

	notificationService notification.Service
	roleService         role.Service

	metrics metrics
}

// NewService returns a new event service.
func NewService(
	db *sql.DB,
	rdb *redis.Client,
	notificationService notification.Service,
	roleService role.Service,
) Service {
	return &service{
		db:                  db,
		rdb:                 rdb,
		notificationService: notificationService,
		roleService:         roleService,
		metrics:             initMetrics(),
	}
}

// AvailableSlots returns an even'ts number of slots available.
func (s *service) AvailableSlots(ctx context.Context, eventID string) (int64, error) {
	s.metrics.incMethodCalls("AvailableSlots")

	q := "SELECT slots FROM events WHERE id=$1"
	slots, err := postgres.Query[int64](ctx, s.db, q, eventID)
	if err != nil {
		return 0, errors.Wrap(err, "scanning slots")
	}

	membersCount, err := s.roleService.GetMembersCount(ctx, eventID)
	if err != nil {
		return 0, err
	}

	return slots - membersCount, nil
}

// Ban bans a user from an event.
func (s *service) Ban(ctx context.Context, eventID, userID string) error {
	s.metrics.incMethodCalls("Ban")

	sqlTx := txgroup.SQLTx(ctx)

	q := "INSERT INTO events_bans (event_id, user_id) VALUES ($1, $2)"
	if _, err := sqlTx.ExecContext(ctx, q, eventID, userID); err != nil {
		return err
	}

	return nil
}

// Create creates a new event.
func (s *service) Create(ctx context.Context, event model.CreateEvent) (string, error) {
	s.metrics.incMethodCalls("Create")

	sqlTx := txgroup.SQLTx(ctx)

	id := ulid.NewString()
	q1 := `INSERT INTO events 
	(id, name, description, type, ticket_type, virtual, url, logo_url, header_url, address, 
	latitude, longitude, public, cron, start_date, end_date, slots, min_age, updated_at)
	VALUES 
	($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)`
	_, err := sqlTx.ExecContext(ctx, q1, id, event.Name, event.Description, event.Type,
		event.TicketType, event.Virtual, event.URL, event.LogoURL, event.HeaderURL, event.Location.Address,
		event.Location.Coordinates.Latitude, event.Location.Coordinates.Longitude, event.Public,
		event.Cron, event.StartDate, event.EndDate, event.Slots, event.MinAge, time.Time{})
	if err != nil {
		return "", errors.Wrap(err, "creating event")
	}

	if err := s.roleService.SetRole(ctx, id, roles.Host, event.HostID); err != nil {
		return "", err
	}

	return id, nil
}

// Delete removes an event and all its edges.
func (s *service) Delete(ctx context.Context, eventID string) error {
	s.metrics.incMethodCalls("Delete")

	sqlTx := txgroup.SQLTx(ctx)

	if _, err := sqlTx.ExecContext(ctx, "DELETE FROM events WHERE id=$1", eventID); err != nil {
		return errors.Wrap(err, "postgres: deleting event")
	}

	if err := s.rdb.Del(ctx, model.T.Event.CacheKey(eventID)).Err(); err != nil {
		return errors.Wrap(err, "deleting event")
	}

	return nil
}

// GetBanned returns event's banned guests.
func (s *service) GetBanned(ctx context.Context, eventID string, params params.Query) ([]model.User, error) {
	s.metrics.incMethodCalls("GetBanned")

	q := "SELECT {fields} FROM {table} WHERE id IN (SELECT user_id FROM events_bans WHERE event_id=$1) {pag}"
	return s.scanUsers(ctx, params, q, eventID)
}

// GetBannedCount returns event's banned guests count.
func (s *service) GetBannedCount(ctx context.Context, eventID string) (int64, error) {
	s.metrics.incMethodCalls("GetBannedCount")

	return postgres.Query[int64](ctx, s.db, "SELECT COUNT(*) FROM events_bans WHERE event_id=$1", eventID)
}

// GetBannedFriends returns event likes users that are friend of the user passed.
func (s *service) GetBannedFriends(ctx context.Context, eventID, userID string, params params.Query) ([]model.User, error) {
	s.metrics.incMethodCalls("GetBannedFriends")

	q := `SELECT {fields} FROM {table} WHERE id IN (
		SELECT user_id FROM events_bans WHERE event_id=$1
		INTERSECT
		(
			SELECT friend_id FROM users_friends WHERE user_id=$2
			UNION
			SELECT user_id FROM users_friends WHERE friend_id=$2
		)
	) {pag}`
	return s.scanUsers(ctx, params, q, eventID, userID)
}

// GetBannedFriendsCount returns event's banned friends count.
func (s *service) GetBannedFriendsCount(ctx context.Context, eventID, userID string) (int64, error) {
	s.metrics.incMethodCalls("GetBannedFriendsCount")

	q := `SELECT COUNT(*) FROM events_bans WHERE event_id=$1 AND user_id IN (
		SELECT friend_id FROM users_friends WHERE user_id=$2
		UNION
		SELECT user_id FROM users_friends WHERE friend_id=$2
	)`
	return postgres.Query[int64](ctx, s.db, q, eventID, userID)
}

// GetByID returns the event with the id passed.
func (s *service) GetByID(ctx context.Context, eventID string) (model.Event, error) {
	s.metrics.incMethodCalls("GetByID")

	q := `SELECT id, name, description, virtual, url, logo_url, header_url, address, latitude, longitude, 
	type, ticket_type, public, cron, start_date, end_date, slots, min_age, created_at, updated_at 
	FROM events WHERE id=$1`
	rows, err := s.db.QueryContext(ctx, q, eventID)
	if err != nil {
		return model.Event{}, errors.Wrap(err, "querying event")
	}

	var event model.Event
	if err := sqan.Row(&event, rows); err != nil {
		return model.Event{}, errors.Wrap(err, "scanning event")
	}

	return event, nil
}

// GetHosts returns event's hosts.
func (s *service) GetHosts(ctx context.Context, eventID string, params params.Query) ([]model.User, error) {
	s.metrics.incMethodCalls("GetHosts")

	q := "SELECT {fields} FROM {table} WHERE id IN (SELECT user_id FROM events_users_roles WHERE event_id=$1 AND role_name=$2) {pag}"
	query := postgres.Select(model.T.User, q, params)
	rows, err := s.db.QueryContext(ctx, query, eventID, roles.Host)
	if err != nil {
		return nil, errors.Wrap(err, "querying users")
	}

	var users []model.User
	if err := sqan.Rows(&users, rows); err != nil {
		return nil, err
	}

	return users, nil
}

// GetInvited returns event's invited users.
func (s *service) GetInvited(ctx context.Context, eventID string, params params.Query) ([]model.User, error) {
	s.metrics.incMethodCalls("GetInvited")

	return s.roleService.GetUsersByRole(ctx, eventID, string(roles.Viewer), params)
}

// GetInvited returns event's invited users count.
func (s *service) GetInvitedCount(ctx context.Context, eventID string) (int64, error) {
	s.metrics.incMethodCalls("GetInvitedCount")

	return s.roleService.GetUsersCountByRole(ctx, eventID, string(roles.Viewer))
}

// GetInvitedFriends returns event invited users that are friends of the user passed.
func (s *service) GetInvitedFriends(ctx context.Context, eventID, userID string, params params.Query) ([]model.User, error) {
	s.metrics.incMethodCalls("GetInvitedFriends")

	return s.roleService.GetUserFriendsByRole(ctx, eventID, userID, string(roles.Viewer), params)
}

// GetInvitedFriendsCount returns event's invited friends count.
func (s *service) GetInvitedFriendsCount(ctx context.Context, eventID, userID string) (int64, error) {
	s.metrics.incMethodCalls("GetInvitedFriendsCount")

	return s.roleService.GetUserFriendsCountByRole(ctx, eventID, userID, string(roles.Viewer))
}

// GetLikes returns users liking the event.
func (s *service) GetLikes(ctx context.Context, eventID string, params params.Query) ([]model.User, error) {
	s.metrics.incMethodCalls("GetLikes")

	q := "SELECT {fields} FROM {table} WHERE id IN (SELECT user_id FROM events_likes WHERE event_id=$1) {pag}"
	return s.scanUsers(ctx, params, q, eventID)
}

// GetRecommended returns a list of events that may be interesting for the logged in user, all of them must be public.
func (s *service) GetRecommended(ctx context.Context, userID string, userCoords model.Coordinates, params params.Query) ([]model.Event, error) {
	s.metrics.incMethodCalls("GetRecommended")
	q := `SELECT DISTINCT {fields} FROM {table} WHERE public=true
	AND
		((latitude BETWEEN $1 AND $2) AND (longitude BETWEEN $3 AND $4))
	OR
		id IN (SELECT event_id FROM events_likes GROUP BY event_id ORDER BY COUNT(*) DESC LIMIT 50)
	OR
		id IN (SELECT event_id FROM events_users_roles WHERE user_id IN (
			SELECT user_id FROM users_friends WHERE friend_id=$5
			UNION
			SELECT friend_id FROM users_friends WHERE user_id=$5
		) LIMIT 50) {pag}`
	query := postgres.Select(model.T.Event, q, params)

	latMin := math.Mod(userCoords.Latitude-1, 90)
	latMax := math.Mod(userCoords.Latitude+1, 90)
	longMin := math.Mod(userCoords.Longitude-1, 180)
	longMax := math.Mod(userCoords.Longitude+1, 180)

	rows, err := s.db.QueryContext(ctx, query, latMin, latMax, longMin, longMax, userID)
	if err != nil {
		return nil, errors.Wrap(err, "querying recommended events")
	}

	var events []model.Event
	if err := sqan.Rows(&events, rows); err != nil {
		return nil, errors.Wrap(err, "scanning events")
	}

	return events, nil
}

// GetLikesCount returns the number of users liking the event.
func (s *service) GetLikesCount(ctx context.Context, eventID string) (int64, error) {
	s.metrics.incMethodCalls("GetLikesCount")

	q := "SELECT COUNT(*) FROM events_likes WHERE event_id=$1"
	return postgres.Query[int64](ctx, s.db, q, eventID)
}

// GetLikesByFriends returns the users that are friends of the user passed and liked the event.
func (s *service) GetLikesByFriends(ctx context.Context, eventID, userID string, params params.Query) ([]model.User, error) {
	s.metrics.incMethodCalls("GetLikesByFriends")

	q := `SELECT {fields} FROM {table} WHERE id IN (
		SELECT user_id FROM events_likes WHERE event_id=$1
		INTERSECT
		(
			SELECT 1 FROM users_friends WHERE user_id=$2
			UNION
			SELECT 1 FROM users_friends WHERE friend_id=$2
		)
	) {pag}`
	return s.scanUsers(ctx, params, q, eventID, userID)
}

// GetLikesByFriendsCount returns event's liked by friends count.
func (s *service) GetLikesByFriendsCount(ctx context.Context, eventID, userID string) (int64, error) {
	s.metrics.incMethodCalls("GetLikesByFriendsCount")

	q := `SELECT COUNT(*) FROM events_likes WHERE event_id=$1 AND user_id IN (
		SELECT 1 FROM users_friends WHERE user_id=$2
		UNION
		SELECT 1 FROM users_friends WHERE friend_id=$2
	)`
	return postgres.Query[int64](ctx, s.db, q, eventID, userID)
}

// GetStatistics returns events' predicates statistics.
func (s *service) GetStatistics(ctx context.Context, eventID string) (model.EventStatistics, error) {
	s.metrics.incMethodCalls("GetStatistics")

	q := `SELECT
	(SELECT COUNT(*) FROM events_bans WHERE event_id=$1) AS banned_count,
	(SELECT COUNT(*) FROM events_users_roles WHERE event_id=$1 AND role_name NOT IN ($2, $3)) AS members_count,
	(SELECT COUNT(*) FROM events_users_roles WHERE event_id=$1 AND role_name=$3) AS invited_count,
	(SELECT COUNT(*) FROM events_likes WHERE event_id=$1) AS likes_count`
	rows, err := s.db.QueryContext(ctx, q, eventID, roles.Host, roles.Viewer)
	if err != nil {
		return model.EventStatistics{}, err
	}

	var stats model.EventStatistics
	if err := sqan.Row(&stats, rows); err != nil {
		return model.EventStatistics{}, err
	}

	return stats, nil
}

// IsBanned returns if the user is banned or not from the event.
func (s *service) IsBanned(ctx context.Context, eventID, userID string) (bool, error) {
	s.metrics.incMethodCalls("IsBanned")

	q := "SELECT EXISTS (SELECT 1 FROM events_bans WHERE event_id=$1 AND user_id=$2)"
	return postgres.Query[bool](ctx, s.db, q, eventID, userID)
}

// IsInvited returns if the user is invited or not to the event.
func (s *service) IsInvited(ctx context.Context, eventID, userID string) (bool, error) {
	s.metrics.incMethodCalls("IsInvited")

	role, err := s.roleService.GetUserRole(ctx, eventID, userID)
	if err != nil {
		return false, err
	}
	return role.Name == string(roles.Viewer), nil
}

// IsPublic returns if the event is public or not.
func (s *service) IsPublic(ctx context.Context, eventID string) (bool, error) {
	s.metrics.incMethodCalls("IsPublic")

	cacheKey := cache.EventPrivacy(eventID)
	if v, err := s.rdb.Get(ctx, cacheKey).Bool(); err == nil {
		return v, nil
	}

	var public bool
	row := s.db.QueryRowContext(ctx, "SELECT public FROM events WHERE id=$1", eventID)
	if err := row.Scan(&public); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, httperr.BadRequest(fmt.Sprintf("event with id %q does not exists", eventID))
		}
		return false, err
	}

	if err := s.rdb.Set(ctx, cacheKey, public, redi.ItemExpiration).Err(); err != nil {
		return false, errors.Wrap(err, "setting event visibility to the cache")
	}

	return public, nil
}

// Like adds the like of a user to an event.
func (s *service) Like(ctx context.Context, eventID, userID string) error {
	s.metrics.incMethodCalls("Like")

	sqlTx := txgroup.SQLTx(ctx)

	q := "INSERT INTO events_likes (event_id, user_id) VALUES ($1, $2)"
	if _, err := sqlTx.ExecContext(ctx, q, eventID, userID); err != nil {
		return err
	}

	return nil
}

// PrivacyFilter lets through only users that can fetch the event data if it's private,
// if it's public it lets anyone in.
func (s *service) PrivacyFilter(ctx context.Context, eventID, userID string) error {
	isPublic, err := s.IsPublic(ctx, eventID)
	if err != nil {
		return err
	}

	if isPublic {
		return nil
	}

	// If the user has a role in the event, then he's able to retrieve its information
	hasRole, err := s.roleService.HasRole(ctx, eventID, userID)
	if err != nil {
		return errors.Wrap(err, "privacy filter: querying user role")
	}
	if !hasRole {
		return errAccessDenied
	}

	return nil
}

func (s *service) RemoveBan(ctx context.Context, eventID, userID string) error {
	s.metrics.incMethodCalls("RemoveBan")

	sqlTx := txgroup.SQLTx(ctx)

	q := "DELETE FROM events_bans WHERE event_id=$1 AND user_id=$2"
	if _, err := sqlTx.ExecContext(ctx, q, eventID, userID); err != nil {
		return err
	}

	return nil
}

func (s *service) RemoveLike(ctx context.Context, eventID, userID string) error {
	s.metrics.incMethodCalls("RemoveLike")

	sqlTx := txgroup.SQLTx(ctx)

	q := "DELETE FROM events_likes WHERE event_id=$1 AND user_id=$2"
	if _, err := sqlTx.ExecContext(ctx, q, eventID, userID); err != nil {
		return err
	}

	return nil
}

// Search returns events matching the given query.
func (s *service) Search(ctx context.Context, query, userID string, params params.Query) ([]model.Event, error) {
	s.metrics.incMethodCalls("Search")

	q := `SELECT {fields} FROM {table} 
	WHERE search @@ to_tsquery($1) AND 
	(public=true OR id IN (SELECT event_id FROM events_users_roles WHERE user_id=$2)) {pag}`
	q = postgres.Select(model.T.Event, q, params)
	rows, err := s.db.QueryContext(ctx, q, postgres.ToTSQuery(query), userID)
	if err != nil {
		return nil, errors.Wrap(err, "events searching")
	}

	var events []model.Event
	if err := sqan.Rows(&events, rows); err != nil {
		return nil, err
	}

	return events, nil
}

// SearchByLocation returns the events located within the coordinates given.
func (s *service) SearchByLocation(ctx context.Context, userID string, location model.LocationSearch) ([]model.Event, error) {
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
	rows, err := s.db.QueryContext(ctx, q, latMin, latMax, longMin, longMax, userID)
	if err != nil {
		return nil, err
	}

	var events []model.Event
	if err := sqan.Rows(&events, rows); err != nil {
		return nil, err
	}

	return events, nil
}

// Update updates an event.
func (s *service) Update(ctx context.Context, eventID string, event model.UpdateEvent) error {
	s.metrics.incMethodCalls("Update")

	sqlTx := txgroup.SQLTx(ctx)

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

		if *event.Slots < membersCount && *event.Slots != -1 {
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
	_, err := sqlTx.ExecContext(ctx, q, eventID, event.Name, event.Description, event.Type,
		event.URL, event.LogoURL, event.HeaderURL, event.Location.Address,
		event.Location.Coordinates.Latitude, event.Location.Coordinates.Longitude,
		event.Cron, event.StartDate, event.EndDate, event.Slots, time.Now())
	if err != nil {
		return errors.Wrap(err, "postgres: updating event")
	}

	if err := s.rdb.Del(ctx, model.T.Event.CacheKey(eventID)).Err(); err != nil {
		return errors.Wrap(err, "updating event")
	}

	return nil
}

func (s *service) scanUsers(ctx context.Context, params params.Query, query string, args ...any) ([]model.User, error) {
	q := postgres.Select(model.T.User, query, params)
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, errors.Wrap(err, "postgres: querying users")
	}

	var users []model.User
	if err := sqan.Rows(&users, rows); err != nil {
		return nil, err
	}

	return users, nil
}
