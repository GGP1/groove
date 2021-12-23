package product

import (
	"context"
	"database/sql"
	"time"

	"github.com/GGP1/groove/internal/cache"
	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/txgroup"
	"github.com/GGP1/groove/internal/ulid"
	"github.com/GGP1/groove/model"
	"github.com/GGP1/groove/storage/postgres"
	"github.com/GGP1/sqan"

	"github.com/pkg/errors"
)

// Service interface for the products service.
type Service interface {
	Create(ctx context.Context, eventID string, product model.Product) (string, error)
	Delete(ctx context.Context, eventID, productID string) error
	Get(ctx context.Context, eventID string, params params.Query) ([]model.Product, error)
	Update(ctx context.Context, eventID, productID string, product model.UpdateProduct) error
}

type service struct {
	db    *sql.DB
	cache cache.Client
}

// NewService returns a new products service.
func NewService(db *sql.DB, cache cache.Client) Service {
	return &service{
		db:    db,
		cache: cache,
	}
}

// Create adds a product to the event.
func (s service) Create(ctx context.Context, eventID string, product model.Product) (string, error) {
	sqlTx := txgroup.SQLTx(ctx)

	id := ulid.NewString()
	q := `INSERT INTO events_products 
	(id, event_id, stock, brand, type, description, discount, taxes, subtotal, total) 
	VALUES 
	($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := sqlTx.ExecContext(ctx, q, id, product.EventID, product.Stock,
		product.Brand, product.Type, product.Description, product.Discount, product.Taxes,
		product.Subtotal, product.Total)
	if err != nil {
		return "", errors.Wrap(err, "creating product")
	}

	return id, nil
}

// Delete removes a product from an event.
func (s *service) Delete(ctx context.Context, eventID, productID string) error {
	sqlTx := txgroup.SQLTx(ctx)

	q := "DELETE FROM events_products WHERE event_id=$1 AND id=$2"
	if _, err := sqlTx.ExecContext(ctx, q, eventID, productID); err != nil {
		return errors.Wrap(err, "deleting product")
	}
	return nil
}

// Get returns the products from an event.
func (s *service) Get(ctx context.Context, eventID string, params params.Query) ([]model.Product, error) {
	q := "SELECT {fields} FROM {table} WHERE event_id=$1 {pag}"
	query := postgres.Select(model.T.Product, q, params)
	rows, err := s.db.QueryContext(ctx, query, eventID)
	if err != nil {
		return nil, err
	}

	var products []model.Product
	if err := sqan.Rows(&products, rows); err != nil {
		return nil, err
	}

	return products, nil
}

// Update updates an event product.
func (s service) Update(ctx context.Context, eventID, productID string, product model.UpdateProduct) error {
	sqlTx := txgroup.SQLTx(ctx)

	q := `UPDATE events_products SET
	brand = COALESCE($3,brand),
	type = COALESCE($4,type),
	description = COALESCE($5,description),
	stock = COALESCE($6,stock),
	discount = COALESCE($7,discount),
	taxes = COALESCE($8,taxes),
	subtotal = COALESCE($9,subtotal),
	total = COALESCE($10,total),
	updated_at = $11
	WHERE event_id=$1 AND id=$2`
	_, err := sqlTx.ExecContext(ctx, q, eventID, productID, product.Brand, product.Type,
		product.Description, product.Stock, product.Discount, product.Taxes, product.Subtotal,
		product.Total, time.Now())
	if err != nil {
		return errors.Wrap(err, "updating product")
	}

	return nil
}
