package postgres_test

import (
	"context"
	"testing"

	"github.com/GGP1/groove/config"
	"github.com/GGP1/groove/storage/postgres"
	"github.com/GGP1/groove/test"

	"github.com/stretchr/testify/assert"
)

func TestConnect(t *testing.T) {
	env := []string{"POSTGRES_USER=postgres", "POSTGRES_PASSWORD=postgres", "listen_addresses = '*'"}
	pool, resource := test.NewResource(t, "postgres", "13.3-alpine", env)

	err := pool.Retry(func() error {
		db, err := postgres.Connect(context.Background(), config.Postgres{
			Username: "postgres",
			Host:     "localhost",
			Port:     resource.GetPort("5432/tcp"),
			Name:     "postgres",
			Password: "postgres",
			SSLMode:  "disable",
		})
		assert.NoError(t, err)

		return db.Close()
	})
	assert.NoError(t, err)
}
