package event

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/GGP1/groove/auth"
	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/permissions"
	"github.com/GGP1/groove/internal/response"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
)

var errAccessDenied = errors.New("Access denied")

type userIDBody struct {
	UserID string `json:"user_id,omitempty"`
}

type edgeMuResponse struct {
	EventID   string    `json:"event_id,omitempty"`
	Predicate predicate `json:"predicate,omitempty"`
	UserID    string    `json:"user_id,omitempty"`
}

// Handler handles events endpoints.
type Handler struct {
	service Service
	cache   *memcache.Client
}

// NewHandler returns an event handler.
func NewHandler(service Service, cache *memcache.Client) Handler {
	return Handler{
		service: service,
		cache:   cache,
	}
}

// AddBanned bans a user in an event.
func (h *Handler) AddBanned() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var reqBody userIDBody
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		sessionInfo, err := auth.GetSessionInfo(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		eventID := httprouter.ParamsFromContext(ctx).ByName("id")
		if err := params.ValidateUUIDs(eventID, reqBody.UserID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		errStatus, err := h.service.SQLTx(ctx, true, func(tx *sql.Tx) (int, error) {
			if err := h.requireHost(ctx, tx, sessionInfo.ID, eventID); err != nil {
				return http.StatusForbidden, err
			}
			return 0, nil
		})
		if err != nil {
			response.Error(w, errStatus, err)
			return
		}

		if err := h.service.AddEdge(ctx, eventID, Banned, reqBody.UserID); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, edgeMuResponse{
			EventID:   eventID,
			Predicate: Banned,
			UserID:    reqBody.UserID,
		})
	}
}

// AddConfirmed confirms a user in an event.
func (h *Handler) AddConfirmed() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var reqBody userIDBody
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		sessionInfo, err := auth.GetSessionInfo(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		eventID := httprouter.ParamsFromContext(ctx).ByName("id")
		if err := params.ValidateUUIDs(eventID, reqBody.UserID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		errStatus, err := h.service.SQLTx(ctx, false, func(tx *sql.Tx) (int, error) {
			err := h.requirePermissions(ctx, tx, sessionInfo.ID, eventID, []string{permissions.InviteUsers})
			if err != nil {
				return http.StatusForbidden, err
			}

			if err := h.service.AddEdge(ctx, eventID, Confirmed, reqBody.UserID); err != nil {
				return http.StatusInternalServerError, err
			}

			err = h.service.SetRole(ctx, tx, eventID, reqBody.UserID, permissions.Attendant)
			if err != nil {
				return http.StatusInternalServerError, err
			}

			return 0, nil
		})
		if err != nil {
			response.Error(w, errStatus, err)
			return
		}

		response.JSON(w, http.StatusOK, edgeMuResponse{
			EventID:   eventID,
			Predicate: Confirmed,
			UserID:    reqBody.UserID,
		})
	}
}

// AddInvited invites a user to an event.
func (h *Handler) AddInvited() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var reqBody userIDBody
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		eventID := httprouter.ParamsFromContext(ctx).ByName("id")
		if err := params.ValidateUUIDs(eventID, reqBody.UserID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sessionInfo, err := auth.GetSessionInfo(ctx, r)
		if err != nil {
			response.Error(w, http.StatusUnauthorized, err)
			return
		}

		errStatus, err := h.service.SQLTx(ctx, true, func(tx *sql.Tx) (int, error) {
			err := h.requirePermissions(ctx, tx, sessionInfo.ID, eventID, []string{permissions.InviteUsers})
			if err != nil {
				return http.StatusForbidden, err
			}

			// Check the invited user settings to verify the invitation can be performed
			// TODO: this should be in the user service
			canInvite, err := h.service.CanInvite(ctx, tx, sessionInfo.ID, reqBody.UserID)
			if err != nil {
				return http.StatusInternalServerError, err
			}
			if !canInvite {
				return http.StatusForbidden, errors.New("user settings do not allow this invitation")
			}

			return 0, nil
		})
		if err != nil {
			response.Error(w, errStatus, err)
			return
		}

		if err := h.service.AddEdge(ctx, eventID, Invited, reqBody.UserID); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, edgeMuResponse{
			EventID:   eventID,
			Predicate: Invited,
			UserID:    reqBody.UserID,
		})
	}
}

// AddLike adds the like of a user to an event.
func (h *Handler) AddLike() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var reqBody userIDBody
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		eventID := httprouter.ParamsFromContext(ctx).ByName("id")
		if err := params.ValidateUUIDs(eventID, reqBody.UserID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sessionInfo, err := auth.GetSessionInfo(ctx, r)
		if err != nil {
			response.Error(w, http.StatusUnauthorized, err)
			return
		}

		errStatus, err := h.service.SQLTx(ctx, false, func(tx *sql.Tx) (int, error) {
			role, err := h.service.GetUserRole(ctx, tx, eventID, sessionInfo.ID)
			if err != nil {
				return http.StatusInternalServerError, err
			}

			if role.Name != permissions.Attendant {
				return http.StatusForbidden, errors.New("must have attended to the event to like it")
			}
			return 0, nil
		})
		if err != nil {
			response.Error(w, errStatus, err)
			return
		}

		if err := h.service.AddEdge(ctx, eventID, LikedBy, reqBody.UserID); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, edgeMuResponse{
			EventID:   eventID,
			Predicate: LikedBy,
			UserID:    reqBody.UserID,
		})
	}
}

// ClonePermissions clones the permissions from one event to another.
func (h *Handler) ClonePermissions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		sessionInfo, err := auth.GetSessionInfo(ctx, r)
		if err != nil {
			response.Error(w, http.StatusUnauthorized, err)
			return
		}

		var req struct {
			ImporterEventID string `json:"importer_event_id,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		exporterEventID, err := params.UUIDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		errStatus, err := h.service.SQLTx(ctx, false, func(tx *sql.Tx) (int, error) {
			if err := h.requireHost(ctx, tx, sessionInfo.ID, exporterEventID, req.ImporterEventID); err != nil {
				return http.StatusForbidden, err
			}

			if err := h.service.ClonePermissions(ctx, tx, exporterEventID, req.ImporterEventID); err != nil {
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

// CloneRoles imports the roles from an event and saves them into another, it also clones the permissions.
func (h *Handler) CloneRoles() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		sessionInfo, err := auth.GetSessionInfo(ctx, r)
		if err != nil {
			response.Error(w, http.StatusUnauthorized, err)
			return
		}

		var req struct {
			ImporterEventID string `json:"importer_event_id,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		exporterEventID, err := params.UUIDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		errStatus, err := h.service.SQLTx(ctx, false, func(tx *sql.Tx) (int, error) {
			// Verify that the user is a host in both events
			if err := h.requireHost(ctx, tx, sessionInfo.ID, exporterEventID, req.ImporterEventID); err != nil {
				return http.StatusForbidden, err
			}

			if err := h.service.ClonePermissions(ctx, tx, exporterEventID, req.ImporterEventID); err != nil {
				return http.StatusInternalServerError, err
			}

			if err := h.service.CloneRoles(ctx, tx, exporterEventID, req.ImporterEventID); err != nil {
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

// Create creates an event.
func (h *Handler) Create() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var event CreateEvent
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		if err := event.Validate(); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		eventID := uuid.NewString()
		if err := h.service.Create(ctx, eventID, event); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		type eventResp struct {
			ID    string      `json:"id,omitempty"`
			Event CreateEvent `json:"event,omitempty"`
		}

		response.JSON(w, http.StatusCreated, eventResp{
			ID:    eventID,
			Event: event,
		})
	}
}

// CreateMedia creates a media inside an event.
func (h *Handler) CreateMedia() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.UUIDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		var media Media
		if err := json.NewDecoder(r.Body).Decode(&media); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		if err := media.Validate(); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sessionInfo, err := auth.GetSessionInfo(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		errStatus, err := h.service.SQLTx(ctx, false, func(tx *sql.Tx) (int, error) {
			if err := h.requireHost(ctx, tx, sessionInfo.ID); err != nil {
				return http.StatusForbidden, err
			}
			if err := h.service.CreateMedia(ctx, tx, eventID, media); err != nil {
				return http.StatusInternalServerError, err
			}

			return 0, nil
		})
		if err != nil {
			response.Error(w, errStatus, err)
			return
		}

		response.JSON(w, http.StatusOK, media)
	})
}

// CreatePermission creates a new permission inside an event.
func (h *Handler) CreatePermission() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.UUIDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		var permission Permission
		if err := json.NewDecoder(r.Body).Decode(&permission); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		if err := permission.Validate(); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sessionInfo, err := auth.GetSessionInfo(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		errStatus, err := h.service.SQLTx(ctx, false, func(tx *sql.Tx) (int, error) {
			if err := h.requireHost(ctx, tx, sessionInfo.ID); err != nil {
				return http.StatusForbidden, err
			}

			permission.Key = strings.ToLower(permission.Key)
			if err := h.service.CreatePermission(ctx, tx, eventID, permission); err != nil {
				return http.StatusInternalServerError, err
			}
			return 0, nil
		})
		if err != nil {
			response.Error(w, errStatus, err)
			return
		}

		response.JSON(w, http.StatusOK, permission)
	})
}

// CreateProduct creates an image/video inside an event.
func (h *Handler) CreateProduct() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.UUIDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		var product Product
		if err := json.NewDecoder(r.Body).Decode(&product); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		if err := product.Validate(); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sessionInfo, err := auth.GetSessionInfo(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		errStatus, err := h.service.SQLTx(ctx, false, func(tx *sql.Tx) (int, error) {
			if err := h.requireHost(ctx, tx, sessionInfo.ID); err != nil {
				return http.StatusForbidden, err
			}
			if err := h.service.CreateProduct(ctx, tx, eventID, product); err != nil {
				return http.StatusInternalServerError, err
			}

			return 0, nil
		})
		if err != nil {
			response.Error(w, errStatus, err)
			return
		}

		response.JSON(w, http.StatusOK, product)
	})
}

// CreateRole creates a new role inside an event.
func (h *Handler) CreateRole() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.UUIDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		var role Role
		if err := json.NewDecoder(r.Body).Decode(&role); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		if err := role.Validate(); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sessionInfo, err := auth.GetSessionInfo(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		errStatus, err := h.service.SQLTx(ctx, false, func(tx *sql.Tx) (int, error) {
			permKeys := []string{permissions.CreateRole}
			if err := h.requirePermissions(ctx, tx, sessionInfo.ID, eventID, permKeys); err != nil {
				return http.StatusForbidden, err
			}

			role.Name = strings.ToLower(role.Name)
			if err := h.service.CreateRole(ctx, tx, eventID, role); err != nil {
				return http.StatusInternalServerError, err
			}
			return 0, nil
		})
		if err != nil {
			response.Error(w, errStatus, err)
			return
		}

		response.JSON(w, http.StatusOK, role)
	})
}

// Delete removes an event from the system.
func (h *Handler) Delete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.UUIDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sessionInfo, err := auth.GetSessionInfo(ctx, r)
		if err != nil {
			response.Error(w, http.StatusUnauthorized, err)
			return
		}

		errStatus, err := h.service.SQLTx(ctx, false, func(tx *sql.Tx) (int, error) {
			if err := h.requireHost(ctx, tx, sessionInfo.ID, eventID); err != nil {
				return http.StatusForbidden, err
			}

			if err := h.service.Delete(ctx, tx, eventID); err != nil {
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

// GetBans gets an event's banned users.
func (h *Handler) GetBans() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.UUIDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		params, err := params.ParseQuery(r.URL.RawQuery, params.User)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if params.Count {
			count, err := h.service.GetBannedCount(ctx, eventID)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, err)
				return
			}

			response.JSONCount(w, http.StatusOK, count)
			return
		}

		sessionInfo, err := auth.GetSessionInfo(ctx, r)
		if err != nil {
			response.Error(w, http.StatusUnauthorized, err)
			return
		}

		sqlTx := h.service.BeginSQLTx(ctx, true)
		defer sqlTx.Rollback()

		if err := h.privacyFilter(ctx, sqlTx, sessionInfo.ID, eventID); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		bans, err := h.service.GetBanned(ctx, sqlTx, eventID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, bans)
	}
}

// GetBansFollowing returns event banned users that are followed by the user passed.
func (h *Handler) GetBansFollowing() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		routerParams := httprouter.ParamsFromContext(ctx)
		eventID := routerParams.ByName("id")
		userID := routerParams.ByName("user_id")
		if err := params.ValidateUUIDs(eventID, userID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		params, err := params.ParseQuery(r.URL.RawQuery, params.Event)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx := h.service.BeginSQLTx(ctx, true)
		users, err := h.service.GetBannedFollowing(ctx, sqlTx, eventID, userID, params)
		if err != nil {
			sqlTx.Rollback()
			response.Error(w, http.StatusInternalServerError, err)
			return
		}
		sqlTx.Rollback()

		response.JSON(w, http.StatusOK, users)
	}
}

// GetByID gets an event by its id.
func (h *Handler) GetByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.UUIDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if item, err := h.cache.Get(eventID); err == nil {
			response.EncodedJSON(w, item.Value)
			return
		}

		sessionInfo, err := auth.GetSessionInfo(ctx, r)
		if err != nil {
			response.Error(w, http.StatusUnauthorized, err)
			return
		}

		sqlTx := h.service.BeginSQLTx(ctx, true)
		defer sqlTx.Rollback()

		if err := h.privacyFilter(ctx, sqlTx, sessionInfo.ID, eventID); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		event, err := h.service.GetByID(ctx, sqlTx, eventID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSONAndCache(h.cache, w, eventID, event)
	}
}

// GetConfirmed gets an event's confirmed users.
func (h *Handler) GetConfirmed() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.UUIDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		params, err := params.ParseQuery(r.URL.RawQuery, params.User)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if params.Count {
			count, err := h.service.GetConfirmedCount(ctx, eventID)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, err)
				return
			}

			response.JSONCount(w, http.StatusOK, count)
			return
		}

		sessionInfo, err := auth.GetSessionInfo(ctx, r)
		if err != nil {
			response.Error(w, http.StatusUnauthorized, err)
			return
		}

		sqlTx := h.service.BeginSQLTx(ctx, true)
		defer sqlTx.Rollback()

		if err := h.privacyFilter(ctx, sqlTx, sessionInfo.ID, eventID); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		confirmed, err := h.service.GetConfirmed(ctx, sqlTx, eventID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, confirmed)
	}
}

// GetConfirmedFollowing returns event confirmed users that are followed by the user passed.
func (h *Handler) GetConfirmedFollowing() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		routerParams := httprouter.ParamsFromContext(ctx)
		eventID := routerParams.ByName("id")
		userID := routerParams.ByName("user_id")
		if err := params.ValidateUUIDs(eventID, userID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		params, err := params.ParseQuery(r.URL.RawQuery, params.Event)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx := h.service.BeginSQLTx(ctx, true)
		users, err := h.service.GetConfirmedFollowing(ctx, sqlTx, eventID, userID, params)
		if err != nil {
			sqlTx.Rollback()
			response.Error(w, http.StatusInternalServerError, err)
			return
		}
		sqlTx.Rollback()

		response.JSON(w, http.StatusOK, users)
	}
}

// GetHosts gets an event's host users.
func (h *Handler) GetHosts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.UUIDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		params, err := params.ParseQuery(r.URL.RawQuery, params.User)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if params.Count {
			count, err := h.service.GetInvitedCount(ctx, eventID)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, err)
				return
			}

			response.JSONCount(w, http.StatusOK, count)
			return
		}

		sessionInfo, err := auth.GetSessionInfo(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		sqlTx := h.service.BeginSQLTx(ctx, true)
		defer sqlTx.Rollback()

		if err := h.privacyFilter(ctx, sqlTx, sessionInfo.ID, eventID); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		hosts, err := h.service.GetHosts(ctx, sqlTx, eventID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, hosts)
	}
}

// GetInvited gets an event's invited users.
func (h *Handler) GetInvited() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.UUIDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		params, err := params.ParseQuery(r.URL.RawQuery, params.User)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if params.Count {
			count, err := h.service.GetInvitedCount(ctx, eventID)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, err)
				return
			}

			response.JSONCount(w, http.StatusOK, count)
			return
		}

		sessionInfo, err := auth.GetSessionInfo(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		sqlTx := h.service.BeginSQLTx(ctx, true)
		defer sqlTx.Rollback()

		if err := h.privacyFilter(ctx, sqlTx, sessionInfo.ID, eventID); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		invited, err := h.service.GetInvited(ctx, sqlTx, eventID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, invited)
	}
}

// GetInvitedFollowing returns event invited users that are followed by the user passed.
func (h *Handler) GetInvitedFollowing() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		routerParams := httprouter.ParamsFromContext(ctx)
		eventID := routerParams.ByName("id")
		userID := routerParams.ByName("user_id")
		if err := params.ValidateUUIDs(eventID, userID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		params, err := params.ParseQuery(r.URL.RawQuery, params.Event)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx := h.service.BeginSQLTx(ctx, true)
		users, err := h.service.GetInvitedFollowing(ctx, sqlTx, eventID, userID, params)
		if err != nil {
			sqlTx.Rollback()
			response.Error(w, http.StatusInternalServerError, err)
			return
		}
		sqlTx.Rollback()

		response.JSON(w, http.StatusOK, users)
	}
}

// GetLikes gets the users liking an event.
func (h *Handler) GetLikes() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.UUIDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		params, err := params.ParseQuery(r.URL.RawQuery, params.User)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if params.Count {
			count, err := h.service.GetLikedByCount(ctx, eventID)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, err)
				return
			}

			response.JSONCount(w, http.StatusOK, count)
			return
		}

		sessionInfo, err := auth.GetSessionInfo(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		sqlTx := h.service.BeginSQLTx(ctx, true)
		defer sqlTx.Rollback()

		if err := h.privacyFilter(ctx, sqlTx, sessionInfo.ID, eventID); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		likes, err := h.service.GetLikedBy(ctx, sqlTx, eventID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, likes)
	}
}

// GetLikesFollowing returns users liking the event that are followed by the user passed.
func (h *Handler) GetLikesFollowing() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		routerParams := httprouter.ParamsFromContext(ctx)
		eventID := routerParams.ByName("id")
		userID := routerParams.ByName("user_id")
		if err := params.ValidateUUIDs(eventID, userID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		params, err := params.ParseQuery(r.URL.RawQuery, params.Event)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx := h.service.BeginSQLTx(ctx, true)
		users, err := h.service.GetLikedByFollowing(ctx, sqlTx, eventID, userID, params)
		if err != nil {
			sqlTx.Rollback()
			response.Error(w, http.StatusInternalServerError, err)
			return
		}
		sqlTx.Rollback()

		response.JSON(w, http.StatusOK, users)
	}
}

// GetMedia gets the media of an event.
func (h *Handler) GetMedia() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.UUIDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		params, err := params.ParseQuery(r.URL.RawQuery, params.Media)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sessionInfo, err := auth.GetSessionInfo(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		sqlTx := h.service.BeginSQLTx(ctx, true)
		defer sqlTx.Rollback()

		if err := h.privacyFilter(ctx, sqlTx, sessionInfo.ID, eventID); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		cacheKey := eventID + "_media"
		if item, err := h.cache.Get(cacheKey); err == nil {
			response.EncodedJSON(w, item.Value)
			return
		}

		media, err := h.service.GetMedia(ctx, sqlTx, eventID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSONAndCache(h.cache, w, cacheKey, media)
	}
}

// GetPermissions retrives all event's permissions.
func (h *Handler) GetPermissions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.UUIDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sessionInfo, err := auth.GetSessionInfo(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		sqlTx := h.service.BeginSQLTx(ctx, true)
		defer sqlTx.Rollback()

		if err := h.requireHost(ctx, sqlTx, sessionInfo.ID, eventID); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		cacheKey := eventID + "_permissions"
		if item, err := h.cache.Get(cacheKey); err == nil {
			response.EncodedJSON(w, item.Value)
			return
		}

		permissions, err := h.service.GetPermissions(ctx, sqlTx, eventID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSONAndCache(h.cache, w, cacheKey, permissions)
	}
}

// GetProducts gets the products of an event.
func (h *Handler) GetProducts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.UUIDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		params, err := params.ParseQuery(r.URL.RawQuery, params.Media)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		sessionInfo, err := auth.GetSessionInfo(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		sqlTx := h.service.BeginSQLTx(ctx, true)
		defer sqlTx.Rollback()

		if err := h.privacyFilter(ctx, sqlTx, sessionInfo.ID, eventID); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		cacheKey := eventID + "_products"
		if item, err := h.cache.Get(cacheKey); err == nil {
			response.EncodedJSON(w, item.Value)
			return
		}

		products, err := h.service.GetProducts(ctx, sqlTx, eventID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSONAndCache(h.cache, w, cacheKey, products)
	}
}

// GetRole gets the role of a user inside an event
func (h *Handler) GetRole() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var reqBody userIDBody
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		eventID := httprouter.ParamsFromContext(ctx).ByName("id")
		if err := params.ValidateUUIDs(eventID, reqBody.UserID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sessionInfo, err := auth.GetSessionInfo(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		sqlTx := h.service.BeginSQLTx(ctx, true)
		defer sqlTx.Rollback()

		if err := h.privacyFilter(ctx, sqlTx, sessionInfo.ID, eventID); err != nil {
			response.Error(w, http.StatusForbidden, err)
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

		eventID, err := params.UUIDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sessionInfo, err := auth.GetSessionInfo(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		sqlTx := h.service.BeginSQLTx(ctx, true)
		defer sqlTx.Rollback()

		if err := h.requireHost(ctx, sqlTx, sessionInfo.ID, eventID); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		cacheKey := eventID + "_roles"
		if item, err := h.cache.Get(cacheKey); err == nil {
			response.EncodedJSON(w, item.Value)
			return
		}

		permissions, err := h.service.GetRoles(ctx, sqlTx, eventID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSONAndCache(h.cache, w, cacheKey, permissions)
	}
}

// GetReports gets an event's reports.
func (h *Handler) GetReports() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.UUIDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sessionInfo, err := auth.GetSessionInfo(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		sqlTx := h.service.BeginSQLTx(ctx, true)
		defer sqlTx.Rollback()

		if err := h.privacyFilter(ctx, sqlTx, sessionInfo.ID, eventID); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		reports, err := h.service.GetReports(ctx, sqlTx, eventID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, reports)
	}
}

// RemoveBanned removes the ban on a user.
func (h *Handler) RemoveBanned() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var reqBody userIDBody
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		eventID := httprouter.ParamsFromContext(ctx).ByName("id")
		if err := params.ValidateUUIDs(eventID, reqBody.UserID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sessionInfo, err := auth.GetSessionInfo(ctx, r)
		if err != nil {
			response.Error(w, http.StatusUnauthorized, err)
			return
		}

		errStatus, err := h.service.SQLTx(ctx, true, func(tx *sql.Tx) (int, error) {
			if err := h.requireHost(ctx, tx, sessionInfo.ID, eventID); err != nil {
				return http.StatusForbidden, err
			}
			return 0, nil
		})
		if err != nil {
			response.Error(w, errStatus, err)
			return
		}

		if err := h.service.RemoveEdge(ctx, eventID, Banned, reqBody.UserID); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, edgeMuResponse{
			EventID:   eventID,
			Predicate: Banned,
			UserID:    reqBody.UserID,
		})
	}
}

// RemoveConfirmed removes the confirmation of a user.
func (h *Handler) RemoveConfirmed() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var reqBody userIDBody
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		eventID := httprouter.ParamsFromContext(ctx).ByName("id")
		if err := params.ValidateUUIDs(eventID, reqBody.UserID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if err := h.service.RemoveEdge(ctx, eventID, Confirmed, reqBody.UserID); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, edgeMuResponse{
			EventID:   eventID,
			Predicate: Confirmed,
			UserID:    reqBody.UserID,
		})
	}
}

// RemoveInvited removes an invitation from a user.
func (h *Handler) RemoveInvited() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var reqBody userIDBody
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		eventID := httprouter.ParamsFromContext(ctx).ByName("id")
		if err := params.ValidateUUIDs(eventID, reqBody.UserID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if err := h.service.RemoveEdge(ctx, eventID, Invited, reqBody.UserID); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, edgeMuResponse{
			EventID:   eventID,
			Predicate: Invited,
			UserID:    reqBody.UserID,
		})
	}
}

// RemoveLike removes a like from a user.
func (h *Handler) RemoveLike() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var reqBody userIDBody
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		eventID := httprouter.ParamsFromContext(ctx).ByName("id")
		if err := params.ValidateUUIDs(eventID, reqBody.UserID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if err := h.service.RemoveEdge(ctx, eventID, LikedBy, reqBody.UserID); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, edgeMuResponse{
			EventID:   eventID,
			Predicate: LikedBy,
			UserID:    reqBody.UserID,
		})
	}
}

// Search performs an event search.
func (h *Handler) Search() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		query := httprouter.ParamsFromContext(ctx).ByName("query")

		events, err := h.service.Search(ctx, query)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, events)
	}
}

// SetRole sets a role to a user inside the event passed.
func (h *Handler) SetRole() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.UUIDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		var reqBody struct {
			UserID   string `json:"user_id,omitempty"`
			RoleName string `json:"role_name,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		sessionInfo, err := auth.GetSessionInfo(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		errStatus, err := h.service.SQLTx(ctx, false, func(tx *sql.Tx) (int, error) {
			if err := h.requireHost(ctx, tx, sessionInfo.ID, eventID); err != nil {
				return http.StatusForbidden, err
			}

			err := h.service.SetRole(ctx, tx, eventID, reqBody.UserID, reqBody.RoleName)
			if err != nil {
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

// Update updates an event.
func (h *Handler) Update() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.UUIDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		var uptEvent UpdateEvent
		if err := json.NewDecoder(r.Body).Decode(&uptEvent); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}
		defer r.Body.Close()

		sessionInfo, err := auth.GetSessionInfo(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		errStatus, err := h.service.SQLTx(ctx, false, func(tx *sql.Tx) (int, error) {
			if err := h.requireHost(ctx, tx, sessionInfo.ID); err != nil {
				return http.StatusForbidden, err
			}

			if err := h.service.Update(ctx, tx, eventID, uptEvent); err != nil {
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

// UpdateMedia updates a media of an event.
func (h *Handler) UpdateMedia() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.UUIDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		var media Media
		if err := json.NewDecoder(r.Body).Decode(&media); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}
		defer r.Body.Close()

		sessionInfo, err := auth.GetSessionInfo(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		errStatus, err := h.service.SQLTx(ctx, false, func(tx *sql.Tx) (int, error) {
			if err := h.requireHost(ctx, tx, sessionInfo.ID); err != nil {
				return http.StatusForbidden, err
			}

			if err := h.service.UpdateMedia(ctx, tx, eventID, media); err != nil {
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

// UpdateProduct updates a product of an event.
func (h *Handler) UpdateProduct() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.UUIDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		var product Product
		if err := json.NewDecoder(r.Body).Decode(&product); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}
		defer r.Body.Close()

		sessionInfo, err := auth.GetSessionInfo(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		errStatus, err := h.service.SQLTx(ctx, false, func(tx *sql.Tx) (int, error) {
			if err := h.requireHost(ctx, tx, sessionInfo.ID); err != nil {
				return http.StatusForbidden, err
			}

			if err := h.service.UpdateProduct(ctx, tx, eventID, product); err != nil {
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

// privacyFilter lets through only users that can fetch the event data if it's private,
// if it's public it lets anyone in.
func (h *Handler) privacyFilter(ctx context.Context, tx *sql.Tx, authUserID, eventID string) error {
	isPublic, err := h.service.IsPublic(ctx, tx, eventID)
	if err != nil {
		return errors.Wrap(err, "privacyFilter: scanning event privacy")
	}

	if isPublic {
		// Event is public, no restrictions applied
		return nil
	}

	// If the user has a role in the event, then he's able to retrieve its information
	hasRole, err := h.service.UserHasRole(ctx, tx, eventID, authUserID)
	if err != nil {
		return errors.Wrap(err, "privacyFilter: scanning user role")
	}
	if !hasRole {
		return errAccessDenied
	}

	return nil
}

// requireHost returns an error if the user is not a host of any of the event passed.
func (h *Handler) requireHost(ctx context.Context, tx *sql.Tx, authUserID string, eventIDs ...string) error {
	isHost, err := h.service.IsHost(ctx, tx, authUserID, eventIDs...)
	if err != nil {
		return errors.Wrap(err, "requireHost: scanning user role")
	}

	if !isHost {
		return errAccessDenied
	}

	return nil
}

// requirePermissions returns an error if the user hasn't the permissions required on the event passed.
func (h *Handler) requirePermissions(ctx context.Context, tx *sql.Tx, authUserID, eventID string, permRequired []string) error {
	role, err := h.service.GetUserRole(ctx, tx, eventID, authUserID)
	if err != nil {
		return errors.Wrap(err, "requirePermissions: scanning user role")
	}

	if err := permissions.Require(role.PermissionKeys, permRequired...); err != nil {
		return errAccessDenied
	}

	return nil
}
