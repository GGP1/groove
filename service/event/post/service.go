package post

import (
	"context"
	"database/sql"
	"unicode"

	"github.com/GGP1/groove/internal/cache"
	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/scan"
	"github.com/GGP1/groove/internal/sqltx"
	"github.com/GGP1/groove/internal/ulid"
	"github.com/GGP1/groove/model"
	"github.com/GGP1/groove/service/auth"
	"github.com/GGP1/groove/service/notification"
	"github.com/GGP1/groove/storage/dgraph"
	"github.com/GGP1/groove/storage/postgres"

	"github.com/dgraph-io/dgo/v210"
	"github.com/pkg/errors"
)

// Service interface for the post service.
type Service interface {
	CreateComment(ctx context.Context, session auth.Session, comment CreateComment) error
	CreatePost(ctx context.Context, session auth.Session, eventID string, post CreatePost) error
	DeleteComment(ctx context.Context, commentID string) error
	DeletePost(ctx context.Context, eventID, postID string) error
	GetComment(ctx context.Context, commentID string) (Comment, error)
	GetCommentLikes(ctx context.Context, commentID string, params params.Query) ([]model.ListUser, error)
	GetPost(ctx context.Context, eventID, postID string) (Post, error)
	GetPostComments(ctx context.Context, parentID string, params params.Query) ([]Comment, error)
	GetPostLikes(ctx context.Context, postID string, params params.Query) ([]model.ListUser, error)
	GetPosts(ctx context.Context, eventID string, params params.Query) ([]Post, error)
	LikeComment(ctx context.Context, commentID, userID string) error
	LikePost(ctx context.Context, postID, userID string) error
	UpdatePost(ctx context.Context, eventID, postID string, post UpdatePost) error
}

type service struct {
	db                  *sql.DB
	dc                  *dgo.Dgraph
	cache               cache.Client
	notificationService notification.Service
}

// NewService returns a new post service.
func NewService(db *sql.DB, dc *dgo.Dgraph, cache cache.Client, notificationService notification.Service) Service {
	return service{
		db:                  db,
		dc:                  dc,
		cache:               cache,
		notificationService: notificationService,
	}
}

// ContentMentions handles post and comments mentions by scraping their content.
func (s service) ContentMentions(ctx context.Context, session auth.Session, content string) error {
	if len(content) < 2 {
		return nil
	}

	sqlTx := sqltx.FromContext(ctx)

	// Reuse objects
	var (
		stmt *sql.Stmt
		err  error
		ntn  notification.CreateNotification
	)
	for i, c := range content {
		if c == '@' {
			if stmt == nil {
				stmt, err = sqlTx.PrepareContext(ctx, "SELECT id FROM users WHERE username=$1")
				if err != nil {
					return errors.Wrap(err, "preparing statement")
				}
				defer stmt.Close()
				ntn = notification.CreateNotification{
					SenderID: session.ID,
					Content:  notification.MentionContent(session),
					Type:     notification.Mention,
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

			row := stmt.QueryRowContext(ctx, username)
			var userID string
			if err := row.Scan(&userID); err != nil {
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
func (s service) CreateComment(ctx context.Context, session auth.Session, comment CreateComment) error {
	sqlTx := sqltx.FromContext(ctx)

	commentID := ulid.NewString()
	q := "INSERT INTO events_posts_comments (id, parent_comment_id, post_id, user_id, content) VALUES ($1, $2, $3, $4, $5)"
	_, err := sqlTx.ExecContext(ctx, q, commentID,
		comment.ParentCommentID, comment.PostID, session.ID, comment.Content)
	if err != nil {
		return errors.Wrap(err, "creating comment")
	}

	if comment.PostID != nil {
		if _, err := sqlTx.ExecContext(ctx, "UPDATE events_posts SET comments_count = comments_count + 1"); err != nil {
			return errors.Wrap(err, "updating post comments count")
		}
	} else if comment.ParentCommentID != nil {
		if _, err := sqlTx.ExecContext(ctx, "UPDATE events_posts_comments SET replies_count = replies_count + 1"); err != nil {
			return errors.Wrap(err, "updating comment replies count")
		}
	}

	if *comment.ContainsMentions {
		if err := s.ContentMentions(ctx, session, comment.Content); err != nil {
			return err
		}
	}

	return dgraph.Mutation(ctx, s.dc, func(tx *dgo.Txn) error {
		return dgraph.CreateNode(ctx, tx, model.Comment, commentID)
	})
}

// CreatePost adds a post to the event.
func (s service) CreatePost(ctx context.Context, session auth.Session, eventID string, post CreatePost) error {
	sqlTx := sqltx.FromContext(ctx)

	postID := ulid.NewString()
	q := "INSERT INTO events_posts (id, event_id, content, media) VALUES ($1, $2, $3, $4)"
	if _, err := sqlTx.ExecContext(ctx, q, postID, eventID, post.Content, post.Media); err != nil {
		return errors.Wrap(err, "creating post")
	}

	if *post.ContainsMentions {
		if err := s.ContentMentions(ctx, session, post.Content); err != nil {
			return err
		}
	}

	return dgraph.Mutation(ctx, s.dc, func(tx *dgo.Txn) error {
		return dgraph.CreateNode(ctx, tx, model.Post, postID)
	})
}

// DeleteComment removes a comment from a post.
func (s service) DeleteComment(ctx context.Context, commentID string) error {
	sqlTx := sqltx.FromContext(ctx)

	q := "DELETE FROM events_posts_comments WHERE id=$1"
	if _, err := sqlTx.ExecContext(ctx, q, commentID); err != nil {
		return errors.Wrap(err, "deleting comment")
	}
	return nil
}

// DeletePost removes a post from an event.
func (s service) DeletePost(ctx context.Context, eventID, postID string) error {
	sqlTx := sqltx.FromContext(ctx)

	q := "DELETE FROM events_posts WHERE event_id=$1 AND id=$2"
	if _, err := sqlTx.ExecContext(ctx, q, eventID, postID); err != nil {
		return errors.Wrap(err, "deleting post")
	}
	return nil
}

// GetComment returns a comment.
func (s service) GetComment(ctx context.Context, commentID string) (Comment, error) {
	sqlTx := sqltx.FromContext(ctx)

	q := `SELECT 
	id, parent_comment_id, post_id, user_id, content, 
	likes_count, replies_count, created_at 
	FROM events_posts_comments WHERE id=$1`
	rows, err := sqlTx.QueryContext(ctx, q, commentID)
	if err != nil {
		return Comment{}, errors.Wrap(err, "querying comment")
	}

	var comment Comment
	if err := scan.Row(&comment, rows); err != nil {
		return Comment{}, errors.Wrap(err, "scanning comment")
	}

	q2 := `SELECT 
	id, user_id, content, likes_count, 
	replies_count, created_at 
	FROM events_posts_comments WHERE parent_comment_id=$1`
	rows2, err := sqlTx.QueryContext(ctx, q2, comment.ID)
	if err != nil {
		return Comment{}, errors.Wrap(err, "fetching comment replies")
	}

	var replies []Comment
	if err := scan.Rows(&replies, rows2); err != nil {
		return Comment{}, err
	}

	comment.Replies = replies
	return comment, nil
}

// GetCommentLikes returns a comment's likes.
func (s service) GetCommentLikes(ctx context.Context, commentID string, params params.Query) ([]model.ListUser, error) {
	query := commentLikes
	if params.LookupID != "" {
		query = commentLikesLookup
	}
	return s.getLikes(ctx, query, commentID, params)
}

// GetPost returns a post from an event.
func (s service) GetPost(ctx context.Context, eventID, postID string) (Post, error) {
	sqlTx := sqltx.FromContext(ctx)

	q := "SELECT id, event_id, content, media, likes_count, comments_count FROM events_posts WHERE event_id=$1 AND id=$2"
	row := sqlTx.QueryRowContext(ctx, q, eventID, postID)

	var post Post
	if err := row.Scan(&post.ID, &post.EventID, &post.Content,
		&post.Media, &post.LikesCount, &post.CommentsCount); err != nil {
		return Post{}, errors.Wrap(err, "fetching post")
	}

	return post, nil
}

// GetPostComments returns all the comments in a post.
func (s service) GetPostComments(ctx context.Context, postID string, params params.Query) ([]Comment, error) {
	sqlTx := sqltx.FromContext(ctx)

	q := postgres.SelectWhere(model.Comment, "post_id=$1", "id", params)
	stmt, err := sqlTx.PrepareContext(ctx, q)
	if err != nil {
		return nil, errors.Wrap(err, "preparing statement")
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, postID)
	if err != nil {
		return nil, errors.Wrap(err, "fetching comments")
	}

	var comments []Comment
	if err := scan.Rows(&comments, rows); err != nil {
		return nil, errors.Wrap(err, "scanning comments")
	}

	for _, comment := range comments {
		repliesRows, err := stmt.QueryContext(ctx, comment.ID)
		if err != nil {
			return nil, errors.Wrap(err, "fetching replies")
		}

		var replies []Comment
		if err := scan.Rows(&replies, repliesRows); err != nil {
			return nil, errors.Wrap(err, "scanning replies")
		}
		comment.Replies = replies
	}

	return comments, nil
}

// GetPostLikes returns a post's likes.
func (s service) GetPostLikes(ctx context.Context, postID string, params params.Query) ([]model.ListUser, error) {
	query := postLikes
	if params.LookupID != "" {
		query = postLikesLookup
	}
	return s.getLikes(ctx, query, postID, params)
}

// GetPosts returns all the posts corresponding to an event
func (s service) GetPosts(ctx context.Context, eventID string, params params.Query) ([]Post, error) {
	sqlTx := sqltx.FromContext(ctx)

	q := postgres.SelectWhere(model.Post, "event_id=$1", "id", params)
	rows, err := sqlTx.QueryContext(ctx, q, eventID)
	if err != nil {
		return nil, err
	}

	var posts []Post
	if err := scan.Rows(&posts, rows); err != nil {
		return nil, err
	}

	return posts, nil
}

// LikeComment adds a like to a comment, if the like already exists, it removes it.
func (s service) LikeComment(ctx context.Context, commentID, userID string) error {
	sqlTx := sqltx.FromContext(ctx)

	set := true
	q := "UPDATE events_posts_comments SET likes_count = likes_count+1 WHERE id=$1"

	exists, err := s.likeExists(ctx, commentLikesLookup, commentID, userID)
	if err != nil {
		return err
	}
	if exists {
		set = false
		q = "UPDATE events_posts_comments SET likes_count = likes_count-1 WHERE id=$1"
	}

	if _, err = sqlTx.ExecContext(ctx, q, commentID); err != nil {
		return errors.Wrap(err, "updating comment likes count")
	}

	return dgraph.Mutation(ctx, s.dc, func(tx *dgo.Txn) error {
		if _, err := tx.Do(ctx, commentMutationReq(commentID, userID, set)); err != nil {
			return err
		}
		return nil
	})
}

// LikePost adds a like to a post, if the like already exists, it removes it.
func (s service) LikePost(ctx context.Context, postID, userID string) error {
	sqlTx := sqltx.FromContext(ctx)

	set := true
	q := "UPDATE events_posts SET likes_count = likes_count+1 WHERE id=$1"

	exists, err := s.likeExists(ctx, postLikesLookup, postID, userID)
	if err != nil {
		return err
	}
	if exists {
		set = false
		q = "UPDATE events_posts SET likes_count = likes_count-1 WHERE id=$1"
	}

	if _, err = sqlTx.ExecContext(ctx, q, postID); err != nil {
		return errors.Wrap(err, "updating post likes count")
	}

	return dgraph.Mutation(ctx, s.dc, func(tx *dgo.Txn) error {
		if _, err := tx.Do(ctx, postMutationReq(postID, userID, set)); err != nil {
			return err
		}
		return nil
	})
}

// UpdatePost updates an event's post.
func (s service) UpdatePost(ctx context.Context, eventID, postID string, post UpdatePost) error {
	sqlTx := sqltx.FromContext(ctx)

	q := `UPDATE events_posts SET
	content = COALESCE($3,content)
	likes = likes + $4
	WHERE event_id=$1 AND id=$2`
	if _, err := sqlTx.ExecContext(ctx, q, eventID, postID); err != nil {
		return errors.Wrap(err, "updating post")
	}
	return nil
}

// getLikes is a helper for retrieving posts and comments likes.
func (s service) getLikes(ctx context.Context, query query, id string, params params.Query) ([]model.ListUser, error) {
	vars := dgraph.QueryVars(id, params)
	res, err := s.dc.NewReadOnlyTxn().QueryRDFWithVars(ctx, queries[query], vars)
	if err != nil {
		return nil, errors.Wrap(err, "dgraph: fetching users ids")
	}

	usersIDs := dgraph.ParseRDFULIDs(res.Rdf)
	if len(usersIDs) == 0 {
		return nil, nil
	}

	sqlTx := sqltx.FromContext(ctx)
	q := postgres.SelectInID(model.User, usersIDs, params.Fields)
	rows, err := sqlTx.QueryContext(ctx, q)
	if err != nil {
		return nil, errors.Wrap(err, "postgres: fetching users")
	}

	var users []model.ListUser
	if err := scan.Rows(&users, rows); err != nil {
		return nil, err
	}

	return users, nil
}

func (s service) likeExists(ctx context.Context, query query, id, userID string) (bool, error) {
	vars := dgraph.QueryVars(id, params.Query{LookupID: userID})
	res, err := s.dc.NewReadOnlyTxn().QueryRDFWithVars(ctx, queries[query], vars)
	if err != nil {
		return false, errors.Wrap(err, "dgraph: fetching users ids")
	}

	usersIDs := dgraph.ParseRDFULIDs(res.Rdf)
	if len(usersIDs) == 0 {
		return false, nil
	}

	return true, nil
}
