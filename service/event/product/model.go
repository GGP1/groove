package product

import (
	"time"

	"github.com/GGP1/groove/internal/validate"

	"github.com/pkg/errors"
)

// Product represents a market commodity.
//
// Amounts to be provided in a currency’s smallest unit.
type Product struct {
	CreatedAt   *time.Time `json:"created_at,omitempty" db:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty" db:"updated_at"`
	ID          string     `json:"id,omitempty"`
	EventID     string     `json:"event_id,omitempty" db:"event_id"`
	Brand       string     `json:"brand,omitempty"`
	Type        string     `json:"type,omitempty"`
	Description string     `json:"description,omitempty"`
	Stock       uint64     `json:"stock,omitempty"`
	Discount    uint64     `json:"discount,omitempty"`
	Taxes       uint64     `json:"taxes,omitempty"`
	Subtotal    uint64     `json:"subtotal,omitempty"`
	Total       uint64     `json:"total,omitempty"`
}

// Validate ..
func (p Product) Validate() error {
	if err := validate.ULID(p.EventID); err != nil {
		return errors.Wrap(err, "invalid event_id")
	}
	if len(p.Brand) > 60 {
		return errors.New("invalid brand, maximum length is 60 characters")
	}
	if len(p.Type) > 60 {
		return errors.New("invalid type, maximum length is 60 characters")
	}
	if len(p.Description) > 200 {
		return errors.New("invalid description, maximum length is 200 characters")
	}
	if p.Discount < 0 {
		return errors.New("invalid discount, minimum is 0")
	}
	if p.Taxes < 0 {
		return errors.New("invalid taxes, minimum is 0")
	}
	if p.Total < 0 {
		return errors.New("invalid total, minimum is 0")
	}
	return nil
}

// UpdateProduct is the structure used to update products.
type UpdateProduct struct {
	Stock       *uint64 `json:"stock,omitempty"`
	Brand       *string `json:"brand,omitempty"`
	Type        *string `json:"type,omitempty"`
	Description *string `json:"description,omitempty"`
	Discount    *uint64 `json:"discount,omitempty"`
	Taxes       *uint64 `json:"taxes,omitempty"`
	Subtotal    *uint64 `json:"subtotal,omitempty"`
	Total       *uint64 `json:"total,omitempty"`
}

// Validate ..
func (p UpdateProduct) Validate() error {
	if p.Discount != nil && *p.Discount < 0 {
		return errors.New("invalid discount, minimum is 0")
	}
	if p.Taxes != nil && *p.Taxes < 0 {
		return errors.New("invalid taxes, minimum is 0")
	}
	if p.Total != nil && *p.Total < 0 {
		return errors.New("invalid total, minimum is 0")
	}
	return nil
}
