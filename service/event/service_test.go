package event_test

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"
	"time"

	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/ulid"
	"github.com/GGP1/groove/service/event"
	"github.com/GGP1/groove/service/event/role"
	"github.com/GGP1/groove/service/user"
	"github.com/GGP1/groove/test"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/dgraph-io/dgo/v210"
	"github.com/stretchr/testify/assert"
)

var (
	userSv  user.Service
	eventSv event.Service
	db      *sql.DB
	sqlTx   *sql.Tx
	dc      *dgo.Dgraph
	mc      *memcache.Client
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
	sqlTx, err = db.BeginTx(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}
	dc = dgraph
	mc = memcached

	eventSv = event.NewService(postgres, dgraph, memcached)

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
	ctx := context.Background()
	eventID := ulid.NewString()
	userID := ulid.NewString()

	err := test.CreateEvent(ctx, db, dc, eventID, "banned")
	assert.NoError(t, err)
	err = test.CreateUser(ctx, db, dc, userID, "banned@email.com", "banned", "1")
	assert.NoError(t, err)

	err = eventSv.AddEdge(ctx, eventID, event.Banned, userID)
	assert.NoError(t, err)

	users, err := eventSv.GetBanned(ctx, sqlTx, eventID, params.Query{LookupID: userID})
	assert.NoError(t, err)

	count, err := eventSv.GetBannedCount(ctx, eventID)
	assert.NoError(t, err)

	assert.Equal(t, *count, uint64(len(users)))
	assert.Equal(t, userID, users[0].ID)

	err = eventSv.RemoveEdge(ctx, eventID, event.Banned, userID)
	assert.NoError(t, err)
}

func TestConfirmed(t *testing.T) {
	ctx := context.Background()
	eventID := ulid.NewString()
	userID := ulid.NewString()

	err := test.CreateEvent(ctx, db, dc, eventID, "confirmed")
	assert.NoError(t, err)
	err = test.CreateUser(ctx, db, dc, userID, "confirmed@email.com", "confirmed", "1")
	assert.NoError(t, err)

	err = eventSv.AddEdge(ctx, eventID, event.Confirmed, userID)
	assert.NoError(t, err)

	users, err := eventSv.GetConfirmed(ctx, sqlTx, eventID, params.Query{LookupID: userID})
	assert.NoError(t, err)

	count, err := eventSv.GetConfirmedCount(ctx, eventID)
	assert.NoError(t, err)

	assert.Equal(t, *count, uint64(len(users)))
	assert.Equal(t, userID, users[0].ID)

	err = eventSv.RemoveEdge(ctx, eventID, event.Confirmed, userID)
	assert.NoError(t, err)
}

func TestInvited(t *testing.T) {
	ctx := context.Background()
	eventID := ulid.NewString()
	userID := ulid.NewString()

	err := test.CreateEvent(ctx, db, dc, eventID, "invited")
	assert.NoError(t, err)
	err = test.CreateUser(ctx, db, dc, userID, "invited@email.com", "invited", "1")
	assert.NoError(t, err)

	err = eventSv.AddEdge(ctx, eventID, event.Invited, userID)
	assert.NoError(t, err)

	users, err := eventSv.GetInvited(ctx, sqlTx, eventID, params.Query{LookupID: userID})
	assert.NoError(t, err)

	count, err := eventSv.GetInvitedCount(ctx, eventID)
	assert.NoError(t, err)

	assert.Equal(t, *count, uint64(len(users)))
	assert.Equal(t, userID, users[0].ID)

	err = eventSv.RemoveEdge(ctx, eventID, event.Invited, userID)
	assert.NoError(t, err)
}

func TestLikedBy(t *testing.T) {
	ctx := context.Background()
	eventID := ulid.NewString()
	userID := ulid.NewString()

	err := test.CreateEvent(ctx, db, dc, eventID, "liked_by")
	assert.NoError(t, err)
	err = test.CreateUser(ctx, db, dc, userID, "liked_by@email.com", "liked_by", "1")
	assert.NoError(t, err)

	err = eventSv.AddEdge(ctx, eventID, event.LikedBy, userID)
	assert.NoError(t, err)

	users, err := eventSv.GetLikedBy(ctx, sqlTx, eventID, params.Query{LookupID: userID})
	assert.NoError(t, err)

	count, err := eventSv.GetLikedByCount(ctx, eventID)
	assert.NoError(t, err)

	assert.Equal(t, *count, uint64(len(users)))
	assert.Equal(t, userID, users[0].ID)

	err = eventSv.RemoveEdge(ctx, eventID, event.LikedBy, userID)
	assert.NoError(t, err)
}

func TestCanInvite(t *testing.T) {
	ctx := context.Background()
	userID := ulid.NewString()
	invitedID := ulid.NewString()

	err := test.CreateUser(ctx, db, dc, userID, "can_invite@email.com", "can_invite", "1")
	assert.NoError(t, err)
	err = test.CreateUser(ctx, db, dc, invitedID, "can_invite2@email.com", "can_invite2", "1")
	assert.NoError(t, err)

	// TODO: test scenarios where the invited user has nobody and mutual_follow settings
	ok, err := eventSv.CanInvite(ctx, sqlTx, userID, invitedID)
	assert.NoError(t, err)

	assert.True(t, ok)
}

func TestCreate(t *testing.T) {
	ctx := context.Background()
	eventID := ulid.NewString()
	creatorID := ulid.NewString()

	err := test.CreateUser(ctx, db, dc, creatorID, "create@email.com", "create", "1")
	assert.NoError(t, err)

	boolean := false
	createEvent := event.CreateEvent{
		HostID:     creatorID,
		Name:       "Create",
		Type:       event.Ceremony,
		Public:     &boolean,
		StartTime:  uint64(time.Now().Unix()),
		EndTime:    uint64(time.Now().Unix() + 1500),
		MinAge:     18,
		Slots:      200,
		TicketCost: 150,
	}
	err = eventSv.Create(ctx, eventID, createEvent)
	assert.NoError(t, err)

	_, err = eventSv.GetByID(ctx, sqlTx, eventID)
	assert.NoError(t, err)
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	eventID := ulid.NewString()

	err := test.CreateEvent(ctx, db, dc, eventID, "delete")
	assert.NoError(t, err)

	err = eventSv.Delete(ctx, sqlTx, eventID)
	assert.NoError(t, err)

	_, err = eventSv.GetByID(ctx, sqlTx, eventID)
	assert.Error(t, err)
}

func TestGetByID(t *testing.T) {
	ctx := context.Background()
	eventID := ulid.NewString()

	name := "get_by_id"
	err := test.CreateEvent(ctx, db, dc, eventID, name)
	assert.NoError(t, err)

	event, err := eventSv.GetByID(ctx, sqlTx, eventID)
	assert.NoError(t, err)

	assert.Equal(t, name, event.Name)
}

func TestGetBannedFollowing(t *testing.T) {

}

func TestGetConfirmedFollowing(t *testing.T) {

}

func TestGetHosts(t *testing.T) {
	ctx := context.Background()
	eventID := ulid.NewString()
	userID := ulid.NewString()

	email := "host@email.com"
	err := test.CreateUser(ctx, db, dc, userID, email, "host", "1")
	assert.NoError(t, err)

	err = test.CreateEvent(ctx, db, dc, eventID, "hosts")
	assert.NoError(t, err)

	err = eventSv.SetRoles(ctx, sqlTx, eventID, role.Host, userID)
	assert.NoError(t, err)

	users, err := eventSv.GetHosts(ctx, sqlTx, eventID, params.Query{LookupID: userID})
	assert.NoError(t, err)

	assert.Equal(t, 1, len(users))
	assert.Equal(t, email, users[0].Email)
}

func TestGetInvitedFollowing(t *testing.T) {

}

func TestGetLikedByFollowing(t *testing.T) {

}

func TestIsPublic(t *testing.T) {
	ctx := context.Background()
	eventID := ulid.NewString()

	err := test.CreateEvent(ctx, db, dc, eventID, "reports")
	assert.NoError(t, err)

	got, err := eventSv.IsPublic(ctx, sqlTx, eventID)
	assert.NoError(t, err)

	assert.Equal(t, true, got)
}

func TestSearch(t *testing.T) {}

func TestPgTx(t *testing.T) {
	assert.NotPanics(t, func() {
		tx := eventSv.BeginSQLTx(context.Background(), true)
		assert.NoError(t, tx.Rollback())
	})
}

func TestUpdate(t *testing.T) {
	ctx := context.Background()
	eventID := ulid.NewString()

	err := test.CreateEvent(ctx, db, dc, eventID, "update")
	assert.NoError(t, err)

	name := "update_updated"
	updateEvent := event.UpdateEvent{
		Name: &name,
	}
	err = eventSv.Update(ctx, sqlTx, eventID, updateEvent)
	assert.NoError(t, err)

	event, err := eventSv.GetByID(ctx, sqlTx, eventID)
	assert.NoError(t, err)

	assert.Equal(t, name, event.Name)
}
