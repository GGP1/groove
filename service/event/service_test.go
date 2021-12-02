package event_test

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"

	"github.com/GGP1/groove/config"
	"github.com/GGP1/groove/internal/cache"
	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/roles"
	"github.com/GGP1/groove/internal/sqltx"
	"github.com/GGP1/groove/internal/ulid"
	"github.com/GGP1/groove/service/auth"
	"github.com/GGP1/groove/service/event"
	"github.com/GGP1/groove/service/event/role"
	"github.com/GGP1/groove/service/notification"
	"github.com/GGP1/groove/service/user"
	"github.com/GGP1/groove/storage/dgraph"
	"github.com/GGP1/groove/test"

	"github.com/dgraph-io/dgo/v210"
	"github.com/stretchr/testify/assert"
)

var (
	userSv      user.Service
	eventSv     event.Service
	db          *sql.DB
	ctx         context.Context
	dc          *dgo.Dgraph
	cacheClient cache.Client
	roleService role.Service
)

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
	sqlTx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}
	ctx = sqltx.NewContext(ctx, sqlTx)
	dc = dgraph
	cacheClient = memcached

	authService := auth.NewService(db, nil, config.Sessions{})
	roleService = role.NewService(db, dc, cacheClient)
	notifService := notification.NewService(db, dc, config.Notifications{}, authService, roleService)
	eventSv = event.NewService(postgres, dgraph, cacheClient, notifService, roleService)

	code := m.Run()

	if err := sqlTx.Rollback(); err != nil {
		log.Fatal(err)
	}

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

func TestBanned(t *testing.T) {
	eventID := ulid.NewString()
	userID := ulid.NewString()

	err := test.CreateEvent(ctx, db, dc, eventID, "banned")
	assert.NoError(t, err)
	err = test.CreateUser(ctx, db, dc, userID, "banned@email.com", "banned", "1")
	assert.NoError(t, err)

	err = eventSv.AddEdge(ctx, eventID, dgraph.Banned, userID)
	assert.NoError(t, err)

	users, err := eventSv.GetBanned(ctx, eventID, params.Query{LookupID: userID})
	assert.NoError(t, err)

	count, err := eventSv.GetBannedCount(ctx, eventID)
	assert.NoError(t, err)

	assert.Equal(t, *count, uint64(len(users)))
	assert.Equal(t, userID, users[0].ID)

	err = eventSv.RemoveEdge(ctx, eventID, dgraph.Banned, userID)
	assert.NoError(t, err)
}

func TestInvited(t *testing.T) {
	eventID := ulid.NewString()
	userID := ulid.NewString()

	err := test.CreateEvent(ctx, db, dc, eventID, "invited")
	assert.NoError(t, err)
	err = test.CreateUser(ctx, db, dc, userID, "invited@email.com", "invited", "1")
	assert.NoError(t, err)

	err = roleService.SetReservedRole(ctx, eventID, userID, roles.Viewer)
	assert.NoError(t, err)

	users, err := eventSv.GetInvited(ctx, eventID, params.Query{LookupID: userID})
	assert.NoError(t, err)

	count, err := eventSv.GetInvitedCount(ctx, eventID)
	assert.NoError(t, err)

	assert.Equal(t, count, int64(len(users)))
	assert.Equal(t, userID, users[0].ID)

	err = roleService.UnsetRole(ctx, eventID, userID)
	assert.NoError(t, err)
}

func TestLikedBy(t *testing.T) {
	eventID := ulid.NewString()
	userID := ulid.NewString()

	err := test.CreateEvent(ctx, db, dc, eventID, "liked_by")
	assert.NoError(t, err)
	err = test.CreateUser(ctx, db, dc, userID, "liked_by@email.com", "liked_by", "1")
	assert.NoError(t, err)

	err = eventSv.AddEdge(ctx, eventID, dgraph.LikedBy, userID)
	assert.NoError(t, err)

	users, err := eventSv.GetLikedBy(ctx, eventID, params.Query{LookupID: userID})
	assert.NoError(t, err)

	count, err := eventSv.GetLikedByCount(ctx, eventID)
	assert.NoError(t, err)

	assert.Equal(t, *count, uint64(len(users)))
	assert.Equal(t, userID, users[0].ID)

	err = eventSv.RemoveEdge(ctx, eventID, dgraph.LikedBy, userID)
	assert.NoError(t, err)
}

func TestCreate(t *testing.T) {
	eventID := ulid.NewString()
	creatorID := ulid.NewString()

	err := test.CreateUser(ctx, db, dc, creatorID, "create@email.com", "create", "1")
	assert.NoError(t, err)

	boolean := false
	createEvent := event.CreateEvent{
		HostID: creatorID,
		Name:   "Create",
		Type:   event.Ceremony,
		Public: &boolean,
		Cron:   "0 0 * * * 60",
		MinAge: 18,
		Slots:  200,
	}
	err = eventSv.Create(ctx, eventID, createEvent)
	assert.NoError(t, err)

	_, err = eventSv.GetByID(ctx, eventID)
	assert.NoError(t, err)
}

func TestDelete(t *testing.T) {
	eventID := ulid.NewString()

	err := test.CreateEvent(ctx, db, dc, eventID, "delete")
	assert.NoError(t, err)

	err = eventSv.Delete(ctx, eventID)
	assert.NoError(t, err)

	_, err = eventSv.GetByID(ctx, eventID)
	assert.Error(t, err)
}

func TestGetBannedFriends(t *testing.T) {

}

func TestGetByID(t *testing.T) {
	eventID := ulid.NewString()

	name := "get_by_id"
	err := test.CreateEvent(ctx, db, dc, eventID, name)
	assert.NoError(t, err)

	event, err := eventSv.GetByID(ctx, eventID)
	assert.NoError(t, err)

	assert.Equal(t, name, event.Name)
}

func TestGetHosts(t *testing.T) {
	eventID := ulid.NewString()
	userID := ulid.NewString()

	email := "host@email.com"
	err := test.CreateUser(ctx, db, dc, userID, email, "host", "1")
	assert.NoError(t, err)

	err = test.CreateEvent(ctx, db, dc, eventID, "hosts")
	assert.NoError(t, err)

	err = roleService.SetReservedRole(ctx, eventID, userID, roles.Host)
	assert.NoError(t, err)

	users, err := eventSv.GetHosts(ctx, eventID, params.Query{LookupID: userID})
	assert.NoError(t, err)

	assert.Equal(t, 1, len(users))
	assert.Equal(t, email, users[0].Email)
}

func TestGetInvitedFriends(t *testing.T) {

}

func TestGetLikedByFriends(t *testing.T) {

}

func TestGetMembers(t *testing.T) {

}

func TestGetMembersFriends(t *testing.T) {

}

func TestIsPublic(t *testing.T) {
	eventID := ulid.NewString()

	err := test.CreateEvent(ctx, db, dc, eventID, "reports")
	assert.NoError(t, err)

	got, err := eventSv.IsPublic(ctx, eventID)
	assert.NoError(t, err)

	assert.Equal(t, true, got)
}

func TestSearch(t *testing.T) {
	eventID := ulid.NewString()

	err := test.CreateEvent(ctx, db, dc, eventID, "search")
	assert.NoError(t, err)

	events, err := eventSv.Search(ctx, "sea", auth.Session{ID: ulid.NewString()}, params.Query{})
	assert.NoError(t, err)

	assert.Equal(t, 1, len(events))
	assert.Equal(t, events[0].ID, eventID)
}

func TestUpdate(t *testing.T) {
	eventID := ulid.NewString()

	err := test.CreateEvent(ctx, db, dc, eventID, "update")
	assert.NoError(t, err)

	name := "update_updated"
	updateEvent := event.UpdateEvent{
		Name: &name,
	}
	err = eventSv.Update(ctx, eventID, updateEvent)
	assert.NoError(t, err)

	event, err := eventSv.GetByID(ctx, eventID)
	assert.NoError(t, err)

	assert.Equal(t, name, event.Name)
}
