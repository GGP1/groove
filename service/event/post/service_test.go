package post_test

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"

	"github.com/GGP1/groove/config"
	"github.com/GGP1/groove/internal/cache"
	"github.com/GGP1/groove/internal/txgroup"
	"github.com/GGP1/groove/internal/ulid"
	"github.com/GGP1/groove/service/auth"
	"github.com/GGP1/groove/service/event/post"
	"github.com/GGP1/groove/service/event/role"
	"github.com/GGP1/groove/service/notification"
	"github.com/GGP1/groove/test"

	"github.com/dgraph-io/dgo/v210"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

var (
	postSv      post.Service
	db          *sql.DB
	dc          *dgo.Dgraph
	cacheClient cache.Client
)

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

	cacheClient = memcached
	db = postgres
	dc = dgraph

	authService := auth.NewService(postgres, nil, config.Sessions{})
	roleService := role.NewService(postgres, dc, cacheClient)
	notifService := notification.NewService(postgres, dc, config.Notifications{}, authService, roleService)
	postSv = post.NewService(postgres, dgraph, cacheClient, notifService)

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

func TestCreateComment(t *testing.T) {
	ctx := context.Background()
	g, ctx := txgroup.WithContext(ctx, txgroup.NewTxs(db, dc)...)
	userID := ulid.NewString()
	err := test.CreateUser(ctx, db, dc, userID, "random@email.test", "test", "ao121")
	assert.NoError(t, err)

	session := auth.Session{
		ID: userID,
	}
	comment := post.CreateComment{
		Content: "post comment",
	}
	commentID := ulid.NewString()
	err = postSv.CreateComment(ctx, session, commentID, comment)
	assert.NoError(t, err)

	assert.NoError(t, g.Commit())
}

func TestCreatePost(t *testing.T) {
	ctx := context.Background()
	g, ctx := txgroup.WithContext(ctx, txgroup.NewTxs(db, dc)...)

	eventID := ulid.NewString()
	err := test.CreateEvent(ctx, db, dc, eventID, "create_post")
	assert.NoError(t, err)

	session := auth.Session{ID: ulid.NewString()}
	post := post.CreatePost{
		Content: "post content",
		Media:   pq.StringArray{"create_post.com/images/a.jpg"},
	}
	postID := ulid.NewString()
	err = postSv.CreatePost(ctx, session, postID, eventID, post)
	assert.NoError(t, err)

	assert.NoError(t, g.Commit())
}

func TestPost(t *testing.T) {

}

func TestComment(t *testing.T) {

}
