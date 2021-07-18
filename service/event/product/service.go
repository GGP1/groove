package product

import (
	"context"
	"database/sql"

	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/ulid"
	"github.com/GGP1/groove/storage/postgres"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/pkg/errors"
)

// Service interface for the products service.
type Service interface {
	CreateProduct(ctx context.Context, sqlTx *sql.Tx, eventID string, product Product) error
	GetProducts(ctx context.Context, sqlTx *sql.Tx, eventID string, params params.Query) ([]Product, error)
	UpdateProduct(ctx context.Context, sqlTx *sql.Tx, eventID string, product Product) error
}

type service struct {
	db *sql.DB
	mc *memcache.Client
}

// NewService returns a new products service.
func NewService(db *sql.DB, mc *memcache.Client) Service {
	return service{
		db: db,
		mc: mc,
	}
}

// CreateProduct adds a product to the event.
func (s service) CreateProduct(ctx context.Context, sqlTx *sql.Tx, eventID string, product Product) error {
	q := `INSERT INTO events_products 
	(id, event_id, stock, brand, type, description, discount, taxes, subtotal, total) 
	VALUES 
	($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := sqlTx.ExecContext(ctx, q, ulid.New(), product.EventID, product.Stock,
		product.Brand, product.Type, product.Description, product.Discount, product.Taxes,
		product.Subtotal, product.Total)
	if err != nil {
		return errors.Wrap(err, "creating product")
	}

	return nil
}

// GetProducts returns the products from an event.
func (s service) GetProducts(ctx context.Context, sqlTx *sql.Tx, eventID string, params params.Query) ([]Product, error) {
	// TODO: add pagination
	q := postgres.SelectWhereID(postgres.Products, params.Fields, "event_id", eventID)
	if params.LookupID != "" {
		q += "AND id='" + params.LookupID + "'"
	}
	rows, err := sqlTx.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}

	products, err := scanProducts(rows)
	if err != nil {
		return nil, err
	}

	return products, nil
}

// UpdateProduct updates an event product.
func (s service) UpdateProduct(ctx context.Context, sqlTx *sql.Tx, eventID string, product Product) error {
	q := `UPDATE events_products SET 
	stock=$3 brand=$4 type=$5 description=$6 discount=$7 taxes=$8 subtotal=$9 total=$10 
	WHERE id=$1 AND event_id=$2`
	_, err := sqlTx.ExecContext(ctx, q, product.ID, eventID, product.Stock, product.Brand, product.Type,
		product.Description, product.Discount, product.Taxes, product.Subtotal, product.Total)
	if err != nil {
		return errors.Wrap(err, "updating products")
	}

	if err := s.mc.Delete(eventID + "_products"); err != nil && err != memcache.ErrCacheMiss {
		return errors.Wrap(err, "memcached: deleting products")
	}

	return nil
}

func scanProducts(rows *sql.Rows) ([]Product, error) {
	var (
		// Reuse object, there's no need to reset fields as they will be always overwritten
		product  Product
		products []Product
	)

	cols, _ := rows.Columns()
	if len(cols) > 0 {
		columns := productColumns(&product, cols)

		for rows.Next() {
			if err := rows.Scan(columns...); err != nil {
				return nil, errors.Wrap(err, "scanning rows")
			}

			products = append(products, product)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return products, nil
}

func productColumns(p *Product, columns []string) []interface{} {
	result := make([]interface{}, 0, len(columns))

	for _, c := range columns {
		switch c {
		case "id":
			result = append(result, &p.ID)
		case "event_id":
			result = append(result, &p.EventID)
		case "stock":
			result = append(result, &p.Stock)
		case "brand":
			result = append(result, &p.Brand)
		case "description":
			result = append(result, &p.Description)
		case "discount":
			result = append(result, &p.Discount)
		case "taxes":
			result = append(result, &p.Taxes)
		case "subtotal":
			result = append(result, &p.Subtotal)
		case "total":
			result = append(result, &p.Total)
		}
	}

	return result
}
