package report

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"
)

// Service interface for the reports service.
type Service interface {
	Create(ctx context.Context, report Report) error
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
func (s service) Create(ctx context.Context, report Report) error {
	q := `INSERT INTO events_reports
	(id, reported_id, reporter_id, type, details)
	VALUES
	($1, $2, $3, $4, $5)`
	_, err := s.db.ExecContext(ctx, q, report.ID, report.ReportedID, report.ReporterID, report.Type, report.Details)
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
	report := Report{
		ReportedID: reportedID,
	}
	for rows.Next() {
		if err := rows.Scan(&report.ID, &report.ReporterID, &report.Type, &report.Details); err != nil {
			return nil, errors.Wrap(err, "scanning rows")
		}
		reports = append(reports, report)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return reports, nil
}
