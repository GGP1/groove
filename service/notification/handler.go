package notification

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/response"
	"github.com/GGP1/groove/model"
	"github.com/GGP1/groove/service/auth"
	"github.com/GGP1/groove/storage/postgres"
)

// Handler handles notifications endpoint.
type Handler struct {
	db      *sql.DB
	service Service
}

// NewHandler returns a new notifications handler.
func NewHandler(db *sql.DB, service Service) Handler {
	return Handler{db: db, service: service}
}

// Answer handles the accept or decline of a notification.
func (h Handler) Answer() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		session, err := auth.GetSession(rctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		id, err := params.IDFromCtx(rctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		var accepted bool
		if err := json.NewDecoder(r.Body).Decode(&accepted); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, false)
		defer sqlTx.Rollback()

		if err := h.service.Answer(ctx, id, session.ID, accepted); err != nil {
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

// GetFromUser returns a user's notifications.
func (h Handler) GetFromUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		session, err := auth.GetSession(rctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		userID, err := params.IDFromCtx(rctx, "user_id")
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if session.ID != userID {
			response.Error(w, http.StatusForbidden, errors.New("access denied"))
			return
		}

		params, err := params.Parse(r.URL.RawQuery, model.Notification)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if params.Count {
			count, err := h.service.GetFromUserCount(rctx, userID)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, err)
				return
			}

			response.JSONCount(w, http.StatusOK, "notifications_count", count)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, false)
		defer sqlTx.Rollback()

		notifications, err := h.service.GetFromUser(ctx, userID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		if err := sqlTx.Commit(); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		var nextCursor string
		if len(notifications) > 0 {
			nextCursor = notifications[len(notifications)-1].ID
		}
		response.JSONCursor(w, nextCursor, "notifications", notifications)
	}
}
