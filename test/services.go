// Package test contains testing helpers.
package test

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"testing"

	"github.com/GGP1/groove/storage/postgres"

	"github.com/go-redis/redis/v8"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/assert"

	// Used to force the driver registration in order to establish a
	// connection with postgres
	_ "github.com/lib/pq"
)

const resourceExpiration uint = 120 // Seconds

// External dependency
const (
	Postgres dependency = iota
	Redis
)

type setupFunc func(s *sql.DB, r *redis.Client)

type dependency uint8

// Main starts up all the dependencies and runs a function with them as parameters.
func Main(m *testing.M, setup setupFunc, dependencies ...dependency) {
	var (
		pgContainer *dockertest.Resource
		db          *sql.DB

		rdbContainer *dockertest.Resource
		rdb          *redis.Client

		err error
	)
	fatal := func(err error) {
		if err != nil {
			log.Fatal(err)
		}
	}

	deps := make(map[dependency]struct{}, len(dependencies))
	for _, dep := range dependencies {
		deps[dep] = struct{}{}
	}

	if _, ok := deps[Postgres]; ok {
		pgContainer, db, err = RunPostgres()
		fatal(err)
	}
	if _, ok := deps[Redis]; ok {
		rdbContainer, rdb, err = RunRedis()
		fatal(err)
	}

	setup(db, rdb)

	code := m.Run()

	if pgContainer != nil {
		fatal(db.Close())
		fatal(pgContainer.Close())
	}
	if rdbContainer != nil {
		fatal(rdb.Close())
		fatal(rdbContainer.Close())
	}

	os.Exit(code)
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

// RunPostgres initializes a docker container with postgres running in it.
func RunPostgres() (*dockertest.Resource, *sql.DB, error) {
	// The database name will be taken from the user name
	env := []string{"POSTGRES_USER=postgres", "POSTGRES_PASSWORD=postgres", "listen_addresses = '*'"}
	pool, container, err := NewDockerContainer("postgres", "14.1-alpine", env)
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
	if err := postgres.CreateProcedures(context.Background(), db); err != nil {
		return nil, nil, err
	}

	return container, db, nil
}

// RunRedis initializes a docker container with redis running in it.
func RunRedis() (*dockertest.Resource, *redis.Client, error) {
	pool, container, err := NewDockerContainer("redis", "6.2.6-alpine", nil)
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
