package product_test

import (
	"context"
	"database/sql"
	"log"
	"testing"

	"github.com/GGP1/groove/internal/cache"
	"github.com/GGP1/groove/internal/txgroup"
	"github.com/GGP1/groove/model"
	"github.com/GGP1/groove/service/event/product"
	"github.com/GGP1/groove/test"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
)

var (
	db          *sql.DB
	cacheClient cache.Client
	productSv   product.Service
	ctx         context.Context
)

func TestMain(m *testing.M) {
	test.Main(m, func(s *sql.DB, r *redis.Client, c cache.Client) {
		sqlTx, err := s.BeginTx(context.Background(), nil)
		if err != nil {
			log.Fatal(err)
		}
		_, ctx = txgroup.WithContext(ctx, txgroup.NewSQLTx(sqlTx))
		db = s
		cacheClient = c
		productSv = product.NewService(s, cacheClient)
	}, test.Postgres, test.Memcached)
}

func TestCreateProduct(t *testing.T) {
	eventID := test.CreateEvent(t, db, "create_product")

	product := model.Product{
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
	_, err := productSv.Create(ctx, eventID, product)
	assert.NoError(t, err)
}
