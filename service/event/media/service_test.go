package media_test

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"

	"github.com/GGP1/groove/internal/ulid"
	"github.com/GGP1/groove/service/event/media"
	"github.com/GGP1/groove/test"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/stretchr/testify/assert"
)

var (
	mediaSv media.Service
	sqlTx   *sql.Tx
	mc      *memcache.Client
)

func TestMain(m *testing.M) {
	poolPg, resourcePg, postgres, err := test.RunPostgres()
	if err != nil {
		log.Fatal(err)
	}
	poolMc, resourceMc, memcached, err := test.RunMemcached()
	if err != nil {
		log.Fatal(err)
	}
	sqlTx, err = postgres.BeginTx(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}
	mc = memcached

	mediaSv = media.NewService(postgres, memcached)

	code := m.Run()

	if err := sqlTx.Rollback(); err != nil {
		log.Fatal(err)
	}

	if err := poolPg.Purge(resourcePg); err != nil {
		log.Fatal(err)
	}
	if err := poolMc.Purge(resourceMc); err != nil {
		log.Fatal(err)
	}

	os.Exit(code)
}

func TestCreateMedia(t *testing.T) {
	ctx := context.Background()
	eventID := ulid.NewString()

	err := createEvent(ctx, eventID, "create_media")
	assert.NoError(t, err)

	media := media.Media{
		EventID: eventID,
		URL:     "create_media.com/images/a.jpg",
	}
	err = mediaSv.CreateMedia(ctx, sqlTx, eventID, media)
	assert.NoError(t, err)
}

func TestUpdateMedia(t *testing.T) {

}

func createEvent(ctx context.Context, id, name string) error {
	q := `INSERT INTO events 
	(id, name, type, public, virtual, ticket_cost, slots, start_time, end_Time) 
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`
	_, err := sqlTx.ExecContext(ctx, q, id, name, 1, true, false, 10, 100, 15000, 320000)
	return err
}
