package user

import (
	"context"
	"database/sql"
	"time"

	"github.com/GGP1/groove/internal/cache"
	"github.com/GGP1/groove/internal/httperr"
	"github.com/GGP1/groove/internal/log"
	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/roles"
	"github.com/GGP1/groove/internal/txgroup"
	"github.com/GGP1/groove/internal/ulid"
	"github.com/GGP1/groove/model"
	"github.com/GGP1/groove/service/auth"
	"github.com/GGP1/groove/service/notification"
	"github.com/GGP1/groove/storage/postgres"
	"github.com/GGP1/sqan"

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
	Create(ctx context.Context, user model.CreateUser) (string, error)
	Delete(ctx context.Context, userID string) error
	Follow(ctx context.Context, userID, businessID string) error
	GetAttendingEvents(ctx context.Context, userID string, params params.Query) ([]model.Event, error)
	GetAttendingEventsCount(ctx context.Context, userID string) (int64, error)
	GetBannedEvents(ctx context.Context, userID string, params params.Query) ([]model.Event, error)
	GetBannedEventsCount(ctx context.Context, userID string) (int64, error)
	GetBlocked(ctx context.Context, userID string, params params.Query) ([]model.User, error)
	GetBlockedCount(ctx context.Context, userID string) (int64, error)
	GetBlockedBy(ctx context.Context, userID string, params params.Query) ([]model.User, error)
	GetBlockedByCount(ctx context.Context, userID string) (int64, error)
	GetByEmail(ctx context.Context, value string) (model.User, error)
	GetByID(ctx context.Context, value string) (model.User, error)
	GetByUsername(ctx context.Context, value string) (model.User, error)
	GetFollowers(ctx context.Context, userID string, params params.Query) ([]model.User, error)
	GetFollowersCount(ctx context.Context, userID string) (int64, error)
	GetFollowing(ctx context.Context, userID string, params params.Query) ([]model.User, error)
	GetFollowingCount(ctx context.Context, userID string) (int64, error)
	GetFriends(ctx context.Context, userID string, params params.Query) ([]model.User, error)
	GetFriendsCount(ctx context.Context, userID string) (int64, error)
	GetFriendsInCommon(ctx context.Context, userID, friendID string, params params.Query) ([]model.User, error)
	GetFriendsInCommonCount(ctx context.Context, userID, friendID string) (int64, error)
	GetFriendsNotInCommon(ctx context.Context, userID, friendID string, params params.Query) ([]model.User, error)
	GetFriendsNotInCommonCount(ctx context.Context, userID, friendID string) (int64, error)
	GetHostedEvents(ctx context.Context, userID string, params params.Query) ([]model.Event, error)
	GetHostedEventsCount(ctx context.Context, userID string) (int64, error)
	GetInvitedEvents(ctx context.Context, userID string, params params.Query) ([]model.Event, error)
	GetInvitedEventsCount(ctx context.Context, userID string) (int64, error)
	GetLikedEvents(ctx context.Context, userID string, params params.Query) ([]model.Event, error)
	GetLikedEventsCount(ctx context.Context, userID string) (int64, error)
	GetStatistics(ctx context.Context, userID string) (model.UserStatistics, error)
	InviteToEvent(ctx context.Context, session auth.Session, invite model.Invite) error
	IsAdmin(ctx context.Context, userID string) (bool, error)
	IsBlocked(ctx context.Context, userID, blockedID string) (bool, error)
	ProfileIsPrivate(ctx context.Context, userID string) (bool, error)
	RemoveFriend(ctx context.Context, userID string, friendID string) error
	Search(ctx context.Context, query string, params params.Query) ([]model.User, error)
	SendFriendRequest(ctx context.Context, session auth.Session, friendID string) error
	Type(ctx context.Context, userID string) (model.UserType, error)
	Unblock(ctx context.Context, userID, blockedID string) error
	Update(ctx context.Context, userID string, user model.UpdateUser) error
}

type service struct {
	db    *sql.DB
	cache cache.Client

	admins  map[string]interface{}
	metrics metrics

	notificationService notification.Service
}

// NewService returns a new user service.
func NewService(
	db *sql.DB,
	cache cache.Client,
	admins map[string]interface{},
	notificationService notification.Service,
) Service {
	return &service{
		db:                  db,
		cache:               cache,
		admins:              admins,
		metrics:             initMetrics(),
		notificationService: notificationService,
	}
}

// AddFriend adds a new friend.
func (s *service) AddFriend(ctx context.Context, userID, friendID string) error {
	s.metrics.incMethodCalls("AddFriend")

	sqlTx := txgroup.SQLTx(ctx)

	q := "INSERT INTO users_friends (user_id, friend_id) VALUES ($1, $2)"
	if _, err := sqlTx.ExecContext(ctx, q, userID, friendID); err != nil {
		return errors.Wrap(err, "adding friend")
	}

	return nil
}

// AreFriends returns if the users are friends or not.
func (s *service) AreFriends(ctx context.Context, userID, targetID string) (bool, error) {
	s.metrics.incMethodCalls("AreFriends")

	q := `SELECT EXISTS (
		SELECT 1 FROM users_friends WHERE user_id=$1 AND friend_id=$2
		UNION
		SELECT 1 FROM users_friends WHERE user_id=$2 AND friend_id=$1
	)`
	row := s.db.QueryRowContext(ctx, q, userID, targetID)

	var areFriends bool
	if err := row.Scan(&areFriends); err != nil {
		return false, errors.Wrap(err, "checking friendship")
	}

	return areFriends, nil
}

// Block blocks a user.
func (s *service) Block(ctx context.Context, userID, blockedID string) error {
	s.metrics.incMethodCalls("Block")

	sqlTx := txgroup.SQLTx(ctx)

	q := "INSERT INTO users_blocked (user_id, blocked_id) VALUES ($1, $2)"
	if _, err := sqlTx.ExecContext(ctx, q, userID, blockedID); err != nil {
		return errors.Wrap(err, "blocking user")
	}

	return nil
}

func (s *service) CanInvite(ctx context.Context, authUserID, invitedID string) (bool, error) {
	s.metrics.incMethodCalls("CanInvite")
	sqlTx := txgroup.SQLTx(ctx)

	var (
		blocked     bool
		invitations model.Invitations
	)
	q := `SELECT 
	u.invitations, EXISTS(SELECT 1 FROM users_blocked WHERE user_id = u.id AND blocked_id=$2) as blocked
	FROM users u LEFT JOIN users_blocked ub 
	ON u.id = ub.user_id 
	WHERE u.id=$1`
	if err := sqlTx.QueryRowContext(ctx, q, invitedID, authUserID).Scan(&invitations, &blocked); err != nil {
		return false, errors.Wrap(err, "querying invitations")
	}
	if invitations == model.Nobody || blocked {
		return false, nil
	}

	return s.AreFriends(ctx, authUserID, invitedID)
}

// Create creates a new user.
func (s *service) Create(ctx context.Context, user model.CreateUser) (string, error) {
	s.metrics.incMethodCalls("Create")

	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		log.DPanic("failed generating user's password hash", zap.Error(err))
		return "", errors.Wrap(err, "generating password hash")
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

	id := ulid.NewString()
	q := `INSERT INTO users 
	(id, name, username, email, password, birth_date, description, 
	profile_image_url, type, is_admin, invitations, updated_at) 
	VALUES 
	($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	_, err = s.db.ExecContext(ctx, q, id, user.Name, user.Username,
		user.Email, hash, user.BirthDate, user.Description, user.ProfileImageURL,
		user.Type, isAdmin, user.Invitations, time.Time{})
	if err != nil {
		return "", errors.Wrap(err, "creating user")
	}

	return id, nil
}

// Delete a user from the system.
func (s *service) Delete(ctx context.Context, userID string) error {
	s.metrics.incMethodCalls("Delete")

	if _, err := s.db.ExecContext(ctx, "DELETE FROM users WHERE id=$1", userID); err != nil {
		return errors.Wrap(err, "deleting user")
	}

	return s.cache.Delete(model.T.User.CacheKey(userID))
}

// Follow follows an business.
func (s *service) Follow(ctx context.Context, userID, businessID string) error {
	s.metrics.incMethodCalls("Follow")

	typ, err := s.Type(ctx, businessID)
	if err != nil {
		return err
	}
	if typ != model.Business {
		return httperr.Forbidden("only businesses can be followed")
	}

	q := "INSERT INTO users_followers (user_id, follower_id) VALUES ($1, $2)"
	sqlTx := txgroup.SQLTx(ctx)
	if _, err := sqlTx.ExecContext(ctx, q, userID, businessID); err != nil {
		return errors.Wrap(err, "following business")
	}

	return nil
}

// GetAttendingEvents returns the events the user is assiting to.
func (s *service) GetAttendingEvents(ctx context.Context, userID string, params params.Query) ([]model.Event, error) {
	s.metrics.incMethodCalls("GetAttendingEvents")

	q := `SELECT {fields} FROM {table} WHERE id IN 
	(SELECT event_id FROM events_users_roles WHERE user_id=$1 AND role_name NOT IN ($2, $3))
	{pag}`
	query := postgres.Select(model.T.Event, q, params)
	rows, err := s.db.QueryContext(ctx, query, userID, roles.Viewer, roles.Host)
	if err != nil {
		return nil, err
	}

	var events []model.Event
	if err := sqan.Rows(&events, rows); err != nil {
		return nil, err
	}

	return events, nil
}

// GetAttendingEventsCount returns the events the user is assiting to.
func (s *service) GetAttendingEventsCount(ctx context.Context, userID string) (int64, error) {
	s.metrics.incMethodCalls("GetAttendingEventsCount")

	q := "SELECT COUNT(*) FROM events_users_roles WHERE user_id=$1 AND role_name NOT IN ($2, $3)"
	return postgres.QueryInt(ctx, s.db, q, userID, roles.Viewer, roles.Host)
}

// GetBannedEvents returns the events that the user is banned from.
func (s *service) GetBannedEvents(ctx context.Context, userID string, params params.Query) ([]model.Event, error) {
	s.metrics.incMethodCalls("GetBannedEvents")

	q := "SELECT {fields} FROM {table} WHERE id IN (SELECT event_id FROM events_banned WHERE user_id=$1) {pag}"
	query := postgres.Select(model.T.Event, q, params)
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}

	var events []model.Event
	if err := sqan.Rows(&events, rows); err != nil {
		return nil, err
	}

	return events, nil
}

// GetBannedEvents returns the number of events that the user is banned from.
func (s *service) GetBannedEventsCount(ctx context.Context, userID string) (int64, error) {
	s.metrics.incMethodCalls("GetBannedEventsCount")

	return postgres.QueryInt(ctx, s.db, "SELECT COUNT(*) FROM events_banned WHERE user_id=$1", userID)
}

func (s *service) GetBlocked(ctx context.Context, userID string, params params.Query) ([]model.User, error) {
	s.metrics.incMethodCalls("GetBlocked")

	q := "SELECT {fields} FROM {table} WHERE id IN (SELECT user_id FROM users_blocked WHERE user_id=$1) {pag}"
	return s.scanUsers(ctx, params, q, userID)
}

func (s *service) GetBlockedCount(ctx context.Context, userID string) (int64, error) {
	s.metrics.incMethodCalls("GetBlockedCount")

	return postgres.QueryInt(ctx, s.db, "SELECT COUNT(*) FROM users_blocked WHERE user_id=$1", userID)
}

func (s *service) GetBlockedBy(ctx context.Context, userID string, params params.Query) ([]model.User, error) {
	s.metrics.incMethodCalls("GetBlockedBy")

	q := "SELECT {fields} FROM {table} WHERE id IN (SELECT user_id FROM users_blocked WHERE blocked_id=$1) {pag}"
	return s.scanUsers(ctx, params, q, userID)
}

func (s *service) GetBlockedByCount(ctx context.Context, userID string) (int64, error) {
	s.metrics.incMethodCalls("GetBlockedByCount")

	return postgres.QueryInt(ctx, s.db, "SELECT COUNT(*) FROM users_blocked WHERE blocked_id=$1", userID)
}

func (s *service) GetByEmail(ctx context.Context, email string) (model.User, error) {
	s.metrics.incMethodCalls("GetByEmail")

	q := `SELECT 
	id, name, username, email, birth_date, description, private, type,
	verified_email, profile_image_url, invitations, created_at, updated_at
	FROM users WHERE email=$1`
	return s.scanUser(ctx, q, email)
}

func (s *service) GetByID(ctx context.Context, userID string) (model.User, error) {
	s.metrics.incMethodCalls("GetByID")

	q := `SELECT 
	id, name, username, email, birth_date, description, private, type,
	verified_email, profile_image_url, invitations, created_at, updated_at
	FROM users WHERE id=$1`
	return s.scanUser(ctx, q, userID)
}

func (s *service) GetByUsername(ctx context.Context, username string) (model.User, error) {
	s.metrics.incMethodCalls("GetByUsername")

	q := `SELECT 
	id, name, username, email, birth_date, description, private, type,
	verified_email, profile_image_url, invitations, created_at, updated_at
	FROM users WHERE username=$1`
	return s.scanUser(ctx, q, username)
}

// GetFollowers returns a user's followers. Only businesses can have followers, calling this on
// a standard user will return always nil.
func (s *service) GetFollowers(ctx context.Context, userID string, params params.Query) ([]model.User, error) {
	s.metrics.incMethodCalls("GetFollowers")

	q := "SELECT {fields} FROM {table} WHERE id IN (SELECT follower_id FROM users_followers WHERE user_id=$1) {pag}"
	return s.scanUsers(ctx, params, q, userID)
}

// GetFollowersCount returns a user's number of followers. Only businesses can have followers, calling this on
// a standard user will return always 0.
func (s *service) GetFollowersCount(ctx context.Context, userID string) (int64, error) {
	s.metrics.incMethodCalls("GetFollowersCount")

	return postgres.QueryInt(ctx, s.db, "SELECT COUNT(*) FROM users_followers WHERE user_id=$1", userID)
}

// GetFollowing returns the businesses the user is following.
func (s *service) GetFollowing(ctx context.Context, userID string, params params.Query) ([]model.User, error) {
	s.metrics.incMethodCalls("GetFollowing")

	q := "SELECT {fields} FROM {table} WHERE id IN (SELECT user_id FROM users_followers WHERE follower_id=$1) {pag}"
	return s.scanUsers(ctx, params, q, userID)
}

// GetFollowingCount returns the number of businesses the user is following.
func (s *service) GetFollowingCount(ctx context.Context, userID string) (int64, error) {
	s.metrics.incMethodCalls("GetFollowingCount")

	return postgres.QueryInt(ctx, s.db, "SELECT COUNT(*) FROM users_followers WHERE follower_id=$1", userID)
}

// GetFriends returns people the user fetched is friend of.
func (s *service) GetFriends(ctx context.Context, userID string, params params.Query) ([]model.User, error) {
	s.metrics.incMethodCalls("GetFriends")

	q := `SELECT {fields} FROM {table} WHERE id IN (
		SELECT friend_id FROM users_friends WHERE user_id=$1
		UNION
		SELECT user_id FROM users_friends WHERE friend_id=$1
	) {pag}`
	return s.scanUsers(ctx, params, q, userID)
}

// GetFriendsCount returns the number of users friends of the one fetched.
func (s *service) GetFriendsCount(ctx context.Context, userID string) (int64, error) {
	s.metrics.incMethodCalls("GetFriendsCount")

	q := `SELECT COUNT(*) FROM (
		SELECT 1 FROM users_friends WHERE user_id=$1
		UNION
		SELECT 1 FROM users_friends WHERE friend_id=$1
	) AS x`
	return postgres.QueryInt(ctx, s.db, q, userID)
}

// GetFriendsInCommon returns the friends in common between userID and friendID.
func (s *service) GetFriendsInCommon(ctx context.Context, userID, friendID string, params params.Query) ([]model.User, error) {
	s.metrics.incMethodCalls("GetFriendsInCommon")

	q := `SELECT {fields} FROM {table} WHERE id IN (
		SELECT friend_id FROM users_friends WHERE user_id=$1
		UNION
		SELECT user_id FROM users_friends WHERE friend_id=$1
		UNION
		SELECT friend_id FROM users_friends WHERE user_id=$2
		UNION
		SELECT user_id FROM users_friends WHERE friend_id=$2
	) {pag}`
	return s.scanUsers(ctx, params, q, userID, friendID)
}

// GetFriendsInCommonCount returns the number of matching friends between userID and friendID.
func (s *service) GetFriendsInCommonCount(ctx context.Context, userID, friendID string) (int64, error) {
	s.metrics.incMethodCalls("GetFriendsInCommonCount")

	q := `SELECT COUNT(*) FROM (
		SELECT 1 FROM users_friends WHERE user_id=$1
		UNION
		SELECT 1 FROM users_friends WHERE friend_id=$1
		UNION
		SELECT 1 FROM users_friends WHERE user_id=$2
		UNION
		SELECT 1 FROM users_friends WHERE friend_id=$2
	) AS x`
	return postgres.QueryInt(ctx, s.db, q, userID, friendID)
}

// GetFriendsNotInCommon returns the friends that are not in common between userID and friendID.
func (s *service) GetFriendsNotInCommon(ctx context.Context, userID, friendID string, params params.Query) ([]model.User, error) {
	s.metrics.incMethodCalls("GetFriendsNotInCommon")

	q := `SELECT {fields} FROM {table} WHERE id IN (
		(
			SELECT friend_id FROM users_friends WHERE user_id=$1
			UNION
			SELECT user_id FROM users_friends WHERE friend_id=$1
		)
		EXCEPT
		(
			SELECT friend_id FROM users_friends WHERE user_id=$2
			UNION
			SELECT user_id FROM users_friends WHERE friend_id=$2
		)
	) {pag}`
	return s.scanUsers(ctx, params, q, userID, friendID)
}

// GetFriendsNotInCommonCount returns the number of non-matching friends between userID and friendID.
func (s *service) GetFriendsNotInCommonCount(ctx context.Context, userID, friendID string) (int64, error) {
	s.metrics.incMethodCalls("GetFriendsNotInCommonCount")

	q := `SELECT COUNT(*) FROM (
		(
			SELECT 1 FROM users_friends WHERE user_id=$1
			UNION
			SELECT 1 FROM users_friends WHERE friend_id=$1
		)
		EXCEPT
		(
			SELECT 1 FROM users_friends WHERE user_id=$2
			UNION
			SELECT 1 FROM users_friends WHERE friend_id=$2
		)
	)`
	return postgres.QueryInt(ctx, s.db, q, userID, friendID)
}

// GetHostedEvents returns the events hosted by the user with the given id.
func (s *service) GetHostedEvents(ctx context.Context, userID string, params params.Query) ([]model.Event, error) {
	s.metrics.incMethodCalls("GetHostedEvents")

	q := `SELECT {fields} FROM {table} WHERE id IN 
	(SELECT event_id FROM events_users_roles WHERE user_id=$1 AND role_name=$2)
	{pag}`
	return s.scanEvents(ctx, params, q, userID, string(roles.Host))
}

// GetHostedEventsCount returns the number of events hosted by the user with the given id.
func (s *service) GetHostedEventsCount(ctx context.Context, userID string) (int64, error) {
	s.metrics.incMethodCalls("GetHostedEventsCount")

	q := "SELECT COUNT(*) FROM events_users_roles WHERE user_id=$1 AND role_name=$2"
	return postgres.QueryInt(ctx, s.db, q, userID, roles.Host)
}

// GetInvitedEvents returns the events that the user is invited to.
func (s *service) GetInvitedEvents(ctx context.Context, userID string, params params.Query) ([]model.Event, error) {
	s.metrics.incMethodCalls("GetInvitedEvents")

	q := `SELECT {fields} FROM {table} WHERE id IN 
	(SELECT event_id FROM events_users_roles WHERE user_id=$1 AND role_name=$2)
	{pag}`
	return s.scanEvents(ctx, params, q, userID, string(roles.Viewer))
}

// GetInvitedEventsCount returns the number of events that the user is invited to.
func (s *service) GetInvitedEventsCount(ctx context.Context, userID string) (int64, error) {
	s.metrics.incMethodCalls("GetInvitedEventsCount")

	q := "SELECT COUNT(*) FROM events_users_roles WHERE user_id=$1 AND role_name=$2"
	return postgres.QueryInt(ctx, s.db, q, userID, roles.Viewer)
}

// GetLikedEvents returns the events that the user likes.
func (s *service) GetLikedEvents(ctx context.Context, userID string, params params.Query) ([]model.Event, error) {
	s.metrics.incMethodCalls("GetLikedEvents")

	q := `SELECT {fields} FROM {table} WHERE id IN 
	(SELECT event_id FROM events_likes WHERE user_id=$1)
	{pag}`
	return s.scanEvents(ctx, params, q, userID)
}

// GetLikedEventsCount returns the number of events that the user likes.
func (s *service) GetLikedEventsCount(ctx context.Context, userID string) (int64, error) {
	s.metrics.incMethodCalls("GetLikedEventsCount")

	return postgres.QueryInt(ctx, s.db, "SELECT COUNT(*) FROM events_likes WHERE user_id=$1", userID)
}

// GetStatistics returns a users' predicates statistics.
func (s *service) GetStatistics(ctx context.Context, userID string) (model.UserStatistics, error) {
	s.metrics.incMethodCalls("GetStatistics")

	q := `SELECT
	(SELECT COUNT(*) FROM users_blocked WHERE user_id=$1) AS blocked_count,
	(SELECT COUNT(*) FROM users_blocked WHERE blocked_id=$1) AS blocked_by_count,
	(SELECT COUNT(*) FROM
		(
			SELECT 1 FROM users_friends WHERE user_id=$1
			UNION
			SELECT 1 FROM users_friends WHERE friend_id=$1
		) AS x
	) AS friends_count,
	(SELECT COUNT(*) FROM users_followers WHERE user_id=$1) AS followers_count,
	(SELECT COUNT(*) FROM users_followers WHERE follower_id=$1) AS following_count,
	(SELECT COUNT(*) FROM events_users_roles WHERE user_id=$1 AND role_name NOT IN ($2, $3)) AS attending_events_count,
	(SELECT COUNT(*) FROM events_users_roles WHERE user_id=$1 AND role_name=$2) AS hosted_events_count,
	(SELECT COUNT(*) FROM events_bans WHERE user_id=$1) AS banned_events_count,
	(SELECT COUNT(*) FROM events_likes WHERE user_id=$1) AS liked_events_count,
	(SELECT COUNT(*) FROM events_users_roles WHERE user_id=$1 AND role_name=$3) AS invitations_count`

	rows, err := s.db.QueryContext(ctx, q, userID, roles.Host, roles.Viewer)
	if err != nil {
		return model.UserStatistics{}, err
	}

	var stats model.UserStatistics
	if err := sqan.Row(&stats, rows); err != nil {
		return model.UserStatistics{}, err
	}

	return stats, nil
}

// InviteToEvent invites a user to an event.
func (s *service) InviteToEvent(ctx context.Context, session auth.Session, invite model.Invite) error {
	s.metrics.incMethodCalls("InviteToEvent")

	for _, userID := range invite.UserIDs {
		canInvite, err := s.CanInvite(ctx, session.ID, userID)
		if err != nil {
			return err
		}
		if !canInvite {
			return httperr.Forbidden("you aren't allowed to invite the user " + userID)
		}
	}

	err := s.notificationService.CreateMany(ctx, session, model.CreateNotificationMany{
		SenderID:    session.ID,
		ReceiverIDs: invite.UserIDs,
		EventID:     &invite.EventID,
		Content:     notification.InvitationContent(session),
		Type:        model.Invitation,
	})
	if err != nil {
		return errors.Wrap(err, "creating invitation notifications")
	}

	return nil
}

// IsAdmin returns if the user is an administrator or not.
func (s *service) IsAdmin(ctx context.Context, userID string) (bool, error) {
	s.metrics.incMethodCalls("IsAdmin")

	return postgres.QueryBool(ctx, s.db, "SELECT is_admin FROM users WHERE id=$1", userID)
}

// IsBlocked returns if the blockedID user is blocked by the userID one or not.
func (s *service) IsBlocked(ctx context.Context, userID, blockedID string) (bool, error) {
	s.metrics.incMethodCalls("IsBlocked")

	q := "SELECT EXISTS(SELECT 1 FROM users_blocked WHERE user_id=$1 AND blocked_id=$2)"
	return postgres.QueryBool(ctx, s.db, q, userID, blockedID)
}

// ProfileIsPrivate returns if the user's profile is private or not.
func (s *service) ProfileIsPrivate(ctx context.Context, userID string) (bool, error) {
	s.metrics.incMethodCalls("ProfileIsPrivate")

	return postgres.QueryBool(ctx, s.db, "SELECT private FROM users WHERE id=$1", userID)
}

// RemoveFriend removes a friend.
func (s *service) RemoveFriend(ctx context.Context, userID string, friendID string) error {
	s.metrics.incMethodCalls("RemoveFriend")

	sqlTx := txgroup.SQLTx(ctx)

	q := "DELETE FROM users_friends WHERE user_id=$1 AND friend_id=$2"
	if _, err := sqlTx.ExecContext(ctx, q, userID, friendID); err != nil {
		return errors.Wrap(err, "removing friend")
	}

	return nil
}

// Search returns users matching the given query.
func (s *service) Search(ctx context.Context, query string, params params.Query) ([]model.User, error) {
	s.metrics.incMethodCalls("Search")

	q := `SELECT {fields} FROM {table} WHERE search @@ to_tsquery($1) AND private=false {pag}`
	return s.scanUsers(ctx, params, q, postgres.ToTSQuery(query))
}

// SendFriendRequest sends a friend request to the user indicated.
func (s *service) SendFriendRequest(ctx context.Context, session auth.Session, friendID string) error {
	s.metrics.incMethodCalls("SendFriendRequest")

	areFriends, err := s.AreFriends(ctx, session.ID, friendID)
	if err != nil {
		return err
	}
	if areFriends {
		return httperr.Forbidden(friendID + " is already your friend")
	}

	isBlocked, err := s.IsBlocked(ctx, friendID, session.ID)
	if err != nil {
		return err
	}
	if isBlocked {
		return httperr.Forbidden(friendID + " blocked you")
	}

	userType, err := s.Type(ctx, friendID)
	if err != nil {
		return err
	}
	if userType == model.Business {
		return httperr.Forbidden("cannot invite a business")
	}

	err = s.notificationService.Create(ctx, session, model.CreateNotification{
		SenderID:   session.ID,
		ReceiverID: friendID,
		Content:    notification.FriendRequestContent(session),
		Type:       model.FriendRequest,
	})
	if err != nil {
		return errors.Wrap(err, "creating friend request notification")
	}
	return nil
}

// Type returns the user's type.
func (s *service) Type(ctx context.Context, userID string) (model.UserType, error) {
	s.metrics.incMethodCalls("Type")

	cacheKey := cache.UserTypeKey(userID)
	if v, err := s.cache.Get(cacheKey); err == nil {
		return model.UserType(cache.BytesToInt(v)), nil
	}

	accType, err := postgres.QueryInt(ctx, s.db, "SELECT type FROM users WHERE id=$1", userID)
	if err != nil {
		return 0, errors.Wrap(err, "querying user type")
	}

	if err := s.cache.Set(cacheKey, cache.IntToBytes(accType)); err != nil {
		return 0, errors.Wrap(err, "saving user type to the cache")
	}

	return model.UserType(accType), nil
}

// Unblock removes the block from one user to other.
func (s *service) Unblock(ctx context.Context, userID string, blockedID string) error {
	s.metrics.incMethodCalls("Unblock")

	sqlTx := txgroup.SQLTx(ctx)

	q := "DELETE FROM users_blocked WHERE user_id=$1 AND blocked_id=$2"
	if _, err := sqlTx.ExecContext(ctx, q, userID, blockedID); err != nil {
		return errors.Wrap(err, "removing block")
	}

	return nil
}

// Update updates a user.
func (s *service) Update(ctx context.Context, userID string, user model.UpdateUser) error {
	s.metrics.incMethodCalls("Update")

	typ, err := s.Type(ctx, userID)
	if err != nil {
		return err
	}
	if typ == model.Business && user.Private != nil {
		return httperr.Forbidden("cannot update an business' visibility")
	}

	sqlTx := txgroup.SQLTx(ctx)

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

	if err := s.cache.Delete(model.T.User.CacheKey(userID)); err != nil {
		return errors.Wrap(err, "deleting user")
	}
	return nil
}

func (s *service) scanEvents(ctx context.Context, params params.Query, query string, args ...interface{}) ([]model.Event, error) {
	q := postgres.Select(model.T.Event, query, params)
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, errors.Wrap(err, "querying events")
	}

	var events []model.Event
	if err := sqan.Rows(&events, rows); err != nil {
		return nil, errors.Wrap(err, "scanning events")
	}

	return events, nil
}

func (s *service) scanUser(ctx context.Context, query, value string) (model.User, error) {
	rows, err := s.db.QueryContext(ctx, query, value)
	if err != nil {
		return model.User{}, errors.Wrap(err, "querying user")
	}

	var user model.User
	if err := sqan.Row(&user, rows); err != nil {
		return model.User{}, errors.Wrap(err, "scanning user")
	}

	return user, nil
}

func (s *service) scanUsers(ctx context.Context, params params.Query, query string, args ...interface{}) ([]model.User, error) {
	q := postgres.Select(model.T.User, query, params)
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, errors.Wrap(err, "querying users")
	}

	var users []model.User
	if err := sqan.Rows(&users, rows); err != nil {
		return nil, errors.Wrap(err, "scanning users")
	}

	return users, nil
}
