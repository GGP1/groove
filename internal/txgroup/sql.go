package txgroup

import (
	"context"
	"database/sql"
)

const sqlKey = "sql"

// SQL represents an sql transaction.
type SQL struct {
	tx *sql.Tx
}

// NewSQLTx returns a new SQL transaction wrapped by a structure
// that satisfies the txgroup.Tx interface.
func NewSQLTx(tx *sql.Tx) *SQL {
	return &SQL{tx: tx}
}

// SQLTx returns an sql transaction from the context.
func SQLTx(ctx context.Context) *sql.Tx {
	tx, err := TxFromContext(ctx, sqlKey)
	if err != nil {
		panic(err)
	}
	sql, ok := tx.(*SQL)
	if !ok {
		panic("transaction is not of type sql")
	}
	return sql.tx
}

// Key returns the key used for storing SQL transactions in a context.
func (s *SQL) Key() string {
	return sqlKey
}

// Commit commits the transaction.
func (s *SQL) Commit() error {
	return s.tx.Commit()
}

// Rollback discards the transaction.
func (s *SQL) Rollback() error {
	return s.tx.Rollback()
}
