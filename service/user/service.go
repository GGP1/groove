package user

import (
	"context"
	"database/sql"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/GGP1/groove/internal/cache"
	"github.com/GGP1/groove/internal/httperr"
	"github.com/GGP1/groove/internal/log"
	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/roles"
	"github.com/GGP1/groove/internal/sqltx"
	"github.com/GGP1/groove/model"
	"github.com/GGP1/groove/service/auth"
	"github.com/GGP1/groove/service/event"
	"github.com/GGP1/groove/service/event/role"
	"github.com/GGP1/groove/service/notification"
	"github.com/GGP1/groove/storage/dgraph"
	"github.com/GGP1/groove/storage/postgres"
	"github.com/GGP1/sqan"

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
	Follow(ctx context.Context, session auth.Session, businessID string) error
	GetAttendingEvents(ctx context.Context, userID string, params params.Query) ([]event.Event, error)
	GetAttendingEventsCount(ctx context.Context, userID string) (int64, error)
	GetBannedEvents(ctx context.Context, userID string, params params.Query) ([]event.Event, error)
	GetBannedEventsCount(ctx context.Context, userID string) (*uint64, error)
	GetBlocked(ctx context.Context, userID string, params params.Query) ([]User, error)
	GetBlockedCount(ctx context.Context, userID string) (*uint64, error)
	GetBlockedBy(ctx context.Context, userID string, params params.Query) ([]User, error)
	GetBlockedByCount(ctx context.Context, userID string) (*uint64, error)
	GetByEmail(ctx context.Context, value string) (User, error)
	GetByID(ctx context.Context, value string) (User, error)
	GetByUsername(ctx context.Context, value string) (User, error)
	GetFollowers(ctx context.Context, userID string, params params.Query) ([]User, error)
	GetFollowersCount(ctx context.Context, userID string) (*uint64, error)
	GetFollowing(ctx context.Context, userID string, params params.Query) ([]User, error)
	GetFollowingCount(ctx context.Context, userID string) (*uint64, error)
	GetFriends(ctx context.Context, userID string, params params.Query) ([]User, error)
	GetFriendsCount(ctx context.Context, userID string) (*uint64, error)
	GetFriendsInCommon(ctx context.Context, userID, friendID string, params params.Query) ([]User, error)
	GetFriendsInCommonCount(ctx context.Context, userID, friendID string) (*uint64, error)
	GetFriendsNotInCommon(ctx context.Context, userID, friendID string, params params.Query) ([]User, error)
	GetFriendsNotInCommonCount(ctx context.Context, userID, friendID string) (*uint64, error)
	GetHostedEvents(ctx context.Context, userID string, params params.Query) ([]event.Event, error)
	GetHostedEventsCount(ctx context.Context, userID string) (int64, error)
	GetInvitedEvents(ctx context.Context, userID string, params params.Query) ([]event.Event, error)
	GetInvitedEventsCount(ctx context.Context, userID string) (int64, error)
	GetLikedEvents(ctx context.Context, userID string, params params.Query) ([]event.Event, error)
	GetLikedEventsCount(ctx context.Context, userID string) (*uint64, error)
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
	roleService         role.Service
}

// NewService returns a new user service.
func NewService(
	db *sql.DB,
	dc *dgo.Dgraph,
	cache cache.Client,
	admins map[string]interface{},
	notificationService notification.Service,
	roleService role.Service,
) Service {
	return &service{
		db:                  db,
		dc:                  dc,
		cache:               cache,
		admins:              admins,
		metrics:             initMetrics(),
		notificationService: notificationService,
		roleService:         roleService,
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

	var invitations model.Invitations
	q := "SELECT invitations FROM users WHERE id=$1"
	if err := sqlTx.QueryRowContext(ctx, q, invitedID).Scan(&invitations); err != nil {
		return false, errors.Wrap(err, "querying invitations")
	}

	switch invitations {
	case model.Friends:
		// user and invited must be friends
		areFriends, err := s.AreFriends(ctx, authUserID, invitedID)
		if err != nil {
			return false, err
		}
		return areFriends, nil
	case model.Nobody:
		return false, nil
	default:
		return false, errors.New("internal inconsistency")
	}
}

// Create creates a new user.
//
// TODO: store API key to return on login (not necessary to return on creation).
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
	case model.Personal:
		user.Invitations = model.Friends
	case model.Business:
		user.Invitations = model.Nobody
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

// Follow follows an business.
func (s service) Follow(ctx context.Context, session auth.Session, businessID string) error {
	s.metrics.incMethodCalls("Follow")

	typ, err := s.Type(ctx, businessID)
	if err != nil {
		return err
	}
	if typ != model.Business {
		return httperr.Forbidden("only businesses can be followed")
	}

	return dgraph.Mutation(ctx, s.dc, func(tx *dgo.Txn) error {
		req := dgraph.UserEdgeRequest(session.ID, dgraph.Follows, businessID, true)
		if _, err := tx.Do(ctx, req); err != nil {
			return errors.Wrap(err, "creating follows edge")
		}
		return nil
	})
}

// GetAttendingEvents returns the events the user is assiting to.
func (s service) GetAttendingEvents(ctx context.Context, userID string, params params.Query) ([]event.Event, error) {
	s.metrics.incMethodCalls("GetAttendingEvents")

	whereCond := "id IN (SELECT event_id FROM events_users_roles WHERE user_id=$1 AND role_name NOT IN ($2, $3))"
	q := postgres.SelectWhere(model.Event, whereCond, "id", params)
	rows, err := s.db.QueryContext(ctx, q, userID, roles.Viewer, roles.Host)
	if err != nil {
		return nil, err
	}

	var events []event.Event
	if err := sqan.Rows(&events, rows); err != nil {
		return nil, err
	}

	return events, nil
}

// GetAttendingEventsCount returns the events the user is assiting to.
func (s service) GetAttendingEventsCount(ctx context.Context, userID string) (int64, error) {
	s.metrics.incMethodCalls("GetAttendingEventsCount")

	q := "SELECT COUNT(*) FROM events_users_roles WHERE user_id=$1 AND role_name NOT IN ($2, $3)"
	count, err := postgres.QueryInt(ctx, s.db, q, userID, roles.Viewer, roles.Host)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// GetBannedEvents returns the events that the user is banned from.
func (s service) GetBannedEvents(ctx context.Context, userID string, params params.Query) ([]event.Event, error) {
	s.metrics.incMethodCalls("GetBannedEvents")

	query := banned
	if params.LookupID != "" {
		query = bannedLookup
	}

	return s.getEventsEdge(ctx, userID, query, params)
}

// GetBannedEvents returns the number of events that the user is banned from.
func (s service) GetBannedEventsCount(ctx context.Context, userID string) (*uint64, error) {
	s.metrics.incMethodCalls("GetBannedEventsCount")

	vars := map[string]string{"$id": userID}
	res, err := s.dc.NewReadOnlyTxn().QueryRDFWithVars(ctx, getQuery[bannedCount], vars)
	if err != nil {
		return nil, errors.Wrap(err, "dgraph: fetching banned events count")
	}

	return dgraph.ParseCount(res.Rdf)
}

func (s service) GetBlocked(ctx context.Context, userID string, params params.Query) ([]User, error) {
	s.metrics.incMethodCalls("GetBlocked")

	query := blocked
	if params.LookupID != "" {
		query = blockedLookup
	}

	return s.getUsersEdge(ctx, userID, query, params)
}

func (s service) GetBlockedCount(ctx context.Context, userID string) (*uint64, error) {
	s.metrics.incMethodCalls("GetBlockedCount")

	return dgraph.GetCount(ctx, s.dc, getQuery[blockedCount], userID)
}

func (s service) GetBlockedBy(ctx context.Context, userID string, params params.Query) ([]User, error) {
	s.metrics.incMethodCalls("GetBlockedBy")

	query := blockedBy
	if params.LookupID != "" {
		query = blockedByLookup
	}

	return s.getUsersEdge(ctx, userID, query, params)
}

func (s service) GetBlockedByCount(ctx context.Context, userID string) (*uint64, error) {
	s.metrics.incMethodCalls("GetBlockedByCount")

	return dgraph.GetCount(ctx, s.dc, getQuery[blockedByCount], userID)
}

func (s service) GetByEmail(ctx context.Context, email string) (User, error) {
	s.metrics.incMethodCalls("GetByEmail")

	q := `SELECT 
	id, name, username, email, birth_date, description, private, type,
	verified_email, profile_image_url, invitations, created_at, updated_at
	FROM users WHERE email=$1`
	return s.getBy(ctx, q, email)
}

func (s service) GetByID(ctx context.Context, userID string) (User, error) {
	s.metrics.incMethodCalls("GetByID")

	q := `SELECT 
	id, name, username, email, birth_date, description, private, type,
	verified_email, profile_image_url, invitations, created_at, updated_at
	FROM users WHERE id=$1`
	return s.getBy(ctx, q, userID)
}

func (s service) GetByUsername(ctx context.Context, username string) (User, error) {
	s.metrics.incMethodCalls("GetByUsername")

	q := `SELECT 
	id, name, username, email, birth_date, description, private, type,
	verified_email, profile_image_url, invitations, created_at, updated_at
	FROM users WHERE username=$1`
	return s.getBy(ctx, q, username)
}

// GetFollowers returns a user's followers. Only businesses can have followers, calling this on
// a standard user will return always nil.
func (s service) GetFollowers(ctx context.Context, userID string, params params.Query) ([]User, error) {
	s.metrics.incMethodCalls("GetFollowers")

	query := followers
	if params.LookupID != "" {
		query = followersLookup
	}

	return s.getUsersEdge(ctx, userID, query, params)
}

// GetFollowersCount returns a user's number of followers. Only businesses can have followers, calling this on
// a standard user will return always 0.
func (s service) GetFollowersCount(ctx context.Context, userID string) (*uint64, error) {
	s.metrics.incMethodCalls("GetFollowersCount")

	return dgraph.GetCount(ctx, s.dc, getQuery[followersCount], userID)
}

// GetFollowing returns the businesses the user is following.
func (s service) GetFollowing(ctx context.Context, userID string, params params.Query) ([]User, error) {
	s.metrics.incMethodCalls("GetFollowing")

	query := following
	if params.LookupID != "" {
		query = followingLookup
	}

	return s.getUsersEdge(ctx, userID, query, params)
}

// GetFollowingCount returns the number of businesses the user is following.
func (s service) GetFollowingCount(ctx context.Context, userID string) (*uint64, error) {
	s.metrics.incMethodCalls("GetFollowingCount")

	return dgraph.GetCount(ctx, s.dc, getQuery[followingCount], userID)
}

// GetFriends returns people the user fetched is friend of.
func (s service) GetFriends(ctx context.Context, userID string, params params.Query) ([]User, error) {
	s.metrics.incMethodCalls("GetFriends")

	query := friends
	if params.LookupID != "" {
		query = friendsLookup
	}

	return s.getUsersEdge(ctx, userID, query, params)
}

// GetFriendsCount returns the number of users friends of the one fetched.
func (s service) GetFriendsCount(ctx context.Context, userID string) (*uint64, error) {
	s.metrics.incMethodCalls("GetFriendsCount")

	return dgraph.GetCount(ctx, s.dc, getQuery[friendsCount], userID)
}

// GetFriendsInCommon returns the friends in common between userID and friendID.
func (s service) GetFriendsInCommon(ctx context.Context, userID, friendID string, params params.Query) ([]User, error) {
	s.metrics.incMethodCalls("GetFriendsInCommon")

	query := friendsInCommon
	if params.LookupID != "" {
		query = friendsInCommonLookup
	}

	return s.getUsersMixedEdge(ctx, userID, friendID, query, params)
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

	query := friendsNotInCommon
	if params.LookupID != "" {
		query = friendsNotInCommonLookup
	}

	return s.getUsersMixedEdge(ctx, userID, friendID, query, params)
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

	return s.getUserEvents(ctx, userID, string(roles.Host), params)
}

// GetHostedEventsCount returns the number of events hosted by the user with the given id.
func (s service) GetHostedEventsCount(ctx context.Context, userID string) (int64, error) {
	s.metrics.incMethodCalls("GetHostedEventsCount")

	return s.roleService.GetUserEventsCount(ctx, userID, string(roles.Host))
}

// GetInvitedEvents returns the events that the user is invited to.
func (s service) GetInvitedEvents(ctx context.Context, userID string, params params.Query) ([]event.Event, error) {
	s.metrics.incMethodCalls("GetInvitedEvents")

	return s.getUserEvents(ctx, userID, string(roles.Viewer), params)
}

// GetInvitedEventsCount returns the number of events that the user is invited to.
func (s service) GetInvitedEventsCount(ctx context.Context, userID string) (int64, error) {
	s.metrics.incMethodCalls("GetInvitedEventsCount")

	return s.roleService.GetUserEventsCount(ctx, userID, string(roles.Viewer))
}

// GetLikedEvents returns the events that the user likes.
func (s service) GetLikedEvents(ctx context.Context, userID string, params params.Query) ([]event.Event, error) {
	s.metrics.incMethodCalls("GetLikedEvents")

	query := likedBy
	if params.LookupID != "" {
		query = likedByLookup
	}
	return s.getEventsEdge(ctx, userID, query, params)
}

// GetLikedEventsCount returns the number of events that the user likes.
func (s service) GetLikedEventsCount(ctx context.Context, userID string) (*uint64, error) {
	s.metrics.incMethodCalls("GetLikedEventsCount")

	return dgraph.GetCount(ctx, s.dc, getQuery[likedByCount], userID)
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
			count(follows)
			count(~liked_by)
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

	hostedCount, err := s.GetHostedEventsCount(ctx, userID)
	if err != nil {
		return Statistics{}, err
	}

	return Statistics{
		Blocked:         mp["blocked"],
		BlockedBy:       mp["~blocked"],
		Friends:         mp["friend"],
		Following:       mp["follows"],
		Followers:       mp["~follows"],
		AttendingEvents: &attendingCount,
		HostedEvents:    &hostedCount,
		Invitations:     mp["~invited"],
		LikedEvents:     mp["~liked_by"],
	}, nil
}

// InviteToEvent invites a user to an event.
func (s service) InviteToEvent(ctx context.Context, session auth.Session, invite Invite) error {
	s.metrics.incMethodCalls("InviteToEvent")

	for _, userID := range invite.UserIDs {
		canInvite, err := s.CanInvite(ctx, session.ID, userID)
		if err != nil {
			return err
		}
		if !canInvite {
			return httperr.Forbidden(fmt.Sprintf("you aren't allowed to invite the user %q", userID))
		}
	}

	req := &api.Request{
		Vars: map[string]string{"$user_id": session.ID},
		Query: `query q($user_id: string, $target_id: string) {
			user as var(func: eq(user_id, $user_id))
			target as var(func: eq(user_id, $target_id))
		}`,
		Mutations: []*api.Mutation{{
			Cond:      "@if(eq(len(user), 1) AND eq(len(target), 1))",
			SetNquads: dgraph.TripleUID("uid(user)", string(dgraph.Invited), "uid(target)"),
		}},
	}

	err := dgraph.Mutation(ctx, s.dc, func(tx *dgo.Txn) error {
		for _, userID := range invite.UserIDs {
			// Replace the target id on each iteration
			req.Vars["$target_id"] = userID
			if _, err := tx.Do(ctx, req); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "creating invited edges")
	}

	err = s.notificationService.CreateMany(ctx, session, notification.CreateNotificationMany{
		SenderID:    session.ID,
		ReceiverIDs: invite.UserIDs,
		EventID:     &invite.EventID,
		Content:     notification.InvitationContent(session),
		Type:        notification.Invitation,
	})
	if err != nil {
		return errors.Wrap(err, "creating invitation notifications")
	}

	return nil
}

// IsAdmin returns if the user is an administrator or not.
func (s service) IsAdmin(ctx context.Context, userID string) (bool, error) {
	s.metrics.incMethodCalls("IsAdmin")

	var isAdmin bool
	if err := s.db.QueryRowContext(ctx, "SELECT is_admin FROM users WHERE id=$1", userID).Scan(&isAdmin); err != nil {
		return false, errors.Wrap(err, "fetching is admin")
	}

	return isAdmin, nil
}

// PrivateProfile returns if the user's profile is private or not.
func (s service) PrivateProfile(ctx context.Context, userID string) (bool, error) {
	s.metrics.incMethodCalls("PrivateProfile")

	isPrivate, err := postgres.QueryBool(ctx, s.db, "SELECT private FROM users WHERE id=$1", userID)
	if err != nil {
		return false, err
	}

	return isPrivate, nil
}

// RemoveFriend removes a friend.
func (s service) RemoveFriend(ctx context.Context, userID string, friendID string) error {
	s.metrics.incMethodCalls("RemoveFriend")

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
	if err := sqan.Rows(&users, rows); err != nil {
		return nil, err
	}

	return users, nil
}

// SendFriendRequest sends a friend request to the user indicated.
func (s service) SendFriendRequest(ctx context.Context, session auth.Session, friendID string) error {
	s.metrics.incMethodCalls("SendFriendRequest")

	typ, err := s.Type(ctx, friendID)
	if err != nil {
		return err
	}
	if typ == model.Business {
		return httperr.Forbidden("cannot invite a business")
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

	cacheKey := userID + "_type"
	if v, err := s.cache.Get(cacheKey); err == nil {
		x, _ := binary.Varint(v)
		return model.UserType(x), nil
	}

	accType, err := postgres.QueryInt(ctx, s.db, "SELECT type FROM users WHERE id=$1", userID)
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
	if typ == model.Business && user.Private != nil {
		return httperr.Forbidden("cannot update an business' visibility")
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

	q := postgres.SelectInID(model.Event, params.Fields, eventIDs)
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, errors.Wrap(err, "postgres: fetching events")
	}

	var events []event.Event
	if err := sqan.Rows(&events, rows); err != nil {
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

func (s service) getUserEvents(ctx context.Context, userID, roleName string, params params.Query) ([]event.Event, error) {
	whereCond := "id IN (SELECT event_id FROM events_users_roles WHERE user_id=$1 AND role_name=$2)"
	q := postgres.SelectWhere(model.Event, whereCond, "id", params)
	rows, err := s.db.QueryContext(ctx, q, userID, roleName)
	if err != nil {
		return nil, errors.Wrapf(err, "fetching events with role %s", roleName)
	}

	var events []event.Event
	if err := sqan.Rows(&events, rows); err != nil {
		return nil, err
	}

	return events, nil
}

func (s service) parseIDsAndScan(ctx context.Context, rdf []byte, params params.Query) ([]User, error) {
	userIDs := dgraph.ParseRDFULIDs(rdf)
	if len(userIDs) == 0 {
		return nil, nil
	}

	q := postgres.SelectInID(model.User, params.Fields, userIDs)
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, errors.Wrap(err, "postgres: fetching users")
	}

	var users []User
	if err := sqan.Rows(&users, rows); err != nil {
		return nil, err
	}

	return users, nil
}

func (s service) getBy(ctx context.Context, query, value string) (User, error) {
	rows, err := s.db.QueryContext(ctx, query, value)
	if err != nil {
		return User{}, errors.Wrap(err, "fetching user")
	}

	var user User
	if err := sqan.Row(&user, rows); err != nil {
		return User{}, errors.Wrap(err, "scanning user")
	}

	return user, nil
}
