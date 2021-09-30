package post

import (
	"time"

	"github.com/GGP1/groove/internal/validate"

	"github.com/lib/pq"
	"github.com/pkg/errors"
)

// Post ..
type Post struct {
	ID            string          `json:"id,omitempty"`
	EventID       string          `json:"event_id,omitempty" db:"event_id"`
	Media         *pq.StringArray `json:"media,omitempty"`
	Content       string          `json:"content,omitempty"`
	LikesCount    *int            `json:"likes_count,omitempty" db:"likes_count"`
	CommentsCount *int            `json:"comments_count,omitempty" db:"comments_count"`
	CreatedAt     *time.Time      `json:"created_at,omitempty" db:"created_at"`
	UpdatedAt     *time.Time      `json:"updated_at,omitempty" db:"updated_at"`
}

// CreatePost ..
type CreatePost struct {
	Content          string         `json:"content,omitempty"`
	Media            pq.StringArray `json:"media,omitempty"`
	ContainsMentions *bool          `json:"contains_mentions,omitempty"`
}

// Validate ..
func (cp CreatePost) Validate() error {
	if cp.Content == "" {
		errors.New("content required")
	}
	if cp.ContainsMentions == nil {
		return errors.New("contains_mentions required")
	}
	return nil
}

// UpdatePost ..
type UpdatePost struct {
	Content    *string `json:"content,omitempty"`
	LikesDelta *int    `json:"likes_delta,omitempty"` // Can be + or -
}

// Validate ..
func (up UpdatePost) Validate() error {
	if up.Content != nil && *up.Content == "" {
		return errors.New("invalid content")
	}
	if up.LikesDelta != nil && *up.LikesDelta == 0 {
		return errors.New("likes_delta mustn't be zero")
	}
	return nil
}

// Comment ..
//
// A comment can be a post comment (PostID != null) or a reply on another comment (ParentCommentID != null)
type Comment struct {
	ID              string    `json:"id,omitempty"`
	ParentCommentID string    `json:"parent_comment_id,omitempty" db:"parent_comment_id"`
	PostID          string    `json:"post_id,omitempty" db:"post_id"`
	UserID          string    `json:"user_id,omitempty" db:"user_id"`
	Content         string    `json:"content,omitempty"`
	LikesCount      int       `json:"likes_count,omitempty" db:"likes_count"`
	RepliesCount    int       `json:"replies_count,omitempty" db:"replies_count"`
	Replies         []Comment `json:"replies,omitempty"`
	CreatedAt       time.Time `json:"created_at,omitempty" db:"created_at"`
}

// CreateComment ..
type CreateComment struct {
	ParentCommentID  *string `json:"parent_comment_id,omitempty" db:"parent_comment_id"`
	PostID           *string `json:"post_id,omitempty" db:"post_id"`
	Content          string  `json:"content,omitempty"`
	ContainsMentions *bool   `json:"contains_mentions,omitempty"`
}

// Validate ..
func (cc CreateComment) Validate() error {
	if cc.ParentCommentID == nil && cc.PostID == nil {
		return errors.New("must reference a post or another comment")
	}
	if cc.ParentCommentID != nil && cc.PostID != nil {
		return errors.New("cannot reference both a post and another comment")
	}
	if cc.ParentCommentID != nil {
		if err := validate.ULID(*cc.ParentCommentID); err != nil {
			return err
		}
	}
	if cc.PostID != nil {
		if err := validate.ULID(*cc.PostID); err != nil {
			return err
		}
	}
	if cc.Content == "" {
		return errors.New("content required")
	}
	if cc.ContainsMentions == nil {
		return errors.New("contains_mentions required")
	}
	return nil
}
