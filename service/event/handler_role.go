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
	"github.com/GGP1/groove/service/event/role"

	"github.com/julienschmidt/httprouter"
)

// ClonePermissions clones the permissions from one event to another.
func (h *Handler) ClonePermissions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var req struct {
			ExporterEventID string `json:"exporter_event_id,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		importerEventID := httprouter.ParamsFromContext(ctx).ByName("id")
		if err := ulid.ValidateN(importerEventID, req.ExporterEventID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		errStatus, err := h.service.SQLTx(ctx, false, func(tx *sql.Tx) (int, error) {
			if err := h.requirePermissions(ctx, r, tx, req.ExporterEventID, permissions.ModifyPermissions); err != nil {
				return http.StatusForbidden, err
			}
			if err := h.requirePermissions(ctx, r, tx, importerEventID, permissions.ModifyPermissions); err != nil {
				return http.StatusForbidden, err
			}
			if err := h.service.ClonePermissions(ctx, tx, req.ExporterEventID, importerEventID); err != nil {
				return http.StatusInternalServerError, err
			}
			return 0, nil
		})
		if err != nil {
			response.Error(w, errStatus, err)
			return
		}

		response.JSONMessage(w, http.StatusOK, "Permissions cloned successfully")
	}
}

// CloneRoles imports the roles from an event and saves them into another.
func (h *Handler) CloneRoles() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var req struct {
			ExporterEventID string `json:"exporter_event_id,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		importerEventID := httprouter.ParamsFromContext(ctx).ByName("id")
		if err := ulid.ValidateN(importerEventID, req.ExporterEventID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		errStatus, err := h.service.SQLTx(ctx, false, func(tx *sql.Tx) (int, error) {
			// Verify that the user is a host in both events
			if err := h.requirePermissions(ctx, r, tx, req.ExporterEventID, permissions.ModifyRoles); err != nil {
				return http.StatusForbidden, err
			}
			if err := h.requirePermissions(ctx, r, tx, importerEventID, permissions.ModifyRoles); err != nil {
				return http.StatusForbidden, err
			}
			if err := h.service.CloneRoles(ctx, tx, req.ExporterEventID, importerEventID); err != nil {
				return http.StatusInternalServerError, err
			}
			return 0, nil
		})
		if err != nil {
			response.Error(w, errStatus, err)
			return
		}

		response.JSONMessage(w, http.StatusOK, "Roles cloned successfully")
	}
}

// CreatePermission creates a new permission inside an event.
func (h *Handler) CreatePermission() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx := h.service.BeginSQLTx(ctx, false)
		defer sqlTx.Rollback()

		if err := h.requirePermissions(ctx, r, sqlTx, eventID, permissions.ModifyPermissions); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		var permission role.Permission
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

		if err := h.service.CreatePermission(ctx, sqlTx, eventID, permission); err != nil {
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
func (h *Handler) CreateRole() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx := h.service.BeginSQLTx(ctx, false)
		defer sqlTx.Rollback()

		if err := h.requirePermissions(ctx, r, sqlTx, eventID, permissions.ModifyRoles); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		var role role.Role
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

		if err := h.service.CreateRole(ctx, sqlTx, eventID, role); err != nil {
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
func (h *Handler) DeletePermission() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		routerParams := httprouter.ParamsFromContext(ctx)
		eventID := routerParams.ByName("id")
		if err := ulid.Validate(eventID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		key := strings.ToLower(routerParams.ByName("key"))

		errStatus, err := h.service.SQLTx(ctx, false, func(tx *sql.Tx) (int, error) {
			if err := h.requirePermissions(ctx, r, tx, eventID, permissions.ModifyPermissions); err != nil {
				return http.StatusForbidden, err
			}
			if err := h.service.DeletePermission(ctx, tx, eventID, key); err != nil {
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

// DeleteRole removes a role from an event.
func (h *Handler) DeleteRole() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		routerParams := httprouter.ParamsFromContext(ctx)
		eventID := routerParams.ByName("id")
		if err := ulid.Validate(eventID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		roleName := strings.ToLower(routerParams.ByName("name"))

		errStatus, err := h.service.SQLTx(ctx, false, func(tx *sql.Tx) (int, error) {
			if err := h.requirePermissions(ctx, r, tx, eventID, permissions.ModifyRoles); err != nil {
				return http.StatusForbidden, err
			}
			if err := h.service.DeleteRole(ctx, tx, eventID, roleName); err != nil {
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

// GetPermissions retrives all event's permissions.
func (h *Handler) GetPermissions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		cacheKey := eventID + "_permissions"
		if item, err := h.mc.Get(cacheKey); err == nil {
			response.EncodedJSON(w, item.Value)
			return
		}

		sqlTx := h.service.BeginSQLTx(ctx, true)
		defer sqlTx.Rollback()

		if err := h.requirePermissions(ctx, r, sqlTx, eventID, permissions.ModifyPermissions); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		permissions, err := h.service.GetPermissions(ctx, sqlTx, eventID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSONAndCache(h.mc, w, cacheKey, permissions)
	}
}

// GetUserRole gets the role of a user inside an event
func (h *Handler) GetUserRole() http.HandlerFunc {
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

		var reqBody userIDBody
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		if err := ulid.Validate(reqBody.UserID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		role, err := h.service.GetUserRole(ctx, sqlTx, eventID, reqBody.UserID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, role)
	}
}

// GetRoles retrives all event's roles.
func (h *Handler) GetRoles() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		cacheKey := eventID + "_roles"
		if item, err := h.mc.Get(cacheKey); err == nil {
			response.EncodedJSON(w, item.Value)
			return
		}

		sqlTx := h.service.BeginSQLTx(ctx, true)
		defer sqlTx.Rollback()

		if err := h.requirePermissions(ctx, r, sqlTx, eventID, permissions.ModifyRoles); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		permissions, err := h.service.GetRoles(ctx, sqlTx, eventID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSONAndCache(h.mc, w, cacheKey, permissions)
	}
}

// SetRoles sets a role to n users inside the event passed.
func (h *Handler) SetRoles() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx := h.service.BeginSQLTx(ctx, false)
		defer sqlTx.Rollback()

		if err := h.requirePermissions(ctx, r, sqlTx, eventID, permissions.SetUserRole); err != nil {
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

		err = h.service.SetRoles(ctx, sqlTx, eventID, reqBody.RoleName, reqBody.UserIDs...)
		if err != nil {
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

// UpdatePermission updates a permission.
func (h *Handler) UpdatePermission() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		routerParams := httprouter.ParamsFromContext(ctx)
		eventID := routerParams.ByName("id")
		if err := ulid.Validate(eventID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		key := strings.ToLower(routerParams.ByName("key"))

		sqlTx := h.service.BeginSQLTx(ctx, false)
		defer sqlTx.Rollback()

		if err := h.requirePermissions(ctx, r, sqlTx, eventID, permissions.ModifyPermissions); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		var permission role.UpdatePermission
		if err := json.NewDecoder(r.Body).Decode(&permission); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		if err := permission.Validate(); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if err := h.service.UpdatePermission(ctx, sqlTx, eventID, key, permission); err != nil {
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
func (h *Handler) UpdateRole() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		routerParams := httprouter.ParamsFromContext(ctx)
		eventID := routerParams.ByName("id")
		if err := ulid.Validate(eventID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		name := strings.ToLower(routerParams.ByName("name"))

		sqlTx := h.service.BeginSQLTx(ctx, false)
		defer sqlTx.Rollback()

		if err := h.requirePermissions(ctx, r, sqlTx, eventID, permissions.ModifyRoles); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		var role role.UpdateRole
		if err := json.NewDecoder(r.Body).Decode(&role); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		if err := role.Validate(); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if err := h.service.UpdateRole(ctx, sqlTx, eventID, name, role); err != nil {
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
