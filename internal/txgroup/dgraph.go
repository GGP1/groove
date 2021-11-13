package txgroup

import (
	"context"

	"github.com/dgraph-io/dgo/v210"
	"github.com/pkg/errors"
)

const dgoKey = "dgo"

// Dgraph represents a dgraph transaction.
type Dgraph struct {
	tx *dgo.Txn
}

// NewDgraphTx ..
func NewDgraphTx(tx *dgo.Txn) *Dgraph {
	return &Dgraph{tx: tx}
}

// DgraphTx returns an sql transaction from the context.
func DgraphTx(ctx context.Context) *dgo.Txn {
	tx, err := TxFromContext(ctx, dgoKey)
	if err != nil {
		panic(err)
	}
	dgraph, ok := tx.(*Dgraph)
	if !ok {
		panic("transaction is not of type dgraph")
	}
	return dgraph.tx
}

// Key ..
func (d *Dgraph) Key() string {
	return dgoKey
}

// Commit ..
func (d *Dgraph) Commit() error {
	if err := d.tx.Commit(context.Background()); err != nil && !errors.Is(err, dgo.ErrFinished) {
		return errors.Wrap(err, "dgraph: committing transaction")
	}
	return nil
}

// Rollback ..
func (d *Dgraph) Rollback() error {
	return d.tx.Discard(context.Background())
}
