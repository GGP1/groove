package event

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/GGP1/groove/auth"
	"github.com/GGP1/groove/internal/cache"
	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/permissions"
	"github.com/GGP1/groove/internal/response"
	"github.com/GGP1/groove/internal/validate"
	"github.com/GGP1/groove/service/event/zone"

	"github.com/julienschmidt/httprouter"
)

// AccessZone checks if the authenticated user is allowed to enter the zone or not.
func (h *Handler) AccessZone() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		ctxParams := httprouter.ParamsFromContext(ctx)
		eventID := ctxParams.ByName("id")
		if err := validate.ULID(eventID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		name := strings.ToLower(ctxParams.ByName("name"))

		session, err := auth.GetSession(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		sqlTx := h.service.BeginSQLTx(ctx, true)
		defer sqlTx.Rollback()

		if err := h.privacyFilter(ctx, r, sqlTx, eventID); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		role, err := h.service.GetUserRole(ctx, sqlTx, eventID, session.ID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		zone, err := h.service.GetZone(ctx, sqlTx, eventID, name)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		userPermissionKeys := sliceToMap(role.PermissionKeys)
		if err := permissions.Require(userPermissionKeys, zone.RequiredPermissionKeys...); err != nil {
			response.Error(w, http.StatusForbidden, errAccessDenied)
			return
		}

		type resp struct {
			Access bool `json:"access,omitempty"`
		}
		response.JSON(w, http.StatusOK, resp{Access: true})
	}
}

// CreateZone creates a new zone inside an event.
func (h *Handler) CreateZone() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		var zone zone.Zone
		if err := json.NewDecoder(r.Body).Decode(&zone); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		sqlTx := h.service.BeginSQLTx(ctx, false)
		defer sqlTx.Rollback()

		if err := h.requirePermissions(ctx, r, sqlTx, eventID, permissions.ModifyZones); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		zone.Name = strings.ToLower(zone.Name)
		if err := h.service.CreateZone(ctx, sqlTx, eventID, zone); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		if err := sqlTx.Commit(); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, zone)
	})
}

// DeleteZone removes a zone from an event.
func (h *Handler) DeleteZone() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		ctxParams := httprouter.ParamsFromContext(ctx)
		eventID := ctxParams.ByName("id")
		if err := validate.ULID(eventID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		name := strings.ToLower(ctxParams.ByName("name"))

		errStatus, err := h.service.SQLTx(ctx, false, func(tx *sql.Tx) (int, error) {
			if err := h.requirePermissions(ctx, r, tx, eventID, permissions.ModifyZones); err != nil {
				return http.StatusForbidden, err
			}
			if err := h.service.DeleteZone(ctx, tx, eventID, name); err != nil {
				return http.StatusInternalServerError, err
			}
			return 0, nil
		})
		if err != nil {
			response.Error(w, errStatus, err)
			return
		}

		response.NoContent(w)
	}
}

// GetZone retrieves a zone in an event with the given name.
func (h *Handler) GetZone() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		ctxParams := httprouter.ParamsFromContext(ctx)
		eventID := ctxParams.ByName("id")
		name := strings.ToLower(ctxParams.ByName("name"))

		if err := validate.ULID(eventID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx := h.service.BeginSQLTx(ctx, true)
		defer sqlTx.Rollback()

		if err := h.privacyFilter(ctx, r, sqlTx, eventID); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		zone, err := h.service.GetZone(ctx, sqlTx, eventID, name)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, zone)
	})
}

// GetZones fetches all the zones from an event.
func (h *Handler) GetZones() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		cacheKey := cache.ZonesKey(eventID)
		if item, err := h.cache.Get(cacheKey); err == nil {
			response.EncodedJSON(w, item.Value)
			return
		}

		sqlTx := h.service.BeginSQLTx(ctx, true)
		defer sqlTx.Rollback()

		if err := h.privacyFilter(ctx, r, sqlTx, eventID); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		zones, err := h.service.GetZones(ctx, sqlTx, eventID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSONAndCache(h.cache, w, cacheKey, zones)
	})
}

// UpdateZone updates an event's zone.
func (h *Handler) UpdateZone() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		ctxParams := httprouter.ParamsFromContext(ctx)
		eventID := ctxParams.ByName("id")
		if err := validate.ULID(eventID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		name := strings.ToLower(ctxParams.ByName("name"))

		sqlTx := h.service.BeginSQLTx(ctx, false)
		defer sqlTx.Rollback()

		if err := h.requirePermissions(ctx, r, sqlTx, eventID, permissions.ModifyRoles); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		var zone zone.UpdateZone
		if err := json.NewDecoder(r.Body).Decode(&zone); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		if err := zone.Validate(); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if err := h.service.UpdateZone(ctx, sqlTx, eventID, name, zone); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		if err := sqlTx.Commit(); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, zone)
	}
}
