package user_test

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"
	"time"

	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/service/event"
	"github.com/GGP1/groove/service/user"
	"github.com/GGP1/groove/test"
	"github.com/bradfitz/gomemcache/memcache"
	"github.com/dgraph-io/dgo/v210"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

var (
	userSv  user.Service
	eventSv event.Service
	db      *sqlx.DB
	dc      *dgo.Dgraph
	mc      *memcache.Client
)

const adminEmail = "admin@email.com"

// Note: each of the test functions creates users to test but does cleanup them to save time, be sure to create
// unique users for each one of them

func TestMain(m *testing.M) {
	poolPg, resourcePg, postgres, err := test.RunPostgres()
	if err != nil {
		log.Fatal(err)
	}
	poolDc, resourceDc, dgraph, conn, err := test.RunDgraph()
	if err != nil {
		log.Fatal(err)
	}
	poolMc, resourceMc, memcached, err := test.RunMemcached()
	if err != nil {
		log.Fatal(err)
	}

	db = postgres
	dc = dgraph
	mc = memcached
	userSv = user.NewService(db, dc, mc, map[string]interface{}{adminEmail: struct{}{}})
	eventSv = event.NewService(db, dc, mc)

	code := m.Run()

	if err := poolPg.Purge(resourcePg); err != nil {
		log.Fatal(err)
	}
	if err := poolDc.Purge(resourceDc); err != nil {
		log.Fatal(err)
	}
	if err := conn.Close(); err != nil {
		log.Fatal(err)
	}
	if err := poolMc.Purge(resourceMc); err != nil {
		log.Fatal(err)
	}

	os.Exit(code)
}

func TestBlock(t *testing.T) {
	ctx := context.Background()
	userID := uuid.NewString()
	blockedID := uuid.NewString()

	err := test.CreateUser(ctx, db, userID, "block1@email.com", "block1", "1")
	assert.NoError(t, err)
	err = test.CreateUser(ctx, db, blockedID, "block2@email.com", "block2", "2")
	assert.NoError(t, err)

	err = userSv.Block(context.Background(), userID, blockedID)
	assert.NoError(t, err)

	// Test if we receive the blocked user
	blocked, err := userSv.GetBlocked(ctx, userID, params.Query{LookupID: blockedID})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(blocked))
	assert.Equal(t, blockedID, blocked[0].ID)

	// Test if we receive the blocker user
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

func TestCreate(t *testing.T) {
	ctx := context.Background()
	id := uuid.NewString()
	tm := time.Now()
	createUser := user.CreateUser{
		Name:      "Create",
		Username:  "Create",
		Email:     "create@email.com",
		Password:  "1",
		BirthDate: &tm,
	}
	err := userSv.Create(ctx, id, createUser)
	assert.NoError(t, err)

	user, err := userSv.GetByUsername(ctx, createUser.Username)
	assert.NoError(t, err)

	assert.Equal(t, id, user.ID)
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	id := uuid.NewString()
	err := test.CreateUser(ctx, db, id, "delete@email.com", "delete", "1")
	assert.NoError(t, err)

	err = userSv.Delete(ctx, id)
	assert.NoError(t, err)

	_, err = userSv.GetByID(ctx, id)
	assert.Error(t, err)
}

func TestFollow(t *testing.T) {
	ctx := context.Background()
	userID := uuid.NewString()
	followedID := uuid.NewString()

	err := test.CreateUser(ctx, db, userID, "follow1@email.com", "follow1", "1")
	assert.NoError(t, err)
	err = test.CreateUser(ctx, db, followedID, "follow2@email.com", "follow2", "2")
	assert.NoError(t, err)

	err = userSv.Follow(context.Background(), userID, followedID)
	assert.NoError(t, err)

	// Test if we receive the followed user
	following, err := userSv.GetFollowing(ctx, userID, params.Query{LookupID: followedID})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(following))
	assert.Equal(t, followedID, following[0].ID)

	// Test if we receive the follower user
	followers, err := userSv.GetFollowers(ctx, followedID, params.Query{LookupID: userID})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(followers))
	assert.Equal(t, userID, followers[0].ID)

	// Remove follow and test again
	err = userSv.Unfollow(ctx, userID, followedID)
	assert.NoError(t, err)

	following2, err := userSv.GetFollowing(ctx, userID, params.Query{LookupID: followedID})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(following2))

	followers2, err := userSv.GetFollowers(ctx, followedID, params.Query{LookupID: userID})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(followers2))
}

func TestGetBy(t *testing.T) {
	ctx := context.Background()
	id := uuid.NewString()
	username := "username"
	email := "email"

	err := test.CreateUser(ctx, db, id, email, username, "1")
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

func TestGetConfirmedEvents(t *testing.T) {
	ctx := context.Background()
	eventID := uuid.NewString()
	userID := uuid.NewString()

	err := test.CreateEvent(ctx, db, eventID, "TestGetConfirmedEvents")
	assert.NoError(t, err)
	err = test.CreateUser(ctx, db, userID, "confirmed@email.com", "confirmed", "1")
	assert.NoError(t, err)

	err = eventSv.AddEdge(ctx, eventID, event.Confirmed, userID)
	assert.NoError(t, err)

	events, err := userSv.GetConfirmedEvents(ctx, userID, params.Query{LookupID: eventID})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(events))
	assert.Equal(t, eventID, events[0].ID)
}

func TestGetInvitedEvents(t *testing.T) {
	ctx := context.Background()
	eventID := uuid.NewString()
	userID := uuid.NewString()

	err := test.CreateEvent(ctx, db, eventID, "TestGetInvitedEvents")
	assert.NoError(t, err)
	err = test.CreateUser(ctx, db, userID, "invited@email.com", "invited", "1")
	assert.NoError(t, err)

	err = eventSv.AddEdge(ctx, eventID, event.Invited, userID)
	assert.NoError(t, err)

	events, err := userSv.GetInvitedEvents(ctx, userID, params.Query{LookupID: eventID})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(events))
	assert.Equal(t, eventID, events[0].ID)
}

func TestGetHostedEvents(t *testing.T) {
	ctx := context.Background()
	eventID := uuid.NewString()
	userID := uuid.NewString()

	err := test.CreateUser(ctx, db, userID, "hosted@email.com", "hosted", "1")
	assert.NoError(t, err)

	createEvent := event.CreateEvent{
		CreatorID: userID,
		Name:      "TestGetHostedEvents",
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
	eventID := uuid.NewString()
	userID := uuid.NewString()

	err := test.CreateEvent(ctx, db, eventID, "TestGetLikedEvents")
	assert.NoError(t, err)
	err = test.CreateUser(ctx, db, userID, "liked@email.com", "liked", "1")
	assert.NoError(t, err)

	err = eventSv.AddEdge(ctx, eventID, event.LikedBy, userID)
	assert.NoError(t, err)

	events, err := userSv.GetLikedEvents(ctx, userID, params.Query{LookupID: eventID})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(events))
	assert.Equal(t, eventID, events[0].ID)
}

func TestIsAdmin(t *testing.T) {
	ctx := context.Background()
	adminID := uuid.NewString()
	nonAdminID := uuid.NewString()

	err := test.CreateUser(ctx, db, adminID, adminEmail, "admin", "1")
	assert.NoError(t, err)
	err = test.CreateUser(ctx, db, nonAdminID, "nonadmin@email.com", "nonadmin", "1")
	assert.NoError(t, err)

	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	assert.NoError(t, err)
	defer tx.Rollback()

	ok, err := userSv.IsAdmin(ctx, tx, adminID)
	assert.NoError(t, err)
	assert.True(t, ok)

	ok2, err := userSv.IsAdmin(ctx, tx, nonAdminID)
	assert.NoError(t, err)
	assert.False(t, ok2)
}

func TestPrivateProfile(t *testing.T) {
	ctx := context.Background()
	id := uuid.NewString()

	err := test.CreateUser(ctx, db, id, "private@email.com", "private", "1")
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
	// TODO
}

func TestUpdate(t *testing.T) {
	ctx := context.Background()
	id := uuid.NewString()

	err := test.CreateUser(ctx, db, id, "update@email.com", "update", "1")
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
