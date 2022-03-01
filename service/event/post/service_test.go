package post_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/GGP1/groove/config"
	"github.com/GGP1/groove/internal/ulid"
	"github.com/GGP1/groove/model"
	"github.com/GGP1/groove/service/auth"
	"github.com/GGP1/groove/service/event/post"
	"github.com/GGP1/groove/service/event/role"
	"github.com/GGP1/groove/service/notification"
	"github.com/GGP1/groove/storage/postgres"
	"github.com/GGP1/groove/test"

	"github.com/go-redis/redis/v8"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

var (
	postSv post.Service
	db     *sql.DB
)

func TestMain(m *testing.M) {
	test.Main(
		m,
		func(s *sql.DB, r *redis.Client) {
			db = s

			authService := auth.NewService(db, nil, config.Sessions{})
			roleService := role.NewService(db, r)
			notifService := notification.NewService(db, config.Notifications{}, authService, roleService)
			postSv = post.NewService(db, r, notifService)
		},
		test.Postgres, test.Redis,
	)
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

func TestPosts(t *testing.T) {

}

func TestComments(t *testing.T) {

}
