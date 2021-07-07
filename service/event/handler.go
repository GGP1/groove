package event

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/GGP1/groove/auth"
	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/permissions"
	"github.com/GGP1/groove/internal/response"
	"github.com/pkg/errors"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
)

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

// AddBanned ..
func (h *Handler) AddBanned() http.HandlerFunc {
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

// AddConfirmed ..
func (h *Handler) AddConfirmed() http.HandlerFunc {
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

		if err := h.service.AddEdge(ctx, eventID, Confirmed, reqBody.UserID); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		if err := h.service.SetRole(ctx, eventID, reqBody.UserID, permissions.Attendant); err != nil {
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

// AddInvited ..
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

		tx, err := h.service.PqTx(ctx, true)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}
		defer tx.Rollback()

		// Check the invited user settings to verify the invitation can be performed
		canInvite, err := h.service.CanInvite(ctx, tx, sessionInfo.ID, reqBody.UserID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}
		if !canInvite {
			response.Error(w, http.StatusForbidden, errors.New("user settings do not allow this invitation"))
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

// AddLike ..
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

// Create ..
func (h *Handler) Create() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		id := uuid.NewString()
		if err := h.service.Create(r.Context(), id, event); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSONMessage(w, http.StatusCreated, id)
	}
}

// CreateMedia creates an image/video inside an event.
func (h *Handler) CreateMedia() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		id, err := params.UUIDFromCtx(ctx)
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

		if err := h.service.CreateMedia(ctx, id, media); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, media)
	})
}

// CreatePermission creates a new permission inside an event.
func (h *Handler) CreatePermission() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		id, err := params.UUIDFromCtx(ctx)
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

		permission.Key = strings.ToLower(permission.Key)
		if err := h.service.CreatePermission(ctx, id, permission); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, permission)
	})
}

// CreateProduct creates an image/video inside an event.
func (h *Handler) CreateProduct() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		id, err := params.UUIDFromCtx(ctx)
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

		if err := h.service.CreateProduct(ctx, id, product); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, product)
	})
}

// CreateRole creates a new role inside an event.
func (h *Handler) CreateRole() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		id, err := params.UUIDFromCtx(ctx)
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

		role.Name = strings.ToLower(role.Name)
		if err := h.service.CreateRole(ctx, id, role); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, role)
	})
}

// Delete ..
func (h *Handler) Delete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		id, err := params.UUIDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if err := h.service.Delete(ctx, id); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSONMessage(w, http.StatusOK, id)
	}
}

// GetBans ..
func (h *Handler) GetBans() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		id, err := params.UUIDFromCtx(ctx)
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
			count, err := h.service.GetBannedCount(ctx, id)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, err)
				return
			}

			response.JSONCount(w, http.StatusOK, count)
			return
		}

		bans, err := h.service.GetBanned(ctx, id, params)
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

		users, err := h.service.GetBannedFollowing(ctx, eventID, userID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, users)
	}
}

// GetByID ..
func (h *Handler) GetByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		id, err := params.UUIDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if item, err := h.cache.Get(id); err == nil {
			response.EncodedJSON(w, item.Value)
			return
		}

		event, err := h.service.GetByID(ctx, id)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSONAndCache(h.cache, w, id, event)
	}
}

// GetConfirmed ..
func (h *Handler) GetConfirmed() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		id, err := params.UUIDFromCtx(ctx)
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
			count, err := h.service.GetConfirmedCount(ctx, id)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, err)
				return
			}

			response.JSONCount(w, http.StatusOK, count)
			return
		}

		confirmed, err := h.service.GetConfirmed(ctx, id, params)
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

		users, err := h.service.GetConfirmedFollowing(ctx, eventID, userID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, users)
	}
}

// GetHosts ..
func (h *Handler) GetHosts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		id, err := params.UUIDFromCtx(ctx)
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
			count, err := h.service.GetInvitedCount(ctx, id)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, err)
				return
			}

			response.JSONCount(w, http.StatusOK, count)
			return
		}

		hosts, err := h.service.GetHosts(ctx, id, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, hosts)
	}
}

// GetInvited ..
func (h *Handler) GetInvited() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		id, err := params.UUIDFromCtx(ctx)
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
			count, err := h.service.GetInvitedCount(ctx, id)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, err)
				return
			}

			response.JSONCount(w, http.StatusOK, count)
			return
		}

		invited, err := h.service.GetInvited(ctx, id, params)
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

		users, err := h.service.GetInvitedFollowing(ctx, eventID, userID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, users)
	}
}

// GetLikes returns the users liking the event.
func (h *Handler) GetLikes() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		id, err := params.UUIDFromCtx(ctx)
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
			count, err := h.service.GetLikedByCount(ctx, id)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, err)
				return
			}

			response.JSONCount(w, http.StatusOK, count)
			return
		}

		likes, err := h.service.GetLikedBy(ctx, id, params)
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

		users, err := h.service.GetLikedByFollowing(ctx, eventID, userID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, users)
	}
}

// GetPermissions retrives all event's permissions.
func (h *Handler) GetPermissions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		id, err := params.UUIDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		cacheKey := id + "_permissions"
		item, err := h.cache.Get(cacheKey)
		if err == nil {
			response.EncodedJSON(w, item.Value)
			return
		}

		permissions, err := h.service.GetPermissions(ctx, id)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSONAndCache(h.cache, w, cacheKey, permissions)
	}
}

// GetRole ..
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

		tx, err := h.service.PqTx(ctx, true)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		role, err := h.service.GetUserRole(ctx, tx, eventID, reqBody.UserID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}
		tx.Commit()

		response.JSON(w, http.StatusOK, role)
	}
}

// GetRoles retrives all event's permissions.
func (h *Handler) GetRoles() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		id, err := params.UUIDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		cacheKey := id + "_roles"
		item, err := h.cache.Get(cacheKey)
		if err == nil {
			response.EncodedJSON(w, item.Value)
			return
		}

		permissions, err := h.service.GetRoles(ctx, id)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSONAndCache(h.cache, w, cacheKey, permissions)
	}
}

// GetReports ..
func (h *Handler) GetReports() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		id, err := params.UUIDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		reports, err := h.service.GetReports(ctx, id)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, reports)
	}
}

// RemoveBanned ..
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

// RemoveConfirmed ..
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

// RemoveInvited ..
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

// RemoveLike ..
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

// Search ..
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

// SetRole ..
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

		if err := h.service.SetRole(ctx, eventID, reqBody.UserID, reqBody.RoleName); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSONMessage(w, http.StatusOK, eventID)
	}
}

// Update ..
func (h *Handler) Update() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		id, err := params.UUIDFromCtx(ctx)
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

		if err := h.service.Update(ctx, id, uptEvent); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSONMessage(w, http.StatusOK, id)
	}
}

// UpdateMedia ..
func (h *Handler) UpdateMedia() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		id, err := params.UUIDFromCtx(ctx)
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

		if err := h.service.UpdateMedia(ctx, id, media); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSONMessage(w, http.StatusOK, id)
	}
}

// UpdateProduct ..
func (h *Handler) UpdateProduct() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		id, err := params.UUIDFromCtx(ctx)
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

		if err := h.service.UpdateProduct(ctx, id, product); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSONMessage(w, http.StatusOK, id)
	}
}
