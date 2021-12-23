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
	"github.com/GGP1/groove/internal/txgroup"
	"github.com/GGP1/groove/internal/ulid"
	"github.com/GGP1/groove/model"
	"github.com/GGP1/groove/service/auth"
	"github.com/GGP1/groove/service/event"
	"github.com/GGP1/groove/service/event/role"
	"github.com/GGP1/groove/service/notification"
	"github.com/GGP1/groove/service/user"
	"github.com/GGP1/groove/test"

	"github.com/stretchr/testify/assert"
)

var (
	userSv      user.Service
	eventSv     event.Service
	db          *sql.DB
	ctx         context.Context
	cacheClient cache.Client
	roleService role.Service
)

func TestMain(m *testing.M) {
	pgContainer, postgres, err := test.RunPostgres()
	if err != nil {
		log.Fatal(err)
	}
	mcContainer, memcached, err := test.RunMemcached()
	if err != nil {
		log.Fatal(err)
	}

	db = postgres
	sqlTx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}
	_, ctx = txgroup.WithContext(ctx, txgroup.NewSQLTx(sqlTx))
	cacheClient = memcached

	authService := auth.NewService(db, nil, config.Sessions{})
	roleService = role.NewService(db, cacheClient)
	notifService := notification.NewService(db, config.Notifications{}, authService, roleService)
	eventSv = event.NewService(postgres, cacheClient, notifService, roleService)

	code := m.Run()

	if err := sqlTx.Rollback(); err != nil {
		log.Fatal(err)
	}
	if err := pgContainer.Close(); err != nil {
		log.Fatal(err)
	}
	if err := mcContainer.Close(); err != nil {
		log.Fatal(err)
	}

	os.Exit(code)
}

func TestBans(t *testing.T) {
	eventID := test.CreateEvent(t, db, "banned")
	userID := test.CreateUser(t, db, "banned@email.com", "banned")

	err := eventSv.Ban(ctx, eventID, userID)
	assert.NoError(t, err)

	users, err := eventSv.GetBanned(ctx, eventID, params.Query{LookupID: userID})
	assert.NoError(t, err)

	count, err := eventSv.GetBannedCount(ctx, eventID)
	assert.NoError(t, err)

	assert.Equal(t, count, len(users))
	assert.Equal(t, userID, users[0].ID)

	err = eventSv.RemoveBan(ctx, eventID, userID)
	assert.NoError(t, err)
}

func TestInvited(t *testing.T) {
	eventID := test.CreateEvent(t, db, "invited")
	userID := test.CreateUser(t, db, "invited@email.com", "invited")

	err := roleService.SetReservedRole(ctx, eventID, userID, roles.Viewer)
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

func TestLikes(t *testing.T) {
	eventID := test.CreateEvent(t, db, "liked_by")
	userID := test.CreateUser(t, db, "liked_by@email.com", "liked_by")

	err := eventSv.Like(ctx, eventID, userID)
	assert.NoError(t, err)

	users, err := eventSv.GetLikes(ctx, eventID, params.Query{LookupID: userID})
	assert.NoError(t, err)

	count, err := eventSv.GetLikesCount(ctx, eventID)
	assert.NoError(t, err)

	assert.Equal(t, count, len(users))
	assert.Equal(t, userID, users[0].ID)

	err = eventSv.RemoveLike(ctx, eventID, userID)
	assert.NoError(t, err)
}

func TestCreate(t *testing.T) {
	creatorID := test.CreateUser(t, db, "create@email.com", "create")

	boolean := false
	eventID := ulid.NewString()
	createEvent := model.CreateEvent{
		HostID: creatorID,
		Name:   "Create",
		Type:   model.Ceremony,
		Public: &boolean,
		Cron:   "0 0 * * * 60",
		MinAge: 18,
		Slots:  200,
	}
	eventID, err := eventSv.Create(ctx, createEvent)
	assert.NoError(t, err)

	_, err = eventSv.GetByID(ctx, eventID)
	assert.NoError(t, err)
}

func TestDelete(t *testing.T) {
	eventID := test.CreateEvent(t, db, "delete")

	err := eventSv.Delete(ctx, eventID)
	assert.NoError(t, err)

	_, err = eventSv.GetByID(ctx, eventID)
	assert.Error(t, err)
}

func TestGetBannedFriends(t *testing.T) {

}

func TestGetByID(t *testing.T) {
	name := "get_by_id"
	eventID := test.CreateEvent(t, db, name)

	event, err := eventSv.GetByID(ctx, eventID)
	assert.NoError(t, err)

	assert.Equal(t, name, event.Name)
}

func TestGetHosts(t *testing.T) {
	email := "host@email.com"
	userID := test.CreateUser(t, db, email, "host")

	eventID := test.CreateEvent(t, db, "hosts")

	err := roleService.SetReservedRole(ctx, eventID, userID, roles.Host)
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
	eventID := test.CreateEvent(t, db, "reports")

	got, err := eventSv.IsPublic(ctx, eventID)
	assert.NoError(t, err)

	assert.Equal(t, true, got)
}

func TestSearch(t *testing.T) {
	eventID := test.CreateEvent(t, db, "search")

	events, err := eventSv.Search(ctx, "sea", ulid.NewString(), params.Query{})
	assert.NoError(t, err)

	assert.Equal(t, 1, len(events))
	assert.Equal(t, events[0].ID, eventID)
}

func TestUpdate(t *testing.T) {
	eventID := test.CreateEvent(t, db, "update")

	name := "update_updated"
	updateEvent := model.UpdateEvent{
		Name: &name,
	}
	err := eventSv.Update(ctx, eventID, updateEvent)
	assert.NoError(t, err)

	event, err := eventSv.GetByID(ctx, eventID)
	assert.NoError(t, err)

	assert.Equal(t, name, event.Name)
}
