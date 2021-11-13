package zone

import (
	"context"
	"database/sql"

	"github.com/GGP1/groove/internal/cache"
	"github.com/GGP1/groove/model"
	"github.com/GGP1/sqan"

	"github.com/pkg/errors"
)

// Service is the interface containing all the method for event zones.
type Service interface {
	Create(ctx context.Context, eventID string, zone Zone) error
	Delete(ctx context.Context, eventID, name string) error
	GetByName(ctx context.Context, eventID, name string) (Zone, error)
	Get(ctx context.Context, eventID string) ([]Zone, error)
	Update(ctx context.Context, eventID, name string, updateZone UpdateZone) error
}

type service struct {
	db    *sql.DB
	cache cache.Client
}

// NewService returns a new zones service.
func NewService(db *sql.DB, cache cache.Client) Service {
	return service{
		db:    db,
		cache: cache,
	}
}

// Create creates a zone inside an event.
func (s service) Create(ctx context.Context, eventID string, zone Zone) error {
	q := "INSERT INTO events_zones (event_id, name, required_permission_keys) VALUES ($1, $2, $3)"
	if _, err := s.db.ExecContext(ctx, q, eventID, zone.Name, zone.RequiredPermissionKeys); err != nil {
		return errors.Wrap(err, "creating zone")
	}
	return nil
}

// Delete removes a zone from the event.
func (s service) Delete(ctx context.Context, eventID, name string) error {
	q := "DELETE FROM events_zones WHERE event_id=$1 AND name=$2"
	if _, err := s.db.ExecContext(ctx, q, eventID, name); err != nil {
		return errors.Wrap(err, "deleting zone")
	}

	if err := s.cache.Delete(model.ZonesCacheKey(eventID)); err != nil {
		return errors.Wrap(err, "deleting zone")
	}

	return nil
}

// GetByName returns the permission keys required to enter a zone.
func (s service) GetByName(ctx context.Context, eventID, name string) (Zone, error) {
	q := "SELECT name, required_permission_keys FROM events_zones WHERE event_id=$1 AND name=$2"
	row := s.db.QueryRowContext(ctx, q, eventID, name)

	var zone Zone
	if err := row.Scan(&zone.Name, &zone.RequiredPermissionKeys); err != nil {
		return Zone{}, errors.Wrap(err, "scanning zone required permission keys")
	}

	return zone, nil
}

// Get gets an event's zones.
func (s service) Get(ctx context.Context, eventID string) ([]Zone, error) {
	q := "SELECT name, required_permission_keys FROM events_zones WHERE event_id=$1"
	rows, err := s.db.QueryContext(ctx, q, eventID)
	if err != nil {
		return nil, errors.Wrap(err, "fetching event zones")
	}

	var zones []Zone
	if err := sqan.Rows(&zones, rows); err != nil {
		return nil, errors.Wrap(err, "scanning zones")
	}

	return zones, nil
}

// Update sets new values for an event's zone.
func (s service) Update(ctx context.Context, eventID, name string, zone UpdateZone) error {
	if err := zone.Validate(); err != nil {
		return err
	}

	q := `UPDATE events_zones SET
	required_permission_keys = COALESCE($3, required_permission_keys)
	WHERE event_id=$1 AND name=$2`
	if _, err := s.db.ExecContext(ctx, q, eventID, name, zone.RequiredPermissionKeys); err != nil {
		return errors.Wrap(err, "updating role")
	}

	return nil
}
