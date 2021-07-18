package product

import (
	"time"

	"github.com/GGP1/groove/internal/ulid"

	"github.com/pkg/errors"
)

// Product represents a market commodity.
//
// Amounts to be provided in a currency’s smallest unit.
type Product struct {
	ID          string     `json:"id,omitempty"`
	EventID     string     `json:"event_id" db:"event_id"`
	Stock       uint       `json:"stock"`
	Brand       string     `json:"brand"`
	Type        string     `json:"type"`
	Description string     `json:"description"`
	Discount    int64      `json:"discount"`
	Taxes       int64      `json:"taxes"`
	Subtotal    int64      `json:"subtotal"`
	Total       int64      `json:"total"`
	CreatedAt   *time.Time `json:"created_at,omitempty" db:"created_at"`
}

// Validate ..
func (p Product) Validate() error {
	if err := ulid.Validate(p.EventID); err != nil {
		return errors.Wrap(err, "invalid event_id")
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
