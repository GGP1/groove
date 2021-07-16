package zone_test

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"

	"github.com/GGP1/groove/service/event/zone"
	"github.com/GGP1/groove/test"

	"github.com/bradfitz/gomemcache/memcache"
)

var (
	zoneSv zone.Service
	sqlTx  *sql.Tx
	mc     *memcache.Client
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

	zoneSv = zone.NewService(postgres, memcached)

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

func TestCreateZone(t *testing.T) {

}

func TestGetZonePermissionKeys(t *testing.T) {

}

func TestGetZones(t *testing.T) {

}
