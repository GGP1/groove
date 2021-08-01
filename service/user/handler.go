package user

import (
	"encoding/json"
	"net/http"

	"github.com/GGP1/groove/internal/apikey"
	"github.com/GGP1/groove/internal/cache"
	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/response"
	"github.com/GGP1/groove/internal/sanitize"
	"github.com/GGP1/groove/internal/ulid"
	"github.com/GGP1/groove/service/event"

	"github.com/julienschmidt/httprouter"
)

type blockedIDBody struct {
	BlockedID string `json:"blocked_id,omitempty"`
}

type friendIDBody struct {
	FriendID string `json:"friend_id,omitempty"`
}

// Handler is the user handler.
type Handler struct {
	service Service
	cache   cache.Client
}

// NewHandler returns a new user handler
func NewHandler(service Service, cache cache.Client) Handler {
	return Handler{
		service: service,
		cache:   cache,
	}
}

// AddFriend adds a new friend to the user.
func (h *Handler) AddFriend() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var reqBody friendIDBody
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		userID := httprouter.ParamsFromContext(ctx).ByName("id")
		if err := ulid.ValidateN(userID, reqBody.FriendID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		// TODO: require confirmation from the other user before performing operation.
		// Probably better to implement SendRequest() and trigger AddFriend as a consequence of the confirmation
		if err := h.service.AddFriend(ctx, userID, reqBody.FriendID); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		type resp struct {
			ID       string `json:"id,omitempty"`
			FriendID string `json:"friend_id,omitempty"`
		}
		response.JSON(w, http.StatusOK, resp{ID: userID, FriendID: reqBody.FriendID})
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
		if err := ulid.ValidateN(userID, reqBody.BlockedID); err != nil {
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

		userID := ulid.NewString()
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

		userID, err := params.IDFromCtx(ctx)
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

// GetBannedEvents gets the events from which the user passed is banned.
func (h *Handler) GetBannedEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID, err := params.IDFromCtx(ctx)
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

		userID, err := params.IDFromCtx(ctx)
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

		userID, err := params.IDFromCtx(ctx)
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

		userID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		cacheKey := cache.UsersKey(userID)
		if item, err := h.cache.Get(cacheKey); err == nil {
			response.EncodedJSON(w, item.Value)
			return
		}

		user, err := h.service.GetByID(ctx, userID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSONAndCache(h.cache, w, cacheKey, user)
	}
}

// GetConfirmedEvents gets the events the user is attending to.
func (h *Handler) GetConfirmedEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID, err := params.IDFromCtx(ctx)
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

// GetFriends gets the users friends of the user passed.
func (h *Handler) GetFriends() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userID, err := params.IDFromCtx(ctx)
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
			count, err := h.service.GetFriendsCount(ctx, userID)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, err)
				return
			}

			response.JSONCount(w, http.StatusOK, count)
			return
		}

		friends, err := h.service.GetFriends(ctx, userID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, friends)
	}
}

// GetHostedEvents gets the events hosted by the user.
func (h *Handler) GetHostedEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID, err := params.IDFromCtx(ctx)
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

		var nextCursor string
		if len(events) > 0 {
			nextCursor = events[len(events)-1].ID
		}

		type resp struct {
			NextCursor string        `json:"next_cursor,omitempty"`
			Events     []event.Event `json:"events,omitempty"`
		}
		response.JSON(w, http.StatusOK, resp{
			NextCursor: nextCursor,
			Events:     events,
		})
	}
}

// GetInvitedEvents gets the events that the user is invited to.
func (h *Handler) GetInvitedEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID, err := params.IDFromCtx(ctx)
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

		userID, err := params.IDFromCtx(ctx)
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

// GetStatistics returns a user's predicates statistics.
func (h *Handler) GetStatistics() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		counts, err := h.service.GetStatistics(ctx, userID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, counts)
	}
}

// RemoveFriend removes a friend.
func (h *Handler) RemoveFriend() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var reqBody friendIDBody
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		userID := httprouter.ParamsFromContext(ctx).ByName("id")
		if err := ulid.ValidateN(userID, reqBody.FriendID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if err := h.service.RemoveFriend(ctx, userID, reqBody.FriendID); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		type resp struct {
			ID       string `json:"id,omitempty"`
			FriendID string `json:"friend_id,omitempty"`
		}
		response.JSON(w, http.StatusOK, resp{ID: userID, FriendID: reqBody.FriendID})
	}
}

// Search performs a user search.
func (h *Handler) Search() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		query := httprouter.ParamsFromContext(ctx).ByName("query")
		query = sanitize.Normalize(query)
		if err := params.ValidateSearchQuery(query); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

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

		var nextCursor string
		if len(users) > 0 {
			nextCursor = users[len(users)-1].ID
		}

		type resp struct {
			NextCursor string     `json:"next_cursor,omitempty"`
			Users      []ListUser `json:"users,omitempty"`
		}
		response.JSON(w, http.StatusOK, resp{NextCursor: nextCursor, Users: users})
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
		if err := ulid.ValidateN(userID, reqBody.BlockedID); err != nil {
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

// Update updates a user.
func (h *Handler) Update() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID, err := params.IDFromCtx(ctx)
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
