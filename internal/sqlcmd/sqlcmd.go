// Package sqlcmd provides a series of interfaces for interacting with SQL commands programatically.
package sqlcmd

import (
	"context"

	"github.com/GGP1/groove/internal/txgroup"

	"github.com/pkg/errors"
)

// Just a draft to see if the implementation is correct or not

// Cmd ..
type Cmd interface {
	Create(ctx context.Context) error
	Exec(ctx context.Context, args ...any) error
}

// likePost ..
type likePost struct{}

// Create creates or replaces the procedure.
func (l *likePost) Create(ctx context.Context) error {
	sqlTx := txgroup.SQLTx(ctx)
	likePost := `CREATE OR REPLACE PROCEDURE likePost(postID text, userID text) AS $$
	BEGIN
		IF EXISTS (SELECT 1 FROM events_posts_likes WHERE post_id=postID AND user_id=userID) THEN
	   		DELETE FROM events_posts_likes WHERE post_id=postID AND user_id=userID;
	   	ELSE
	   		INSERT INTO events_posts_likes (post_id, user_id) VALUES (postID, userID);
	   	END IF;
	END $$ LANGUAGE plpgsql`
	if _, err := sqlTx.ExecContext(ctx, likePost); err != nil {
		return errors.Wrap(err, "creating likePost procedure")
	}

	return nil
}

// Exec runs the procedure.
func (l *likePost) Exec(ctx context.Context, args ...any) error {
	sqlTx := txgroup.SQLTx(ctx)
	if _, err := sqlTx.ExecContext(ctx, "CALL likePost($1, $2)", args); err != nil {
		return errors.Wrap(err, "post like")
	}

	return nil
}
