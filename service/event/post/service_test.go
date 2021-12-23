package post_test

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"

	"github.com/GGP1/groove/config"
	"github.com/GGP1/groove/internal/cache"
	"github.com/GGP1/groove/internal/ulid"
	"github.com/GGP1/groove/model"
	"github.com/GGP1/groove/service/auth"
	"github.com/GGP1/groove/service/event/post"
	"github.com/GGP1/groove/service/event/role"
	"github.com/GGP1/groove/service/notification"
	"github.com/GGP1/groove/storage/postgres"
	"github.com/GGP1/groove/test"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

var (
	postSv      post.Service
	db          *sql.DB
	cacheClient cache.Client
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

	cacheClient = memcached
	db = postgres

	authService := auth.NewService(postgres, nil, config.Sessions{})
	roleService := role.NewService(postgres, cacheClient)
	notifService := notification.NewService(postgres, config.Notifications{}, authService, roleService)
	postSv = post.NewService(postgres, cacheClient, notifService)

	code := m.Run()

	if err := pgContainer.Close(); err != nil {
		log.Fatal(err)
	}
	if err := mcContainer.Close(); err != nil {
		log.Fatal(err)
	}

	os.Exit(code)
}

func TestCreateComment(t *testing.T) {
	sqlTx, ctx := postgres.BeginTx(context.Background(), db)
	userID := test.CreateUser(t, db, "random@email.test", "test")

	session := auth.Session{
		ID: userID,
	}
	comment := model.CreateComment{
		Content: "post comment",
	}
	_, err := postSv.CreateComment(ctx, session, comment)
	assert.NoError(t, err)

	assert.NoError(t, sqlTx.Commit())
}

func TestCreatePost(t *testing.T) {
	sqlTx, ctx := postgres.BeginTx(context.Background(), db)

	eventID := test.CreateEvent(t, db, "create_post")

	session := auth.Session{ID: ulid.NewString()}
	post := model.CreatePost{
		Content: "post content",
		Media:   pq.StringArray{"create_post.com/images/a.jpg"},
	}
	_, err := postSv.CreatePost(ctx, session, eventID, post)
	assert.NoError(t, err)

	assert.NoError(t, sqlTx.Commit())
}

func TestPost(t *testing.T) {

}

func TestComment(t *testing.T) {

}
