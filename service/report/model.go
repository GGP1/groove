package report

import (
	"errors"
	"time"

	"github.com/GGP1/groove/internal/validate"
)

// Report represents a report made by a user on an event/user
type Report struct {
	CreatedAt  *time.Time `json:"created_at,omitempty" db:"created_at"`
	ID         string     `json:"id,omitempty"`
	ReportedID string     `json:"reported_id,omitempty" db:"reported_id"`
	ReporterID string     `json:"reporter_id,omitempty" db:"reporter_id"`
	Type       string     `json:"type,omitempty"`
	Details    string     `json:"details,omitempty"`
}

// CreateReport is used for creating reports.
type CreateReport struct {
	ReportedID string `json:"reported_id,omitempty" db:"reported_id"` // Could be an event or user
	ReporterID string `json:"reporter_id,omitempty" db:"reporter_id"`
	Type       string `json:"type,omitempty"`
	Details    string `json:"details,omitempty"`
}

// Validate makes sure the received report is correct.
func (cr CreateReport) Validate() error {
	if err := validate.ULIDs(cr.ReportedID, cr.ReporterID); err != nil {
		return err
	}
	if cr.Type == "" {
		return errors.New("type required")
	}
	if cr.Details == "" {
		return errors.New("details required")
	}
	if len(cr.Type) > 60 {
		return errors.New("invalid type, maximum length is 60 characters")
	}
	if len(cr.Details) > 1024 {
		return errors.New("invalid details, maximum length is 1024 characters")
	}
	return nil
}
