package user_test

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"
	"time"

	"github.com/GGP1/groove/config"
	"github.com/GGP1/groove/internal/cache"
	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/ulid"
	"github.com/GGP1/groove/service/auth"
	"github.com/GGP1/groove/service/event"
	"github.com/GGP1/groove/service/event/role"
	"github.com/GGP1/groove/service/notification"
	"github.com/GGP1/groove/service/user"
	"github.com/GGP1/groove/storage/dgraph"
	"github.com/GGP1/groove/test"

	"github.com/dgraph-io/dgo/v210"
	"github.com/dgraph-io/dgo/v210/protos/api"
	"github.com/stretchr/testify/assert"
)

var (
	userSv      user.Service
	eventSv     event.Service
	db          *sql.DB
	dc          *dgo.Dgraph
	cacheClient cache.Client
)

const adminEmail = "admin@email.com"

// Note: each of the test functions creates users to test but does cleanup them to save time, be sure to create
// unique users for each one of them

func TestMain(m *testing.M) {
	pgContainer, postgres, err := test.RunPostgres()
	if err != nil {
		log.Fatal(err)
	}
	dcContainer, dgraph, conn, err := test.RunDgraph()
	if err != nil {
		log.Fatal(err)
	}
	mcContainer, memcached, err := test.RunMemcached()
	if err != nil {
		log.Fatal(err)
	}

	db = postgres
	dc = dgraph
	cacheClient = memcached
	authService := auth.NewService(db, nil, config.Sessions{})
	roleService := role.NewService(db, dc, cacheClient)
	notifService := notification.NewService(db, dc, config.Notifications{}, authService, roleService)
	admins := map[string]interface{}{adminEmail: struct{}{}}
	userSv = user.NewService(db, dc, cacheClient, admins, notifService, roleService)
	eventSv = event.NewService(db, dc, cacheClient, notifService, roleService)

	code := m.Run()

	if err := pgContainer.Close(); err != nil {
		log.Fatal(err)
	}
	if err := conn.Close(); err != nil {
		log.Fatal(err)
	}
	if err := dcContainer.Close(); err != nil {
		log.Fatal(err)
	}
	if err := mcContainer.Close(); err != nil {
		log.Fatal(err)
	}

	os.Exit(code)
}

func TestBlock(t *testing.T) {
	ctx := context.Background()
	userID := ulid.NewString()
	blockedID := ulid.NewString()

	err := test.CreateUser(ctx, db, dc, userID, "block1@email.com", "block1", "1")
	assert.NoError(t, err)
	err = test.CreateUser(ctx, db, dc, blockedID, "block2@email.com", "block2", "2")
	assert.NoError(t, err)

	err = userSv.Block(context.Background(), userID, blockedID)
	assert.NoError(t, err)

	blocked, err := userSv.GetBlocked(ctx, userID, params.Query{LookupID: blockedID})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(blocked))
	assert.Equal(t, blockedID, blocked[0].ID)

	blockedBy, err := userSv.GetBlockedBy(ctx, blockedID, params.Query{LookupID: userID})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(blockedBy))
	assert.Equal(t, userID, blockedBy[0].ID)

	// Remove block and test again
	err = userSv.Unblock(ctx, userID, blockedID)
	assert.NoError(t, err)

	blocked2, err := userSv.GetBlocked(ctx, userID, params.Query{LookupID: blockedID})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(blocked2))

	blockedBy2, err := userSv.GetBlockedBy(ctx, blockedID, params.Query{LookupID: userID})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(blockedBy2))
}

func TestCanInvite(t *testing.T) {
	ctx := context.Background()
	userID := ulid.NewString()
	invitedID := ulid.NewString()

	err := test.CreateUser(ctx, db, dc, userID, "can_invite@email.com", "can_invite", "1")
	assert.NoError(t, err)
	err = test.CreateUser(ctx, db, dc, invitedID, "can_invite2@email.com", "can_invite2", "1")
	assert.NoError(t, err)

	t.Run("Friends", func(t *testing.T) {
		_, err := db.Exec("UPDATE users SET invitations=1 WHERE id=$1", invitedID)
		assert.NoError(t, err)

		vars := map[string]string{"$user_id": userID, "$friend_id": invitedID}
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
		_, err = dc.NewTxn().Do(ctx, req)
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
	id := ulid.NewString()
	now := time.Now()
	createUser := user.CreateUser{
		Name:      "Create",
		Username:  "Create",
		Email:     "create@email.com",
		Password:  "1",
		BirthDate: &now,
	}
	err := userSv.Create(ctx, id, createUser)
	assert.NoError(t, err)

	user, err := userSv.GetByUsername(ctx, createUser.Username)
	assert.NoError(t, err)
	assert.Equal(t, id, user.ID)
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	id := ulid.NewString()
	err := test.CreateUser(ctx, db, dc, id, "delete@email.com", "delete", "1")
	assert.NoError(t, err)

	err = userSv.Delete(ctx, id)
	assert.NoError(t, err)

	_, err = userSv.GetByID(ctx, id)
	assert.Error(t, err)
}

func TestAddFriend(t *testing.T) {
	ctx := context.Background()
	userID := ulid.NewString()
	friendID := ulid.NewString()

	err := test.CreateUser(ctx, db, dc, userID, "friend@email.com", "friend", "1")
	assert.NoError(t, err)
	err = test.CreateUser(ctx, db, dc, friendID, "friend2@email.com", "friend2", "2")
	assert.NoError(t, err)

	err = userSv.AddFriend(context.Background(), userID, friendID)
	assert.NoError(t, err)

	// Test if we receive the friend user
	friends, err := userSv.GetFriends(ctx, userID, params.Query{LookupID: friendID})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(friends))
	assert.Equal(t, friendID, friends[0].ID)

	// Remove friendship and test again
	err = userSv.RemoveFriend(ctx, userID, friendID)
	assert.NoError(t, err)

	friends2, err := userSv.GetFriends(ctx, userID, params.Query{LookupID: friendID})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(friends2))
}

func TestGetBy(t *testing.T) {
	ctx := context.Background()
	id := ulid.NewString()
	username := "username"
	email := "email"

	err := test.CreateUser(ctx, db, dc, id, email, username, "1")
	assert.NoError(t, err)

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
	eventID := ulid.NewString()
	userID := ulid.NewString()

	err := test.CreateEvent(ctx, db, dc, eventID, "TestGetAttendingEvents")
	assert.NoError(t, err)
	err = test.CreateUser(ctx, db, dc, userID, "attending@email.com", "attending", "1")
	assert.NoError(t, err)

	events, err := userSv.GetAttendingEvents(ctx, userID, params.Query{LookupID: eventID})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(events))
	assert.Equal(t, eventID, events[0].ID)
}

func TestGetInvitedEvents(t *testing.T) {
	ctx := context.Background()
	eventID := ulid.NewString()
	userID := ulid.NewString()

	err := test.CreateEvent(ctx, db, dc, eventID, "TestGetInvitedEvents")
	assert.NoError(t, err)
	err = test.CreateUser(ctx, db, dc, userID, "invited@email.com", "invited", "1")
	assert.NoError(t, err)

	invite := user.Invite{
		EventID: eventID,
		UserIDs: []string{userID},
	}
	err = userSv.InviteToEvent(ctx, auth.Session{ID: ulid.NewString()}, invite)
	assert.NoError(t, err)

	events, err := userSv.GetInvitedEvents(ctx, userID, params.Query{LookupID: eventID})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(events))
	assert.Equal(t, eventID, events[0].ID)
}

func TestGetHostedEvents(t *testing.T) {
	ctx := context.Background()
	eventID := ulid.NewString()
	userID := ulid.NewString()

	err := test.CreateUser(ctx, db, dc, userID, "hosted@email.com", "hosted", "1")
	assert.NoError(t, err)

	boolean := false
	createEvent := event.CreateEvent{
		HostID: userID,
		Name:   "TestGetHostedEvents",
		Type:   event.Talk,
		Public: &boolean,
		Slots:  100,
		Cron:   "0 0 * * * 60",
	}
	err = eventSv.Create(ctx, eventID, createEvent)
	assert.NoError(t, err)

	events, err := userSv.GetHostedEvents(ctx, userID, params.Query{LookupID: eventID})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(events))
	assert.Equal(t, eventID, events[0].ID)
}

func TestGetLikedEvents(t *testing.T) {
	ctx := context.Background()
	eventID := ulid.NewString()
	userID := ulid.NewString()

	err := test.CreateEvent(ctx, db, dc, eventID, "TestGetLikedEvents")
	assert.NoError(t, err)
	err = test.CreateUser(ctx, db, dc, userID, "liked@email.com", "liked", "1")
	assert.NoError(t, err)

	err = eventSv.AddEdge(ctx, eventID, dgraph.LikedBy, userID)
	assert.NoError(t, err)

	events, err := userSv.GetLikedEvents(ctx, userID, params.Query{LookupID: eventID})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(events))
	assert.Equal(t, eventID, events[0].ID)
}

func TestIsAdmin(t *testing.T) {
	ctx := context.Background()
	adminID := ulid.NewString()
	nonAdminID := ulid.NewString()

	now := time.Now()
	err := userSv.Create(ctx, adminID, user.CreateUser{
		Name:      "admin",
		Username:  "admin",
		Email:     adminEmail,
		Password:  "1",
		BirthDate: &now,
	})
	assert.NoError(t, err)

	err = test.CreateUser(ctx, db, dc, nonAdminID, "nonadmin@email.com", "nonadmin", "1")
	assert.NoError(t, err)

	ok, err := userSv.IsAdmin(ctx, adminID)
	assert.NoError(t, err)
	assert.True(t, ok)

	ok2, err := userSv.IsAdmin(ctx, nonAdminID)
	assert.NoError(t, err)
	assert.False(t, ok2)
}

func TestPrivateProfile(t *testing.T) {
	ctx := context.Background()
	id := ulid.NewString()

	err := test.CreateUser(ctx, db, dc, id, "private@email.com", "private", "1")
	assert.NoError(t, err)

	ok, err := userSv.PrivateProfile(ctx, id)
	assert.NoError(t, err)
	assert.False(t, ok)

	priv := true
	err = userSv.Update(ctx, id, user.UpdateUser{Private: &priv})
	assert.NoError(t, err)

	ok2, err := userSv.PrivateProfile(ctx, id)
	assert.NoError(t, err)
	assert.True(t, ok2)
}

func TestSearch(t *testing.T) {
	ctx := context.Background()
	userID := ulid.NewString()

	err := test.CreateUser(ctx, db, dc, userID, "search@email.com", "search", "1")
	assert.NoError(t, err)

	users, err := userSv.Search(ctx, "sea", params.Query{})
	assert.NoError(t, err)

	assert.Equal(t, 1, len(users))
	assert.Equal(t, userID, users[0].ID)
}

func TestUpdate(t *testing.T) {
	ctx := context.Background()
	id := ulid.NewString()

	err := test.CreateUser(ctx, db, dc, id, "update@email.com", "update", "1")
	assert.NoError(t, err)

	uptUName := "updatedUsername"
	uptUser := user.UpdateUser{
		Username: &uptUName,
	}
	err = userSv.Update(ctx, id, uptUser)
	assert.NoError(t, err)

	user, err := userSv.GetByID(ctx, id)
	assert.NoError(t, err)

	assert.Equal(t, uptUName, user.Username)
}
