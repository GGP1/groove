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

// Block executes a block from the user passed to another one.
func (h *Handler) Block() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var reqBody blockedIDBody
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		userID := httprouter.ParamsFromContext(ctx).ByName("id")
		if err := params.ValidateUUIDs(userID, reqBody.BlockedID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if err := h.service.Block(ctx, userID, reqBody.BlockedID); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		type resp struct {
			ID        string `json:"id,omitempty"`
			BlockedID string `json:"blocked_id,omitempty"`
		}
		response.JSON(w, http.StatusOK, resp{ID: userID, BlockedID: reqBody.BlockedID})
	}
}

// Create creates a new user.
func (h *Handler) Create() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

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

		userID := uuid.NewString()
		if err := h.service.Create(ctx, userID, user); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		apiKey, err := apikey.New(userID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		type resp struct {
			ID     string `json:"id,omitempty"`
			APIKey string `json:"api_key,omitempty"`
		}
		response.JSON(w, http.StatusCreated, resp{ID: userID, APIKey: apiKey})
	}
}

// Delete removes a user from the system.
func (h *Handler) Delete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID, err := params.UUIDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if err := h.service.Delete(ctx, userID); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSONMessage(w, http.StatusOK, userID)
	}
}

// Follow executes the follow from the user passed to another one.
func (h *Handler) Follow() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var reqBody followedIDBody
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		userID := httprouter.ParamsFromContext(ctx).ByName("id")
		if err := params.ValidateUUIDs(userID, reqBody.FollowedID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if err := h.service.Follow(ctx, userID, reqBody.FollowedID); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		type resp struct {
			ID            string `json:"id,omitempty"`
			FollowedID    string `json:"followed_id,omitempty"`
			PendingFollow bool   `json:"pending_follow,omitempty"` // If the follow was already performed or not
		}
		response.JSON(w, http.StatusOK, resp{ID: userID, FollowedID: reqBody.FollowedID})
	}
}

// GetBannedEvents gets the events from which the user passed is banned.
func (h *Handler) GetBannedEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID, err := params.UUIDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		params, err := params.ParseQuery(r.URL.RawQuery, params.Event)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		events, err := h.service.GetConfirmedEvents(ctx, userID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, events)
	}
}

// GetBlocked gets the users the user passed blocked.
func (h *Handler) GetBlocked() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID, err := params.UUIDFromCtx(ctx)
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
			count, err := h.service.GetBlockedCount(ctx, userID)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, err)
				return
			}

			response.JSONCount(w, http.StatusOK, count)
			return
		}

		blocked, err := h.service.GetBlocked(ctx, userID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, blocked)
	}
}

// GetBlockedBy gets the users that blocked the user passed.
func (h *Handler) GetBlockedBy() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID, err := params.UUIDFromCtx(ctx)
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
			count, err := h.service.GetBlockedByCount(ctx, userID)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, err)
				return
			}

			response.JSONCount(w, http.StatusOK, count)
			return
		}

		blockedBy, err := h.service.GetBlockedBy(ctx, userID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, blockedBy)
	}
}

// GetByID gets a user by its id.
func (h *Handler) GetByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID, err := params.UUIDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if item, err := h.cache.Get(userID); err == nil {
			response.EncodedJSON(w, item.Value)
			return
		}

		user, err := h.service.GetByID(ctx, userID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSONAndCache(h.cache, w, userID, user)
	}
}

// GetConfirmedEvents gets the events the user is attending to.
func (h *Handler) GetConfirmedEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID, err := params.UUIDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		params, err := params.ParseQuery(r.URL.RawQuery, params.Event)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		events, err := h.service.GetConfirmedEvents(ctx, userID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, events)
	}
}

// GetFollowers get the users following the user passed.
func (h *Handler) GetFollowers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userID, err := params.UUIDFromCtx(ctx)
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
			count, err := h.service.GetFollowersCount(ctx, userID)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, err)
				return
			}

			response.JSONCount(w, http.StatusOK, count)
			return
		}

		followers, err := h.service.GetFollowers(ctx, userID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, followers)
	}
}

// GetFollowing gets the users followed by the user passed.
func (h *Handler) GetFollowing() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userID, err := params.UUIDFromCtx(ctx)
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
			count, err := h.service.GetFollowingCount(ctx, userID)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, err)
				return
			}

			response.JSONCount(w, http.StatusOK, count)
			return
		}

		following, err := h.service.GetFollowing(ctx, userID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, following)
	}
}

// GetHostedEvents gets the events hosted by the user.
func (h *Handler) GetHostedEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID, err := params.UUIDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		params, err := params.ParseQuery(r.URL.RawQuery, params.Event)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		events, err := h.service.GetHostedEvents(ctx, userID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, events)
	}
}

// GetInvitedEvents gets the events that the user is invited to.
func (h *Handler) GetInvitedEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID, err := params.UUIDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		params, err := params.ParseQuery(r.URL.RawQuery, params.Event)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		events, err := h.service.GetInvitedEvents(ctx, userID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, events)
	}
}

// GetLikedEvents gets the events liked by a user.
func (h *Handler) GetLikedEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID, err := params.UUIDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		params, err := params.ParseQuery(r.URL.RawQuery, params.Event)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		events, err := h.service.GetLikedEvents(ctx, userID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, events)
	}
}

// Search performs a user search.
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

// Unblock removes the block from the user passed to another.
func (h *Handler) Unblock() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var reqBody blockedIDBody
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		userID := httprouter.ParamsFromContext(ctx).ByName("id")
		if err := params.ValidateUUIDs(userID, reqBody.BlockedID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if err := h.service.Unblock(ctx, userID, reqBody.BlockedID); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		type resp struct {
			ID          string `json:"id,omitempty"`
			UnblockedID string `json:"unblocked_id,omitempty"`
		}
		response.JSON(w, http.StatusOK, resp{ID: userID, UnblockedID: reqBody.BlockedID})
	}
}

// Unfollow removes the follow from the user passed to another.
func (h *Handler) Unfollow() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var reqBody followedIDBody
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		userID := httprouter.ParamsFromContext(ctx).ByName("id")
		if err := params.ValidateUUIDs(userID, reqBody.FollowedID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if err := h.service.Unfollow(ctx, userID, reqBody.FollowedID); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		type resp struct {
			ID           string `json:"id,omitempty"`
			UnfollowedID string `json:"unfollowed_id,omitempty"`
		}
		response.JSON(w, http.StatusOK, resp{ID: userID, UnfollowedID: reqBody.FollowedID})
	}
}

// Update updates a user.
func (h *Handler) Update() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID, err := params.UUIDFromCtx(ctx)
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

		if err := h.service.Update(ctx, userID, uptUser); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSONMessage(w, http.StatusOK, userID)
	}
}
