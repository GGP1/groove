package report

import "time"

// Report represents a report made by a user on an event/user
type Report struct {
	ID         string     `json:"id,omitempty"`
	ReportedID string     `json:"reported_id,omitempty" db:"reported_id"` // Could be an event or user
	ReporterID string     `json:"reporter_id,omitempty" db:"reporter_id"`
	Type       string     `json:"type,omitempty"`
	Details    string     `json:"details,omitempty"`
	CreatedAt  *time.Time `json:"created_at,omitempty" db:"created_at"`
}
