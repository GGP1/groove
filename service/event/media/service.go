package media

import (
	"context"
	"database/sql"

	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/ulid"
	"github.com/GGP1/groove/storage/postgres"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/pkg/errors"
)

// Service interface for the media service.
type Service interface {
	CreateMedia(ctx context.Context, sqlTx *sql.Tx, eventID string, media Media) error
	GetMedia(ctx context.Context, sqlTx *sql.Tx, eventID string, params params.Query) ([]Media, error)
	UpdateMedia(ctx context.Context, sqlTx *sql.Tx, eventID string, media Media) error
}

type service struct {
	db *sql.DB
	mc *memcache.Client
}

// NewService returns a new media service.
func NewService(db *sql.DB, mc *memcache.Client) Service {
	return service{
		db: db,
		mc: mc,
	}
}

// CreateMedia adds a photo or video to the event.
func (s service) CreateMedia(ctx context.Context, sqlTx *sql.Tx, eventID string, media Media) error {
	q := "INSERT INTO events_media (id, event_id, url) VALUES ($1, $2, $3)"
	_, err := sqlTx.ExecContext(ctx, q, ulid.New(), media.EventID, media.URL)
	if err != nil {
		return errors.Wrap(err, "creating media")
	}

	if err := s.mc.Delete(eventID + "_media"); err != nil && err != memcache.ErrCacheMiss {
		return errors.Wrap(err, "memcached: deleting media")
	}

	return nil
}

func (s service) GetMedia(ctx context.Context, sqlTx *sql.Tx, eventID string, params params.Query) ([]Media, error) {
	// TODO: add pagination
	q := postgres.SelectWhereID(postgres.Media, params.Fields, "event_id", eventID)
	if params.LookupID != "" {
		q += "AND id='" + params.LookupID + "'"
	}
	rows, err := sqlTx.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}

	media, err := scanMedia(rows)
	if err != nil {
		return nil, err
	}

	return media, nil
}

// UpdateMedia updates event's media.
func (s service) UpdateMedia(ctx context.Context, sqlTx *sql.Tx, eventID string, media Media) error {
	q := "UPDATE events_media SET url=$2 WHERE id=$1 AND event_id=$2"
	_, err := sqlTx.ExecContext(ctx, q, media.ID, eventID, media.URL)
	if err != nil {
		return errors.Wrap(err, "updating media")
	}

	if err := s.mc.Delete(eventID + "_media"); err != nil && err != memcache.ErrCacheMiss {
		return errors.Wrap(err, "memcached: deleting media")
	}

	return nil
}

func scanMedia(rows *sql.Rows) ([]Media, error) {
	var (
		// Reuse object, there's no need to reset fields as they will be always overwritten
		media  Media
		medias []Media
	)

	cols, _ := rows.Columns()
	if len(cols) > 0 {
		columns := mediaColumns(&media, cols)

		for rows.Next() {
			if err := rows.Scan(columns...); err != nil {
				return nil, errors.Wrap(err, "scanning rows")
			}

			medias = append(medias, media)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return medias, nil
}

func mediaColumns(m *Media, columns []string) []interface{} {
	result := make([]interface{}, 0, len(columns))

	for _, c := range columns {
		switch c {
		case "id":
			result = append(result, &m.ID)
		case "event_id":
			result = append(result, &m.EventID)
		case "url":
			result = append(result, &m.URL)
		}
	}

	return result
}
