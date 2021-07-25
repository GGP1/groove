package zone

import (
	"context"
	"database/sql"

	"github.com/GGP1/groove/internal/bufferpool"
	"github.com/bradfitz/gomemcache/memcache"
	"github.com/pkg/errors"
)

// Service is the interface containing all the method for event zones.
type Service interface {
	CreateZone(ctx context.Context, sqlTx *sql.Tx, eventID string, zone Zone) error
	DeleteZone(ctx context.Context, sqlTx *sql.Tx, eventID, name string) error
	GetZone(ctx context.Context, sqlTx *sql.Tx, eventID, name string) (Zone, error)
	GetZones(ctx context.Context, sqlTx *sql.Tx, eventID string) ([]Zone, error)
	UpdateZone(ctx context.Context, sqlTx *sql.Tx, eventID, name string, updateZone UpdateZone) error
}

type service struct {
	db *sql.DB
	mc *memcache.Client
}

// NewService returns a new zones service.
func NewService(db *sql.DB, mc *memcache.Client) Service {
	return service{
		db: db,
		mc: mc,
	}
}

// CreateZone creates a zone inside an event.
func (s service) CreateZone(ctx context.Context, sqlTx *sql.Tx, eventID string, zone Zone) error {
	q := "INSERT INTO events_zones (event_id, name, required_permission_keys) VALUES ($1, $2, $3)"
	if _, err := sqlTx.ExecContext(ctx, q, eventID, zone.Name, zone.RequiredPermissionKeys); err != nil {
		return errors.Wrap(err, "creating zone")
	}

	if err := s.mc.Delete(eventID + "_zones"); err != nil && err != memcache.ErrCacheMiss {
		return errors.Wrap(err, "memcached: deleting event")
	}

	return nil
}

// DeleteZone removes a zone from the event.
func (s service) DeleteZone(ctx context.Context, sqlTx *sql.Tx, eventID, name string) error {
	q := "DELETE FROM events_zones WHERE event_id=$1 AND name=$2"
	if _, err := sqlTx.ExecContext(ctx, q, eventID, name); err != nil {
		return errors.Wrap(err, "deleting zone")
	}
	return nil
}

// GetZoneByName returns the permission keys required to enter a zone.
func (s service) GetZone(ctx context.Context, sqlTx *sql.Tx, eventID, name string) (Zone, error) {
	q := "SELECT name, required_permission_keys FROM events_zones WHERE event_id=$1 AND name=$2"
	row := sqlTx.QueryRowContext(ctx, q, eventID, name)

	var zone Zone
	if err := row.Scan(&zone.Name, &zone.RequiredPermissionKeys); err != nil {
		return Zone{}, errors.Wrap(err, "scanning zone required permission keys")
	}

	return zone, nil
}

// GetZones gets an event's zones.
func (s service) GetZones(ctx context.Context, sqlTx *sql.Tx, eventID string) ([]Zone, error) {
	q := "SELECT name, required_permission_keys FROM events_zones WHERE event_id=$1"

	rows, err := sqlTx.QueryContext(ctx, q, eventID)
	if err != nil {
		return nil, errors.Wrap(err, "fetching event zones")
	}

	var (
		zone  Zone
		zones []Zone
	)
	for rows.Next() {
		if err := rows.Scan(&zone.Name, &zone.RequiredPermissionKeys); err != nil {
			return nil, errors.Wrap(err, "scanning zone")
		}
		zones = append(zones, zone)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return zones, nil
}

// UpdateZone sets new values for an event's zone.
func (s service) UpdateZone(ctx context.Context, sqlTx *sql.Tx, eventID, name string, updateZone UpdateZone) error {
	buf := bufferpool.Get()
	buf.WriteString("UPDATE events_zones SET")

	if updateZone.RequiredPermissionKeys != nil {
		buf.WriteString(" required_permission_keys='")
		buf.WriteString("'{")
		for i, key := range *updateZone.RequiredPermissionKeys {
			if i != 0 {
				buf.WriteByte(',')
			}
			buf.WriteByte('"')
			buf.WriteString(key)
			buf.WriteByte('"')
		}
		buf.WriteString("}'")
	}

	buf.WriteString(" WHERE event_id='")
	buf.WriteString(eventID)
	buf.WriteByte('\'')
	buf.WriteString("AND name='")
	buf.WriteString(name)
	buf.WriteByte('\'')

	if _, err := sqlTx.ExecContext(ctx, buf.String()); err != nil {
		return errors.Wrap(err, "updating role")
	}

	bufferpool.Put(buf)
	return nil
}
