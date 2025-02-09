package post

import (
	"context"
	"database/sql"
	"net/http"
	"unicode"

	"github.com/GGP1/groove/internal/httperr"
	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/roles"
	"github.com/GGP1/groove/internal/txgroup"
	"github.com/GGP1/groove/internal/ulid"
	"github.com/GGP1/groove/model"
	"github.com/GGP1/groove/service/auth"
	"github.com/GGP1/groove/service/notification"
	"github.com/GGP1/groove/storage/postgres"
	"github.com/GGP1/sqan"

	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
)

// Service interface for the post service.
type Service interface {
	CreateComment(ctx context.Context, session auth.Session, comment model.CreateComment) (string, error)
	CreatePost(ctx context.Context, session auth.Session, eventID string, post model.CreatePost) (string, error)
	DeleteComment(ctx context.Context, commentID string, session auth.Session) error
	DeletePost(ctx context.Context, eventID, postID string) error
	GetComment(ctx context.Context, commentID, userID string) (model.Comment, error)
	GetCommentLikes(ctx context.Context, commentID string, params params.Query) ([]model.User, error)
	GetCommentLikesCount(ctx context.Context, commentID string) (int64, error)
	GetHomePosts(ctx context.Context, session auth.Session, params params.Query) ([]model.Post, error)
	GetPost(ctx context.Context, eventID, postID, userID string) (model.Post, error)
	GetPostLikes(ctx context.Context, postID string, params params.Query) ([]model.User, error)
	GetPostLikesCount(ctx context.Context, postID string) (int64, error)
	GetPosts(ctx context.Context, eventID, userID string, params params.Query) ([]model.Post, error)
	GetReplies(ctx context.Context, parentID, userID string, params params.Query) ([]model.Comment, error)
	LikeComment(ctx context.Context, commentID, userID string) error
	LikePost(ctx context.Context, postID, userID string) error
	UpdatePost(ctx context.Context, postID string, post model.UpdatePost) error
}

type service struct {
	db                  *sql.DB
	rdb                 *redis.Client
	notificationService notification.Service
}

// NewService returns a new post service.
func NewService(db *sql.DB, rdb *redis.Client, notificationService notification.Service) Service {
	return &service{
		db:                  db,
		rdb:                 rdb,
		notificationService: notificationService,
	}
}

// ContentMentions handles post and comments mentions by scraping their content.
func (s *service) ContentMentions(ctx context.Context, session auth.Session, content string) error {
	if len(content) < 2 {
		return nil
	}

	sqlTx := txgroup.SQLTx(ctx)

	// Reuse objects
	var (
		stmt *sql.Stmt
		err  error
		ntn  model.CreateNotification
	)
	for i, c := range content {
		if c == '@' {
			if stmt == nil {
				stmt, err = sqlTx.PrepareContext(ctx, "SELECT id FROM users WHERE username=$1")
				if err != nil {
					return errors.Wrap(err, "preparing statement")
				}
				defer stmt.Close()
				ntn = model.CreateNotification{
					SenderID: session.ID,
					Content:  notification.MentionContent(session),
					Type:     model.Mention,
				}
			}

			// end represents the index of the username's last character
			end := len(content)
			for j, ch := range content[i+1:] {
				if (unicode.IsPunct(ch) || unicode.IsSpace(ch) || unicode.IsSymbol(ch)) &&
					ch != '_' && ch != '.' {
					end = j + i + 1
					break
				}
			}
			username := content[i+1 : end]
			i = end

			var userID string
			if err := stmt.QueryRowContext(ctx, username).Scan(&userID); err != nil {
				continue // user does not exist, skip
			}

			ntn.ReceiverID = userID
			if err := s.notificationService.Create(ctx, session, ntn); err != nil {
				return err
			}
		}
	}

	return nil
}

// CreateComments creates a comment inside a post.
func (s *service) CreateComment(ctx context.Context, session auth.Session, comment model.CreateComment) (string, error) {
	sqlTx := txgroup.SQLTx(ctx)

	id := ulid.NewString()
	q := "INSERT INTO events_posts_comments (id, parent_comment_id, post_id, user_id, content) VALUES ($1, $2, $3, $4, $5)"
	_, err := sqlTx.ExecContext(ctx, q, id,
		comment.ParentCommentID, comment.PostID, session.ID, comment.Content)
	if err != nil {
		return "", errors.Wrap(err, "creating comment")
	}

	if comment.PostID != nil {
		if _, err := sqlTx.ExecContext(ctx, "UPDATE events_posts SET comments_count = comments_count + 1 WHERE id = $1", comment.PostID); err != nil {
			return "", errors.Wrap(err, "updating post comments count")
		}
	} else if comment.ParentCommentID != nil {
		if _, err := sqlTx.ExecContext(ctx, "UPDATE events_posts_comments SET replies_count = replies_count + 1 WHERE id = $1", comment.ParentCommentID); err != nil {
			return "", errors.Wrap(err, "updating comment replies count")
		}
	}

	if err := s.ContentMentions(ctx, session, comment.Content); err != nil {
		return "", err
	}

	return id, nil
}

// CreatePost adds a post to the event.
func (s *service) CreatePost(ctx context.Context, session auth.Session, eventID string, post model.CreatePost) (string, error) {
	sqlTx := txgroup.SQLTx(ctx)

	id := ulid.NewString()
	q := "INSERT INTO events_posts (id, event_id, content, media) VALUES ($1, $2, $3, $4)"
	if _, err := sqlTx.ExecContext(ctx, q, id, eventID, post.Content, post.Media); err != nil {
		return "", errors.Wrap(err, "creating post")
	}

	if err := s.ContentMentions(ctx, session, post.Content); err != nil {
		return "", err
	}

	return id, nil
}

// DeleteComment removes a comment from a post.
func (s *service) DeleteComment(ctx context.Context, commentID string, session auth.Session) error {
	sqlTx := txgroup.SQLTx(ctx)

	var userID string
	q1 := "SELECT user_id FROM events_posts_comments WHERE id=$1"
	if err := sqlTx.QueryRowContext(ctx, q1, commentID).Scan(&userID); err != nil {
		return errors.Wrap(err, "querying comment owner")
	}

	if userID != session.ID {
		return httperr.New("can't delete this comment", http.StatusForbidden)
	}

	q2 := "DELETE FROM events_posts_comments WHERE id=$1"
	if _, err := sqlTx.ExecContext(ctx, q2, commentID); err != nil {
		return errors.Wrap(err, "deleting comment")
	}
	return nil
}

// DeletePost removes a post from an event.
func (s *service) DeletePost(ctx context.Context, eventID, postID string) error {
	sqlTx := txgroup.SQLTx(ctx)

	q := "DELETE FROM events_posts WHERE event_id=$1 AND id=$2"
	if _, err := sqlTx.ExecContext(ctx, q, eventID, postID); err != nil {
		return errors.Wrap(err, "deleting post")
	}
	return nil
}

// GetComment returns a comment.
func (s *service) GetComment(ctx context.Context, commentID, userID string) (model.Comment, error) {
	q := `SELECT 
	c.id, c.parent_comment_id, c.post_id, c.user_id, c.content, c.replies_count, c.created_at,
	(SELECT COUNT(*) FROM events_posts_comments_likes WHERE comment_id = c.id) as likes_count,
	(SELECT EXISTS(SELECT 1 FROM events_posts_comments_likes WHERE comment_id = c.id AND user_id=$2)) as auth_user_liked
	FROM events_posts_comments AS c WHERE id=$1`
	rows, err := s.db.QueryContext(ctx, q, commentID, userID)
	if err != nil {
		return model.Comment{}, errors.Wrap(err, "querying comment")
	}

	var comment model.Comment
	if err := sqan.Row(&comment, rows); err != nil {
		return model.Comment{}, errors.Wrap(err, "scanning comment")
	}

	return comment, nil
}

// GetCommentLikes returns a comment's likes.
func (s *service) GetCommentLikes(ctx context.Context, commentID string, params params.Query) ([]model.User, error) {
	q := "SELECT {fields} FROM {table} WHERE id IN (SELECT user_id FROM events_posts_comments_likes WHERE comment_id=$1) {pag}"
	query := postgres.Select(model.T.User, q, params)
	rows, err := s.db.QueryContext(ctx, query, commentID)
	if err != nil {
		return nil, err
	}

	var users []model.User
	if err := sqan.Rows(&users, rows); err != nil {
		return nil, err
	}

	return users, nil
}

func (s *service) GetCommentLikesCount(ctx context.Context, commentID string) (int64, error) {
	q := "SELECT COUNT(*) FROM events_posts_comments_likes WHERE comment_id=$1"
	return postgres.Query[int64](ctx, s.db, q, commentID)
}

// GetHomePosts returns a user's home posts.
func (s *service) GetHomePosts(ctx context.Context, session auth.Session, params params.Query) ([]model.Post, error) {
	q := `SELECT {fields},
	(SELECT COUNT(*) FROM events_posts_likes WHERE post_id = p.id) as likes_count,
	(SELECT EXISTS(SELECT 1 FROM events_posts_likes WHERE post_id = p.id AND user_id=$1)) as auth_user_liked
	FROM {table} WHERE
	event_id IN (SELECT event_id FROM events_users_roles WHERE user_id=$1 AND role_name != $2) {pag}`
	query := postgres.Select(model.T.Post, q, params)
	rows, err := s.db.QueryContext(ctx, query, session.ID, roles.Viewer)
	if err != nil {
		return nil, errors.Wrap(err, "querying posts")
	}

	var posts []model.Post
	if err := sqan.Rows(&posts, rows); err != nil {
		return nil, errors.Wrap(err, "scanning posts")
	}

	return posts, nil
}

// GetPost returns a post from an event.
func (s *service) GetPost(ctx context.Context, eventID, postID, userID string) (model.Post, error) {
	q := `SELECT 
	p.id, p.event_id, p.content, p.media, p.comments_count, p.created_at, p.updated_at,
	(SELECT COUNT(*) FROM events_posts_likes WHERE post_id = p.id) as likes_count,
	(SELECT EXISTS(SELECT 1 FROM events_posts_likes WHERE post_id = p.id AND user_id=$3)) as auth_user_liked
	FROM events_posts AS p WHERE event_id=$1 AND id=$2`
	row := s.db.QueryRowContext(ctx, q, eventID, postID, userID)

	var post model.Post
	err := row.Scan(&post.ID, &post.EventID, &post.Content,
		&post.Media, &post.CommentsCount,
		&post.CreatedAt, &post.UpdatedAt, &post.LikesCount,
		&post.AuthUserLiked)
	if err != nil {
		return model.Post{}, errors.Wrap(err, "querying post")
	}

	return post, nil
}

// GetPostLikes returns a post's likes.
func (s *service) GetPostLikes(ctx context.Context, postID string, params params.Query) ([]model.User, error) {
	q := "SELECT {fields} FROM {table} WHERE id IN (SELECT user_id FROM events_posts_likes WHERE post_id=$1) {pag}"
	query := postgres.Select(model.T.User, q, params)
	rows, err := s.db.QueryContext(ctx, query, postID)
	if err != nil {
		return nil, err
	}

	var users []model.User
	if err := sqan.Rows(&users, rows); err != nil {
		return nil, err
	}

	return users, nil
}

// GetPostLikesCount returns the number of likes in a comment.
func (s *service) GetPostLikesCount(ctx context.Context, postID string) (int64, error) {
	q := "SELECT COUNT(*) FROM events_posts_likes WHERE post_id=$1"
	return postgres.Query[int64](ctx, s.db, q, postID)
}

// GetPosts returns all the posts corresponding to an event.
func (s *service) GetPosts(ctx context.Context, eventID, userID string, params params.Query) ([]model.Post, error) {
	q := `SELECT 
	{fields},
	(SELECT COUNT(*) FROM events_posts_likes WHERE post_id = p.id) as likes_count,
	(SELECT EXISTS(SELECT 1 FROM events_posts_likes WHERE post_id = p.id AND user_id=$2)) as auth_user_liked
	FROM {table}
	WHERE event_id=$1 {pag}`
	query := postgres.Select(model.T.Post, q, params)
	rows, err := s.db.QueryContext(ctx, query, eventID, userID)
	if err != nil {
		return nil, err
	}

	var posts []model.Post
	if err := sqan.Rows(&posts, rows); err != nil {
		return nil, err
	}

	return posts, nil
}

// GetReplies returns a comment's replies.
func (s *service) GetReplies(ctx context.Context, parentID, userID string, params params.Query) ([]model.Comment, error) {
	q := `SELECT 
	{fields},
	(SELECT COUNT(*) FROM events_posts_comments_likes WHERE comment_id = c.id) as likes_count,
	(SELECT EXISTS(SELECT 1 FROM events_posts_comments_likes WHERE comment_id = c.id AND user_id=$2)) as auth_user_liked
	FROM {table} WHERE parent_comment_id=$1 OR post_id=$1 {pag}`
	query := postgres.Select(model.T.Comment, q, params)
	rows, err := s.db.QueryContext(ctx, query, parentID, userID)
	if err != nil {
		return nil, errors.Wrap(err, "querying replies")
	}

	var comments []model.Comment
	if err := sqan.Rows(&comments, rows); err != nil {
		return nil, errors.Wrap(err, "scanning replies")
	}

	return comments, nil
}

// LikeComment adds a like to a comment, if the like already exists, it removes it.
func (s *service) LikeComment(ctx context.Context, commentID, userID string) error {
	sqlTx := txgroup.SQLTx(ctx)
	if _, err := sqlTx.ExecContext(ctx, "CALL likeComment($1, $2)", commentID, userID); err != nil {
		return errors.Wrap(err, "comment like")
	}

	return nil
}

// LikePost adds a like to a post, if the like already exists, it removes it.
func (s *service) LikePost(ctx context.Context, postID, userID string) error {
	sqlTx := txgroup.SQLTx(ctx)
	if _, err := sqlTx.ExecContext(ctx, "CALL likePost($1, $2)", postID, userID); err != nil {
		return errors.Wrap(err, "post like")
	}
	return nil
}

// UpdatePost updates an event's post.
func (s service) UpdatePost(ctx context.Context, postID string, post model.UpdatePost) error {
	q := `UPDATE events_posts SET
	content = COALESCE($2,content)
	WHERE id=$1`
	if _, err := s.db.ExecContext(ctx, q, postID, post.Content); err != nil {
		return errors.Wrap(err, "updating post")
	}
	return nil
}
