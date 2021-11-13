package ticket

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/response"
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
func (h Handler) Available() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, ticketName, err := params.IDAndNameFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		availableTickets, err := h.service.AvailableTickets(ctx, eventID, ticketName)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSONCount(w, http.StatusOK, "available_tickets_count", availableTickets)
	}
}

// Buy performs the operations necessary when a ticket is bought.
func (h Handler) Buy() http.HandlerFunc {
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

		err = h.service.BuyTicket(ctx, eventID, session.ID, ticketName)
		if err != nil {
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
func (h Handler) Create() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		var tickets []Ticket
		if err := json.NewDecoder(r.Body).Decode(&tickets); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}
		defer r.Body.Close()

		sqlTx, ctx := postgres.BeginTx(ctx, h.db)
		defer sqlTx.Rollback()

		err = h.service.CreateTickets(ctx, eventID, tickets)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		if err := sqlTx.Commit(); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusCreated, tickets)
	}
}

// Delete removes a ticket from the event.
func (h Handler) Delete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, ticketName, err := params.IDAndNameFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(ctx, h.db)
		defer sqlTx.Rollback()

		if err := h.service.DeleteTicket(ctx, eventID, ticketName); err != nil {
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
func (h Handler) Get() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		tickets, err := h.service.GetTickets(ctx, eventID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, tickets)
	}
}

// GetByName returns the ticket with the given name.
func (h Handler) GetByName() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, ticketName, err := params.IDAndNameFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		ticket, err := h.service.GetTicket(ctx, eventID, ticketName)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, ticket)
	}
}

// Refund performs the operations necessary when a ticket is refunded.
func (h Handler) Refund() http.HandlerFunc {
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

		err = h.service.RefundTicket(ctx, eventID, session.ID, ticketName)
		if err != nil {
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
func (h Handler) Update() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, ticketName, err := params.IDAndNameFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		var updateTicket UpdateTicket
		if err := json.NewDecoder(r.Body).Decode(&updateTicket); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}
		defer r.Body.Close()

		sqlTx, ctx := postgres.BeginTx(ctx, h.db)
		defer sqlTx.Rollback()

		if err := h.service.UpdateTicket(ctx, eventID, ticketName, updateTicket); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		if err := sqlTx.Commit(); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, updateTicket)
	}
}
