package report

import (
	"context"
	"database/sql"

	"github.com/GGP1/groove/internal/ulid"
	"github.com/GGP1/groove/model"
	"github.com/GGP1/sqan"
	"github.com/pkg/errors"
)

// Service interface for the reports service.
type Service interface {
	Create(ctx context.Context, report model.CreateReport) (string, error)
	Get(ctx context.Context, reportedID string) ([]model.Report, error)
	GetByID(ctx context.Context, reportID string) (model.Report, error)
}

type service struct {
	db *sql.DB
}

// NewService returns a new reports service.
func NewService(db *sql.DB) Service {
	return &service{db: db}
}

// Create adds a report to the event.
func (s *service) Create(ctx context.Context, report model.CreateReport) (string, error) {
	id := ulid.NewString()
	q := `INSERT INTO events_reports
	(id, reported_id, reporter_id, type, details)
	VALUES
	($1, $2, $3, $4, $5)`
	_, err := s.db.ExecContext(ctx, q, id, report.ReportedID, report.ReporterID, report.Type, report.Details)
	if err != nil {
		return "", errors.Wrap(err, "creating report")
	}

	return id, nil
}

// Get returns an event/user reports.
func (s *service) Get(ctx context.Context, reportedID string) ([]model.Report, error) {
	q := "SELECT id, reporter_id, type, details FROM events_reports WHERE reported_id=$1"
	rows, err := s.db.QueryContext(ctx, q, reportedID)
	if err != nil {
		return nil, err
	}

	var reports []model.Report
	if err := sqan.Rows(&reports, rows); err != nil {
		return nil, errors.Wrap(err, "scanning reports")
	}

	return reports, nil
}

// GetByID looks for a report by its id and returns it.
func (s *service) GetByID(ctx context.Context, reportID string) (model.Report, error) {
	q := "SELECT id, reporter_id, type, details FROM events_reports WHERE id=$1"
	rows, err := s.db.QueryContext(ctx, q, reportID)
	if err != nil {
		return model.Report{}, err
	}

	var report model.Report
	if err := sqan.Row(&report, rows); err != nil {
		return model.Report{}, errors.Wrap(err, "scanning reports")
	}

	return report, nil
}
