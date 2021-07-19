package media

import (
	"time"

	"github.com/pkg/errors"
)

// Media reprensents images, videos and audio.
type Media struct {
	ID        string     `json:"id,omitempty"`
	EventID   string     `json:"event_id,omitempty" db:"event_id"`
	URL       string     `json:"url,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty" db:"created_at"`
}

// CreateMedia is the stucture used to create a media inside an event.
type CreateMedia struct {
	URL string `json:"url,omitempty"`
}

// Validate ..
func (cm CreateMedia) Validate() error {
	if cm.URL == "" {
		return errors.New("url required")
	}
	return nil
}
