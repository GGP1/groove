package event_test

import (
	"context"
	"database/sql"
	"log"
	"testing"
	"time"

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

	"github.com/go-redis/redis/v8"
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
	test.Main(m, func(s *sql.DB, _ *redis.Client, c cache.Client) {
		db = s
		sqlTx, err := db.BeginTx(context.Background(), nil)
		if err != nil {
			log.Fatal(err)
		}
		_, ctx = txgroup.WithContext(ctx, txgroup.NewSQLTx(sqlTx))
		cacheClient = c

		authService := auth.NewService(db, nil, config.Sessions{})
		roleService = role.NewService(db, cacheClient)
		notifService := notification.NewService(db, config.Notifications{}, authService, roleService)
		eventSv = event.NewService(s, cacheClient, notifService, roleService)
	}, test.Postgres, test.Memcached)
}

func TestBans(t *testing.T) {
	eventID := test.CreateEvent(t, db, "banned")
	userID := test.CreateUser(t, db, "banned@email.com", "banned")

	t.Run("Ban", func(t *testing.T) {
		err := eventSv.Ban(ctx, eventID, userID)
		assert.NoError(t, err)
		ctx = test.CommitTx(ctx, t, db)
	})

	t.Run("IsBanned", func(t *testing.T) {
		isBanned, err := eventSv.IsBanned(ctx, eventID, userID)
		assert.NoError(t, err)
		assert.True(t, isBanned)
	})

	t.Run("GetBanned", func(t *testing.T) {
		users, err := eventSv.GetBanned(ctx, eventID, params.Query{})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(users))
		assert.Equal(t, userID, users[0].ID)
	})

	t.Run("GetBannedCount", func(t *testing.T) {
		count, err := eventSv.GetBannedCount(ctx, eventID)
		assert.NoError(t, err)
		assert.Equal(t, 1, int(count))
	})

	t.Run("RemoveBan", func(t *testing.T) {
		err := eventSv.RemoveBan(ctx, eventID, userID)
		assert.NoError(t, err)
		ctx = test.CommitTx(ctx, t, db)

		isBanned, err := eventSv.IsBanned(ctx, eventID, userID)
		assert.NoError(t, err)
		assert.False(t, isBanned)
	})
}

func TestInvited(t *testing.T) {
	eventID := test.CreateEvent(t, db, "invited")
	userID := test.CreateUser(t, db, "invited@email.com", "invited")

	t.Run("SetRole", func(t *testing.T) {
		err := roleService.SetRole(ctx, eventID, roles.Viewer, userID)
		assert.NoError(t, err)
		ctx = test.CommitTx(ctx, t, db)
	})

	t.Run("IsInvited", func(t *testing.T) {
		isInvited, err := eventSv.IsInvited(ctx, eventID, userID)
		assert.NoError(t, err)
		assert.True(t, isInvited)
	})

	t.Run("GetInvited", func(t *testing.T) {
		users, err := eventSv.GetInvited(ctx, eventID, params.Query{})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(users))
		assert.Equal(t, userID, users[0].ID)
	})

	t.Run("GetInvitedCount", func(t *testing.T) {
		count, err := eventSv.GetInvitedCount(ctx, eventID)
		assert.NoError(t, err)
		assert.Equal(t, 1, int(count))
	})

	t.Run("UnsetRole", func(t *testing.T) {
		err := roleService.UnsetRole(ctx, eventID, userID)
		assert.NoError(t, err)
		ctx = test.CommitTx(ctx, t, db)

		isInvited, err := eventSv.IsInvited(ctx, eventID, userID)
		assert.NoError(t, err)
		assert.False(t, isInvited)
	})
}

func TestLikes(t *testing.T) {
	eventID := test.CreateEvent(t, db, "liked_by")
	userID := test.CreateUser(t, db, "liked_by@email.com", "liked_by")

	t.Run("Like", func(t *testing.T) {
		err := eventSv.Like(ctx, eventID, userID)
		assert.NoError(t, err)
		ctx = test.CommitTx(ctx, t, db)
	})

	t.Run("GetLikes", func(t *testing.T) {
		users, err := eventSv.GetLikes(ctx, eventID, params.Query{})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(users))
		assert.Equal(t, userID, users[0].ID)
	})

	t.Run("GetLikesCount", func(t *testing.T) {
		count, err := eventSv.GetLikesCount(ctx, eventID)
		assert.NoError(t, err)
		assert.Equal(t, 1, int(count))
	})

	t.Run("RemoveLike", func(t *testing.T) {
		err := eventSv.RemoveLike(ctx, eventID, userID)
		assert.NoError(t, err)
		ctx = test.CommitTx(ctx, t, db)

		users, err := eventSv.GetLikes(ctx, eventID, params.Query{})
		assert.NoError(t, err)
		assert.Equal(t, 0, len(users))
	})
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
	ctx = test.CommitTx(ctx, t, db)

	_, err = eventSv.GetByID(ctx, eventID)
	assert.NoError(t, err)
}

func TestDelete(t *testing.T) {
	eventID := test.CreateEvent(t, db, "delete")

	err := eventSv.Delete(ctx, eventID)
	assert.NoError(t, err)
	ctx = test.CommitTx(ctx, t, db)

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

	err := roleService.SetRole(ctx, eventID, roles.Host, userID)
	assert.NoError(t, err)
	ctx = test.CommitTx(ctx, t, db)

	users, err := eventSv.GetHosts(ctx, eventID, params.Query{LookupID: userID})
	assert.NoError(t, err)

	assert.Equal(t, 1, len(users))
	assert.Equal(t, email, users[0].Email)
}

func TestGetInvitedFriends(t *testing.T) {

}

func TestGetLikedByFriends(t *testing.T) {

}

func TestGetRecommended(t *testing.T) {
	eventID1 := createEventWithLocation(t, db, model.Coordinates{Latitude: 11, Longitude: 15})
	eventID2 := createEventWithLocation(t, db, model.Coordinates{Latitude: 100, Longitude: 50})
	eventID3 := createEventWithLocation(t, db, model.Coordinates{Latitude: 12.5, Longitude: 16})
	eventID4 := createEventWithLocation(t, db, model.Coordinates{Latitude: 120, Longitude: 80})
	eventID5 := createEventWithLocation(t, db, model.Coordinates{Latitude: -38, Longitude: -20})
	userID := test.CreateUser(t, db, "user@mail.com", "username")

	userCoordinates := model.Coordinates{
		Latitude:  12,
		Longitude: 15,
	}
	events, err := eventSv.GetRecommended(ctx, userID, userCoordinates, params.Query{})
	assert.NoError(t, err)

	expectedNum := 2
	assert.Equal(t, expectedNum, len(events))

	eventIDs := make(map[string]struct{}, expectedNum)
	for _, event := range events {
		eventIDs[event.ID] = struct{}{}
	}

	assert.NotNil(t, eventIDs[eventID1])
	assert.Nil(t, eventIDs[eventID2])
	assert.NotNil(t, eventIDs[eventID3])
	assert.Nil(t, eventIDs[eventID4])
	assert.Nil(t, eventIDs[eventID5])
}

func TestGetStatistics(t *testing.T) {
	eventID := test.CreateEvent(t, db, "stats")
	userID := test.CreateUser(t, db, "user@mail.com", "username")

	stats, err := eventSv.GetStatistics(ctx, eventID)
	assert.NoError(t, err)
	assert.Equal(t, model.EventStatistics{}, stats)

	err = eventSv.Like(ctx, eventID, userID)
	assert.NoError(t, err)
	ctx = test.CommitTx(ctx, t, db)

	stats2, err := eventSv.GetStatistics(ctx, eventID)
	assert.NoError(t, err)
	expectedStats := model.EventStatistics{Likes: 1}
	assert.Equal(t, expectedStats, stats2)
}

func TestIsPublic(t *testing.T) {
	eventID := test.CreateEvent(t, db, "reports")

	got, err := eventSv.IsPublic(ctx, eventID)
	assert.NoError(t, err)

	assert.Equal(t, true, got)
}

func TestPrivacyFilter(t *testing.T) {
	eventID := test.CreateEvent(t, db, "public")
	userID := test.CreateUser(t, db, "random@mail.com", "username")

	err := eventSv.PrivacyFilter(ctx, eventID, userID)
	assert.NoError(t, err)
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

func createEventWithLocation(t testing.TB, db *sql.DB, coordinates model.Coordinates) string {
	ctx := context.Background()
	id := ulid.NewString()
	q := `INSERT INTO events 
	(id, name, type, public, virtual, slots, cron, start_date, end_date, ticket_type, latitude, longitude) 
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`
	_, err := db.ExecContext(ctx, q,
		id, "name", model.GrandPrix, true, false, 100, "30 12 * * * 15", time.Now(),
		time.Now().Add(time.Hour*2400), 1, coordinates.Latitude, coordinates.Longitude)
	assert.NoError(t, err)

	return id
}
