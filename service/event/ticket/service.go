package ticket

import (
	"context"
	"database/sql"

	"github.com/GGP1/groove/internal/cache"
	"github.com/GGP1/groove/internal/roles"
	"github.com/GGP1/groove/internal/sqltx"
	"github.com/GGP1/groove/service/event/role"
	"github.com/GGP1/groove/storage/postgres"

	"github.com/pkg/errors"
)

/*
Ticket purchase flow:
1. User presses the but button and is taken to the payment gateway to perform the purchase
2. The payment is set to pending
3. When the user authenticates in the event's client, the payment is executed.
5% of the cost goes to Groove, ~3-4% to the payment gateway fees and the rest to the host (~91%)
3.5. If the user refunds the ticket, put the transaction to cancelled and return the cost-fees-penalty to the user.
*/

// Service interface for the ticket service.
type Service interface {
	AvailableTickets(ctx context.Context, eventID, ticketName string) (int64, error)
	BuyTicket(ctx context.Context, eventID, userID, ticketName string) error
	CreateTickets(ctx context.Context, eventID string, tickets []Ticket) error
	DeleteTicket(ctx context.Context, eventID, ticketName string) error
	GetTicket(ctx context.Context, eventID, ticketName string) (Ticket, error)
	GetTickets(ctx context.Context, eventID string) ([]Ticket, error)
	RefundTicket(ctx context.Context, eventID, userID, ticketName string) error
	UpdateTicket(ctx context.Context, eventID, ticketName string, updateTicket UpdateTicket) error
}

type service struct {
	db    *sql.DB
	cache cache.Client

	roleService role.Service
}

// NewService returns a new service
func NewService(db *sql.DB, cache cache.Client, roleService role.Service) Service {
	return service{
		db:          db,
		cache:       cache,
		roleService: roleService,
	}
}

// AvailableTickets returns the number of available tickets.
func (s service) AvailableTickets(ctx context.Context, eventID, ticketName string) (int64, error) {
	sqlTx := sqltx.FromContext(ctx)

	q := "SELECT available_count FROM events_tickets WHERE event_id=$1 AND name=$2"
	return postgres.QueryInt(ctx, sqlTx, q, eventID, ticketName)
}

// BuyTicket performs the operations necessary when a ticket is bought.
func (s service) BuyTicket(ctx context.Context, eventID, userID, ticketName string) error {
	sqlTx := sqltx.FromContext(ctx)

	// TODO: create user payment with a pending status. Add cost to RETURNING to get the ticket's cost.
	// Updating will fail if there are not enough available tickets but it's not the best way to check it
	q := `UPDATE events_tickets SET 
	available_count = available_count - 1 
	WHERE event_id=$1 AND name=$2
	RETURNING linked_role`
	linkedRole, err := postgres.QueryString(ctx, sqlTx, q, eventID, ticketName)
	if err != nil {
		return errors.Wrap(err, "updating ticket availability")
	}

	return s.roleService.SetRoles(ctx, eventID, linkedRole, userID)
}

// CreateTickets adds n tickets to the event.
func (s service) CreateTickets(ctx context.Context, eventID string, tickets []Ticket) error {
	sqlTx := sqltx.FromContext(ctx)

	stmt, err := postgres.BulkInsert(ctx, sqlTx, "events_tickets", "event_id", "available_count", "name", "cost", "linked_role")
	if err != nil {
		return err
	}
	defer stmt.Close()

	q := "SELECT EXISTS(SELECT 1 FROM events_roles WHERE name=$1)"
	for _, ticket := range tickets {
		if ticket.LinkedRole != "" && !roles.Reserved.Exists(ticket.LinkedRole) {
			exists, err := postgres.QueryBool(ctx, sqlTx, q, ticket.LinkedRole)
			if err != nil {
				return errors.Wrap(err, "querying role name")
			}
			if !exists {
				return errors.Errorf("role %q does not exists in the event", ticket.LinkedRole)
			}
		}

		_, err = stmt.ExecContext(ctx, eventID, ticket.AvailableCount, ticket.Name, ticket.Cost, ticket.LinkedRole)
		if err != nil {
			return errors.Wrapf(err, "creating %q ticket", ticket.Name)
		}
	}

	// Flush buffered data
	if _, err := stmt.Exec(); err != nil {
		return errors.Wrap(err, "flushing buffered data")
	}

	return nil
}

// DeleteTicket removes a ticket from the event.
func (s service) DeleteTicket(ctx context.Context, eventID, ticketName string) error {
	sqlTx := sqltx.FromContext(ctx)

	q := "DELETE FROM events_tickets WHERE event_id=$1 AND name=$2"
	if _, err := sqlTx.ExecContext(ctx, q, eventID, ticketName); err != nil {
		return errors.Wrap(err, "deleting ticket")
	}
	return nil
}

// GetTicket returns the ticket with the given name.
func (s service) GetTicket(ctx context.Context, eventID, ticketName string) (Ticket, error) {
	sqlTx := sqltx.FromContext(ctx)

	q := "SELECT available_count, name, cost, linked_role FROM events_tickets WHERE event_id=$1"
	row := sqlTx.QueryRowContext(ctx, q, eventID, ticketName)
	var ticket Ticket
	err := row.Scan(&ticket.AvailableCount, &ticket.Name, &ticket.Cost, &ticket.LinkedRole)
	if err != nil {
		return Ticket{}, errors.Wrap(err, "scanning ticket")
	}

	return ticket, nil
}

// GetTickets returns an event's tickets.
func (s service) GetTickets(ctx context.Context, eventID string) ([]Ticket, error) {
	sqlTx := sqltx.FromContext(ctx)

	q := "SELECT available_count, name, cost, linked_role FROM events_tickets WHERE event_id=$1"
	rows, err := sqlTx.QueryContext(ctx, q, eventID)
	if err != nil {
		return nil, errors.Wrap(err, "fetching tickets")
	}

	var (
		tickets []Ticket
		ticket  Ticket
	)
	for rows.Next() {
		err := rows.Scan(&ticket.AvailableCount, &ticket.Name, &ticket.Cost, &ticket.LinkedRole)
		if err != nil {
			return nil, errors.Wrap(err, "scanning tickets")
		}

		tickets = append(tickets, ticket)
	}

	return tickets, nil
}

// RefundTicket performs the operations necessary when a ticket is refunded.
func (s service) RefundTicket(ctx context.Context, eventID, userID, ticketName string) error {
	sqlTx := sqltx.FromContext(ctx)

	// TODO: refund user with the ticket cost - penalties/fees and remove the pending state.
	q := "UPDATE events_tickets SET available_count = available_count + 1 WHERE event_id=$1 AND name=$2"
	if _, err := sqlTx.ExecContext(ctx, q, eventID, ticketName); err != nil {
		return errors.Wrap(err, "updating ticket availability")
	}
	return nil
}

// UpdateTicket updates a ticket from the event.
func (s service) UpdateTicket(ctx context.Context, eventID, ticketName string, updateTicket UpdateTicket) error {
	sqlTx := sqltx.FromContext(ctx)

	q := `UPDATE events_tickets SET 
	available_count = COALESCE($3,available_count), 
	cost = COALESCE($4,cost), 
	linked_role = COALESCE($5,linked_role) 
	WHERE event_id=$1 AND name=$2
	RETURNING (SELECT available_count FROM events_tickets WHERE event_id=$1 AND name=$2`
	oldAvailableCount, err := postgres.QueryInt(ctx, sqlTx, q, eventID, ticketName,
		updateTicket.AvailableCount, updateTicket.Cost, updateTicket.LinkedRole)
	if err != nil {
		return errors.Wrap(err, "updating ticket")
	}

	if updateTicket.AvailableCount != nil {
		// if available_count is updated, update the event's total slots
		q2 := `UPDATE events SET slots = slots + $2 WHERE event_id=$1`
		if _, err := sqlTx.ExecContext(ctx, q2, eventID, *updateTicket.AvailableCount-oldAvailableCount); err != nil {
			return errors.Wrap(err, "updating event slots")
		}
	}

	return nil
}
