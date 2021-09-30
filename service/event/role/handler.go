package role

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
	"github.com/GGP1/groove/service/auth"
	"github.com/GGP1/groove/storage/postgres"

	"github.com/julienschmidt/httprouter"
)

// Handler handles ticket service endpoints.
type Handler struct {
	db    *sql.DB
	cache cache.Client

	service Service
}

// NewHandler returns a new ticket handler.
func NewHandler(db *sql.DB, cache cache.Client, service Service) Handler {
	return Handler{
		db:      db,
		cache:   cache,
		service: service,
	}
}

// ClonePermissions clones the permissions from one event to another.
func (h Handler) ClonePermissions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		var req struct {
			ExporterEventID string `json:"exporter_event_id,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		importerEventID := httprouter.ParamsFromContext(rctx).ByName("id")
		if err := validate.ULIDs(importerEventID, req.ExporterEventID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, false)
		defer sqlTx.Rollback()

		if err := h.service.RequirePermissions(ctx, r, req.ExporterEventID, permissions.ModifyPermissions); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}
		if err := h.service.RequirePermissions(ctx, r, importerEventID, permissions.ModifyPermissions); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}
		if err := h.service.ClonePermissions(ctx, req.ExporterEventID, importerEventID); err != nil {
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

// CloneRoles imports the roles from an event and saves them into another.
func (h Handler) CloneRoles() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		var req struct {
			ExporterEventID string `json:"exporter_event_id,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		importerEventID := httprouter.ParamsFromContext(rctx).ByName("id")
		if err := validate.ULIDs(importerEventID, req.ExporterEventID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, false)
		defer sqlTx.Rollback()

		// Verify the user has permissions in both events
		if err := h.service.RequirePermissions(ctx, r, req.ExporterEventID, permissions.ModifyRoles); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}
		if err := h.service.RequirePermissions(ctx, r, importerEventID, permissions.ModifyRoles); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}
		if err := h.service.CloneRoles(ctx, req.ExporterEventID, importerEventID); err != nil {
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

// CreatePermission creates a new permission inside an event.
func (h Handler) CreatePermission() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		eventID, err := params.IDFromCtx(rctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, false)
		defer sqlTx.Rollback()

		if err := h.service.RequirePermissions(ctx, r, eventID, permissions.ModifyPermissions); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		var permission Permission
		if err := json.NewDecoder(r.Body).Decode(&permission); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		permission.Key = strings.ToLower(permission.Key)
		if err := permission.Validate(); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if err := h.service.CreatePermission(ctx, eventID, permission); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		if err := sqlTx.Commit(); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, permission)
	})
}

// CreateRole creates a new role inside an event.
func (h Handler) CreateRole() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		eventID, err := params.IDFromCtx(rctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, false)
		defer sqlTx.Rollback()

		if err := h.service.RequirePermissions(ctx, r, eventID, permissions.ModifyRoles); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		var role Role
		if err := json.NewDecoder(r.Body).Decode(&role); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		role.Name = strings.ToLower(role.Name)
		if err := role.Validate(); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if err := h.service.CreateRole(ctx, eventID, role); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		if err := sqlTx.Commit(); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, role)
	})
}

// DeletePermission removes a permission from an event.
func (h Handler) DeletePermission() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		ctxParams := httprouter.ParamsFromContext(rctx)
		eventID := ctxParams.ByName("id")
		if err := validate.ULID(eventID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		key := strings.ToLower(ctxParams.ByName("key"))

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, false)
		defer sqlTx.Rollback()

		if err := h.service.RequirePermissions(ctx, r, eventID, permissions.ModifyPermissions); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}
		if err := h.service.DeletePermission(ctx, eventID, key); err != nil {
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

// DeleteRole removes a role from an event.
func (h Handler) DeleteRole() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		eventID, roleName, err := params.IDAndNameFromCtx(rctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, false)
		defer sqlTx.Rollback()

		if err := h.service.RequirePermissions(ctx, r, eventID, permissions.ModifyRoles); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}
		if err := h.service.DeleteRole(ctx, eventID, roleName); err != nil {
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

// GetMembers returns the members of an event.
func (h Handler) GetMembers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		eventID, err := params.IDFromCtx(rctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		params, err := params.Parse(r.URL.RawQuery, model.User)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, true)
		defer sqlTx.Rollback()

		if params.Count {
			count, err := h.service.GetMembersCount(ctx, eventID)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, err)
				return
			}

			response.JSONCount(w, http.StatusOK, "members_count", count)
			return
		}

		members, err := h.service.GetMembers(ctx, eventID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		var nextCursor string
		if len(members) > 0 {
			nextCursor = members[len(members)-1].ID
		}

		response.JSONCursor(w, nextCursor, "users", members)
	}
}

// GetMembersFriends returns the members of an event that are friends of a user.
func (h Handler) GetMembersFriends() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		eventID, err := params.IDFromCtx(rctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		params, err := params.Parse(r.URL.RawQuery, model.User)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		session, err := auth.GetSession(rctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, true)
		defer sqlTx.Rollback()

		if params.Count {
			count, err := h.service.GetMembersFriendsCount(ctx, eventID, session.ID)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, err)
				return
			}

			response.JSONCount(w, http.StatusOK, "members_friends_count", count)
			return
		}

		members, err := h.service.GetMembersFriends(ctx, eventID, session.ID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		var nextCursor string
		if len(members) > 0 {
			nextCursor = members[len(members)-1].ID
		}

		response.JSONCursor(w, nextCursor, "users", members)
	}
}

// GetPermission returns a permission from an event with the given key.
func (h Handler) GetPermission() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		ctxParams := httprouter.ParamsFromContext(rctx)
		eventID := ctxParams.ByName("id")
		if err := validate.ULID(eventID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		key := strings.ToLower(ctxParams.ByName("key"))

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, true)
		defer sqlTx.Rollback()

		if err := h.service.RequirePermissions(ctx, r, eventID, permissions.ModifyPermissions); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		permission, err := h.service.GetPermission(ctx, eventID, key)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, permission)
	}
}

// GetPermissions retrives all event's permissions.
func (h Handler) GetPermissions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		eventID, err := params.IDFromCtx(rctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		cacheKey := model.PermissionsCacheKey(eventID)
		if item, err := h.cache.Get(cacheKey); err == nil {
			response.EncodedJSON(w, item.Value)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, true)
		defer sqlTx.Rollback()

		if err := h.service.RequirePermissions(ctx, r, eventID, permissions.ModifyPermissions); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		permissions, err := h.service.GetPermissions(ctx, eventID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSONAndCache(h.cache, w, cacheKey, permissions)
	}
}

// GetRole returns a role from an event with the given name.
func (h Handler) GetRole() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		eventID, roleName, err := params.IDAndNameFromCtx(rctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, true)
		defer sqlTx.Rollback()

		if err := h.service.RequirePermissions(ctx, r, eventID, permissions.ModifyRoles); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		role, err := h.service.GetRole(ctx, eventID, roleName)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, role)
	}
}

// GetRoles retrives all event's roles.
func (h Handler) GetRoles() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		eventID, err := params.IDFromCtx(rctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		cacheKey := model.RolesCacheKey(eventID)
		if item, err := h.cache.Get(cacheKey); err == nil {
			response.EncodedJSON(w, item.Value)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, true)
		defer sqlTx.Rollback()

		if err := h.service.RequirePermissions(ctx, r, eventID, permissions.ModifyRoles); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		roles, err := h.service.GetRoles(ctx, eventID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSONAndCache(h.cache, w, cacheKey, roles)
	}
}

// GetUserRole gets the role of a user inside an event
func (h Handler) GetUserRole() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		eventID, err := params.IDFromCtx(rctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, true)
		defer sqlTx.Rollback()

		if err := h.service.PrivacyFilter(ctx, r, eventID); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		var reqBody model.UserID
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		if err := validate.ULID(reqBody.UserID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		role, err := h.service.GetUserRole(ctx, eventID, reqBody.UserID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, role)
	}
}

// SetRoles sets a role to n users inside the event passed.
func (h Handler) SetRoles() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		eventID, err := params.IDFromCtx(rctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, false)
		defer sqlTx.Rollback()

		if err := h.service.RequirePermissions(ctx, r, eventID, permissions.SetUserRole); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		var reqBody struct {
			UserIDs  []string `json:"user_ids,omitempty"`
			RoleName string   `json:"role_name,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		if err := validate.ULIDs(reqBody.UserIDs...); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		err = h.service.SetRoles(ctx, eventID, reqBody.RoleName, reqBody.UserIDs...)
		if err != nil {
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

// UpdatePermission updates a permission.
func (h Handler) UpdatePermission() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		ctxParams := httprouter.ParamsFromContext(rctx)
		eventID := ctxParams.ByName("id")
		if err := validate.ULID(eventID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		key := strings.ToLower(ctxParams.ByName("key"))

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, false)
		defer sqlTx.Rollback()

		if err := h.service.RequirePermissions(ctx, r, eventID, permissions.ModifyPermissions); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		var permission UpdatePermission
		if err := json.NewDecoder(r.Body).Decode(&permission); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		if err := permission.Validate(); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if err := h.service.UpdatePermission(ctx, eventID, key, permission); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		if err := sqlTx.Commit(); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, permission)
	}
}

// UpdateRole updates a role.
func (h Handler) UpdateRole() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		eventID, roleName, err := params.IDAndNameFromCtx(rctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, false)
		defer sqlTx.Rollback()

		if err := h.service.RequirePermissions(ctx, r, eventID, permissions.ModifyRoles); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		var role UpdateRole
		if err := json.NewDecoder(r.Body).Decode(&role); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		if err := role.Validate(); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if err := h.service.UpdateRole(ctx, eventID, roleName, role); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		if err := sqlTx.Commit(); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, role)
	}
}
