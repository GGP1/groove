package model

import (
	"time"

	"github.com/GGP1/groove/internal/validate"

	"github.com/pkg/errors"
)

// Comment represents a comment.
//
// A comment can be a post comment (PostID != null) or a reply on another comment (ParentCommentID != null)
type Comment struct {
	CreatedAt       time.Time `json:"created_at,omitempty" db:"created_at"`
	ParentCommentID *string   `json:"parent_comment_id,omitempty" db:"parent_comment_id"`
	PostID          *string   `json:"post_id,omitempty" db:"post_id"`
	LikesCount      *int      `json:"likes_count,omitempty" db:"likes_count"`
	RepliesCount    *int      `json:"replies_count,omitempty" db:"replies_count"`
	ID              string    `json:"id,omitempty"`
	UserID          string    `json:"user_id,omitempty" db:"user_id"`
	Content         string    `json:"content,omitempty"`
	AuthUserLiked   bool      `json:"auth_user_liked,omitempty" db:"auth_user_liked"`
}

// Reply is implemented to avoid sql.NullString as its fields are already known.
type Reply struct {
	CreatedAt       time.Time `json:"created_at,omitempty" db:"created_at"`
	RepliesCount    *int      `json:"replies_count,omitempty" db:"replies_count"`
	LikesCount      *int      `json:"likes_count,omitempty" db:"likes_count"`
	UserID          string    `json:"user_id,omitempty" db:"user_id"`
	Content         string    `json:"content,omitempty"`
	ParentCommentID string    `json:"parent_comment_id,omitempty" db:"parent_comment_id"`
	ID              string    `json:"id,omitempty"`
	AuthUserLiked   bool      `json:"auth_user_liked,omitempty" db:"auth_user_liked"`
}

// CreateComment holds the values needed for the creation of a comment.
type CreateComment struct {
	ParentCommentID *string `json:"parent_comment_id,omitempty" db:"parent_comment_id"`
	PostID          *string `json:"post_id,omitempty" db:"post_id"`
	Content         string  `json:"content,omitempty"`
}

// Validate returns an error if the comment contains invalid information.
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
	return nil
}
