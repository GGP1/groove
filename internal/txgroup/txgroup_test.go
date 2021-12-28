package txgroup_test

import (
	"context"
	"testing"

	"github.com/GGP1/groove/internal/txgroup"
	"github.com/stretchr/testify/assert"
)

func TestNewContext(t *testing.T) {
	ctx := context.Background()
	tx := &mockTx{}

	ctx = txgroup.NewContext(ctx, tx)

	got, err := txgroup.TxFromContext(ctx, tx.Key())
	assert.NoError(t, err)

	assert.Equal(t, tx, got)
}

func TestWithContext(t *testing.T) {
	ctx := context.Background()
	tx := &mockTx{}

	txg, ctx := txgroup.WithContext(ctx, tx)
	assert.NoError(t, txg.Commit())
	assert.NoError(t, txg.Rollback())

	got, err := txgroup.TxFromContext(ctx, tx.Key())
	assert.NoError(t, err)

	assert.Equal(t, tx, got)
}

func TestTxFromContextNotFound(t *testing.T) {
	_, err := txgroup.TxFromContext(context.Background(), "notFound")
	assert.Error(t, err)
}

func TestUniqueKey(t *testing.T) {
	ctx := context.Background()
	tx := &mockTx{}

	_, ctx = txgroup.WithContext(ctx, tx)
	got := ctx.Value(tx.Key())
	assert.Nil(t, got)
}

func TestAddTx(t *testing.T) {
	txg, ctx := txgroup.WithContext(context.Background())

	tx := &mockTx{}
	ctx = txg.AddTx(ctx, tx)

	got, err := txgroup.TxFromContext(ctx, tx.Key())
	assert.NoError(t, err)

	assert.Equal(t, tx, got)
}

type mockTx struct{}

func (*mockTx) Key() string {
	return "mock"
}

func (*mockTx) Commit() error {
	return nil
}

func (*mockTx) Rollback() error {
	return nil
}
