// Package test contains testing helpers.
package test

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/GGP1/groove/config"
	"github.com/GGP1/groove/internal/cache"
	"github.com/GGP1/groove/service/event"
	"github.com/GGP1/groove/storage/dgraph"
	"github.com/GGP1/groove/storage/memcached"
	"github.com/GGP1/groove/storage/postgres"

	"github.com/dgraph-io/dgo/v210"
	"github.com/dgraph-io/dgo/v210/protos/api"
	"github.com/go-redis/redis/v8"
	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/encoding/gzip"

	// Used to open a connection with postgres database
	_ "github.com/lib/pq"
)

// CreateEvent creates a new user for testing purposes.
func CreateEvent(ctx context.Context, db *sql.DB, dc *dgo.Dgraph, id, name string) error {
	typ := event.GrandPrix
	public := true
	virtual := false
	ticketCost := 10
	slots := 100
	startTime := 150000
	endTime := 320000
	q := `INSERT INTO events 
	(id, name, type, public, virtual, ticket_cost, slots, start_time, end_Time) 
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`
	_, err := db.ExecContext(ctx, q, id, name, typ, public, virtual,
		ticketCost, slots, startTime, endTime)
	if err != nil {
		return err
	}

	dcTx := dc.NewTxn()
	if err := dgraph.CreateNode(ctx, dcTx, dgraph.Event, id); err != nil {
		dcTx.Discard(ctx)
		return err
	}

	return dcTx.Commit(ctx)
}

// CreateUser creates a new user for testing purposes.
func CreateUser(ctx context.Context, db *sql.DB, dc *dgo.Dgraph, id, email, username, password string) error {
	pwd, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	password = string(pwd)
	q := "INSERT INTO users (id, name, email, username, password, birth_date) VALUES ($1,$2,$3,$4,$5,$6)"
	_, err = db.ExecContext(ctx, q, id, "test", email, username, password, time.Now())
	if err != nil {
		return err
	}

	dcTx := dc.NewTxn()
	if err := dgraph.CreateNode(ctx, dcTx, dgraph.User, id); err != nil {
		dcTx.Discard(ctx)
		return err
	}

	return dcTx.Commit(ctx)
}

// NewResource returns a new pool, a docker container and handles its purge.
func NewResource(t testing.TB, repository, tags string, env []string) (*dockertest.Pool, *dockertest.Resource) {
	pool, err := dockertest.NewPool("")
	assert.NoError(t, err)

	resource, err := pool.Run(repository, tags, env)
	assert.NoError(t, err)

	t.Cleanup(func() {
		assert.NoError(t, pool.Purge(resource), "Couldn't free resources")
	})

	return pool, resource
}

// RunDgraph initializes a docker container with memcached running in it.
func RunDgraph() (*dockertest.Pool, *dockertest.Resource, *dgo.Dgraph, *grpc.ClientConn, error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, nil, nil, nil, err
	}

	resource, err := pool.Run("dgraph/standalone", "v21.03.0", nil)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	var (
		dc   *dgo.Dgraph
		conn *grpc.ClientConn
	)
	err = pool.Retry(func() error {
		conn, err = grpc.Dial(
			net.JoinHostPort("localhost", resource.GetPort("9080/tcp")),
			grpc.WithInsecure(),
			grpc.WithDefaultCallOptions(grpc.UseCompressor(gzip.Name)),
		)
		if err != nil {
			return err
		}
		dc = dgo.NewDgraphClient(api.NewDgraphClient(conn))
		return nil
	})
	if err != nil {
		return nil, nil, nil, nil, err
	}

	ctx := context.Background()
	// Wait for the connection to establish before running tests
	ticker := time.NewTicker(1 * time.Second)
	for {
		<-ticker.C
		state := conn.GetState()
		if state == connectivity.Ready {
			break
		}
	}

	// The connection is established but the server is still initiating
	// retry creating schema until success
	for {
		<-ticker.C
		if err := dgraph.CreateSchema(ctx, dc); err == nil {
			break
		}
	}
	ticker.Stop()

	return pool, resource, dc, conn, nil
}

// StartDgraph initializes a docker container with dgraph running in it.
func StartDgraph(t testing.TB) *dgo.Dgraph {
	pool, resource, dc, conn, err := RunDgraph()
	assert.NoError(t, err)

	t.Cleanup(func() {
		assert.NoError(t, conn.Close(), "Couldn't close the connection")
		assert.NoError(t, pool.Purge(resource), "Couldn't free resources")
	})

	return dc
}

// RunMemcached initializes a docker container with memcached running in it.
func RunMemcached() (*dockertest.Pool, *dockertest.Resource, cache.Client, error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, nil, nil, err
	}

	resource, err := pool.Run("memcached", "1.6.9-alpine", nil)
	if err != nil {
		return nil, nil, nil, err
	}

	var cache cache.Client
	err = pool.Retry(func() error {
		cache, err = memcached.NewClient(config.Memcached{
			Servers: []string{fmt.Sprintf("localhost:%s", resource.GetPort("11211/tcp"))},
		})
		return err
	})
	if err != nil {
		return nil, nil, nil, err
	}

	return pool, resource, cache, nil
}

// StartMemcached starts a memcached container and makes the cleanup.
func StartMemcached(t testing.TB) cache.Client {
	pool, resource, mc, err := RunMemcached()
	assert.NoError(t, err)

	t.Cleanup(func() {
		assert.NoError(t, pool.Purge(resource), "Couldn't free resources")
	})

	return mc
}

// RunPostgres initializes a docker container with postgres running in it.
func RunPostgres() (*dockertest.Pool, *dockertest.Resource, *sql.DB, error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, nil, nil, err
	}
	// The database name will be taken from the user name
	env := []string{"POSTGRES_USER=postgres", "POSTGRES_PASSWORD=postgres", "listen_addresses = '*'"}
	resource, err := pool.Run("postgres", "13.2-alpine", env)
	if err != nil {
		return nil, nil, nil, err
	}

	var db *sql.DB
	err = pool.Retry(func() error {
		url := fmt.Sprintf("host=localhost port=%s user=postgres password=postgres dbname=postgres sslmode=disable",
			resource.GetPort("5432/tcp"))
		db, err = sql.Open("postgres", url)
		if err != nil {
			return err
		}
		return db.Ping()
	})
	if err != nil {
		return nil, nil, nil, err
	}

	if err := postgres.CreateTables(context.Background(), db, config.Postgres{Username: "postgres"}); err != nil {
		return nil, nil, nil, err
	}

	return pool, resource, db, nil
}

// StartPostgres starts a postgres container and makes the cleanup.
func StartPostgres(t testing.TB) *sql.DB {
	pool, resource, db, err := RunPostgres()
	assert.NoError(t, err)

	t.Cleanup(func() {
		assert.NoError(t, db.Close(), "Couldn't close the connection with postgres")
		assert.NoError(t, pool.Purge(resource), "Couldn't free resources")
	})

	return db
}

// RunRedis initializes a docker container with redis running in it.
func RunRedis() (*dockertest.Pool, *dockertest.Resource, *redis.Client, error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, nil, nil, err
	}

	resource, err := pool.Run("redis", "6.2.1-alpine", nil)
	if err != nil {
		return nil, nil, nil, err
	}

	var rdb *redis.Client
	err = pool.Retry(func() error {
		rdb = redis.NewClient(&redis.Options{
			Network: "tcp",
			Addr:    net.JoinHostPort("localhost", resource.GetPort("6379/tcp")),
		})
		return rdb.Ping(rdb.Context()).Err()
	})
	if err != nil {
		return nil, nil, nil, err
	}

	return pool, resource, rdb, nil
}

// StartRedis starts a redis container and makes the cleanup.
func StartRedis(t testing.TB) *redis.Client {
	pool, resource, rdb, err := RunRedis()
	assert.NoError(t, err)

	t.Cleanup(func() {
		assert.NoError(t, rdb.Close(), "Couldn't close connection with redis")
		assert.NoError(t, pool.Purge(resource), "Couldn't free resources")
	})

	return rdb
}
