package zone

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/GGP1/groove/internal/cache"
	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/permissions"
	"github.com/GGP1/groove/internal/response"
	"github.com/GGP1/groove/internal/validate"
	"github.com/GGP1/groove/model"
	"github.com/GGP1/groove/service/event/role"
	"github.com/GGP1/groove/storage/postgres"
)

// Handler handles zone service endpoints.
type Handler struct {
	db    *sql.DB
	cache cache.Client

	service     Service
	roleService role.Service
}

// NewHandler returns a new zone handler.
func NewHandler(db *sql.DB, cache cache.Client, service Service, roleService role.Service) Handler {
	return Handler{
		db:          db,
		cache:       cache,
		service:     service,
		roleService: roleService,
	}
}

// Access checks if the authenticated user is allowed to enter the zone or not.
func (h Handler) Access() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		eventID, zoneName, err := params.IDAndNameFromCtx(rctx)
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

		zone, err := h.service.GetByName(ctx, eventID, zoneName)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		if err := h.roleService.RequirePermissions(ctx, r, eventID, zone.RequiredPermissionKeys...); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		type resp struct {
			Access bool `json:"access,omitempty"`
		}
		response.JSON(w, http.StatusOK, resp{Access: true})
	}
}

// Create creates a new zone inside an event.
func (h Handler) Create() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		eventID, err := params.IDFromCtx(rctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		var zone Zone
		if err := json.NewDecoder(r.Body).Decode(&zone); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, true)
		defer sqlTx.Rollback()

		if err := h.roleService.RequirePermissions(ctx, r, eventID, permissions.ModifyZones); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		zone.Name = strings.ToLower(zone.Name)
		if err := h.service.Create(ctx, eventID, zone); err != nil {
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

// Delete removes a zone from an event.
func (h Handler) Delete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		eventID, zoneName, err := params.IDAndNameFromCtx(rctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, true)
		defer sqlTx.Rollback()

		if err := h.roleService.RequirePermissions(ctx, r, eventID, permissions.ModifyZones); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		if err := h.service.Delete(ctx, eventID, zoneName); err != nil {
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

// GetByName retrieves a zone in an event with the given name.
func (h Handler) GetByName() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		eventID, zoneName, err := params.IDAndNameFromCtx(rctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if err := validate.ULID(eventID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, true)
		defer sqlTx.Rollback()

		if err := h.roleService.PrivacyFilter(ctx, r, eventID); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		zone, err := h.service.GetByName(ctx, eventID, zoneName)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, zone)
	})
}

// Get fetches all the zones from an event.
func (h Handler) Get() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		eventID, err := params.IDFromCtx(rctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		cacheKey := model.ZonesCacheKey(eventID)
		if item, err := h.cache.Get(cacheKey); err == nil {
			response.EncodedJSON(w, item.Value)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, true)
		defer sqlTx.Rollback()

		if err := h.roleService.PrivacyFilter(ctx, r, eventID); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		zones, err := h.service.Get(ctx, eventID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSONAndCache(h.cache, w, cacheKey, zones)
	})
}

// Update updates an event's zone.
func (h Handler) Update() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		eventID, zoneName, err := params.IDAndNameFromCtx(rctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, true)
		defer sqlTx.Rollback()

		if err := h.roleService.RequirePermissions(ctx, r, eventID, permissions.ModifyRoles); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		var zone UpdateZone
		if err := json.NewDecoder(r.Body).Decode(&zone); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		if err := zone.Validate(); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if err := h.service.Update(ctx, eventID, zoneName, zone); err != nil {
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
