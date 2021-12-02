// Package test contains testing helpers.
package test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/GGP1/groove/config"
	"github.com/GGP1/groove/internal/cache"
	"github.com/GGP1/groove/model"
	"github.com/GGP1/groove/service/event"
	"github.com/GGP1/groove/storage/dgraph"
	"github.com/GGP1/groove/storage/memcached"
	"github.com/GGP1/groove/storage/postgres"

	"github.com/dgraph-io/dgo/v210"
	"github.com/dgraph-io/dgo/v210/protos/api"
	"github.com/go-redis/redis/v8"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/encoding/gzip"

	// Used to force the driver registration in order to establish a
	// connection with postgres
	_ "github.com/lib/pq"
)

const resourceExpiration uint = 120 // Seconds

// CreateEvent creates a new user for testing purposes.
func CreateEvent(ctx context.Context, db *sql.DB, dc *dgo.Dgraph, id, name string) error {
	q := `INSERT INTO events 
	(id, name, type, public, virtual, slots, cron, start_date, end_date, ticket_type) 
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`
	_, err := db.ExecContext(ctx, q,
		id, name, event.GrandPrix, true, false, 100, "48 12 * * * 15", time.Now(), time.Now().Add(time.Hour*2400), 1)
	if err != nil {
		return err
	}

	return dgraph.Mutation(ctx, dc, func(tx *dgo.Txn) error {
		return dgraph.CreateNode(ctx, tx, model.Event, id)
	})
}

// CreateUser creates a new user for testing purposes.
func CreateUser(ctx context.Context, db *sql.DB, dc *dgo.Dgraph, id, email, username, password string) error {
	pwd, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	password = string(pwd)
	q := "INSERT INTO users (id, name, email, username, password, birth_date, type) VALUES ($1,$2,$3,$4,$5,$6,$7)"
	_, err = db.ExecContext(ctx, q, id, "test", email, username, password, time.Now(), model.Personal)
	if err != nil {
		return err
	}

	return dgraph.Mutation(ctx, dc, func(tx *dgo.Txn) error {
		return dgraph.CreateNode(ctx, tx, model.User, id)
	})
}

// NewDockerContainer returns a pool with a connection to the docker API
// and a docker container configured to be removed after its use.
func NewDockerContainer(repository, tag string, env []string) (*dockertest.Pool, *dockertest.Resource, error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, nil, err
	}

	container, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: repository,
		Tag:        tag,
		Env:        env,
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{
			Name: "no",
		}
	})
	if err != nil {
		return nil, nil, err
	}
	container.Expire(resourceExpiration)

	return pool, container, nil
}

// RunDgraph initializes a docker container with memcached running in it.
func RunDgraph() (*dockertest.Resource, *dgo.Dgraph, *grpc.ClientConn, error) {
	pool, container, err := NewDockerContainer("dgraph/standalone", "v21.03.2", nil)
	if err != nil {
		return nil, nil, nil, err
	}

	var (
		dc   *dgo.Dgraph
		conn *grpc.ClientConn
	)
	err = pool.Retry(func() error {
		conn, err = grpc.Dial(
			net.JoinHostPort("localhost", container.GetPort("9080/tcp")),
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
		return nil, nil, nil, err
	}

	ctx := context.Background()
	// Wait for the connection to establish before running tests
	ticker := time.NewTicker(200 * time.Millisecond)
	timeout := time.Now().Add(5 * time.Second)
	for {
		if t := <-ticker.C; t.After(timeout) {
			return nil, nil, nil, errors.New("connection: timeout reached")
		}
		state := conn.GetState()
		if state == connectivity.Ready || state == connectivity.Idle {
			break
		}
	}

	// The connection is established but the server is still initiating
	// retry creating schema until success
	timeout = time.Now().Add(5 * time.Second)
	for {
		if t := <-ticker.C; t.After(timeout) {
			return nil, nil, nil, errors.New("create schema: timeout reached")
		}
		if err := dgraph.CreateSchema(ctx, dc); err == nil {
			break
		}
	}
	ticker.Stop()

	return container, dc, conn, nil
}

// RunMemcached initializes a docker container with memcached running in it.
func RunMemcached() (*dockertest.Resource, cache.Client, error) {
	pool, container, err := NewDockerContainer("memcached", "1.6.10-alpine", nil)
	if err != nil {
		return nil, nil, err
	}

	var cache cache.Client
	err = pool.Retry(func() error {
		cache, err = memcached.NewClient(config.Memcached{
			Servers:      []string{fmt.Sprintf("localhost:%s", container.GetPort("11211/tcp"))},
			MaxIdleConns: 1,
			Timeout:      2 * time.Second,
		})
		return err
	})
	if err != nil {
		return nil, nil, err
	}

	return container, cache, nil
}

// RunPostgres initializes a docker container with postgres running in it.
func RunPostgres() (*dockertest.Resource, *sql.DB, error) {
	// The database name will be taken from the user name
	env := []string{"POSTGRES_USER=postgres", "POSTGRES_PASSWORD=postgres", "listen_addresses = '*'"}
	pool, container, err := NewDockerContainer("postgres", "14.0-alpine", env)
	if err != nil {
		return nil, nil, err
	}

	var db *sql.DB
	err = pool.Retry(func() error {
		url := fmt.Sprintf("host=localhost port=%s user=postgres password=postgres dbname=postgres sslmode=disable",
			container.GetPort("5432/tcp"))
		db, err = sql.Open("postgres", url)
		if err != nil {
			return err
		}
		return db.Ping()
	})
	if err != nil {
		return nil, nil, err
	}

	if err := postgres.CreateTables(context.Background(), db); err != nil {
		return nil, nil, err
	}

	return container, db, nil
}

// RunRedis initializes a docker container with redis running in it.
func RunRedis() (*dockertest.Resource, *redis.Client, error) {
	pool, container, err := NewDockerContainer("redis", "6.2.5-alpine", nil)
	if err != nil {
		return nil, nil, err
	}

	var rdb *redis.Client
	err = pool.Retry(func() error {
		rdb = redis.NewClient(&redis.Options{
			Network: "tcp",
			Addr:    net.JoinHostPort("localhost", container.GetPort("6379/tcp")),
		})
		return rdb.Ping(rdb.Context()).Err()
	})
	if err != nil {
		return nil, nil, err
	}

	return container, rdb, nil
}

// StartDgraph initializes a docker container with dgraph running in it.
func StartDgraph(t testing.TB) *dgo.Dgraph {
	container, dc, conn, err := RunDgraph()
	assert.NoError(t, err)

	t.Cleanup(func() {
		assert.NoError(t, conn.Close(), "Couldn't close the connection")
		assert.NoError(t, container.Close(), "Couldn't remove container")
	})

	return dc
}

// StartMemcached starts a memcached container and makes the cleanup.
func StartMemcached(t testing.TB) cache.Client {
	container, mc, err := RunMemcached()
	assert.NoError(t, err)

	t.Cleanup(func() {
		assert.NoError(t, container.Close(), "Couldn't remove container")
	})

	return mc
}

// StartPostgres starts a postgres container and makes the cleanup.
func StartPostgres(t testing.TB) *sql.DB {
	container, db, err := RunPostgres()
	assert.NoError(t, err)

	t.Cleanup(func() {
		assert.NoError(t, db.Close(), "Couldn't close the connection with postgres")
		assert.NoError(t, container.Close(), "Couldn't remove container")
	})

	return db
}

// StartRedis starts a redis container and makes the cleanup.
func StartRedis(t testing.TB) *redis.Client {
	container, rdb, err := RunRedis()
	assert.NoError(t, err)

	t.Cleanup(func() {
		assert.NoError(t, rdb.Close(), "Couldn't close connection with redis")
		assert.NoError(t, container.Close(), "Couldn't remove container")
	})

	return rdb
}
