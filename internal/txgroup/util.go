package txgroup

import (
	"database/sql"

	"github.com/dgraph-io/dgo/v210"
)

// NewTxs returns a slice with an sql and a dgraph transaction.
func NewTxs(db *sql.DB, dc *dgo.Dgraph) []Tx {
	sqlTx, err := db.Begin()
	if err != nil {
		panic(err)
	}
	return []Tx{
		NewDgraphTx(dc.NewTxn()),
		NewSQLTx(sqlTx),
	}
}
