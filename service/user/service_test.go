package user_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/ulid"
	"github.com/GGP1/groove/model"
	"github.com/GGP1/groove/service/auth"
	"github.com/GGP1/groove/service/event"
	"github.com/GGP1/groove/service/event/role"
	"github.com/GGP1/groove/service/user"
	"github.com/GGP1/groove/storage/postgres"
	"github.com/GGP1/groove/test"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
)

var (
	userSv  user.Service
	eventSv event.Service
	db      *sql.DB
)

const adminEmail = "admin@email.com"

func TestMain(m *testing.M) {
	test.Main(
		m,
		func(pg *sql.DB, r *redis.Client) {
			db = pg
			roleService := role.NewService(db, r)
			admins := map[string]struct{}{adminEmail: {}}
			// TODO: mock notification service (firabase api)
			userSv = user.NewService(db, r, admins, nil)
			eventSv = event.NewService(db, r, nil, roleService)
		},
		test.Postgres, test.Redis,
	)
}

func TestBlock(t *testing.T) {
	ctx := context.Background()

	blockerID := test.CreateUser(t, db, "block1@email.com", "block1")
	blockedID := test.CreateUser(t, db, "block2@email.com", "block2")

	sqlTx, ctx := postgres.BeginTx(ctx, db)

	err := userSv.Block(ctx, blockerID, blockedID)
	assert.NoError(t, err)

	assert.NoError(t, sqlTx.Commit())

	blocked, err := userSv.GetBlocked(ctx, blockerID, params.Query{LookupID: blockedID})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(blocked))
	assert.Equal(t, blockedID, blocked[0].ID)

	blockedBy, err := userSv.GetBlockedBy(ctx, blockedID, params.Query{LookupID: blockerID})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(blockedBy))
	assert.Equal(t, blockerID, blockedBy[0].ID)

	// Remove block and test again
	err = userSv.Unblock(ctx, blockerID, blockedID)
	assert.NoError(t, err)

	blocked2, err := userSv.GetBlocked(ctx, blockerID, params.Query{LookupID: blockedID})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(blocked2))

	blockedBy2, err := userSv.GetBlockedBy(ctx, blockedID, params.Query{LookupID: blockerID})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(blockedBy2))
}

func TestCanInvite(t *testing.T) {
	ctx := context.Background()

	userID := test.CreateUser(t, db, "can_invite@email.com", "can_invite")
	invitedID := test.CreateUser(t, db, "can_invite2@email.com", "can_invite2")

	t.Run("Friends", func(t *testing.T) {
		_, err := db.Exec("UPDATE users SET invitations=1 WHERE id=$1", invitedID)
		assert.NoError(t, err)

		err = userSv.AddFriend(ctx, userID, invitedID)
		assert.NoError(t, err)

		ok, err := userSv.CanInvite(ctx, userID, invitedID)
		assert.NoError(t, err)

		assert.True(t, ok)
	})

	t.Run("Nobody", func(t *testing.T) {
		_, err := db.Exec("UPDATE users SET invitations=2 WHERE id=$1", invitedID)
		assert.NoError(t, err)

		ok, err := userSv.CanInvite(ctx, userID, invitedID)
		assert.NoError(t, err)

		assert.False(t, ok)
	})
}

func TestCreate(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	typ := model.Personal
	createUser := model.CreateUser{
		Name:      "Create",
		Username:  "Create",
		Email:     "create@email.com",
		Password:  "1",
		BirthDate: &now,
		Type:      &typ,
	}
	id, err := userSv.Create(ctx, createUser)
	assert.NoError(t, err)

	user, err := userSv.GetByUsername(ctx, createUser.Username)
	assert.NoError(t, err)
	assert.Equal(t, id, user.ID)
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	id := test.CreateUser(t, db, "delete@email.com", "delete")

	err := userSv.Delete(ctx, id)
	assert.NoError(t, err)

	_, err = userSv.GetByID(ctx, id)
	assert.Error(t, err)
}

func TestGetBy(t *testing.T) {
	ctx := context.Background()
	username := "username"
	email := "email"

	id := test.CreateUser(t, db, email, username)

	eUser, err := userSv.GetByEmail(ctx, email)
	assert.NoError(t, err)

	uUser, err := userSv.GetByUsername(ctx, username)
	assert.NoError(t, err)

	idUser, err := userSv.GetByID(ctx, id)
	assert.NoError(t, err)

	assert.Equal(t, id, eUser.ID)
	assert.Equal(t, eUser, uUser)
	assert.Equal(t, eUser, idUser)
}

func TestGetAttendingEvents(t *testing.T) {
	ctx := context.Background()

	eventID := test.CreateEvent(t, db, "TestGetAttendingEvents")
	userID := test.CreateUser(t, db, "attending@email.com", "attending")

	events, err := userSv.GetAttendingEvents(ctx, userID, params.Query{LookupID: eventID})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(events))
	assert.Equal(t, eventID, events[0].ID)
}

func TestGetInvitedEvents(t *testing.T) {
	ctx := context.Background()

	eventID := test.CreateEvent(t, db, "TestGetInvitedEvents")
	userID := test.CreateUser(t, db, "invited@email.com", "invited")

	invite := model.Invite{
		EventID: eventID,
		UserIDs: []string{userID},
	}
	err := userSv.InviteToEvent(ctx, auth.Session{ID: ulid.NewString()}, invite)
	assert.NoError(t, err)

	events, err := userSv.GetInvitedEvents(ctx, userID, params.Query{LookupID: eventID})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(events))
	assert.Equal(t, eventID, events[0].ID)
}

func TestGetHostedEvents(t *testing.T) {
	ctx := context.Background()

	userID := test.CreateUser(t, db, "hosted@email.com", "hosted")

	boolean := false
	createEvent := model.CreateEvent{
		HostID: userID,
		Name:   "TestGetHostedEvents",
		Type:   model.Talk,
		Public: &boolean,
		Slots:  100,
		Cron:   "0 0 * * * 60",
	}
	eventID, err := eventSv.Create(ctx, createEvent)
	assert.NoError(t, err)

	events, err := userSv.GetHostedEvents(ctx, userID, params.Query{LookupID: eventID})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(events))
	assert.Equal(t, eventID, events[0].ID)
}

func TestGetLikedEvents(t *testing.T) {
	ctx := context.Background()

	eventID := test.CreateEvent(t, db, "TestGetLikedEvents")
	userID := test.CreateUser(t, db, "liked@email.com", "liked")

	err := eventSv.Like(ctx, eventID, userID)
	assert.NoError(t, err)

	events, err := userSv.GetLikedEvents(ctx, userID, params.Query{LookupID: eventID})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(events))
	assert.Equal(t, eventID, events[0].ID)
}

func TestFriends(t *testing.T) {
	ctx := context.Background()

	userID := test.CreateUser(t, db, "friend@email.com", "friend")
	friendID := test.CreateUser(t, db, "friend2@email.com", "friend2")

	err := userSv.AddFriend(context.Background(), userID, friendID)
	assert.NoError(t, err)

	// Test if we receive the friend user
	friends, err := userSv.GetFriends(ctx, userID, params.Query{LookupID: friendID})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(friends))
	assert.Equal(t, friendID, friends[0].ID)

	friendsCount, err := userSv.GetFriendsCount(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, 1, friendsCount)

	areFriends, err := userSv.AreFriends(ctx, userID, friendID)
	assert.NoError(t, err)
	assert.True(t, areFriends)

	// Remove friendship and test again
	err = userSv.RemoveFriend(ctx, userID, friendID)
	assert.NoError(t, err)

	friends2, err := userSv.GetFriends(ctx, userID, params.Query{LookupID: friendID})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(friends2))
}

func TestIsAdmin(t *testing.T) {
	adminID := test.CreateUser(t, db, adminEmail, "admin")
	nonAdminID := test.CreateUser(t, db, "nonadmin@email.com", "nonadmin")

	ctx := context.Background()
	ok, err := userSv.IsAdmin(ctx, adminID)
	assert.NoError(t, err)
	assert.True(t, ok)

	ok2, err := userSv.IsAdmin(ctx, nonAdminID)
	assert.NoError(t, err)
	assert.False(t, ok2)
}

func TestProfileIsPrivate(t *testing.T) {
	ctx := context.Background()

	userID := test.CreateUser(t, db, "private@email.com", "private")

	ok, err := userSv.ProfileIsPrivate(ctx, userID)
	assert.NoError(t, err)
	assert.False(t, ok)

	priv := true
	err = userSv.Update(ctx, userID, model.UpdateUser{Private: &priv})
	assert.NoError(t, err)

	ok2, err := userSv.ProfileIsPrivate(ctx, userID)
	assert.NoError(t, err)
	assert.True(t, ok2)
}

func TestSearch(t *testing.T) {
	ctx := context.Background()

	userID := test.CreateUser(t, db, "search@email.com", "search")

	users, err := userSv.Search(ctx, "sea", params.Query{})
	assert.NoError(t, err)

	assert.Equal(t, 1, len(users))
	assert.Equal(t, userID, users[0].ID)
}

func TestUpdate(t *testing.T) {
	ctx := context.Background()

	userID := test.CreateUser(t, db, "update@email.com", "update")

	uptUName := "updatedUsername"
	uptUser := model.UpdateUser{
		Username: &uptUName,
	}
	err := userSv.Update(ctx, userID, uptUser)
	assert.NoError(t, err)

	user, err := userSv.GetByID(ctx, userID)
	assert.NoError(t, err)

	assert.Equal(t, uptUName, user.Username)
}
