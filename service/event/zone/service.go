package zone

import (
	"context"
	"database/sql"
	"strings"

	"github.com/GGP1/groove/storage/postgres"
	"github.com/bradfitz/gomemcache/memcache"
	"github.com/pkg/errors"
)

// Service is the interface containing all the method for event zones.
type Service interface {
	CreateZone(ctx context.Context, sqlTx *sql.Tx, eventID string, zone Zone) error
	GetZoneByName(ctx context.Context, sqlTx *sql.Tx, eventID, name string) (Zone, error)
	GetZones(ctx context.Context, sqlTx *sql.Tx, eventID string) ([]Zone, error)
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
	q1 := "SELECT EXISTS(SELECT 1 FROM events_zones WHERE event_id=$1 AND name=$2)"

	exists, err := postgres.QueryBool(ctx, sqlTx, q1, eventID, zone.Name)
	if err != nil {
		return err
	}
	if exists {
		return errors.Errorf("already exists a zone named %q", zone.Name)
	}

	q2 := "INSERT INTO events_zones (event_id, name, required_permission_keys) VALUES ($1, $2, $3)"
	keys := strings.Join(zone.RequiredPermissionKeys, ",")

	if _, err := sqlTx.ExecContext(ctx, q2, eventID, zone.Name, keys); err != nil {
		return errors.Wrap(err, "creating zone")
	}

	if err := s.mc.Delete(eventID + "_zones"); err != nil && err != memcache.ErrCacheMiss {
		return errors.Wrap(err, "memcached: deleting event")
	}

	return nil
}

// GetZoneByName returns the permission keys required to enter a zone.
func (s service) GetZoneByName(ctx context.Context, sqlTx *sql.Tx, eventID, name string) (Zone, error) {
	q := "SELECT name, required_permission_keys FROM events_zones WHERE event_id=$1 AND name=$2"
	row := sqlTx.QueryRowContext(ctx, q, eventID, name)

	var zone Zone
	var requiredPermKeys string
	if err := row.Scan(&zone.Name, &requiredPermKeys); err != nil {
		return Zone{}, errors.Wrap(err, "scanning permission keys")
	}
	zone.RequiredPermissionKeys = strings.Split(requiredPermKeys, ",")

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
		zone                   Zone
		zones                  []Zone
		requiredPermissionKeys sql.NullString
	)
	for rows.Next() {
		if err := rows.Scan(&zone.Name, &requiredPermissionKeys); err != nil {
			return nil, errors.Wrap(err, "scanning zone")
		}

		zone.RequiredPermissionKeys = strings.Split(requiredPermissionKeys.String, ",")
		zones = append(zones, zone)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return zones, nil
}
