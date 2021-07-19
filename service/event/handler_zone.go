package event

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/permissions"
	"github.com/GGP1/groove/internal/response"
	"github.com/GGP1/groove/internal/ulid"
	"github.com/GGP1/groove/service/event/zone"

	"github.com/julienschmidt/httprouter"
)

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

		routerParams := httprouter.ParamsFromContext(ctx)
		eventID := routerParams.ByName("id")
		if err := ulid.Validate(eventID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		name := routerParams.ByName("name")

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

		response.JSONMessage(w, http.StatusOK, eventID)
	}
}

// GetZoneByName retrieves a zone in an event with the given name.
func (h *Handler) GetZoneByName() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		routerParams := httprouter.ParamsFromContext(ctx)
		eventID := routerParams.ByName("id")
		name := routerParams.ByName("name")

		if err := ulid.Validate(eventID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx := h.service.BeginSQLTx(ctx, true)
		defer sqlTx.Rollback()

		if err := h.privacyFilter(ctx, r, sqlTx, eventID); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		zone, err := h.service.GetZoneByName(ctx, sqlTx, eventID, name)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, zone)
	})
}

// GetZones ..
func (h *Handler) GetZones() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		cacheKey := eventID + "_zones"
		if item, err := h.mc.Get(cacheKey); err == nil {
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

		response.JSONAndCache(h.mc, w, cacheKey, zones)
	})
}
