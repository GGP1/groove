package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/GGP1/groove/internal/log"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// Client ..
type Client struct {
	sql     *sql.DB
	metrics metrics
}

// func NewClient(config config.Postgres) (Client, error) {
// 	return client{}, nil
// }

// QueryContext ..
func (c *Client) QueryContext(ctx context.Context, query string, a ...interface{}) (*sql.Rows, error) {
	start := time.Now()

	rows, err := c.sql.QueryContext(ctx, query, a)
	if err != nil {
		log.Error(query, zap.Any("values", a))
		return nil, errors.Wrap(err, "postgres")
	}

	since := time.Since(start)
	if since > 200*time.Millisecond { // 200*time.Millisecond to be set in config
		log.Warn("slow query", zap.Duration(query, since))
	}

	return rows, nil
}
