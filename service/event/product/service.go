package product

import (
	"context"
	"database/sql"
	"strconv"

	"github.com/GGP1/groove/internal/bufferpool"
	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/ulid"
	"github.com/GGP1/groove/storage/postgres"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/pkg/errors"
)

// Service interface for the products service.
type Service interface {
	CreateProduct(ctx context.Context, sqlTx *sql.Tx, eventID string, product Product) error
	DeleteProduct(ctx context.Context, sqlTx *sql.Tx, eventID, productID string) error
	GetProducts(ctx context.Context, sqlTx *sql.Tx, eventID string, params params.Query) ([]Product, error)
	UpdateProduct(ctx context.Context, sqlTx *sql.Tx, eventID string, product UpdateProduct) error
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
	_, err := sqlTx.ExecContext(ctx, q, ulid.NewString(), product.EventID, product.Stock,
		product.Brand, product.Type, product.Description, product.Discount, product.Taxes,
		product.Subtotal, product.Total)
	if err != nil {
		return errors.Wrap(err, "creating product")
	}

	return nil
}

// DeleteProduct removes a product from an event.
func (s service) DeleteProduct(ctx context.Context, sqlTx *sql.Tx, eventID, productID string) error {
	q := "DELETE FROM events_products WHERE event_id=$1 AND id=$2"
	if _, err := sqlTx.ExecContext(ctx, q, eventID, productID); err != nil {
		return errors.Wrap(err, "deleting product")
	}
	return nil
}

// GetProducts returns the products from an event.
func (s service) GetProducts(ctx context.Context, sqlTx *sql.Tx, eventID string, params params.Query) ([]Product, error) {
	q := postgres.SelectWhereID(postgres.Products, "event_id", eventID, "id", params)
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
func (s service) UpdateProduct(ctx context.Context, sqlTx *sql.Tx, eventID string, product UpdateProduct) error {
	q := updateProductQuery(eventID, product)
	if _, err := sqlTx.ExecContext(ctx, q); err != nil {
		return errors.Wrap(err, "updating products")
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

func updateProductQuery(eventID string, p UpdateProduct) string {
	buf := bufferpool.Get()
	buf.WriteString("UPDATE events_products SET")

	if p.Brand != nil {
		buf.WriteString(" brand='")
		buf.WriteString(*p.Brand)
		buf.WriteString("',")
	}
	if p.Type != nil {
		buf.WriteString(" type='")
		buf.WriteString(*p.Type)
		buf.WriteString("',")
	}
	if p.Description != nil {
		buf.WriteString(" description='")
		buf.WriteString(*p.Description)
		buf.WriteString("',")
	}
	if p.Stock != nil {
		buf.WriteString(" stock=")
		buf.WriteString(strconv.FormatUint(*p.Stock, 10))
		buf.WriteByte(',')
	}
	if p.Discount != nil {
		buf.WriteString(" discount=")
		buf.WriteString(strconv.FormatUint(*p.Discount, 10))
		buf.WriteByte(',')
	}
	if p.Taxes != nil {
		buf.WriteString(" taxes=")
		buf.WriteString(strconv.FormatUint(*p.Taxes, 10))
		buf.WriteByte(',')
	}
	if p.Subtotal != nil {
		buf.WriteString(" subtotal=")
		buf.WriteString(strconv.FormatUint(*p.Subtotal, 10))
		buf.WriteByte(',')
	}
	if p.Total != nil {
		buf.WriteString(" total=")
		buf.WriteString(strconv.FormatUint(*p.Total, 10))
	}

	buf.WriteString(" WHERE event_id='")
	buf.WriteString(eventID)
	buf.WriteString("' AND id='")
	buf.WriteString(p.ID)
	buf.WriteByte('\'')

	q := buf.String()
	bufferpool.Put(buf)

	return q
}
