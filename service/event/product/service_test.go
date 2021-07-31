package product_test

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"

	"github.com/GGP1/groove/internal/cache"
	"github.com/GGP1/groove/internal/ulid"
	"github.com/GGP1/groove/service/event/product"
	"github.com/GGP1/groove/test"

	"github.com/stretchr/testify/assert"
)

var (
	productSv   product.Service
	sqlTx       *sql.Tx
	cacheClient cache.Client
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
	cacheClient = memcached

	productSv = product.NewService(postgres, cacheClient)

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

func TestCreateProduct(t *testing.T) {
	ctx := context.Background()
	eventID := ulid.NewString()

	err := createEvent(ctx, eventID, "create_product")
	assert.NoError(t, err)

	product := product.Product{
		EventID:     eventID,
		Stock:       12,
		Brand:       "brand",
		Type:        "type",
		Discount:    5,
		Taxes:       2,
		Subtotal:    10,
		Total:       7,
		Description: "TestCreatePermission",
	}
	err = productSv.CreateProduct(ctx, sqlTx, eventID, product)
	assert.NoError(t, err)
}

func TestUserHasRole(t *testing.T) {

}

func createEvent(ctx context.Context, id, name string) error {
	q := `INSERT INTO events 
	(id, name, type, public, virtual, ticket_cost, slots, start_time, end_Time) 
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`
	_, err := sqlTx.ExecContext(ctx, q, id, name, 1, true, false, 10, 100, 15000, 320000)
	return err
}
