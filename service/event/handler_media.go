package event

import (
	"encoding/json"
	"net/http"

	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/response"
	"github.com/GGP1/groove/service/event/media"
)

// CreateMedia creates a media inside an event.
func (h *Handler) CreateMedia() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx := h.service.BeginSQLTx(ctx, false)
		defer sqlTx.Rollback()

		if err := h.requirePermissions(ctx, r, sqlTx, eventID, hostPermissions); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		var media media.CreateMedia
		if err := json.NewDecoder(r.Body).Decode(&media); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		if err := media.Validate(); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if err := h.service.CreateMedia(ctx, sqlTx, eventID, media); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		if err := sqlTx.Commit(); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, media)
	})
}

// GetMedia gets the media of an event.
func (h *Handler) GetMedia() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx := h.service.BeginSQLTx(ctx, true)
		defer sqlTx.Rollback()

		if err := h.privacyFilter(ctx, r, sqlTx, eventID); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		params, err := params.ParseQuery(r.URL.RawQuery, params.Media)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		mediaList, err := h.service.GetMedia(ctx, sqlTx, eventID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		var nextCursor string
		if len(mediaList) > 0 {
			nextCursor = mediaList[len(mediaList)-1].ID
		}

		type resp struct {
			NextCursor string        `json:"next_cursor,omitempty"`
			Media      []media.Media `json:"media,omitempty"`
		}
		response.JSON(w, http.StatusOK, resp{NextCursor: nextCursor, Media: mediaList})
	}
}

// UpdateMedia updates a media of an event.
func (h *Handler) UpdateMedia() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx := h.service.BeginSQLTx(ctx, false)
		defer sqlTx.Rollback()

		if err := h.requirePermissions(ctx, r, sqlTx, eventID, hostPermissions); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		var media media.Media
		if err := json.NewDecoder(r.Body).Decode(&media); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}
		defer r.Body.Close()

		if err := h.service.UpdateMedia(ctx, sqlTx, eventID, media); err != nil {
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
