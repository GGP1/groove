package post

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/permissions"
	"github.com/GGP1/groove/internal/response"
	"github.com/GGP1/groove/internal/validate"
	"github.com/GGP1/groove/model"
	"github.com/GGP1/groove/service/auth"
	"github.com/GGP1/groove/service/event/role"
	"github.com/GGP1/groove/storage/postgres"

	"github.com/julienschmidt/httprouter"
)

// Handler handles ticket service endpoints.
type Handler struct {
	db *sql.DB

	service     Service
	roleService role.Service
}

// NewHandler returns a new ticket handler.
func NewHandler(db *sql.DB, service Service, roleService role.Service) Handler {
	return Handler{
		db:          db,
		service:     service,
		roleService: roleService,
	}
}

// CreateComment creates a new comment.
func (h Handler) CreateComment() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		session, err := auth.GetSession(rctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		eventID, err := params.IDFromCtx(rctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		var comment CreateComment
		if err := json.NewDecoder(r.Body).Decode(&comment); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		if err := comment.Validate(); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, false)
		defer sqlTx.Rollback()

		if err := h.service.CreateComment(ctx, session, comment); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		if err := sqlTx.Commit(); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSONMessage(w, http.StatusCreated, eventID)
	}
}

// CreatePost creates a post in an event.
func (h Handler) CreatePost() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		session, err := auth.GetSession(rctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		eventID, err := params.IDFromCtx(rctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, false)
		defer sqlTx.Rollback()

		if err := h.roleService.RequirePermissions(ctx, r, eventID, permissions.ModifyPosts); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		var post CreatePost
		if err := json.NewDecoder(r.Body).Decode(&post); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		if err := post.Validate(); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if err := h.service.CreatePost(ctx, session, eventID, post); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		if err := sqlTx.Commit(); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSONMessage(w, http.StatusCreated, eventID)
	})
}

// DeleteComment removes a comment from a conversation/post.
func (h Handler) DeleteComment() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		commentID, err := params.IDFromCtx(rctx, "comment_id")
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, false)
		defer sqlTx.Rollback()

		if err := h.service.DeleteComment(ctx, commentID); err != nil {
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
func (h Handler) DeletePost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		ctxParams := httprouter.ParamsFromContext(rctx)
		eventID := ctxParams.ByName("id")
		postID := ctxParams.ByName("post_id")
		if err := validate.ULIDs(eventID, postID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, false)
		defer sqlTx.Rollback()

		if err := h.roleService.RequirePermissions(ctx, r, eventID, permissions.ModifyPosts); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

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
func (h Handler) GetComment() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		ctxParams := httprouter.ParamsFromContext(rctx)
		eventID := ctxParams.ByName("id")
		commentID := ctxParams.ByName("comment_id")
		if err := validate.ULIDs(eventID, commentID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, true)
		defer sqlTx.Rollback()

		if err := h.roleService.PrivacyFilter(ctx, r, eventID); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		comment, err := h.service.GetComment(ctx, commentID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, comment)
	}
}

// GetCommentLikes gets the users liking a post.
func (h Handler) GetCommentLikes() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		ctxParams := httprouter.ParamsFromContext(rctx)
		eventID := ctxParams.ByName("id")
		commentID := ctxParams.ByName("comment_id")
		if err := validate.ULIDs(eventID, commentID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, true)
		defer sqlTx.Rollback()

		if err := h.roleService.PrivacyFilter(ctx, r, eventID); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		params, err := params.Parse(r.URL.RawQuery, model.User)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
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

// GetPostComments gets all the comments in a post.
func (h Handler) GetPostComments() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		ctxParams := httprouter.ParamsFromContext(rctx)
		eventID := ctxParams.ByName("id")
		postID := ctxParams.ByName("post_id")
		if err := validate.ULIDs(eventID, postID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, true)
		defer sqlTx.Rollback()

		if err := h.roleService.PrivacyFilter(ctx, r, eventID); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		params, err := params.Parse(r.URL.RawQuery, model.Post)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		comments, err := h.service.GetPostComments(ctx, postID, params)
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

// GetPost gets a post from an event.
func (h Handler) GetPost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		ctxParams := httprouter.ParamsFromContext(rctx)
		eventID := ctxParams.ByName("id")
		postID := ctxParams.ByName("post_id")
		if err := validate.ULIDs(eventID, postID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, true)
		defer sqlTx.Rollback()

		if err := h.roleService.PrivacyFilter(ctx, r, eventID); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		post, err := h.service.GetPost(ctx, eventID, postID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, post)
	}
}

// GetPostLikes gets the users liking a post.
func (h Handler) GetPostLikes() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		ctxParams := httprouter.ParamsFromContext(rctx)
		eventID := ctxParams.ByName("id")
		postID := ctxParams.ByName("post_id")
		if err := validate.ULIDs(eventID, postID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, true)
		defer sqlTx.Rollback()

		if err := h.roleService.PrivacyFilter(ctx, r, eventID); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		params, err := params.Parse(r.URL.RawQuery, model.User)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		users, err := h.service.GetPostLikes(ctx, postID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		var nextCursor string
		if len(users) > 1 {
			nextCursor = users[len(users)-1].ID
		}

		response.JSONCursor(w, nextCursor, "users", users)
	}
}

// GetPosts gets all the posts from an event.
func (h Handler) GetPosts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		eventID, err := params.IDFromCtx(rctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, true)
		defer sqlTx.Rollback()

		if err := h.roleService.PrivacyFilter(ctx, r, eventID); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		params, err := params.Parse(r.URL.RawQuery, model.Post)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		posts, err := h.service.GetPosts(ctx, eventID, params)
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

// LikeComment adds a like to a comment, if the like already exists, it removes it.
func (h Handler) LikeComment() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		session, err := auth.GetSession(rctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		ctxParams := httprouter.ParamsFromContext(rctx)
		eventID := ctxParams.ByName("id")
		commentID := ctxParams.ByName("comment_id")
		if err := validate.ULIDs(eventID, commentID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, false)
		defer sqlTx.Rollback()

		if err := h.roleService.PrivacyFilter(ctx, r, eventID); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		if err := h.service.LikeComment(ctx, commentID, session.ID); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		if err := sqlTx.Commit(); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		type res struct {
			CommentID string `json:"comment_id,omitempty"`
			Predicate string `json:"predicate,omitempty"`
			UserID    string `json:"user_id,omitempty"`
		}
		response.JSON(w, http.StatusOK, res{
			CommentID: commentID,
			Predicate: "liked_by",
			UserID:    session.ID,
		})
	}
}

// LikePost adds a like to a post, if the like already exists, it removes it.
func (h Handler) LikePost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		session, err := auth.GetSession(rctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		ctxParams := httprouter.ParamsFromContext(rctx)
		eventID := ctxParams.ByName("id")
		postID := ctxParams.ByName("post_id")
		if err := validate.ULIDs(eventID, postID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, false)
		defer sqlTx.Rollback()

		if err := h.roleService.PrivacyFilter(ctx, r, eventID); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		if err := h.service.LikePost(ctx, postID, session.ID); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		if err := sqlTx.Commit(); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		type res struct {
			PostID    string `json:"comment_id,omitempty"`
			Predicate string `json:"predicate,omitempty"`
			UserID    string `json:"user_id,omitempty"`
		}
		response.JSON(w, http.StatusOK, res{
			PostID:    postID,
			Predicate: "liked_by",
			UserID:    session.ID,
		})
	}
}

// UpdatePost updates an event's post.
func (h Handler) UpdatePost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		ctxParams := httprouter.ParamsFromContext(rctx)
		eventID := ctxParams.ByName("id")
		postID := ctxParams.ByName("post_id")
		if err := validate.ULIDs(eventID, postID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, false)
		defer sqlTx.Rollback()

		if err := h.roleService.RequirePermissions(ctx, r, eventID, permissions.ModifyPosts); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		var post UpdatePost
		if err := json.NewDecoder(r.Body).Decode(&post); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		if err := post.Validate(); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if err := h.service.UpdatePost(ctx, eventID, postID, post); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		if err := sqlTx.Commit(); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSONMessage(w, http.StatusOK, eventID)
	}
}
