package sqltx

import (
	"context"
	"database/sql"

	"github.com/GGP1/groove/internal/txgroup"
)

// txKey is the context key for the sql transaction.
var txKey key

type key struct{}

// NewContext returns a new context with a sql transaction in it.
// The new context must replace the old one.
//
// This function should be called inside handlers (not services),
// otherwise the context won't be updated properly as it's passed by value.
func NewContext(ctx context.Context, tx *sql.Tx) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, txKey, tx)
}

// FromContext returns the sql transaction stored in the context.
//
// It panics if there is no transaction.
func FromContext(ctx context.Context) *sql.Tx {
	tx, ok := ctx.Value(txKey).(*sql.Tx)
	if !ok {
		tx = txgroup.SQLTx(ctx)
	}
	return tx
}
