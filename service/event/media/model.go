package media

import (
	"time"

	"github.com/GGP1/groove/internal/params"

	"github.com/pkg/errors"
)

// Media reprensents images, videos and audio.
type Media struct {
	ID        string     `json:"id,omitempty"`
	EventID   string     `json:"event_id,omitempty" db:"event_id"`
	URL       string     `json:"url,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty" db:"created_at"`
}

// Validate ..
func (m Media) Validate() error {
	if err := params.ValidateUUID(m.EventID); err != nil {
		return errors.Wrap(err, "invalid event_id")
	}
	if m.URL == "" {
		return errors.New("url required")
	}
	return nil
}
