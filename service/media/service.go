package media

import (
	"context"
	"path/filepath"

	"github.com/GGP1/groove/internal/ulid"

	"github.com/pkg/errors"
)

const maxSize int64 = 1024000

var errImageTooLarge = errors.Errorf("Image too large. Max Size: %w", maxSize)

// const (
// 	Logo bucket = "logo"
// 	Header bucket = "header"
// 	Post bucket = "post"
// )

// type bucket string

// Service represents a service for storing media.
type Service interface {
	Upload(ctx context.Context, media Media) (string, error)
}

type service struct {
	// https://github.com/cshum/imagor
	// s *session.Session // AWS session
}

// NewService returns a new media service.
func NewService() Service {
	return &service{}
}

// Upload ..
func (s *service) Upload(ctx context.Context, media Media) (string, error) {

	id := filepath.Join(ulid.NewString(), filepath.Ext(media.FileHeader.Filename))
	// url, error
	return id, nil
}
