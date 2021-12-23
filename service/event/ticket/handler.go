package ticket

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/response"
	"github.com/GGP1/groove/internal/validate"
	"github.com/GGP1/groove/model"
	"github.com/GGP1/groove/service/auth"
	"github.com/GGP1/groove/storage/postgres"
)

// Handler handles ticket service endpoints.
type Handler struct {
	db *sql.DB

	service Service
}

// NewHandler returns a new ticket handler.
func NewHandler(db *sql.DB, service Service) Handler {
	return Handler{
		db:      db,
		service: service,
	}
}

// Available returns the number of available tickets.
func (h *Handler) Available() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, ticketName, err := params.IDAndNameFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		availableTickets, err := h.service.Available(ctx, eventID, ticketName)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSONCount(w, http.StatusOK, "available_tickets_count", availableTickets)
	}
}

// Buy performs the operations necessary when a ticket is bought.
func (h *Handler) Buy() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		session, err := auth.GetSession(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		eventID, ticketName, err := params.IDAndNameFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		var body struct {
			UserIDs []string `json:"user_ids,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		body.UserIDs, err = reduceAndValidate(body.UserIDs)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTxOpts(ctx, h.db, sql.LevelSerializable)
		defer sqlTx.Rollback()

		if err := h.service.Buy(ctx, session, eventID, ticketName, body.UserIDs); err != nil {
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

// Create adds n tickets to the event.
func (h *Handler) Create() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		var ticket model.Ticket
		if err := json.NewDecoder(r.Body).Decode(&ticket); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		sqlTx, ctx := postgres.BeginTx(ctx, h.db)
		defer sqlTx.Rollback()

		if err := h.service.Create(ctx, eventID, ticket); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		if err := sqlTx.Commit(); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, response.Name{Name: ticket.Name})
	}
}

// Delete removes a ticket from the event.
func (h *Handler) Delete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, ticketName, err := params.IDAndNameFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(ctx, h.db)
		defer sqlTx.Rollback()

		if err := h.service.Delete(ctx, eventID, ticketName); err != nil {
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

// Get returns an event's tickets.
func (h *Handler) Get() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		tickets, err := h.service.Get(ctx, eventID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, tickets)
	}
}

// GetByName returns the ticket with the given name.
func (h *Handler) GetByName() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, ticketName, err := params.IDAndNameFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		ticket, err := h.service.GetByName(ctx, eventID, ticketName)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, ticket)
	}
}

// Refund performs the operations necessary when a ticket is refunded.
func (h *Handler) Refund() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		session, err := auth.GetSession(ctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		eventID, ticketName, err := params.IDAndNameFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTxOpts(ctx, h.db, sql.LevelSerializable)
		defer sqlTx.Rollback()

		if err := h.service.Refund(ctx, session, eventID, ticketName); err != nil {
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

// Update updates a ticket from the event.
func (h *Handler) Update() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, ticketName, err := params.IDAndNameFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		var updateTicket model.UpdateTicket
		if err := json.NewDecoder(r.Body).Decode(&updateTicket); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		sqlTx, ctx := postgres.BeginTx(ctx, h.db)
		defer sqlTx.Rollback()

		if err := h.service.Update(ctx, eventID, ticketName, updateTicket); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		if err := sqlTx.Commit(); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, response.Name{Name: ticketName})
	}
}

// reduceAndValidate removes duplicates from a slice while verifying that the ids are valid.
func reduceAndValidate(ids []string) ([]string, error) {
	if len(ids) == 1 {
		if err := validate.ULID(ids[0]); err != nil {
			return nil, err
		}
		return ids, nil
	}

	mp := make(map[string]struct{}, len(ids))
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		if _, ok := mp[id]; !ok {
			if err := validate.ULID(id); err != nil {
				return nil, err
			}
			mp[id] = struct{}{}
			result = append(result, id)
		}
	}

	return result, nil
}
