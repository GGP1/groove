package model

import (
	"time"

	"github.com/GGP1/groove/internal/validate"

	"github.com/lib/pq"
	"github.com/pkg/errors"
)

// Post represents an event's post.
type Post struct {
	CreatedAt     *time.Time      `json:"created_at,omitempty" db:"created_at"`
	UpdatedAt     *time.Time      `json:"updated_at,omitempty" db:"updated_at"`
	Media         *pq.StringArray `json:"media,omitempty"`
	LikesCount    *int            `json:"likes_count,omitempty" db:"likes_count"`
	CommentsCount *int            `json:"comments_count,omitempty" db:"comments_count"`
	Content       string          `json:"content,omitempty"`
	ID            string          `json:"id,omitempty"`
	EventID       string          `json:"event_id,omitempty" db:"event_id"`
	AuthUserLiked bool            `json:"auth_user_liked,omitempty" db:"auth_user_liked"`
}

// CreatePost is used for creating posts
type CreatePost struct {
	Content string         `json:"content,omitempty"`
	Media   pq.StringArray `json:"media,omitempty"`
}

// Validate verifies the correctness of the values received.
func (cp CreatePost) Validate() error {
	if cp.Content == "" {
		errors.New("content required")
	}
	for i, m := range cp.Media {
		if err := validate.URL(m); err != nil {
			return errors.Wrapf(err, "media %d", i)
		}
	}
	return nil
}

// UpdatePost contains the fields for updating a post.
type UpdatePost struct {
	Content *string `json:"content,omitempty"`
}

// Validate verifies the correctness of the values received.
func (up UpdatePost) Validate() error {
	if up.Content != nil && *up.Content == "" {
		return errors.New("invalid content")
	}
	return nil
}
