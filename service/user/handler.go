package user

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/GGP1/groove/internal/apikey"
	"github.com/GGP1/groove/internal/cache"
	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/response"
	"github.com/GGP1/groove/internal/sanitize"
	"github.com/GGP1/groove/internal/ulid"
	"github.com/GGP1/groove/internal/validate"
	"github.com/GGP1/groove/model"
	"github.com/GGP1/groove/service/auth"
	"github.com/GGP1/groove/storage/postgres"

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
	db    *sql.DB
	cache cache.Client

	service Service
}

// NewHandler returns a new user handler
func NewHandler(db *sql.DB, cache cache.Client, service Service) Handler {
	return Handler{
		db:      db,
		cache:   cache,
		service: service,
	}
}

// AddFriend adds a new friend to the user.
func (h Handler) AddFriend() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var reqBody friendIDBody
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		userID := httprouter.ParamsFromContext(ctx).ByName("id")
		if err := validate.ULIDs(userID, reqBody.FriendID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

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
func (h Handler) Block() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var reqBody blockedIDBody
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		userID := httprouter.ParamsFromContext(ctx).ByName("id")
		if err := validate.ULIDs(userID, reqBody.BlockedID); err != nil {
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
func (h Handler) Create() http.HandlerFunc {
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
		user.Username = strings.ToLower(user.Username)
		sanitize.Strings(&user.Username, &user.Name)

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
func (h Handler) Delete() http.HandlerFunc {
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

		response.NoContent(w)
	}
}

// Follow follows a business.
func (h Handler) Follow() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		businessID, err := params.IDFromCtx(ctx, "business_id")
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		session, err := auth.GetSession(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		if err := h.service.Follow(ctx, session, businessID); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.NoContent(w)
	}
}

// GetAttendingEvents gets the events the user is attending to.
func (h Handler) GetAttendingEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		params, err := params.Parse(r.URL.RawQuery, model.Event)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if params.Count {
			count, err := h.service.GetAttendingEventsCount(ctx, userID)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, err)
				return
			}

			response.JSONCount(w, http.StatusOK, "attending_events_count", count)
			return
		}

		events, err := h.service.GetAttendingEvents(ctx, userID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, events)
	}
}

// GetBannedEvents gets the events from which the user passed is banned.
func (h Handler) GetBannedEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		params, err := params.Parse(r.URL.RawQuery, model.Event)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if params.Count {
			count, err := h.service.GetBannedEventsCount(ctx, userID)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, err)
				return
			}

			response.JSONCount(w, http.StatusOK, "banned_events_count", count)
			return
		}

		events, err := h.service.GetBannedEvents(ctx, userID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, events)
	}
}

// GetBlocked gets the users the user passed blocked.
func (h Handler) GetBlocked() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		params, err := params.Parse(r.URL.RawQuery, model.User)
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

			response.JSONCount(w, http.StatusOK, "blocked_count", count)
			return
		}

		blocked, err := h.service.GetBlocked(ctx, userID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		var nextCursor string
		if len(blocked) > 1 {
			nextCursor = blocked[len(blocked)-1].ID
		}

		response.JSONCursor(w, nextCursor, "users", blocked)
	}
}

// GetBlockedBy gets the users that blocked the user passed.
func (h Handler) GetBlockedBy() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		params, err := params.Parse(r.URL.RawQuery, model.User)
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

			response.JSONCount(w, http.StatusOK, "blocked_by_count", count)
			return
		}

		blockedBy, err := h.service.GetBlockedBy(ctx, userID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		var nextCursor string
		if len(blockedBy) > 1 {
			nextCursor = blockedBy[len(blockedBy)-1].ID
		}

		response.JSONCursor(w, nextCursor, "users", blockedBy)
	}
}

// GetByID gets a user by its id.
func (h Handler) GetByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		cacheKey := model.User.CacheKey(userID)
		if v, err := h.cache.Get(cacheKey); err == nil {
			response.EncodedJSON(w, v)
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

// GetFollowers gets the followers of a user.
func (h Handler) GetFollowers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		params, err := params.Parse(r.URL.RawQuery, model.User)
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

			response.JSONCount(w, http.StatusOK, "followers_count", count)
			return
		}

		followers, err := h.service.GetFollowers(ctx, userID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		var nextCursor string
		if len(followers) > 1 {
			nextCursor = followers[len(followers)-1].ID
		}

		response.JSONCursor(w, nextCursor, "users", followers)
	}
}

// GetFollowing gets the business the user is following.
func (h Handler) GetFollowing() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		params, err := params.Parse(r.URL.RawQuery, model.User)
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

			response.JSONCount(w, http.StatusOK, "following_count", count)
			return
		}

		following, err := h.service.GetFollowing(ctx, userID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		var nextCursor string
		if len(following) > 1 {
			nextCursor = following[len(following)-1].ID
		}

		response.JSONCursor(w, nextCursor, "users", following)
	}
}

// GetFriends gets the users friends of the user passed.
func (h Handler) GetFriends() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		params, err := params.Parse(r.URL.RawQuery, model.User)
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

			response.JSONCount(w, http.StatusOK, "friends_count", count)
			return
		}

		friends, err := h.service.GetFriends(ctx, userID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		var nextCursor string
		if len(friends) > 1 {
			nextCursor = friends[len(friends)-1].ID
		}

		response.JSONCursor(w, nextCursor, "users", friends)
	}
}

// GetFriendsInCommon returns the friends in common between userID and friendID.
func (h Handler) GetFriendsInCommon() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		ctxParams := httprouter.ParamsFromContext(ctx)
		userID := ctxParams.ByName("id")
		friendID := ctxParams.ByName("friend_id")
		if err := validate.ULIDs(userID, friendID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		params, err := params.Parse(r.URL.RawQuery, model.User)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if params.Count {
			count, err := h.service.GetFriendsInCommonCount(ctx, userID, friendID)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, err)
				return
			}

			response.JSONCount(w, http.StatusOK, "friends_in_common_count", count)
			return
		}

		friends, err := h.service.GetFriendsInCommon(ctx, userID, friendID, params)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		var nextCursor string
		if len(friends) > 1 {
			nextCursor = friends[len(friends)-1].ID
		}

		response.JSONCursor(w, nextCursor, "users", friends)
	}
}

// GetFriendsNotInCommon returns the friends that are not in common between userID and friendID.
func (h Handler) GetFriendsNotInCommon() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		ctxParams := httprouter.ParamsFromContext(ctx)
		userID := ctxParams.ByName("id")
		friendID := ctxParams.ByName("friend_id")
		if err := validate.ULIDs(userID, friendID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		params, err := params.Parse(r.URL.RawQuery, model.User)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if params.Count {
			count, err := h.service.GetFriendsNotInCommonCount(ctx, userID, friendID)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, err)
				return
			}

			response.JSONCount(w, http.StatusOK, "friends_not_in_common_count", count)
			return
		}

		friends, err := h.service.GetFriendsNotInCommon(ctx, userID, friendID, params)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		var nextCursor string
		if len(friends) > 1 {
			nextCursor = friends[len(friends)-1].ID
		}

		response.JSONCursor(w, nextCursor, "users", friends)
	}
}

// GetHostedEvents gets the events hosted by the user.
func (h Handler) GetHostedEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		params, err := params.Parse(r.URL.RawQuery, model.Event)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if params.Count {
			count, err := h.service.GetHostedEventsCount(ctx, userID)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, err)
				return
			}

			response.JSONCount(w, http.StatusOK, "hosted_events_count", count)
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

		response.JSONCursor(w, nextCursor, "events", events)
	}
}

// GetInvitedEvents gets the events that the user is invited to.
func (h Handler) GetInvitedEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		params, err := params.Parse(r.URL.RawQuery, model.Event)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if params.Count {
			count, err := h.service.GetInvitedEventsCount(ctx, userID)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, err)
				return
			}

			response.JSONCount(w, http.StatusOK, "invited_events_count", count)
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
func (h Handler) GetLikedEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		params, err := params.Parse(r.URL.RawQuery, model.Event)
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
func (h Handler) GetStatistics() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		stats, err := h.service.GetStatistics(ctx, userID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, stats)
	}
}

// InviteToEvent invites a user to an event.
func (h Handler) InviteToEvent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		session, err := auth.GetSession(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		var invite Invite
		if err := json.NewDecoder(r.Body).Decode(&invite); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		if err := invite.Validate(); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(ctx, h.db)
		defer sqlTx.Rollback()

		if err := h.service.InviteToEvent(ctx, session, invite); err != nil {
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

// RemoveFriend removes a friend.
func (h Handler) RemoveFriend() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var reqBody friendIDBody
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		userID := httprouter.ParamsFromContext(ctx).ByName("id")
		if err := validate.ULIDs(userID, reqBody.FriendID); err != nil {
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
func (h Handler) Search() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		values, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		query := values.Get("query")
		params, err := params.ParseQuery(values, model.User)
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

		response.JSONCursor(w, nextCursor, "users", users)
	}
}

// SendFriendRequest sends a friend request to a user.
func (h Handler) SendFriendRequest() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		session, err := auth.GetSession(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		var reqBody model.UserID
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		sqlTx, ctx := postgres.BeginTx(ctx, h.db)
		defer sqlTx.Rollback()

		if err := h.service.SendFriendRequest(ctx, session, reqBody.UserID); err != nil {
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

// Unblock removes the block from the user passed to another.
func (h Handler) Unblock() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var reqBody blockedIDBody
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		userID := httprouter.ParamsFromContext(ctx).ByName("id")
		if err := validate.ULIDs(userID, reqBody.BlockedID); err != nil {
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
func (h Handler) Update() http.HandlerFunc {
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

		sqlTx, ctx := postgres.BeginTx(ctx, h.db)
		defer sqlTx.Rollback()

		if err := h.service.Update(ctx, userID, uptUser); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		if err := sqlTx.Commit(); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSONMessage(w, http.StatusOK, userID)
	}
}
