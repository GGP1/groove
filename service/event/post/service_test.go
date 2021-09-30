package post_test

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/GGP1/groove/config"
	"github.com/GGP1/groove/internal/cache"
	"github.com/GGP1/groove/internal/sqltx"
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
	dc          *dgo.Dgraph
	ctx         context.Context
	cacheClient cache.Client
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
	sqlTx, err := postgres.BeginTx(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}
	ctx = sqltx.NewContext(ctx, sqlTx)
	cacheClient = memcached
	dc = dgraph

	authService := auth.NewService(postgres, nil, config.Sessions{})
	roleService := role.NewService(postgres, dc, cacheClient)
	notifService := notification.NewService(postgres, dc, config.Notifications{}, authService, roleService)
	postSv = post.NewService(postgres, dgraph, cacheClient, notifService)

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

func TestCreatePost(t *testing.T) {
	eventID := ulid.NewString()

	err := createEvent(eventID, "create_media")
	assert.NoError(t, err)

	session := auth.Session{ID: ulid.NewString()}
	post := post.CreatePost{
		Content: "post content",
		Media:   pq.StringArray{"create_post.com/images/a.jpg"},
	}
	err = postSv.CreatePost(ctx, session, eventID, post)
	assert.NoError(t, err)
}

func TestUpdatePost(t *testing.T) {

}

func createEvent(id, name string) error {
	sqlTx := sqltx.FromContext(ctx)
	q := `INSERT INTO events 
	(id, name, type, public, virtual, slots, start_time, end_Time) 
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`
	_, err := sqlTx.ExecContext(ctx, q, id, name, 1, true, false, 100, 15000, 320000)
	return err
}
