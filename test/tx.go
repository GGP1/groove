package test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/GGP1/groove/internal/txgroup"

	"github.com/stretchr/testify/assert"
)

// CommitTx commits the sql transaction stored in the context and
// replaces it with a new one for later use.
func CommitTx(ctx context.Context, t testing.TB, db *sql.DB) context.Context {
	assert.NoError(t, txgroup.SQLTx(ctx).Commit())

	tx, err := db.Begin()
	assert.NoError(t, err)
	return txgroup.NewContext(ctx, txgroup.NewSQLTx(tx))
}
