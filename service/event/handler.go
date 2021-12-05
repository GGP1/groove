package event

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/GGP1/groove/internal/cache"
	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/response"
	"github.com/GGP1/groove/internal/roles"
	"github.com/GGP1/groove/internal/ulid"
	"github.com/GGP1/groove/internal/validate"
	"github.com/GGP1/groove/model"
	"github.com/GGP1/groove/service/auth"
	"github.com/GGP1/groove/service/event/role"
	"github.com/GGP1/groove/storage/dgraph"
	"github.com/GGP1/groove/storage/postgres"

	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
)

var errAccessDenied = errors.New("Access denied")

// Handler handles events endpoints.
type Handler struct {
	db    *sql.DB
	cache cache.Client

	service     Service
	roleService role.Service
}

// NewHandler returns an event handler.
func NewHandler(db *sql.DB, cache cache.Client, service Service, roleService role.Service) Handler {
	return Handler{
		db:          db,
		cache:       cache,
		service:     service,
		roleService: roleService,
	}
}

// AddBanned bans a user in an event.
func (h Handler) AddBanned() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
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

		if err := h.service.AddEdge(ctx, eventID, dgraph.Banned, reqBody.UserID); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.NoContent(w)
	}
}

// AddInvited invites a user to an event.
func (h Handler) AddInvited() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
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

		sqlTx, ctx := postgres.BeginTx(ctx, h.db)
		defer sqlTx.Rollback()

		if err := h.roleService.SetReservedRole(ctx, eventID, reqBody.UserID, roles.Viewer); err != nil {
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

// AddLike adds the like of a user to an event.
func (h Handler) AddLike() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		session, err := auth.GetSession(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		hasRole, err := h.roleService.HasRole(ctx, eventID, session.ID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}
		if !hasRole {
			response.Error(w, http.StatusForbidden, errors.New("you must be a member of the event to like it"))
			return
		}

		if err := h.service.AddEdge(ctx, eventID, dgraph.LikedBy, session.ID); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.NoContent(w)
	}
}

// Create creates an event.
func (h Handler) Create() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		session, err := auth.GetSession(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		var event CreateEvent
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		eventID := ulid.NewString()
		event.HostID = session.ID
		if err := h.service.Create(ctx, eventID, event); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusCreated, response.ID{ID: eventID})
	}
}

// Delete removes an event from the system.
func (h Handler) Delete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if err := h.service.Delete(ctx, eventID); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.NoContent(w)
	}
}

// GetBans gets an event's banned users.
func (h Handler) GetBans() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.IDFromCtx(ctx)
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
			count, err := h.service.GetBannedCount(ctx, eventID)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, err)
				return
			}

			response.JSONCount(w, http.StatusOK, "banned_count", count)
			return
		}

		bans, err := h.service.GetBanned(ctx, eventID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		var nextCursor string
		if len(bans) > 0 {
			nextCursor = bans[len(bans)-1].ID
		}

		response.JSONCursor(w, nextCursor, "users", bans)
	}
}

// GetBannedFriends returns event banned users that are friends of the user passed.
func (h Handler) GetBannedFriends() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		session, err := auth.GetSession(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		params, err := params.Parse(r.URL.RawQuery, model.User)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if params.Count {
			count, err := h.service.GetBannedFriendsCount(ctx, eventID, session.ID)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, err)
				return
			}

			response.JSONCount(w, http.StatusOK, "banned_friends_count", count)
			return
		}

		users, err := h.service.GetBannedFriends(ctx, eventID, session.ID, params)
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

// GetByID gets an event by its id.
func (h Handler) GetByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		cacheKey := model.Event.CacheKey(eventID)
		if v, err := h.cache.Get(cacheKey); err == nil {
			response.EncodedJSON(w, v)
			return
		}

		event, err := h.service.GetByID(ctx, eventID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSONAndCache(h.cache, w, cacheKey, event)
	}
}

// GetHosts gets an event's host users.
func (h Handler) GetHosts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		params, err := params.Parse(r.URL.RawQuery, model.User)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		hosts, err := h.service.GetHosts(ctx, eventID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		var nextCursor string
		if len(hosts) > 0 {
			nextCursor = hosts[len(hosts)-1].ID
		}

		response.JSONCursor(w, nextCursor, "users", hosts)
	}
}

// GetInvited gets an event's invited users.
func (h Handler) GetInvited() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.IDFromCtx(ctx)
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
			count, err := h.service.GetInvitedCount(ctx, eventID)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, err)
				return
			}

			response.JSONCount(w, http.StatusOK, "invited_count", count)
			return
		}

		invited, err := h.service.GetInvited(ctx, eventID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		var nextCursor string
		if len(invited) > 0 {
			nextCursor = invited[len(invited)-1].ID
		}

		response.JSONCursor(w, nextCursor, "users", invited)
	}
}

// GetInvitedFriends returns an event's invited users that are friends of the user passed.
func (h Handler) GetInvitedFriends() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		session, err := auth.GetSession(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		params, err := params.Parse(r.URL.RawQuery, model.User)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if params.Count {
			count, err := h.service.GetInvitedFriendsCount(ctx, eventID, session.ID)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, err)
				return
			}

			response.JSONCount(w, http.StatusOK, "invited_friends_count", count)
			return
		}

		users, err := h.service.GetInvitedFriends(ctx, eventID, session.ID, params)
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

// GetLikes gets the users liking an event.
func (h Handler) GetLikes() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.IDFromCtx(ctx)
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
			count, err := h.service.GetLikedByCount(ctx, eventID)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, err)
				return
			}

			response.JSONCount(w, http.StatusOK, "liked_by_count", count)
			return
		}

		likes, err := h.service.GetLikedBy(ctx, eventID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		var nextCursor string
		if len(likes) > 0 {
			nextCursor = likes[len(likes)-1].ID
		}

		response.JSONCursor(w, nextCursor, "users", likes)
	}
}

// GetLikedByFriends returns users liking the event that are friends of the user passed.
func (h Handler) GetLikedByFriends() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		session, err := auth.GetSession(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		params, err := params.Parse(r.URL.RawQuery, model.User)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if params.Count {
			count, err := h.service.GetLikedByFriendsCount(ctx, eventID, session.ID)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, err)
				return
			}

			response.JSONCount(w, http.StatusOK, "liked_by_friends_count", count)
			return
		}

		users, err := h.service.GetLikedByFriends(ctx, eventID, session.ID, params)
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

// GetRecommended returns a list of events that may be interesting for the user.
func (h Handler) GetRecommended() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		session, err := auth.GetSession(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		params, err := params.Parse(r.URL.RawQuery, model.Event)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		var userCoords Coordinates
		if err := json.NewDecoder(r.Body).Decode(&userCoords); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		events, err := h.service.GetRecommended(ctx, session, userCoords, params)
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

// GetStatistics returns an event's predicates statistics.
func (h Handler) GetStatistics() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		stats, err := h.service.GetStatistics(ctx, eventID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, stats)
	}
}

// Join handles the auth user entrance to a free event, paid events are entered by buying a ticket.
func (h Handler) Join() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		session, err := auth.GetSession(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		event, err := h.service.GetByID(ctx, eventID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		// The EventPrivacyFilter middleware already checks if the user can view a private event
		if event.TicketType == Paid {
			response.Error(w, http.StatusBadRequest, errors.New("event is paid, buy a ticket to join"))
			return
		}

		availableSlots, err := h.service.AvailableSlots(ctx, eventID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}
		if availableSlots < 1 {
			response.Error(w, http.StatusForbidden, errors.New("there are no slots available"))
			return
		}

		sqlTx, ctx := postgres.BeginTx(ctx, h.db)
		defer sqlTx.Rollback()

		if err := h.roleService.SetReservedRole(ctx, eventID, session.ID, roles.Attendant); err != nil {
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

// RemoveBanned removes the ban on a user.
func (h Handler) RemoveBanned() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
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

		if err := h.service.RemoveEdge(ctx, eventID, dgraph.Banned, reqBody.UserID); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.NoContent(w)
	}
}

// RemoveLike removes a like from a user.
func (h Handler) RemoveLike() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var reqBody model.UserID
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		eventID := httprouter.ParamsFromContext(ctx).ByName("id")
		if err := validate.ULIDs(eventID, reqBody.UserID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if err := h.service.RemoveEdge(ctx, eventID, dgraph.LikedBy, reqBody.UserID); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.NoContent(w)
	}
}

// Search performs an event search.
func (h Handler) Search() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		values, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		query := values.Get("query")
		if query == "" {
			response.Error(w, http.StatusBadRequest, errors.New("invalid query"))
			return
		}

		session, err := auth.GetSession(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		params, err := params.ParseQuery(values, model.Event)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		events, err := h.service.Search(ctx, query, session, params)
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

// SearchByLocation looks for events given their location.
func (h Handler) SearchByLocation() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		session, err := auth.GetSession(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		var location LocationSearch
		if err := json.NewDecoder(r.Body).Decode(&location); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		events, err := h.service.SearchByLocation(ctx, session, location)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, events)
	}
}

// Update updates an event.
func (h Handler) Update() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.IDFromCtx(ctx)
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

		if err := h.service.Update(ctx, eventID, uptEvent); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, response.ID{ID: eventID})
	}
}
