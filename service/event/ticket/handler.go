package ticket

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/permissions"
	"github.com/GGP1/groove/internal/response"
	"github.com/GGP1/groove/service/auth"
	"github.com/GGP1/groove/service/event/role"
	"github.com/GGP1/groove/storage/postgres"
)

// Handler handles ticket service endpoints.
type Handler struct {
	db *sql.DB

	service     Service
	roleService role.Service
}

// NewHandler returns a new ticket handler.
func NewHandler(db *sql.DB, service Service, roleService role.Service) Handler {
	return Handler{
		db:          db,
		service:     service,
		roleService: roleService,
	}
}

// Available returns the number of available tickets.
func (h Handler) Available() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		eventID, ticketName, err := params.IDAndNameFromCtx(rctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, true)
		defer sqlTx.Rollback()

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
		rctx := r.Context()

		session, err := auth.GetSession(rctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		eventID, ticketName, err := params.IDAndNameFromCtx(rctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, false)
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
		rctx := r.Context()

		eventID, err := params.IDFromCtx(rctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, false)
		defer sqlTx.Rollback()

		if err := h.roleService.RequirePermissions(ctx, r, eventID, permissions.ModifyTickets); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		var tickets []Ticket
		if err := json.NewDecoder(r.Body).Decode(&tickets); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}
		defer r.Body.Close()

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
		rctx := r.Context()

		eventID, ticketName, err := params.IDAndNameFromCtx(rctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, false)
		defer sqlTx.Rollback()

		if err := h.roleService.RequirePermissions(ctx, r, eventID, permissions.ModifyTickets); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

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
		rctx := r.Context()

		eventID, err := params.IDFromCtx(rctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, true)
		defer sqlTx.Rollback()

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
		rctx := r.Context()

		eventID, ticketName, err := params.IDAndNameFromCtx(rctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, true)
		defer sqlTx.Rollback()

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
		rctx := r.Context()

		session, err := auth.GetSession(rctx, r)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		eventID, ticketName, err := params.IDAndNameFromCtx(rctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, false)
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
		rctx := r.Context()

		eventID, ticketName, err := params.IDAndNameFromCtx(rctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, false)
		defer sqlTx.Rollback()

		if err := h.roleService.RequirePermissions(ctx, r, eventID, permissions.ModifyTickets); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		var updateTicket UpdateTicket
		if err := json.NewDecoder(r.Body).Decode(&updateTicket); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}
		defer r.Body.Close()

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
