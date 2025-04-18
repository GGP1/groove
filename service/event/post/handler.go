package post

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/response"
	"github.com/GGP1/groove/internal/validate"
	"github.com/GGP1/groove/model"
	"github.com/GGP1/groove/service/auth"
	"github.com/GGP1/groove/storage/postgres"

	"github.com/julienschmidt/httprouter"
)

// Handler handles ticket service endpoints.
type Handler struct {
	db *sql.DB

	service Service
}

// NewHandler returns a new ticket handler.
func NewHandler(db *sql.DB, service Service) Handler {
	return Handler{
		db:      db,
		service: service,
	}
}

// CreateComment creates a new comment.
func (h *Handler) CreateComment() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		session, err := auth.GetSession(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		var comment model.CreateComment
		if err := json.NewDecoder(r.Body).Decode(&comment); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		if err := comment.Validate(); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(ctx, h.db)
		defer sqlTx.Rollback()

		commentID, err := h.service.CreateComment(ctx, session, comment)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		if err := sqlTx.Commit(); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusCreated, response.ID{ID: commentID})
	}
}

// CreatePost creates a post in an event.
func (h *Handler) CreatePost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		session, err := auth.GetSession(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		var post model.CreatePost
		if err := json.NewDecoder(r.Body).Decode(&post); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		if err := post.Validate(); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(ctx, h.db)
		defer sqlTx.Rollback()

		postID, err := h.service.CreatePost(ctx, session, eventID, post)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		if err := sqlTx.Commit(); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusCreated, response.ID{ID: postID})
	}
}

// DeleteComment removes a comment from a conversation/post.
func (h *Handler) DeleteComment() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		commentID, err := params.IDFromCtx(ctx, "comment_id")
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		session, err := auth.GetSession(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(ctx, h.db)
		defer sqlTx.Rollback()

		if err := h.service.DeleteComment(ctx, commentID, session); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		if err := sqlTx.Commit(); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.NoContent(w)
	}
}

// DeletePost removes a post from an event.
func (h *Handler) DeletePost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		ctxParams := httprouter.ParamsFromContext(ctx)
		eventID := ctxParams.ByName("id")
		postID := ctxParams.ByName("post_id")
		if err := validate.ULIDs(eventID, postID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(ctx, h.db)
		defer sqlTx.Rollback()

		if err := h.service.DeletePost(ctx, eventID, postID); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		if err := sqlTx.Commit(); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.NoContent(w)
	}
}

// GetComment gets a comment.
func (h *Handler) GetComment() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		session, err := auth.GetSession(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		commentID, err := params.IDFromCtx(ctx, "comment_id")
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		comment, err := h.service.GetComment(ctx, commentID, session.ID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, comment)
	}
}

// GetCommentLikes gets the users liking a post.
func (h *Handler) GetCommentLikes() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		commentID, err := params.IDFromCtx(ctx, "comment_id")
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		params, err := params.Parse(r.URL.RawQuery, model.T.Comment)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if params.Count {
			count, err := h.service.GetCommentLikesCount(ctx, commentID)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, err)
				return
			}

			response.JSONCount(w, http.StatusOK, "comment_likes_count", count)
			return
		}

		users, err := h.service.GetCommentLikes(ctx, commentID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		var nextCursor string
		if len(users) > 0 {
			nextCursor = users[len(users)-1].ID
		}

		response.JSONCursor(w, nextCursor, "users", users)
	}
}

// GetHomePosts returns a user's home posts.
func (h *Handler) GetHomePosts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		session, err := auth.GetSession(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		params, err := params.Parse(r.URL.RawQuery, model.T.Post)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		posts, err := h.service.GetHomePosts(ctx, session, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		var nextCursor string
		if len(posts) > 0 {
			nextCursor = posts[len(posts)-1].ID
		}

		response.JSONCursor(w, nextCursor, "posts", posts)
	}
}

// GetPost gets a post from an event.
func (h *Handler) GetPost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		session, err := auth.GetSession(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		ctxParams := httprouter.ParamsFromContext(ctx)
		eventID := ctxParams.ByName("id")
		postID := ctxParams.ByName("post_id")
		if err := validate.ULIDs(eventID, postID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		post, err := h.service.GetPost(ctx, eventID, postID, session.ID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, post)
	}
}

// GetPosts gets all the posts from an event.
func (h *Handler) GetPosts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		session, err := auth.GetSession(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		params, err := params.Parse(r.URL.RawQuery, model.T.Post)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		posts, err := h.service.GetPosts(ctx, eventID, session.ID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		var nextCursor string
		if len(posts) > 0 {
			nextCursor = posts[len(posts)-1].ID
		}

		response.JSONCursor(w, nextCursor, "posts", posts)
	}
}

// GetPostLikes gets the users liking a post.
func (h *Handler) GetPostLikes() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		postID, err := params.IDFromCtx(ctx, "post_id")
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		params, err := params.Parse(r.URL.RawQuery, model.T.Post)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if params.Count {
			count, err := h.service.GetPostLikesCount(ctx, postID)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, err)
				return
			}

			response.JSONCount(w, http.StatusOK, "post_likes_count", count)
			return
		}

		users, err := h.service.GetPostLikes(ctx, postID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		var nextCursor string
		if len(users) > 0 {
			nextCursor = users[len(users)-1].ID
		}

		response.JSONCursor(w, nextCursor, "users", users)
	}
}

// GetReplies gets all the comments in a post/comment.
func (h *Handler) GetReplies() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		session, err := auth.GetSession(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		parentID, err := params.IDFromCtx(ctx, "parent_id")
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		params, err := params.Parse(r.URL.RawQuery, model.T.Comment)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		comments, err := h.service.GetReplies(ctx, parentID, session.ID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		var nextCursor string
		if len(comments) > 0 {
			nextCursor = comments[len(comments)-1].ID
		}

		response.JSONCursor(w, nextCursor, "comments", comments)
	}
}

// LikeComment adds a like to a comment, if the like already exists, it removes it.
func (h *Handler) LikeComment() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		session, err := auth.GetSession(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		commentID, err := params.IDFromCtx(ctx, "comment_id")
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(ctx, h.db)
		defer sqlTx.Rollback()

		if err := h.service.LikeComment(ctx, commentID, session.ID); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		if err := sqlTx.Commit(); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.NoContent(w)
	}
}

// LikePost adds a like to a post, if the like already exists, it removes it.
func (h *Handler) LikePost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		session, err := auth.GetSession(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		postID, err := params.IDFromCtx(ctx, "post_id")
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(ctx, h.db)
		defer sqlTx.Rollback()

		if err := h.service.LikePost(ctx, postID, session.ID); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		if err := sqlTx.Commit(); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.NoContent(w)
	}
}

// UpdatePost updates an event's post.
func (h *Handler) UpdatePost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		postID, err := params.IDFromCtx(ctx, "post_id")
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		var post model.UpdatePost
		if err := json.NewDecoder(r.Body).Decode(&post); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		if err := post.Validate(); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if err := h.service.UpdatePost(ctx, postID, post); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, response.ID{ID: postID})
	}
}
