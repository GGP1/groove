package post

import (
	"time"

	"github.com/GGP1/groove/internal/validate"

	"github.com/lib/pq"
	"github.com/pkg/errors"
)

// TODO: add LoggedInUserLiked to both post and comment to check if the user requesting the posts
// liked them

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
	Content    *string `json:"content,omitempty"`
	LikesDelta *int    `json:"likes_delta,omitempty"` // Can be + or -
}

// Validate verifies the correctness of the values received.
func (up UpdatePost) Validate() error {
	if up.Content != nil && *up.Content == "" {
		return errors.New("invalid content")
	}
	if up.LikesDelta != nil && *up.LikesDelta == 0 {
		return errors.New("likes_delta mustn't be zero")
	}
	return nil
}

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
	Replies         []Reply   `json:"replies,omitempty"`
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
	Replies         []Reply   `json:"replies,omitempty"`
}

// CreateComment ..
type CreateComment struct {
	ParentCommentID *string `json:"parent_comment_id,omitempty" db:"parent_comment_id"`
	PostID          *string `json:"post_id,omitempty" db:"post_id"`
	Content         string  `json:"content,omitempty"`
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
	return nil
}
