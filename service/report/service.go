package report

import (
	"context"
	"database/sql"

	"github.com/GGP1/groove/internal/ulid"
	"github.com/GGP1/sqan"
	"github.com/pkg/errors"
)

// Service interface for the reports service.
type Service interface {
	Create(ctx context.Context, report CreateReport) error
	Get(ctx context.Context, reportedID string) ([]Report, error)
}

type service struct {
	db *sql.DB
}

// NewService returns a new reports service.
func NewService(db *sql.DB) Service {
	return service{db: db}
}

// Create adds a report to the event.
func (s service) Create(ctx context.Context, report CreateReport) error {
	q := `INSERT INTO events_reports
	(id, reported_id, reporter_id, type, details)
	VALUES
	($1, $2, $3, $4, $5)`
	_, err := s.db.ExecContext(ctx, q, ulid.NewString(), report.ReportedID, report.ReporterID, report.Type, report.Details)
	if err != nil {
		return errors.Wrap(err, "creating report")
	}

	return nil
}

// Get returns an event/user reports.
func (s service) Get(ctx context.Context, reportedID string) ([]Report, error) {
	q := "SELECT id, reporter_id, type, details FROM events_reports WHERE reported_id=$1"
	rows, err := s.db.QueryContext(ctx, q, reportedID)
	if err != nil {
		return nil, err
	}

	var reports []Report
	if err := sqan.Rows(&reports, rows); err != nil {
		return nil, errors.Wrap(err, "scanning reports")
	}

	return reports, nil
}
