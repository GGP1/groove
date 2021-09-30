package user

import (
	"context"
	"database/sql"
	"encoding/binary"
	"time"

	"github.com/GGP1/groove/internal/cache"
	"github.com/GGP1/groove/internal/httperr"
	"github.com/GGP1/groove/internal/log"
	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/scan"
	"github.com/GGP1/groove/internal/sqltx"
	"github.com/GGP1/groove/model"
	"github.com/GGP1/groove/service/auth"
	"github.com/GGP1/groove/service/event"
	"github.com/GGP1/groove/service/notification"
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
	AreFriends(ctx context.Context, userID, targetID string) (bool, error)
	Block(ctx context.Context, userID, blockedID string) error
	CanInvite(ctx context.Context, authUserID, invitedID string) (bool, error)
	Create(ctx context.Context, userID string, user CreateUser) error
	Delete(ctx context.Context, userID string) error
	Follow(ctx context.Context, session auth.Session, organizationID string) error
	GetAttendingEvents(ctx context.Context, userID string, params params.Query) ([]event.Event, error)
	GetAttendingEventsCount(ctx context.Context, userID string) (int64, error)
	GetBannedEvents(ctx context.Context, userID string, params params.Query) ([]event.Event, error)
	GetBlocked(ctx context.Context, userID string, params params.Query) ([]User, error)
	GetBlockedCount(ctx context.Context, userID string) (*uint64, error)
	GetBlockedBy(ctx context.Context, userID string, params params.Query) ([]User, error)
	GetBlockedByCount(ctx context.Context, userID string) (*uint64, error)
	GetByEmail(ctx context.Context, value string) (User, error)
	GetByID(ctx context.Context, value string) (User, error)
	GetByUsername(ctx context.Context, value string) (User, error)
	GetFriends(ctx context.Context, userID string, params params.Query) ([]User, error)
	GetFriendsCount(ctx context.Context, userID string) (*uint64, error)
	GetFriendsInCommon(ctx context.Context, userID, friendID string, params params.Query) ([]User, error)
	GetFriendsInCommonCount(ctx context.Context, userID, friendID string) (*uint64, error)
	GetFriendsNotInCommon(ctx context.Context, userID, friendID string, params params.Query) ([]User, error)
	GetFriendsNotInCommonCount(ctx context.Context, userID, friendID string) (*uint64, error)
	GetInvitedEvents(ctx context.Context, userID string, params params.Query) ([]event.Event, error)
	GetHostedEvents(ctx context.Context, userID string, params params.Query) ([]event.Event, error)
	GetLikedEvents(ctx context.Context, userID string, params params.Query) ([]event.Event, error)
	GetStatistics(ctx context.Context, userID string) (Statistics, error)
	InviteToEvent(ctx context.Context, session auth.Session, invite Invite) error
	IsAdmin(ctx context.Context, userID string) (bool, error)
	PrivateProfile(ctx context.Context, userID string) (bool, error)
	RemoveFriend(ctx context.Context, userID string, friendID string) error
	Search(ctx context.Context, query string, params params.Query) ([]User, error)
	SendFriendRequest(ctx context.Context, session auth.Session, friendID string) error
	Type(ctx context.Context, userID string) (model.UserType, error)
	Unblock(ctx context.Context, userID, blockedID string) error
	Update(ctx context.Context, userID string, user UpdateUser) error
}

type service struct {
	db    *sql.DB
	dc    *dgo.Dgraph
	cache cache.Client

	admins  map[string]interface{}
	metrics metrics

	notificationService notification.Service
}

// NewService returns a new user service.
func NewService(
	db *sql.DB,
	dc *dgo.Dgraph,
	cache cache.Client,
	admins map[string]interface{},
	notificationService notification.Service,
) Service {
	return &service{
		db:                  db,
		dc:                  dc,
		cache:               cache,
		admins:              admins,
		metrics:             initMetrics(),
		notificationService: notificationService,
	}
}

// AddFriend adds a new friend.
func (s service) AddFriend(ctx context.Context, userID, friendID string) error {
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
		return errors.Wrap(err, "creating friend edges")
	}

	return nil
}

// AreFriends returns if the users are friends or not.
func (s service) AreFriends(ctx context.Context, userID, targetID string) (bool, error) {
	s.metrics.incMethodCalls("AreFriends")

	vars := map[string]string{
		"$id":        userID,
		"$lookup_id": targetID,
	}
	res, err := s.dc.NewReadOnlyTxn().QueryRDFWithVars(ctx, getQuery[areFriends], vars)
	if err != nil {
		return false, errors.Wrap(err, "checking friend edge")
	}

	count, err := dgraph.ParseCount(res.Rdf)
	if err != nil {
		return false, errors.Wrap(err, "parsing count")
	}

	return *count == 1, nil
}

// Block blocks a user.
func (s service) Block(ctx context.Context, userID, blockedID string) error {
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

func (s service) CanInvite(ctx context.Context, authUserID, invitedID string) (bool, error) {
	s.metrics.incMethodCalls("CanInvite")
	sqlTx := sqltx.FromContext(ctx)

	var invitations invitations
	q := "SELECT invitations FROM users WHERE id=$1"
	if err := sqlTx.QueryRowContext(ctx, q, invitedID).Scan(&invitations); err != nil {
		return false, errors.Wrap(err, "querying invitations")
	}

	switch invitations {
	case Friends:
		// user and invited must be friends
		areFriends, err := s.AreFriends(ctx, authUserID, invitedID)
		if err != nil {
			return false, err
		}
		return areFriends, nil
	case Nobody:
		return false, nil
	default:
		return false, errors.New("internal inconsistency")
	}
}

// Create creates a new user.
func (s service) Create(ctx context.Context, userID string, user CreateUser) error {
	s.metrics.incMethodCalls("Create")

	sqlTx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "starting transaction")
	}
	defer sqlTx.Rollback()

	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		log.DPanic("failed generating user's password hash", zap.Error(err))
		return errors.Wrap(err, "generating password hash")
	}

	var isAdmin bool
	if _, ok := s.admins[user.Email]; ok {
		isAdmin = true
	}

	// Use default invitations settings depending on the user type
	switch *user.Type {
	case model.Standard:
		user.Invitations = Friends
	case model.Organization:
		user.Invitations = Nobody
	}

	q3 := `INSERT INTO users 
	(id, name, username, email, password, birth_date, description, 
	profile_image_url, type, is_admin, invitations, updated_at) 
	VALUES 
	($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	_, err = sqlTx.ExecContext(ctx, q3, userID, user.Name, user.Username,
		user.Email, hash, user.BirthDate, user.Description, user.ProfileImageURL,
		user.Type, isAdmin, user.Invitations, time.Time{})
	if err != nil {
		return errors.Wrap(err, "creating user")
	}

	err = dgraph.Mutation(ctx, s.dc, func(dgraphTx *dgo.Txn) error {
		return dgraph.CreateNode(ctx, dgraphTx, model.User, userID)
	})
	if err != nil {
		return err
	}

	if err := sqlTx.Commit(); err != nil {
		return err
	}

	s.metrics.registeredUsers.Inc()
	return nil
}

// Delete a user from the system.
func (s service) Delete(ctx context.Context, userID string) error {
	s.metrics.incMethodCalls("Delete")

	sqlTx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "postgres: starting transaction")
	}

	if _, err := sqlTx.ExecContext(ctx, "DELETE FROM users WHERE id=$1", userID); err != nil {
		_ = sqlTx.Rollback()
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
		_ = sqlTx.Rollback()
		return errors.Wrap(err, "dgraph: deleting user")
	}

	if err := sqlTx.Commit(); err != nil {
		return err
	}

	if err := s.cache.Delete(model.User.CacheKey(userID)); err != nil {
		return errors.Wrap(err, "deleting user")
	}

	s.metrics.registeredUsers.Dec()
	return nil
}

// Follow follows an organization.
func (s service) Follow(ctx context.Context, session auth.Session, organizationID string) error {
	typ, err := s.Type(ctx, organizationID)
	if err != nil {
		return err
	}
	if typ != model.Organization {
		return httperr.New("only organizations can be followed", httperr.Forbidden)
	}

	return dgraph.Mutation(ctx, s.dc, func(tx *dgo.Txn) error {
		req := dgraph.UserEdgeRequest(session.ID, dgraph.Follows, organizationID, true)
		if _, err := tx.Do(ctx, req); err != nil {
			return errors.Wrap(err, "creating follows edge")
		}
		return nil
	})
}

// GetAttendingEvents returns the events the user is assiting to.
func (s service) GetAttendingEvents(ctx context.Context, userID string, params params.Query) ([]event.Event, error) {
	s.metrics.incMethodCalls("GetAttendingEvents")

	whereCond := "id IN (SELECT event_id FROM events_users_roles WHERE user_id=$1)"
	q := postgres.SelectWhere(model.Event, whereCond, "id", params)
	rows, err := s.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, err
	}

	var events []event.Event
	if err := scan.Rows(&events, rows); err != nil {
		return nil, err
	}

	return events, nil
}

// GetAttendingEventsCount returns the events the user is assiting to.
func (s service) GetAttendingEventsCount(ctx context.Context, userID string) (int64, error) {
	s.metrics.incMethodCalls("GetAttendingEventsCount")

	sqlTx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return 0, err
	}
	defer sqlTx.Rollback()

	q := "SELECT COUNT(*) FROM events_users_roles WHERE user_id=$1"
	count, err := postgres.QueryInt(ctx, sqlTx, q, userID)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// GetBannedEvents returns the events that the user is attending.
func (s service) GetBannedEvents(ctx context.Context, userID string, params params.Query) ([]event.Event, error) {
	s.metrics.incMethodCalls("GetBannedEvents")

	predicate := banned
	if params.LookupID != "" {
		predicate = bannedLookup
	}

	return s.getEventsEdge(ctx, userID, predicate, params)
}

func (s service) GetBlocked(ctx context.Context, userID string, params params.Query) ([]User, error) {
	s.metrics.incMethodCalls("GetBlocked")

	predicate := blocked
	if params.LookupID != "" {
		predicate = blockedLookup
	}

	return s.getUsersEdge(ctx, userID, predicate, params)
}

func (s service) GetBlockedCount(ctx context.Context, userID string) (*uint64, error) {
	s.metrics.incMethodCalls("GetBlockedCount")

	return dgraph.GetCount(ctx, s.dc, getQuery[blockedCount], userID)
}

func (s service) GetBlockedBy(ctx context.Context, userID string, params params.Query) ([]User, error) {
	s.metrics.incMethodCalls("GetBlockedBy")

	predicate := blockedBy
	if params.LookupID != "" {
		predicate = blockedByLookup
	}

	return s.getUsersEdge(ctx, userID, predicate, params)
}

func (s service) GetBlockedByCount(ctx context.Context, userID string) (*uint64, error) {
	s.metrics.incMethodCalls("GetBlockedByCount")

	return dgraph.GetCount(ctx, s.dc, getQuery[blockedByCount], userID)
}

func (s service) GetByEmail(ctx context.Context, email string) (User, error) {
	s.metrics.incMethodCalls("GetByEmail")

	q := `SELECT 
	id, name, username, email, birth_date, description, private, 
	verified_email, profile_image_url, invitations, created_at, updated_at
	FROM users WHERE email=$1`
	return s.getBy(ctx, q, email)
}

func (s service) GetByID(ctx context.Context, userID string) (User, error) {
	s.metrics.incMethodCalls("GetByID")

	q := `SELECT 
	id, name, username, email, birth_date, description, private, 
	verified_email, profile_image_url, invitations, created_at, updated_at
	FROM users WHERE id=$1`
	return s.getBy(ctx, q, userID)
}

func (s service) GetByUsername(ctx context.Context, username string) (User, error) {
	s.metrics.incMethodCalls("GetByUsername")

	q := `SELECT 
	id, name, username, email, birth_date, description, private, 
	verified_email, profile_image_url, invitations, created_at, updated_at
	FROM users WHERE username=$1`
	return s.getBy(ctx, q, username)
}

// GetFriends returns people the user fetched is friend of.
func (s service) GetFriends(ctx context.Context, userID string, params params.Query) ([]User, error) {
	s.metrics.incMethodCalls("GetFriends")

	predicate := friends
	if params.LookupID != "" {
		predicate = friendsLookup
	}

	return s.getUsersEdge(ctx, userID, predicate, params)
}

// GetFriendsCount returns the number of users friends of the one fetched.
func (s service) GetFriendsCount(ctx context.Context, userID string) (*uint64, error) {
	s.metrics.incMethodCalls("GetFriendsCount")

	return dgraph.GetCount(ctx, s.dc, getQuery[friendsCount], userID)
}

// GetFriendsInCommon returns the friends in common between userID and friendID.
func (s service) GetFriendsInCommon(ctx context.Context, userID, friendID string, params params.Query) ([]User, error) {
	s.metrics.incMethodCalls("GetFriendsInCommon")

	predicate := friendsInCommon
	if params.LookupID != "" {
		predicate = friendsInCommonLookup
	}

	return s.getUsersMixedEdge(ctx, userID, friendID, predicate, params)
}

// GetFriendsInCommonCount returns the number of matching friends between userID and friendID.
func (s service) GetFriendsInCommonCount(ctx context.Context, userID, friendID string) (*uint64, error) {
	s.metrics.incMethodCalls("GetFriendsInCommonCount")

	vars := map[string]string{"$id": userID, "$friend_id": friendID}
	return dgraph.GetCountWithVars(ctx, s.dc, getMixedQuery[friendsInCommonCount], vars)
}

// GetFriendsNotInCommon returns the friends that are not in common between userID and friendID.
func (s service) GetFriendsNotInCommon(ctx context.Context, userID, friendID string, params params.Query) ([]User, error) {
	s.metrics.incMethodCalls("GetFriendsNotInCommon")

	predicate := friendsNotInCommon
	if params.LookupID != "" {
		predicate = friendsNotInCommonLookup
	}

	return s.getUsersMixedEdge(ctx, userID, friendID, predicate, params)
}

// GetFriendsNotInCommonCount returns the number of non-matching friends between userID and friendID.
func (s service) GetFriendsNotInCommonCount(ctx context.Context, userID, friendID string) (*uint64, error) {
	s.metrics.incMethodCalls("GetFriendsNotInCommonCount")

	vars := map[string]string{"$id": userID, "$friend_id": friendID}
	return dgraph.GetCountWithVars(ctx, s.dc, getMixedQuery[friendsNotInCommonCount], vars)
}

// GetHostedEvents returns the events hosted by the user with the given id.
func (s service) GetHostedEvents(ctx context.Context, userID string, params params.Query) ([]event.Event, error) {
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

	q2 := postgres.SelectInID(model.User, eventsIds, params.Fields)
	rows2, err := tx.QueryContext(ctx, q2)
	if err != nil {
		return nil, errors.Wrap(err, "fetching events")
	}

	var events []event.Event
	if err := scan.Rows(&events, rows2); err != nil {
		return nil, err
	}

	return events, nil
}

// GetInvitedEvents returns the events that the user is invited to.
func (s service) GetInvitedEvents(ctx context.Context, userID string, params params.Query) ([]event.Event, error) {
	s.metrics.incMethodCalls("GetInvitedEvents")

	predicate := invited
	if params.LookupID != "" {
		predicate = invitedLookup
	}

	return s.getEventsEdge(ctx, userID, predicate, params)
}

// GetLikedEvents returns the events that the user likes.
func (s service) GetLikedEvents(ctx context.Context, userID string, params params.Query) ([]event.Event, error) {
	s.metrics.incMethodCalls("GetLikedEvents")

	predicate := likedBy
	if params.LookupID != "" {
		predicate = likedByLookup
	}

	return s.getEventsEdge(ctx, userID, predicate, params)
}

// GetStatistics returns a users' predicates statistics.
func (s service) GetStatistics(ctx context.Context, userID string) (Statistics, error) {
	s.metrics.incMethodCalls("GetStatistics")

	q := `query q($id: string) {
		q(func: eq(user_id, $id)) {
			count(blocked)
			count(~blocked)
			count(friend)
			count(~invited)
			count(~follows)
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

	attendingCount, err := s.GetAttendingEventsCount(ctx, userID)
	if err != nil {
		return Statistics{}, err
	}

	return Statistics{
		Blocked:         mp["blocked"],
		BlockedBy:       mp["~blocked"],
		Friends:         mp["friend"],
		Followers:       mp["~follows"],
		AttendingEvents: attendingCount,
		InvitedEvents:   mp["~invited"],
	}, nil
}

// InviteToEvent invites a user to an event.
func (s service) InviteToEvent(ctx context.Context, session auth.Session, invite Invite) error {
	canInvite, err := s.CanInvite(ctx, session.ID, invite.UserID)
	if err != nil {
		return err
	}
	if !canInvite {
		return httperr.New("you aren't allowed to invite this user", httperr.Forbidden)
	}

	err = s.notificationService.Create(ctx, session, notification.CreateNotification{
		SenderID:   session.ID,
		ReceiverID: invite.UserID,
		EventID:    &invite.EventID,
		Content:    notification.InvitationContent(session),
		Type:       notification.Invitation,
	})
	if err != nil {
		return errors.Wrap(err, "creating invitation notification")
	}

	return dgraph.AddEventEdge(ctx, s.dc, invite.EventID, dgraph.Invited, invite.UserID)
}

// IsAdmin returns if the user is an administrator or not.
func (s service) IsAdmin(ctx context.Context, userID string) (bool, error) {
	var isAdmin bool
	if err := s.db.QueryRowContext(ctx, "SELECT is_admin FROM users WHERE id=$1", userID).Scan(&isAdmin); err != nil {
		return false, errors.Wrap(err, "fetching is admin")
	}

	return isAdmin, nil
}

// PrivateProfile returns if the user's profile is private or not.
func (s service) PrivateProfile(ctx context.Context, userID string) (bool, error) {
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
func (s service) RemoveFriend(ctx context.Context, userID string, friendID string) error {
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
func (s service) Search(ctx context.Context, query string, params params.Query) ([]User, error) {
	s.metrics.incMethodCalls("Search")

	q := postgres.FullTextSearch(model.User, params)
	rows, err := s.db.QueryContext(ctx, q, postgres.ToTSQuery(query))
	if err != nil {
		return nil, errors.Wrap(err, "users searching")
	}

	var users []User
	if err := scan.Rows(&users, rows); err != nil {
		return nil, err
	}

	return users, nil
}

// SendFriendRequest ..
func (s service) SendFriendRequest(ctx context.Context, session auth.Session, friendID string) error {
	s.metrics.incMethodCalls("SendFriendRequest")

	typ, err := s.Type(ctx, friendID)
	if err != nil {
		return err
	}
	if typ == model.Organization {
		return httperr.New("cannot invite an organization", httperr.Forbidden)
	}

	err = s.notificationService.Create(ctx, session, notification.CreateNotification{
		SenderID:   session.ID,
		ReceiverID: friendID,
		Content:    notification.FriendRequestContent(session),
		Type:       notification.FriendRequest,
	})
	if err != nil {
		return errors.Wrap(err, "creating friend request notification")
	}
	return nil
}

// Type returns the user's type.
func (s service) Type(ctx context.Context, userID string) (model.UserType, error) {
	s.metrics.incMethodCalls("Type")
	sqlTx := sqltx.FromContext(ctx)

	cacheKey := userID + "_type"
	if item, err := s.cache.Get(cacheKey); err == nil {
		x, _ := binary.Varint(item.Value)
		return model.UserType(x), nil
	}

	accType, err := postgres.QueryInt(ctx, sqlTx, "SELECT type FROM users WHERE id=$1", userID)
	if err != nil {
		return 0, errors.Wrap(err, "querying user type")
	}

	n := make([]byte, 1)
	binary.PutVarint(n, accType)
	if err := s.cache.Set(cacheKey, n); err != nil {
		return 0, errors.Wrap(err, "saving user type to the cache")
	}

	return model.UserType(accType), nil
}

// Unblock removes the block from one user to other.
func (s service) Unblock(ctx context.Context, userID string, blockedID string) error {
	s.metrics.incMethodCalls("Unblock")

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
func (s service) Update(ctx context.Context, userID string, user UpdateUser) error {
	s.metrics.incMethodCalls("Update")

	typ, err := s.Type(ctx, userID)
	if err != nil {
		return err
	}
	if typ == model.Organization && user.Private != nil {
		return httperr.New("cannot update an organization's visibility", httperr.Forbidden)
	}

	sqlTx := sqltx.FromContext(ctx)

	q := `UPDATE users SET
	name = COALESCE($2,name),
	username = COALESCE($3,username),
	private = COALESCE($4,private),
	invitations = COALESCE($5,invitations),
	updated_at = $6 
	WHERE id=$1`
	_, err = sqlTx.ExecContext(ctx, q, userID, user.Name, user.Username,
		user.Private, user.Invitations, time.Now())
	if err != nil {
		return errors.Wrap(err, "updating user")
	}

	if err := s.cache.Delete(model.User.CacheKey(userID)); err != nil {
		return errors.Wrap(err, "deleting user")
	}
	return nil
}

func (s service) getEventsEdge(ctx context.Context, userID string, query query, params params.Query) ([]event.Event, error) {
	vars := dgraph.QueryVars(userID, params)
	res, err := s.dc.NewReadOnlyTxn().QueryRDFWithVars(ctx, getQuery[query], vars)
	if err != nil {
		return nil, errors.Wrap(err, "dgraph: fetching event ids")
	}

	eventIDs := dgraph.ParseRDFULIDs(res.Rdf)
	if len(eventIDs) == 0 {
		return nil, nil
	}

	q := postgres.SelectInID(model.Event, eventIDs, params.Fields)
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, errors.Wrap(err, "postgres: fetching events")
	}

	var events []event.Event
	if err := scan.Rows(&events, rows); err != nil {
		return nil, err
	}

	return events, nil
}

func (s service) getUsersEdge(ctx context.Context, userID string, query query, params params.Query) ([]User, error) {
	vars := dgraph.QueryVars(userID, params)
	res, err := s.dc.NewReadOnlyTxn().QueryRDFWithVars(ctx, getQuery[query], vars)
	if err != nil {
		return nil, errors.Wrap(err, "dgraph: fetching user ids")
	}

	return s.parseIDsAndScan(ctx, res.Rdf, params)
}

func (s service) getUsersMixedEdge(ctx context.Context, userID, friendID string, query mixedQuery, params params.Query) ([]User, error) {
	vars := dgraph.QueryVars(userID, params)
	vars["$friend_id"] = friendID
	res, err := s.dc.NewReadOnlyTxn().QueryRDFWithVars(ctx, getMixedQuery[query], vars)
	if err != nil {
		return nil, errors.Wrap(err, "dgraph: fetching user ids")
	}

	return s.parseIDsAndScan(ctx, res.Rdf, params)
}

func (s service) parseIDsAndScan(ctx context.Context, rdf []byte, params params.Query) ([]User, error) {
	userIDs := dgraph.ParseRDFULIDs(rdf)
	if len(userIDs) == 0 {
		return nil, nil
	}

	q := postgres.SelectInID(model.User, userIDs, params.Fields)
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, errors.Wrap(err, "postgres: fetching users")
	}

	var users []User
	if err := scan.Rows(&users, rows); err != nil {
		return nil, err
	}

	return users, nil
}

func (s service) getBy(ctx context.Context, query, value string) (User, error) {
	sqlTx := sqltx.FromContext(ctx)

	rows, err := sqlTx.QueryContext(ctx, query, value)
	if err != nil {
		return User{}, errors.Wrap(err, "fetching user")
	}

	var user User
	if err := scan.Row(&user, rows); err != nil {
		return User{}, errors.Wrap(err, "scanning user")
	}

	return user, nil
}
