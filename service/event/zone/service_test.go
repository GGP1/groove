package zone_test

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"

	"github.com/GGP1/groove/internal/cache"
	"github.com/GGP1/groove/internal/txgroup"
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

	if err := db.Close(); err != nil {
		log.Fatal(err)
	}
	if err := pgContainer.Close(); err != nil {
		log.Fatal(err)
	}
	if err := mcContainer.Close(); err != nil {
		log.Fatal(err)
	}

	os.Exit(code)
}

func TestZoneService(t *testing.T) {
	ctx := context.Background()
	tx, err := db.Begin()
	assert.NoError(t, err)
	ctx = txgroup.NewContext(nil, txgroup.NewSQLTx(tx))
	eventID := test.CreateEvent(t, db, "name")
	perm := model.Permission{
		Key:  "access_zones",
		Name: "Access zones",
	}
	test.CreatePermission(t, db, eventID, perm)

	createZone := model.Zone{
		Name:                   "zone",
		RequiredPermissionKeys: pq.StringArray{perm.Key},
	}
	t.Run("Create", func(t *testing.T) {
		err = zoneSv.Create(ctx, eventID, createZone)
		assert.NoError(t, err)
		ctx = test.CommitTx(ctx, t, db)
	})

	t.Run("Get", func(t *testing.T) {
		zones, err := zoneSv.Get(ctx, eventID)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(zones))
		assert.Equal(t, createZone, zones[0])
	})

	t.Run("GetByName", func(t *testing.T) {
		z, err := zoneSv.GetByName(ctx, eventID, createZone.Name)
		assert.NoError(t, err)
		assert.Equal(t, createZone, z)
	})

	t.Run("Update", func(t *testing.T) {
		updateZone := model.UpdateZone{
			RequiredPermissionKeys: &pq.StringArray{perm.Key},
		}
		err = zoneSv.Update(ctx, eventID, createZone.Name, updateZone)
		assert.NoError(t, err)
		ctx = test.CommitTx(ctx, t, db)

		z, err := zoneSv.GetByName(ctx, eventID, createZone.Name)
		assert.NoError(t, err)
		assert.Equal(t, *updateZone.RequiredPermissionKeys, z.RequiredPermissionKeys)
	})

	t.Run("Delete", func(t *testing.T) {
		err = zoneSv.Delete(ctx, eventID, createZone.Name)
		assert.NoError(t, err)
		ctx = test.CommitTx(ctx, t, db)

		_, err = zoneSv.GetByName(ctx, eventID, createZone.Name)
		assert.Error(t, err)
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})

}
