package user

import (
	"context"
	"database/sql"
	"time"

	"github.com/GGP1/groove/internal/cache"
	"github.com/GGP1/groove/internal/log"
	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/service/event"
	"github.com/GGP1/groove/storage/dgraph"
	"github.com/GGP1/groove/storage/postgres"

	"github.com/dgraph-io/dgo/v210"
	"github.com/dgraph-io/dgo/v210/protos/api"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// Service represents the user service.
type Service interface {
	AddFriend(ctx context.Context, userID, friendID string) error
	Block(ctx context.Context, userID, blockedID string) error
	Create(ctx context.Context, userID string, user CreateUser) error
	Delete(ctx context.Context, userID string) error
	GetBannedEvents(ctx context.Context, userID string, params params.Query) ([]event.Event, error)
	GetBlocked(ctx context.Context, userID string, params params.Query) ([]ListUser, error)
	GetBlockedCount(ctx context.Context, userID string) (*uint64, error)
	GetBlockedBy(ctx context.Context, userID string, params params.Query) ([]ListUser, error)
	GetBlockedByCount(ctx context.Context, userID string) (*uint64, error)
	GetByEmail(ctx context.Context, value string) (ListUser, error)
	GetByID(ctx context.Context, value string) (ListUser, error)
	GetByUsername(ctx context.Context, value string) (ListUser, error)
	GetConfirmedEvents(ctx context.Context, userID string, params params.Query) ([]event.Event, error)
	GetFriends(ctx context.Context, userID string, params params.Query) ([]ListUser, error)
	GetFriendsCount(ctx context.Context, userID string) (*uint64, error)
	GetInvitedEvents(ctx context.Context, userID string, params params.Query) ([]event.Event, error)
	GetHostedEvents(ctx context.Context, userID string, params params.Query) ([]event.Event, error)
	GetLikedEvents(ctx context.Context, userID string, params params.Query) ([]event.Event, error)
	GetStatistics(ctx context.Context, userID string) (Statistics, error)
	IsAdmin(ctx context.Context, tx *sql.Tx, userID string) (bool, error)
	PrivateProfile(ctx context.Context, userID string) (bool, error)
	RemoveFriend(ctx context.Context, userID string, friendID string) error
	Search(ctx context.Context, query string, params params.Query) ([]ListUser, error)
	Unblock(ctx context.Context, userID, blockedID string) error
	Update(ctx context.Context, userID string, user UpdateUser) error
}

type service struct {
	db    *sql.DB
	dc    *dgo.Dgraph
	cache cache.Client

	admins  map[string]interface{} // TODO: let admins modify this on the run? Must use mutexes
	metrics metrics
}

// NewService returns a new user service.
func NewService(db *sql.DB, dc *dgo.Dgraph, cache cache.Client, admins map[string]interface{}) Service {
	return &service{
		db:      db,
		dc:      dc,
		cache:   cache,
		admins:  admins,
		metrics: initMetrics(),
	}
}

// AddFriend adds a new friend.
func (s *service) AddFriend(ctx context.Context, userID, friendID string) error {
	s.metrics.incMethodCalls("AddFriend")

	vars := map[string]string{"$user_id": userID, "$friend_id": friendID}
	query := `query q($user_id: string, $friend_id: string) {
		user as var(func: eq(user_id, $user_id))
		friend as var(func: eq(user_id, $friend_id))
	}`
	mu := &api.Mutation{
		Cond: "@if(eq(len(user), 1) AND eq(len(friend), 1))",
		SetNquads: []byte(`uid(user) <friend> uid(friend) .
		uid(friend) <friend> uid(user) .`),
	}
	req := &api.Request{
		Query:     query,
		Vars:      vars,
		Mutations: []*api.Mutation{mu},
		CommitNow: true,
	}
	if _, err := s.dc.NewTxn().Do(ctx, req); err != nil {
		return errors.Wrap(err, "creating friendship edges")
	}

	return nil
}

// Block blocks a user.
func (s *service) Block(ctx context.Context, userID, blockedID string) error {
	s.metrics.incMethodCalls("Block")

	vars := map[string]string{"$blocker_id": userID, "$blocked_id": blockedID}
	query := `query q($blocker_id: string, $blocked_id: string) {
		blocker as var(func: eq(user_id, $blocker_id))
		blocked as var(func: eq(user_id, $blocked_id))
	}`
	mu := &api.Mutation{
		Cond:      "@if(eq(len(blocker), 1) AND eq(len(blocked), 1))",
		SetNquads: []byte("uid(blocker) <blocked> uid(blocked) ."),
	}
	req := &api.Request{
		Query:     query,
		Vars:      vars,
		Mutations: []*api.Mutation{mu},
		CommitNow: true,
	}
	if _, err := s.dc.NewTxn().Do(ctx, req); err != nil {
		return errors.Wrap(err, "performing block")
	}

	return nil
}

// Create creates a new user.
func (s *service) Create(ctx context.Context, userID string, user CreateUser) error {
	s.metrics.incMethodCalls("Create")

	psqlTx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "starting transaction")
	}
	defer psqlTx.Rollback()

	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		log.DPanic("failed generating user's password hash", zap.Error(err))
		return errors.Wrap(err, "generating password hash")
	}

	var isAdmin bool
	if _, ok := s.admins[user.Email]; ok {
		isAdmin = true
	}

	q3 := `INSERT INTO users 
	(id, name, username, email, password, birth_date, location_id, description, profile_image_url, is_admin, updated_at) 
	VALUES 
	($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err = psqlTx.ExecContext(ctx, q3, userID, user.Name, user.Username,
		user.Email, hash, user.BirthDate, user.LocationID, user.Description,
		user.ProfileImageURL, isAdmin, time.Time{})
	if err != nil {
		return errors.Wrap(err, "creating user")
	}

	err = dgraph.Mutation(ctx, s.dc, func(dgraphTx *dgo.Txn) error {
		return dgraph.CreateNode(ctx, dgraphTx, dgraph.User, userID)
	})
	if err != nil {
		return err
	}

	if err := psqlTx.Commit(); err != nil {
		return errors.Wrap(err, "postgres: committing changes")
	}

	s.metrics.registeredUsers.Inc()
	return nil
}

// Delete a user from the system.
func (s *service) Delete(ctx context.Context, userID string) error {
	s.metrics.incMethodCalls("Delete")

	psqlTx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "postgres: starting transaction")
	}

	if _, err := psqlTx.ExecContext(ctx, "DELETE FROM users WHERE id=$1", userID); err != nil {
		_ = psqlTx.Rollback()
		return errors.Wrap(err, "postgres: deleting user")
	}

	vars := map[string]string{"$id": userID}
	q := `query q($id: string) {
		user as var(func: eq(user_id, $id))
	}`
	mu := &api.Mutation{
		DelNquads: []byte("uid(user) * * ."),
	}
	req := &api.Request{
		Vars:      vars,
		Query:     q,
		Mutations: []*api.Mutation{mu},
		CommitNow: true,
	}

	if _, err := s.dc.NewTxn().Do(ctx, req); err != nil {
		_ = psqlTx.Rollback()
		return errors.Wrap(err, "dgraph: deleting user")
	}

	if err := psqlTx.Commit(); err != nil {
		return err
	}

	if err := s.cache.Delete(cache.UsersKey(userID)); err != nil {
		return errors.Wrap(err, "deleting user")
	}

	s.metrics.registeredUsers.Dec()
	return nil
}

// GetBannedEvents returns the events that the user is attending.
func (s *service) GetBannedEvents(ctx context.Context, userID string, params params.Query) ([]event.Event, error) {
	s.metrics.incMethodCalls("GetBannedEvents")

	predicate := banned
	if params.LookupID != "" {
		predicate = bannedLookup
	}

	return s.getEventsEdge(ctx, userID, predicate, params)
}

func (s *service) GetBlocked(ctx context.Context, userID string, params params.Query) ([]ListUser, error) {
	s.metrics.incMethodCalls("GetBlocked")

	predicate := blocked
	if params.LookupID != "" {
		predicate = blockedLookup
	}

	return s.getUsersEdge(ctx, userID, predicate, params)
}

func (s *service) GetBlockedCount(ctx context.Context, userID string) (*uint64, error) {
	s.metrics.incMethodCalls("GetBlockedCount")

	return dgraph.GetCount(ctx, s.dc, getQuery[blockedCount], userID)
}

func (s *service) GetBlockedBy(ctx context.Context, userID string, params params.Query) ([]ListUser, error) {
	s.metrics.incMethodCalls("GetBlockedBy")

	predicate := blockedBy
	if params.LookupID != "" {
		predicate = blockedByLookup
	}

	return s.getUsersEdge(ctx, userID, predicate, params)
}

func (s *service) GetBlockedByCount(ctx context.Context, userID string) (*uint64, error) {
	s.metrics.incMethodCalls("GetBlockedByCount")

	return dgraph.GetCount(ctx, s.dc, getQuery[blockedByCount], userID)
}

func (s *service) GetByEmail(ctx context.Context, email string) (ListUser, error) {
	s.metrics.incMethodCalls("GetByEmail")

	q := `SELECT 
	id, name, username, email, birth_date, location_id, description, premium, private, 
	verified_email, profile_image_url, invitations, created_at, updated_at
	FROM users WHERE email=$1`
	return s.getBy(ctx, q, email)
}

func (s *service) GetByID(ctx context.Context, userID string) (ListUser, error) {
	s.metrics.incMethodCalls("GetByID")

	q := `SELECT 
	id, name, username, email, birth_date, location_id, description, premium, private, 
	verified_email, profile_image_url, invitations, created_at, updated_at
	FROM users WHERE id=$1`
	return s.getBy(ctx, q, userID)
}

func (s *service) GetByUsername(ctx context.Context, username string) (ListUser, error) {
	s.metrics.incMethodCalls("GetByUsername")

	q := `SELECT 
	id, name, username, email, birth_date, location_id, description, premium, private, 
	verified_email, profile_image_url, invitations, created_at, updated_at
	FROM users WHERE username=$1`
	return s.getBy(ctx, q, username)
}

// GetConfirmedEvents returns the events that the user is attending to.
func (s *service) GetConfirmedEvents(ctx context.Context, userID string, params params.Query) ([]event.Event, error) {
	s.metrics.incMethodCalls("GetConfirmedEvents")

	predicate := confirmed
	if params.LookupID != "" {
		predicate = confirmedLookup
	}

	return s.getEventsEdge(ctx, userID, predicate, params)
}

// GetFriends returns people the user fetched is friend of.
func (s *service) GetFriends(ctx context.Context, userID string, params params.Query) ([]ListUser, error) {
	s.metrics.incMethodCalls("GetFriends")

	predicate := friends
	if params.LookupID != "" {
		predicate = friendsLookup
	}

	return s.getUsersEdge(ctx, userID, predicate, params)
}

// GetFriendsCount returns the number of users friends of the one fetched.
func (s *service) GetFriendsCount(ctx context.Context, userID string) (*uint64, error) {
	s.metrics.incMethodCalls("GetFriendsCount")

	return dgraph.GetCount(ctx, s.dc, getQuery[friendsCount], userID)
}

// GetHostedEvents returns the events hosted by the user with the given id.
func (s *service) GetHostedEvents(ctx context.Context, userID string, params params.Query) ([]event.Event, error) {
	s.metrics.incMethodCalls("GetHostedEvents")

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrap(err, "starting transaction")
	}
	defer tx.Rollback()

	query := "SELECT event_id FROM events_users_roles WHERE user_id=$1 AND role_name='host'"
	q := postgres.AddPagination(query, "event_id", params)
	rows, err := tx.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, errors.Wrap(err, "fetching events")
	}

	eventsIds, err := postgres.ScanStringSlice(rows)
	if err != nil {
		return nil, err
	}

	if len(eventsIds) == 0 {
		return nil, nil
	}

	q2 := postgres.SelectInID(postgres.Events, eventsIds, params.Fields)
	rows2, err := tx.QueryContext(ctx, q2)
	if err != nil {
		return nil, errors.Wrap(err, "fetching events")
	}

	events, err := scanEvents(rows2)
	if err != nil {
		return nil, err
	}

	return events, nil
}

// GetInvitedEvents returns the events that the user is invited to.
func (s *service) GetInvitedEvents(ctx context.Context, userID string, params params.Query) ([]event.Event, error) {
	s.metrics.incMethodCalls("GetInvitedEvents")

	predicate := invited
	if params.LookupID != "" {
		predicate = invitedLookup
	}

	return s.getEventsEdge(ctx, userID, predicate, params)
}

// GetLikedEvents returns the events that the user likes.
func (s *service) GetLikedEvents(ctx context.Context, userID string, params params.Query) ([]event.Event, error) {
	s.metrics.incMethodCalls("GetLikedEvents")

	predicate := likedBy
	if params.LookupID != "" {
		predicate = likedByLookup
	}

	return s.getEventsEdge(ctx, userID, predicate, params)
}

// GetStatistics returns a users' predicates statistics.
func (s *service) GetStatistics(ctx context.Context, userID string) (Statistics, error) {
	s.metrics.incMethodCalls("GetStatistics")

	q := `query q($id: string) {
		q(func: eq(user_id, $id)) {
			count(blocked)
			count(~blocked)
			count(~confirmed)
			count(friend)
			count(~invited)
		}
	}`
	vars := map[string]string{"$id": userID}

	res, err := s.dc.NewReadOnlyTxn().QueryRDFWithVars(ctx, q, vars)
	if err != nil {
		return Statistics{}, errors.Wrap(err, "querying count")
	}

	mp, err := dgraph.ParseCountWithMap(res.Rdf)
	if err != nil {
		return Statistics{}, err
	}

	return Statistics{
		Blocked:         mp["blocked"],
		BlockedBy:       mp["~blocked"],
		ConfirmedEvents: mp["~confirmed"],
		Friends:         mp["friend"],
		InvitedEvents:   mp["~invited"],
	}, nil
}

// IsAdmin returns if the user is an administrator or not.
func (s *service) IsAdmin(ctx context.Context, tx *sql.Tx, userID string) (bool, error) {
	isAdmin, err := postgres.QueryBool(ctx, tx, "SELECT is_admin FROM users WHERE id=$1", userID)
	if err != nil {
		return false, err
	}
	return isAdmin, nil
}

// PrivateProfile returns if the user's profile is private or not.
func (s *service) PrivateProfile(ctx context.Context, userID string) (bool, error) {
	s.metrics.incMethodCalls("PrivateProfile")

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return false, errors.Wrap(err, "starting transaction")
	}
	defer tx.Rollback()

	isPrivate, err := postgres.QueryBool(ctx, tx, "SELECT private FROM users WHERE id=$1", userID)
	if err != nil {
		return false, err
	}

	return isPrivate, nil
}

// RemoveFriend removes a friend.
func (s *service) RemoveFriend(ctx context.Context, userID string, friendID string) error {
	vars := map[string]string{"$user_id": userID, "$friend_id": friendID}
	q := `query q($user_id: string, $friend_id: string) {
		user as var(func: eq(user_id, $user_id))
		friend as var(func: eq(user_id, $friend_id))
	}`
	mu := &api.Mutation{
		DelNquads: []byte(`uid(user) <friend> uid(friend) .
		uid(friend) <friend> uid(user) .`),
	}
	req := &api.Request{
		Vars:      vars,
		Query:     q,
		Mutations: []*api.Mutation{mu},
		CommitNow: true,
	}

	if _, err := s.dc.NewTxn().Do(ctx, req); err != nil {
		return errors.Wrap(err, "removing friendship")
	}

	return nil
}

// Search returns users matching the given query.
func (s *service) Search(ctx context.Context, query string, params params.Query) ([]ListUser, error) {
	s.metrics.incMethodCalls("Search")

	q := postgres.FullTextSearch(postgres.Users, query, params)
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, errors.Wrap(err, "users searching")
	}

	users, err := scanUsers(rows)
	if err != nil {
		return nil, err
	}

	return users, nil
}

// Unblock removes the block from one user to other.
func (s *service) Unblock(ctx context.Context, userID string, blockedID string) error {
	vars := map[string]string{"$blocker_id": userID, "$blocked_id": blockedID}
	q := `query q($blocker_id: string, $blocked_id: string) {
		blocker as var(func: eq(user_id, $blocker_id))
		blocked as var(func: eq(user_id, $blocked_id))
	}`
	mu := &api.Mutation{
		DelNquads: []byte("uid(blocker) <blocked> uid(blocked) ."),
	}
	req := &api.Request{
		Vars:      vars,
		Query:     q,
		Mutations: []*api.Mutation{mu},
		CommitNow: true,
	}

	if _, err := s.dc.NewTxn().Do(ctx, req); err != nil {
		return errors.Wrap(err, "deleting block")
	}

	return nil
}

// Update updates a user.
func (s *service) Update(ctx context.Context, userID string, user UpdateUser) error {
	s.metrics.incMethodCalls("Update")

	q := `UPDATE users SET
	name = COALESCE($2,name),
	username = COALESCE($3,username),
	private = COALESCE($4,private),
	invitations = COALESCE($5,invitations),
	updated_at = $6 
	WHERE id=$1`
	_, err := s.db.ExecContext(ctx, q, userID, user.Name, user.Username,
		user.Private, user.Invitations, time.Now())
	if err != nil {
		return errors.Wrap(err, "updating user")
	}

	if err := s.cache.Delete(cache.UsersKey(userID)); err != nil {
		return errors.Wrap(err, "deleting user")
	}
	return nil
}

func (s *service) getEventsEdge(ctx context.Context, userID string, query query, params params.Query) ([]event.Event, error) {
	vars := dgraph.QueryVars(userID, params)
	res, err := s.dc.NewReadOnlyTxn().QueryRDFWithVars(ctx, getQuery[query], vars)
	if err != nil {
		return nil, errors.Wrap(err, "dgraph: fetching event ids")
	}

	eventIDs := dgraph.ParseRDFULIDs(res.Rdf)
	if len(eventIDs) == 0 {
		return nil, nil
	}

	q := postgres.SelectInID(postgres.Events, eventIDs, params.Fields)
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, errors.Wrap(err, "postgres: fetching events")
	}

	events, err := scanEvents(rows)
	if err != nil {
		return nil, err
	}

	return events, nil
}

func (s *service) getUsersEdge(ctx context.Context, userID string, query query, params params.Query) ([]ListUser, error) {
	vars := dgraph.QueryVars(userID, params)
	res, err := s.dc.NewReadOnlyTxn().QueryRDFWithVars(ctx, getQuery[query], vars)
	if err != nil {
		return nil, errors.Wrap(err, "dgraph: fetching user ids")
	}

	userIDs := dgraph.ParseRDFULIDs(res.Rdf)
	if len(userIDs) == 0 {
		return nil, nil
	}

	q := postgres.SelectInID(postgres.Users, userIDs, params.Fields)
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, errors.Wrap(err, "postgres: fetching users")
	}

	users, err := scanUsers(rows)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (s *service) getBy(ctx context.Context, query, value string) (ListUser, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return ListUser{}, errors.Wrap(err, "starting transaction")
	}
	defer tx.Commit()

	var (
		user ListUser
		// Use NullString to scan the values that can be null
		profileImageURL sql.NullString
		description     sql.NullString
		locationID      int64
	)
	err = tx.QueryRowContext(ctx, query, value).Scan(
		&user.ID, &user.Name, &user.Username, &user.Email,
		&user.BirthDate, &locationID, &description, &user.Premium,
		&user.Private, &user.VerifiedEmail, &profileImageURL,
		&user.Invitations, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return ListUser{}, errors.Wrap(err, "scanning user")
	}
	user.Description = description.String
	user.ProfileImageURL = profileImageURL.String

	if locationID != 0 {
		// TODO: fetch location from location service
		// locationID
	}

	return user, nil
}
