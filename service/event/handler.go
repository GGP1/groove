package event

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/GGP1/groove/auth"
	"github.com/GGP1/groove/internal/cache"
	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/permissions"
	"github.com/GGP1/groove/internal/response"
	"github.com/GGP1/groove/internal/roles"
	"github.com/GGP1/groove/internal/ulid"
	"github.com/GGP1/groove/internal/validate"

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
	cache   cache.Client
}

// NewHandler returns an event handler.
func NewHandler(service Service, cache cache.Client) Handler {
	return Handler{
		service: service,
		cache:   cache,
	}
}

// AddBanned bans a user in an event.
func (h *Handler) AddBanned() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx := h.service.BeginSQLTx(ctx, true)

		if err := h.requirePermissions(ctx, r, sqlTx, eventID, permissions.BanUsers); err != nil {
			sqlTx.Rollback()
			response.Error(w, http.StatusForbidden, err)
			return
		}
		sqlTx.Rollback()

		var reqBody userIDBody
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		if err := validate.ULID(reqBody.UserID); err != nil {
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

		eventID := httprouter.ParamsFromContext(ctx).ByName("id")
		if err := validate.ULIDs(eventID, reqBody.UserID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		errStatus, err := h.service.SQLTx(ctx, false, func(tx *sql.Tx) (int, error) {
			invited, err := h.service.GetInvited(ctx, tx, eventID, params.Query{LookupID: reqBody.UserID})
			if err != nil || len(invited) == 0 {
				return http.StatusForbidden, errors.New("the user is not invited to the event")
			}

			availableSlots, err := h.service.AvailableSlots(ctx, tx, eventID)
			if err != nil {
				return http.StatusInternalServerError, err
			}
			if availableSlots < 1 {
				return http.StatusForbidden, errors.New("there are no slots available")
			}

			if err := h.service.AddEdge(ctx, eventID, Confirmed, reqBody.UserID); err != nil {
				return http.StatusInternalServerError, err
			}

			if err := h.service.SetRoles(ctx, tx, eventID, roles.Attendant, reqBody.UserID); err != nil {
				return http.StatusInternalServerError, err
			}

			if err := h.service.RemoveEdge(ctx, eventID, Invited, reqBody.UserID); err != nil {
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

		sqlTx := h.service.BeginSQLTx(ctx, true)
		defer sqlTx.Rollback()

		err = h.requirePermissions(ctx, r, sqlTx, eventID, permissions.InviteUsers)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		var reqBody userIDBody
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		if err := validate.ULID(reqBody.UserID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		// Check the invited user settings to verify the invitation can be performed
		canInvite, err := h.service.CanInvite(ctx, sqlTx, session.ID, reqBody.UserID)
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

// AddLike adds the like of a user to an event.
func (h *Handler) AddLike() http.HandlerFunc {
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

		sqlTx := h.service.BeginSQLTx(ctx, true)
		defer sqlTx.Rollback()

		users, err := h.service.GetConfirmed(ctx, sqlTx, eventID, params.Query{LookupID: session.ID})
		if err != nil || len(users) == 0 {
			response.Error(w, http.StatusForbidden, errors.New("must be confirmed in the event to like it"))
			return
		}

		if err := h.service.AddEdge(ctx, eventID, LikedBy, session.ID); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, edgeMuResponse{
			EventID:   eventID,
			Predicate: LikedBy,
			UserID:    session.ID,
		})
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

		session, err := auth.GetSession(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		eventID := ulid.NewString()
		event.HostID = session.ID
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

// Delete removes an event from the system.
func (h *Handler) Delete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		errStatus, err := h.service.SQLTx(ctx, false, func(tx *sql.Tx) (int, error) {
			// TODO: require confirmation from all the hosts?
			if err := h.requirePermissions(ctx, r, tx, eventID, permissions.All); err != nil {
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

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx := h.service.BeginSQLTx(ctx, true)
		defer sqlTx.Rollback()

		if err := h.requirePermissions(ctx, r, sqlTx, eventID, permissions.BanUsers); err != nil {
			response.Error(w, http.StatusForbidden, err)
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

		bans, err := h.service.GetBanned(ctx, sqlTx, eventID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, bans)
	}
}

// GetBannedFriends returns event banned users that are friends of the user passed.
func (h *Handler) GetBannedFriends() http.HandlerFunc {
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

		params, err := params.ParseQuery(r.URL.RawQuery, params.User)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx := h.service.BeginSQLTx(ctx, true)
		users, err := h.service.GetBannedFriends(ctx, sqlTx, eventID, session.ID, params)
		if err != nil {
			_ = sqlTx.Rollback()
			response.Error(w, http.StatusInternalServerError, err)
			return
		}
		_ = sqlTx.Rollback()

		response.JSON(w, http.StatusOK, users)
	}
}

// GetByID gets an event by its id.
func (h *Handler) GetByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		cacheKey := cache.EventsKey(eventID)
		if item, err := h.cache.Get(cacheKey); err == nil {
			response.EncodedJSON(w, item.Value)
			return
		}

		sqlTx := h.service.BeginSQLTx(ctx, true)
		defer sqlTx.Rollback()

		if err := h.privacyFilter(ctx, r, sqlTx, eventID); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		event, err := h.service.GetByID(ctx, sqlTx, eventID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSONAndCache(h.cache, w, cacheKey, event)
	}
}

// GetConfirmed gets an event's confirmed users.
func (h *Handler) GetConfirmed() http.HandlerFunc {
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

		confirmed, err := h.service.GetConfirmed(ctx, sqlTx, eventID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, confirmed)
	}
}

// GetConfirmedFriends returns event confirmed users that are friends of the user passed.
func (h *Handler) GetConfirmedFriends() http.HandlerFunc {
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

		params, err := params.ParseQuery(r.URL.RawQuery, params.User)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx := h.service.BeginSQLTx(ctx, true)
		users, err := h.service.GetConfirmedFriends(ctx, sqlTx, eventID, session.ID, params)
		if err != nil {
			_ = sqlTx.Rollback()
			response.Error(w, http.StatusInternalServerError, err)
			return
		}
		_ = sqlTx.Rollback()

		response.JSON(w, http.StatusOK, users)
	}
}

// GetStatistics returns an event's predicates statistics.
func (h *Handler) GetStatistics() http.HandlerFunc {
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

// GetHosts gets an event's host users.
func (h *Handler) GetHosts() http.HandlerFunc {
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

		params, err := params.ParseQuery(r.URL.RawQuery, params.User)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		hosts, err := h.service.GetHosts(ctx, sqlTx, eventID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		var nextCursor string
		if len(hosts) > 0 {
			nextCursor = hosts[len(hosts)-1].ID
		}

		type resp struct {
			NextCursor string `json:"next_cursor,omitempty"`
			Hosts      []User `json:"hosts,omitempty"`
		}
		response.JSON(w, http.StatusOK, resp{NextCursor: nextCursor, Hosts: hosts})
	}
}

// GetInvited gets an event's invited users.
func (h *Handler) GetInvited() http.HandlerFunc {
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

		invited, err := h.service.GetInvited(ctx, sqlTx, eventID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, invited)
	}
}

// GetInvitedFriends returns event invited users that are friends of the user passed.
func (h *Handler) GetInvitedFriends() http.HandlerFunc {
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

		params, err := params.ParseQuery(r.URL.RawQuery, params.User)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx := h.service.BeginSQLTx(ctx, true)
		users, err := h.service.GetInvitedFriends(ctx, sqlTx, eventID, session.ID, params)
		if err != nil {
			_ = sqlTx.Rollback()
			response.Error(w, http.StatusInternalServerError, err)
			return
		}
		_ = sqlTx.Rollback()

		response.JSON(w, http.StatusOK, users)
	}
}

// GetLikes gets the users liking an event.
func (h *Handler) GetLikes() http.HandlerFunc {
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

		likes, err := h.service.GetLikedBy(ctx, sqlTx, eventID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, likes)
	}
}

// GetLikedByFriends returns users liking the event that are friends of the user passed.
func (h *Handler) GetLikedByFriends() http.HandlerFunc {
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

		params, err := params.ParseQuery(r.URL.RawQuery, params.User)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx := h.service.BeginSQLTx(ctx, true)
		users, err := h.service.GetLikedByFriends(ctx, sqlTx, eventID, session.ID, params)
		if err != nil {
			_ = sqlTx.Rollback()
			response.Error(w, http.StatusInternalServerError, err)
			return
		}
		_ = sqlTx.Rollback()

		response.JSON(w, http.StatusOK, users)
	}
}

// RemoveBanned removes the ban on a user.
func (h *Handler) RemoveBanned() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		errStatus, err := h.service.SQLTx(ctx, true, func(tx *sql.Tx) (int, error) {
			if err := h.requirePermissions(ctx, r, tx, eventID, permissions.BanUsers); err != nil {
				return http.StatusForbidden, err
			}
			return 0, nil
		})
		if err != nil {
			response.Error(w, errStatus, err)
			return
		}

		var reqBody userIDBody
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		if err := validate.ULID(reqBody.UserID); err != nil {
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

// RemoveConfirmed removes the confirmation of a user.
func (h *Handler) RemoveConfirmed() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		errStatus, err := h.service.SQLTx(ctx, true, func(tx *sql.Tx) (int, error) {
			if err := h.requirePermissions(ctx, r, tx, eventID, permissions.All); err != nil {
				return http.StatusForbidden, err
			}
			return 0, nil
		})
		if err != nil {
			response.Error(w, errStatus, err)
			return
		}

		var reqBody userIDBody
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		if err := validate.ULID(reqBody.UserID); err != nil {
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

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		errStatus, err := h.service.SQLTx(ctx, true, func(tx *sql.Tx) (int, error) {
			if err := h.requirePermissions(ctx, r, tx, eventID, permissions.All); err != nil {
				return http.StatusForbidden, err
			}
			return 0, nil
		})
		if err != nil {
			response.Error(w, errStatus, err)
			return
		}

		var reqBody userIDBody
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		if err := validate.ULID(reqBody.UserID); err != nil {
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
		if err := validate.ULIDs(eventID, reqBody.UserID); err != nil {
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

		params, err := params.ParseQuery(r.URL.RawQuery, params.Event)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		events, err := h.service.Search(ctx, query, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		var nextCursor string
		if len(events) > 0 {
			nextCursor = events[len(events)-1].ID
		}

		type resp struct {
			NextCursor string  `json:"next_cursor,omitempty"`
			Events     []Event `json:"events,omitempty"`
		}
		response.JSON(w, http.StatusOK, resp{NextCursor: nextCursor, Events: events})
	}
}

// Update updates an event.
func (h *Handler) Update() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx := h.service.BeginSQLTx(ctx, false)
		defer sqlTx.Rollback()

		if err := h.requirePermissions(ctx, r, sqlTx, eventID, permissions.All); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		var uptEvent UpdateEvent
		if err := json.NewDecoder(r.Body).Decode(&uptEvent); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}
		defer r.Body.Close()

		if err := uptEvent.Validate(); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if err := h.service.Update(ctx, sqlTx, eventID, uptEvent); err != nil {
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

// UserJoin joins a user to a private event and sets it the viewer role.
func (h *Handler) UserJoin() http.HandlerFunc {
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

		sqlTx := h.service.BeginSQLTx(ctx, false)

		if err := h.service.UserJoin(ctx, sqlTx, eventID, session.ID); err != nil {
			_ = sqlTx.Rollback()
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		if err := sqlTx.Commit(); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSONMessage(w, http.StatusOK, "Joined event")
	}
}

// privacyFilter lets through only users that can fetch the event data if it's private,
// if it's public it lets anyone in.
func (h *Handler) privacyFilter(ctx context.Context, r *http.Request, tx *sql.Tx, eventID string) error {
	session, err := auth.GetSession(ctx, r)
	if err != nil {
		return err
	}

	isPublic, err := h.service.IsPublic(ctx, tx, eventID)
	if err != nil {
		return errors.Wrap(err, "privacyFilter")
	}

	if isPublic {
		// Event is public, no restrictions applied
		return nil
	}

	// If the user has a role in the event, then he's able to retrieve its information
	hasRole, err := h.service.UserHasRole(ctx, tx, eventID, session.ID)
	if err != nil {
		return errors.Wrap(err, "privacyFilter")
	}
	if !hasRole {
		return errAccessDenied
	}

	return nil
}

// requirePermissions returns an error if the user hasn't the permissions required on the event passed.
func (h *Handler) requirePermissions(ctx context.Context, r *http.Request, tx *sql.Tx, eventID string, permRequired ...string) error {
	session, err := auth.GetSession(ctx, r)
	if err != nil {
		return err
	}

	role, err := h.service.GetUserRole(ctx, tx, eventID, session.ID)
	if err != nil {
		return errors.Wrap(err, "requirePermissions")
	}

	userPermKeys := sliceToMap(role.PermissionKeys)
	if err := permissions.Require(userPermKeys, permRequired...); err != nil {
		return errAccessDenied
	}

	return nil
}

func sliceToMap(slice []string) map[string]struct{} {
	mp := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		mp[s] = struct{}{}
	}
	return mp
}
