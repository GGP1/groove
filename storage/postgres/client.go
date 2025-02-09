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
	db      *sql.DB
	metrics metrics
}

// func NewClient(config config.Postgres, db *sql.DB) (Client, error) {
// 	return client{}, nil
// }

// QueryContext ..
func (c *Client) QueryContext(ctx context.Context, query string, a ...any) (*sql.Rows, error) {
	start := time.Now()

	rows, err := c.db.QueryContext(ctx, query, a)
	if err != nil {
		log.Error(query, zap.Any("values", a))
		return nil, errors.Wrap(err, "postgres")
	}

	since := time.Since(start)
	if since > 200*time.Millisecond { // time to be set in config
		log.Warn("slow query", zap.Duration(query, since))
	}

	return rows, nil
}

func wrapper(f func() error, query string, a ...any) error {
	start := time.Now()

	if err := f(); err != nil {
		log.Error(query, zap.Any("values", a))
		return errors.Wrap(err, "postgres")
	}

	since := time.Since(start)
	if since > 200*time.Millisecond { // time to be set in config
		log.Warn("slow query", zap.Duration(query, since))
	}
	return nil
}
