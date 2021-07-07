package user

import (
	"encoding/json"
	"net/http"

	"github.com/GGP1/groove/internal/apikey"
	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/response"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
)

type blockedIDBody struct {
	BlockedID string `json:"blocked_id,omitempty"`
}

type followedIDBody struct {
	FollowedID string `json:"followed_id,omitempty"`
}

// Handler is the user handler.
type Handler struct {
	service Service
	cache   *memcache.Client
}

// NewHandler returns a new user handler
func NewHandler(service Service, cache *memcache.Client) Handler {
	return Handler{
		service: service,
		cache:   cache,
	}
}

// Block ..
func (h *Handler) Block() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var reqBody blockedIDBody
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		id := httprouter.ParamsFromContext(ctx).ByName("id")
		if err := params.ValidateUUIDs(id, reqBody.BlockedID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if err := h.service.Block(ctx, id, reqBody.BlockedID); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		type resp struct {
			ID        string `json:"id,omitempty"`
			BlockedID string `json:"blocked_id,omitempty"`
		}
		response.JSON(w, http.StatusOK, resp{ID: id, BlockedID: reqBody.BlockedID})
	}
}

// Create ..
func (h *Handler) Create() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user CreateUser
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}
		defer r.Body.Close()

		if err := user.Validate(); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		id := uuid.NewString()
		if err := h.service.Create(r.Context(), id, user); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		apiKey, err := apikey.New(id)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		type resp struct {
			ID     string `json:"id,omitempty"`
			APIKey string `json:"api_key,omitempty"`
		}
		response.JSON(w, http.StatusCreated, resp{ID: id, APIKey: apiKey})
	}
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

// Follow ..
func (h *Handler) Follow() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var reqBody followedIDBody
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		id := httprouter.ParamsFromContext(ctx).ByName("id")
		if err := params.ValidateUUIDs(id, reqBody.FollowedID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if err := h.service.Follow(ctx, id, reqBody.FollowedID); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		type resp struct {
			ID            string `json:"id,omitempty"`
			FollowedID    string `json:"followed_id,omitempty"`
			PendingFollow bool   `json:"pending_follow,omitempty"` // If the follow was already performed or not
		}
		response.JSON(w, http.StatusOK, resp{ID: id, FollowedID: reqBody.FollowedID})
	}
}

// GetBannedEvents ..
func (h *Handler) GetBannedEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		id, err := params.UUIDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		params, err := params.ParseQuery(r.URL.RawQuery, params.Event)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		events, err := h.service.GetConfirmedEvents(ctx, id, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, events)
	}
}

// GetBlocked ..
func (h *Handler) GetBlocked() http.HandlerFunc {
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
			count, err := h.service.GetBlockedCount(ctx, id)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, err)
				return
			}

			response.JSONCount(w, http.StatusOK, count)
			return
		}

		blocked, err := h.service.GetBlocked(ctx, id, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, blocked)
	}
}

// GetBlockedBy ..
func (h *Handler) GetBlockedBy() http.HandlerFunc {
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
			count, err := h.service.GetBlockedByCount(ctx, id)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, err)
				return
			}

			response.JSONCount(w, http.StatusOK, count)
			return
		}

		blockedBy, err := h.service.GetBlockedBy(ctx, id, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, blockedBy)
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

		user, err := h.service.GetByID(ctx, id)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSONAndCache(h.cache, w, id, user)
	}
}

// GetConfirmedEvents ..
func (h *Handler) GetConfirmedEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		id, err := params.UUIDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		params, err := params.ParseQuery(r.URL.RawQuery, params.Event)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		events, err := h.service.GetConfirmedEvents(ctx, id, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, events)
	}
}

// GetFollowers ..
func (h *Handler) GetFollowers() http.HandlerFunc {
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
			count, err := h.service.GetFollowersCount(ctx, id)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, err)
				return
			}

			response.JSONCount(w, http.StatusOK, count)
			return
		}

		followers, err := h.service.GetFollowers(ctx, id, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, followers)
	}
}

// GetFollowing ..
func (h *Handler) GetFollowing() http.HandlerFunc {
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
			count, err := h.service.GetFollowingCount(ctx, id)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, err)
				return
			}

			response.JSONCount(w, http.StatusOK, count)
			return
		}

		following, err := h.service.GetFollowing(ctx, id, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, following)
	}
}

// GetHostedEvents ..
func (h *Handler) GetHostedEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		id, err := params.UUIDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		params, err := params.ParseQuery(r.URL.RawQuery, params.Event)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		events, err := h.service.GetHostedEvents(ctx, id, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, events)
	}
}

// GetInvitedEvents ..
func (h *Handler) GetInvitedEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		id, err := params.UUIDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		params, err := params.ParseQuery(r.URL.RawQuery, params.Event)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		events, err := h.service.GetInvitedEvents(ctx, id, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, events)
	}
}

// GetLikedEvents ..
func (h *Handler) GetLikedEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		id, err := params.UUIDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		params, err := params.ParseQuery(r.URL.RawQuery, params.Event)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		events, err := h.service.GetLikedEvents(ctx, id, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, events)
	}
}

// Search ..
func (h *Handler) Search() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		query := httprouter.ParamsFromContext(ctx).ByName("query")
		params, err := params.ParseQuery(r.URL.RawQuery, params.User)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		users, err := h.service.Search(ctx, query, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, users)
	}
}

// Unblock ..
func (h *Handler) Unblock() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var reqBody blockedIDBody
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		id := httprouter.ParamsFromContext(ctx).ByName("id")
		if err := params.ValidateUUIDs(id, reqBody.BlockedID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if err := h.service.Unblock(ctx, id, reqBody.BlockedID); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		type resp struct {
			ID          string `json:"id,omitempty"`
			UnblockedID string `json:"unblocked_id,omitempty"`
		}
		response.JSON(w, http.StatusOK, resp{ID: id, UnblockedID: reqBody.BlockedID})
	}
}

// Unfollow ..
func (h *Handler) Unfollow() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var reqBody followedIDBody
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		id := httprouter.ParamsFromContext(ctx).ByName("id")
		if err := params.ValidateUUIDs(id, reqBody.FollowedID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if err := h.service.Unfollow(ctx, id, reqBody.FollowedID); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		type resp struct {
			ID           string `json:"id,omitempty"`
			UnfollowedID string `json:"unfollowed_id,omitempty"`
		}
		response.JSON(w, http.StatusOK, resp{ID: id, UnfollowedID: reqBody.FollowedID})
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

		var uptUser UpdateUser
		if err := json.NewDecoder(r.Body).Decode(&uptUser); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}
		defer r.Body.Close()

		if err := uptUser.Validate(); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if err := h.service.Update(ctx, id, uptUser); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSONMessage(w, http.StatusOK, id)
	}
}
