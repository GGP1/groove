package product_test

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/GGP1/groove/internal/cache"
	"github.com/GGP1/groove/internal/sqltx"
	"github.com/GGP1/groove/internal/ulid"
	"github.com/GGP1/groove/service/event/product"
	"github.com/GGP1/groove/test"

	"github.com/stretchr/testify/assert"
)

var (
	productSv   product.Service
	ctx         context.Context
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

	sqlTx, err := postgres.BeginTx(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}
	ctx = sqltx.NewContext(ctx, sqlTx)
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
	eventID := ulid.NewString()

	err := createEvent(eventID, "create_product")
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
	err = productSv.Create(ctx, eventID, product)
	assert.NoError(t, err)
}

func TestUserHasRole(t *testing.T) {

}

func createEvent(id, name string) error {
	sqlTx := sqltx.FromContext(ctx)
	q := `INSERT INTO events 
	(id, name, type, public, virtual, slots, cron) 
	VALUES ($1,$2,$3,$4,$5,$6,$7)`
	_, err := sqlTx.ExecContext(ctx, q, id, name, 1, true, false, 100, "15 20 5 12 2 120")
	return err
}
