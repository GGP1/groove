package txgroup

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// Tx represents a database transaction.
type Tx interface {
	Key() string
	Commit() error
	Rollback() error
}

// Group manages transactions atomically.
type Group struct {
	errOnce sync.Once
	txs     []Tx
}

type unique string

// WithContext returns a transactions manager containing a context with all the transactions stored in it.
func WithContext(ctx context.Context, txs []Tx) (*Group, context.Context) {
	for _, tx := range txs {
		ctx = context.WithValue(ctx, unique(tx.Key()), tx)
	}
	return &Group{txs: txs}, ctx
}

// AddTx adds a new transaction to the group.
func (a *Group) AddTx(ctx context.Context, tx Tx) context.Context {
	a.txs = append(a.txs, tx)
	return context.WithValue(ctx, unique(tx.Key()), tx)
}

// Commit commits all transactions.
func (a *Group) Commit() error {
	for _, tx := range a.txs {
		if err := tx.Commit(); err != nil {
			if err := a.Rollback(); err != nil {
				return err
			}
			return fmt.Errorf("%s commit: %w", tx.Key(), err)
		}
	}
	return nil
}

// Rollback aborts all transactions.
func (a *Group) Rollback() error {
	var err error
	for _, tx := range a.txs {
		if err := tx.Rollback(); err != nil {
			a.errOnce.Do(func() {
				err = fmt.Errorf("%s rollback: %w", tx.Key(), err)
			})
		}
	}
	return err
}

// TxFromContext returns a transaction from the context.
func TxFromContext(ctx context.Context, key string) (Tx, error) {
	tx, ok := ctx.Value(unique(key)).(Tx)
	if !ok {
		return nil, errors.New(key + " transaction not found")
	}
	return tx, nil
}
