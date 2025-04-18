package zone

import (
	"context"
	"database/sql"

	"github.com/GGP1/groove/internal/cache"
	"github.com/GGP1/groove/internal/permissions"
	"github.com/GGP1/groove/internal/txgroup"
	"github.com/GGP1/groove/model"
	"github.com/GGP1/sqan"

	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
)

// Service is the interface containing all the method for event zones.
type Service interface {
	Create(ctx context.Context, eventID string, zone model.Zone) error
	Delete(ctx context.Context, eventID, name string) error
	GetByName(ctx context.Context, eventID, name string) (model.Zone, error)
	Get(ctx context.Context, eventID string) ([]model.Zone, error)
	Update(ctx context.Context, eventID, name string, updateZone model.UpdateZone) error
}

type service struct {
	db  *sql.DB
	rdb *redis.Client
}

// NewService returns a new zone service.
func NewService(db *sql.DB, rdb *redis.Client) Service {
	return &service{
		db:  db,
		rdb: rdb,
	}
}

// Create creates a zone inside an event.
func (s *service) Create(ctx context.Context, eventID string, zone model.Zone) error {
	sqlTx := txgroup.SQLTx(ctx)

	q1 := "SELECT EXISTS(SELECT 1 FROM events_permissions WHERE event_id=$1 AND key=$2)"
	exists := false

	stmt, err := sqlTx.PrepareContext(ctx, q1)
	if err != nil {
		return errors.Wrap(err, "preparing statement")
	}
	defer stmt.Close()

	// Check for the existence of the keys used for the role
	for _, key := range zone.RequiredPermissionKeys {
		if permissions.Reserved.Exists(key) {
			continue
		}
		row := stmt.QueryRowContext(ctx, eventID, key)
		if err := row.Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return errors.Errorf("permission with key %q does not exist", key)
		}
	}

	q := "INSERT INTO events_zones (event_id, name, required_permission_keys) VALUES ($1, $2, $3)"
	if _, err := sqlTx.ExecContext(ctx, q, eventID, zone.Name, zone.RequiredPermissionKeys); err != nil {
		return errors.Wrap(err, "creating zone")
	}
	return nil
}

// Delete removes a zone from the event.
func (s *service) Delete(ctx context.Context, eventID, name string) error {
	sqlTx := txgroup.SQLTx(ctx)

	q := "DELETE FROM events_zones WHERE event_id=$1 AND name=$2"
	if _, err := sqlTx.ExecContext(ctx, q, eventID, name); err != nil {
		return errors.Wrap(err, "deleting zone")
	}

	if err := s.rdb.Del(ctx, cache.ZonesKey(eventID)).Err(); err != nil {
		return errors.Wrap(err, "deleting zone")
	}

	return nil
}

// GetByName returns the permission keys required to enter a zone.
func (s *service) GetByName(ctx context.Context, eventID, name string) (model.Zone, error) {
	q := "SELECT name, required_permission_keys FROM events_zones WHERE event_id=$1 AND name=$2"
	row := s.db.QueryRowContext(ctx, q, eventID, name)

	var zone model.Zone
	if err := row.Scan(&zone.Name, &zone.RequiredPermissionKeys); err != nil {
		return model.Zone{}, errors.Wrap(err, "scanning zone required permission keys")
	}

	return zone, nil
}

// Get gets an event's zones.
func (s *service) Get(ctx context.Context, eventID string) ([]model.Zone, error) {
	q := "SELECT name, required_permission_keys FROM events_zones WHERE event_id=$1"
	rows, err := s.db.QueryContext(ctx, q, eventID)
	if err != nil {
		return nil, errors.Wrap(err, "querying event zones")
	}

	var zones []model.Zone
	if err := sqan.Rows(&zones, rows); err != nil {
		return nil, errors.Wrap(err, "scanning zones")
	}

	return zones, nil
}

// Update sets new values for an event's zone.
func (s *service) Update(ctx context.Context, eventID, name string, zone model.UpdateZone) error {
	sqlTx := txgroup.SQLTx(ctx)

	q := `UPDATE events_zones SET
	name = COALESCE($3,name),
	required_permission_keys = COALESCE($4,required_permission_keys)
	WHERE event_id=$1 AND name=$2`
	if _, err := sqlTx.ExecContext(ctx, q, eventID, name, zone.Name, zone.RequiredPermissionKeys); err != nil {
		return errors.Wrap(err, "updating zone")
	}

	return nil
}
