package zone_test

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"

	"github.com/GGP1/groove/internal/cache"
	"github.com/GGP1/groove/model"
	"github.com/GGP1/groove/service/event/zone"
	"github.com/GGP1/groove/test"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

var (
	zoneSv      zone.Service
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

	db = postgres
	cacheClient = memcached
	zoneSv = zone.NewService(db, cacheClient)

	code := m.Run()

	if err := pgContainer.Close(); err != nil {
		log.Fatal(err)
	}
	if err := mcContainer.Close(); err != nil {
		log.Fatal(err)
	}

	os.Exit(code)
}

func TestZone(t *testing.T) {
	ctx := context.Background()
	eventID := test.CreateEvent(t, db, "name")

	createZone := model.Zone{
		Name:                   "zone",
		RequiredPermissionKeys: pq.StringArray{"access_zones", "edit_zones", "invite_users"},
	}
	err := zoneSv.Create(ctx, eventID, createZone)
	assert.NoError(t, err)

	zones, err := zoneSv.Get(ctx, eventID)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(zones))
	assert.Equal(t, createZone, zones[0])

	updateZone := model.UpdateZone{
		RequiredPermissionKeys: &pq.StringArray{"access_zones"},
	}
	err = zoneSv.Update(ctx, eventID, createZone.Name, updateZone)
	assert.NoError(t, err)

	z, err := zoneSv.GetByName(ctx, eventID, createZone.Name)
	assert.Equal(t, *updateZone.RequiredPermissionKeys, z.RequiredPermissionKeys)

	err = zoneSv.Delete(ctx, eventID, createZone.Name)
	assert.NoError(t, err)

	_, err = zoneSv.GetByName(ctx, eventID, createZone.Name)
	assert.Error(t, err)
}
