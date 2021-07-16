package report

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"
)

// Service interface for the reports service.
type Service interface {
	CreateReport(ctx context.Context, report Report) error
	GetReports(ctx context.Context, reportedID string) ([]Report, error)
}

type service struct {
	db *sql.DB
}

// NewService returns a new reports service.
func NewService(db *sql.DB) Service {
	return service{db: db}
}

// CreateReport adds a report to the event.
func (s service) CreateReport(ctx context.Context, report Report) error {
	q := `INSERT INTO events_reports
	(reported_id, reporter_id, type, details)
	VALUES
	($1, $2, $3, $4)`
	_, err := s.db.ExecContext(ctx, q, report.ReportedID, report.ReporterID, report.Type, report.Details)
	if err != nil {
		return errors.Wrap(err, "creating report")
	}

	return nil
}

// GetReports returns event's reports.
func (s service) GetReports(ctx context.Context, reportedID string) ([]Report, error) {
	q := "SELECT * FROM events_reports WHERE reported_id=$1"
	rows, err := s.db.QueryContext(ctx, q, reportedID)
	if err != nil {
		return nil, err
	}

	var reports []Report
	for rows.Next() {
		var reportedID, reporterID, details string
		if err := rows.Scan(&reportedID, &reporterID, &details); err != nil {
			return nil, errors.Wrap(err, "scanning rows")
		}
		reports = append(reports, Report{
			ReportedID: reportedID,
			ReporterID: reporterID,
			Details:    details,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return reports, nil
}
