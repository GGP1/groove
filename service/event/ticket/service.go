package ticket

import (
	"context"
	"database/sql"

	"github.com/GGP1/groove/internal/cache"
	"github.com/GGP1/groove/internal/roles"
	"github.com/GGP1/groove/internal/txgroup"
	"github.com/GGP1/groove/model"
	"github.com/GGP1/groove/service/auth"
	"github.com/GGP1/groove/service/event/role"
	"github.com/GGP1/groove/storage/postgres"
	"github.com/GGP1/sqan"

	"github.com/pkg/errors"
)

/*
Ticket purchase flow:
1. User presses the but button and is taken to the payment gateway to perform the purchase
2. The payment is set to pending
3. When the user authenticates in the event's client, the payment is executed.
5% of the cost goes to Groove, ~3-4% to the payment gateway fees and the rest to the host (~91%)
3.5. If the user refunds the ticket, put the transaction to cancelled and return the cost-fees-penalty to the user.

TODO: the ticket service will have to be integrated with the user, payment and notification ones.
There's a need to mantain a state of the tickets bought by each user so it's possible to determine
whether they can ask for a refund or not. Should gifted tickets be "refundable"?
Also, when a users buys a ticket for other one, notify the one that received the gift
*/

// Service interface for the ticket service.
type Service interface {
	Available(ctx context.Context, eventID, ticketName string) (int64, error)
	Buy(ctx context.Context, session auth.Session, eventID, ticketName string, userIDs []string) error
	Create(ctx context.Context, eventID string, ticket model.Ticket) error
	Delete(ctx context.Context, eventID, ticketName string) error
	GetByName(ctx context.Context, eventID, ticketName string) (model.Ticket, error)
	Get(ctx context.Context, eventID string) ([]model.Ticket, error)
	Refund(ctx context.Context, session auth.Session, eventID, ticketName string) error
	Update(ctx context.Context, eventID, ticketName string, updateTicket model.UpdateTicket) error
}

type service struct {
	db    *sql.DB
	cache cache.Client

	roleService role.Service
}

// NewService returns a new service
func NewService(db *sql.DB, cache cache.Client, roleService role.Service) Service {
	return &service{
		db:          db,
		cache:       cache,
		roleService: roleService,
	}
}

// Available returns the number of available tickets.
func (s *service) Available(ctx context.Context, eventID, ticketName string) (int64, error) {
	q := "SELECT available_count FROM events_tickets WHERE event_id=$1 AND name=$2"
	return postgres.QueryInt(ctx, s.db, q, eventID, ticketName)
}

// Buy performs the operations necessary when a ticket is bought.
func (s *service) Buy(ctx context.Context, session auth.Session, eventID, ticketName string, userIDs []string) error {
	sqlTx := txgroup.SQLTx(ctx)

	// TODO: create auth user payment with a pending status. Add cost to RETURNING to get the ticket's cost.
	// Updating will fail if there are not enough available tickets but it's not the best way to check it
	q := `UPDATE events_tickets SET 
	available_count = available_count - 1 
	WHERE event_id=$1 AND name=$2
	RETURNING linked_role`
	row := sqlTx.QueryRowContext(ctx, q, eventID, ticketName)
	var linkedRole string
	if err := row.Scan(&linkedRole); err != nil {
		return errors.Wrap(err, "updating ticket availability")
	}

	// TODO: notify the users that are not the authenticated that they have been gifted a ticket to the event
	return s.roleService.SetRole(ctx, eventID, model.SetRole{RoleName: linkedRole, UserIDs: userIDs})
}

// Create adds a ticket to the event.
func (s *service) Create(ctx context.Context, eventID string, ticket model.Ticket) error {
	sqlTx := txgroup.SQLTx(ctx)

	if ticket.LinkedRole != "" && !roles.Reserved.Exists(ticket.LinkedRole) {
		row := sqlTx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM events_roles WHERE name=$1)", ticket.LinkedRole)
		var exists bool
		if err := row.Scan(&exists); err != nil {
			return errors.Wrap(err, "querying role existence")
		}

		if !exists {
			return errors.Errorf("role %q does not exists in the event", ticket.LinkedRole)
		}
	}

	q := `INSERT INTO events_tickets 
	(event_id, available_count, name, description, cost, linked_role)
	VALUES
	($1, $2, $3, $4, $5, $6)`
	_, err := sqlTx.ExecContext(ctx, q, eventID, ticket.AvailableCount, ticket.Name,
		ticket.Description, ticket.Cost, ticket.LinkedRole)
	if err != nil {
		return errors.Wrap(err, "creating ticket")
	}

	return nil
}

// Delete removes a ticket from the event.
func (s *service) Delete(ctx context.Context, eventID, ticketName string) error {
	sqlTx := txgroup.SQLTx(ctx)

	q := "DELETE FROM events_tickets WHERE event_id=$1 AND name=$2"
	if _, err := sqlTx.ExecContext(ctx, q, eventID, ticketName); err != nil {
		return errors.Wrap(err, "deleting ticket")
	}
	return nil
}

// GetByName returns the ticket with the given name.
func (s *service) GetByName(ctx context.Context, eventID, ticketName string) (model.Ticket, error) {
	q := "SELECT available_count, name, description, cost, linked_role FROM events_tickets WHERE event_id=$1"
	rows, err := s.db.QueryContext(ctx, q, eventID, ticketName)
	if err != nil {
		return model.Ticket{}, errors.Wrap(err, "scanning ticket")
	}

	var ticket model.Ticket
	if err := sqan.Row(&ticket, rows); err != nil {
		return model.Ticket{}, err
	}

	return ticket, nil
}

// Get returns an event's tickets.
func (s *service) Get(ctx context.Context, eventID string) ([]model.Ticket, error) {
	q := "SELECT available_count, name, description, cost, linked_role FROM events_tickets WHERE event_id=$1"
	rows, err := s.db.QueryContext(ctx, q, eventID)
	if err != nil {
		return nil, errors.Wrap(err, "querying tickets")
	}

	var tickets []model.Ticket
	if err := sqan.Rows(&tickets, rows); err != nil {
		return nil, errors.Wrap(err, "scanning tickets")
	}

	return tickets, nil
}

// Refund performs the operations necessary when a ticket is refunded.
func (s *service) Refund(ctx context.Context, session auth.Session, eventID, ticketName string) error {
	sqlTx := txgroup.SQLTx(ctx)

	// TODO: refund auth user with the ticket cost - penalties/fees and remove the pending state.
	// ONLY if the had previously bought a ticket
	q := "UPDATE events_tickets SET available_count = available_count + 1 WHERE event_id=$1 AND name=$2"
	if _, err := sqlTx.ExecContext(ctx, q, eventID, ticketName); err != nil {
		return errors.Wrap(err, "updating ticket availability")
	}
	return nil
}

// Update updates a ticket from the event.
func (s *service) Update(ctx context.Context, eventID, ticketName string, updateTicket model.UpdateTicket) error {
	sqlTx := txgroup.SQLTx(ctx)

	q := `UPDATE events_tickets SET 
	description = COALESCE($3,description),
	available_count = COALESCE($4,available_count), 
	cost = COALESCE($5,cost), 
	linked_role = COALESCE($6,linked_role) 
	WHERE event_id=$1 AND name=$2
	RETURNING (SELECT available_count FROM events_tickets WHERE event_id=$1 AND name=$2)`
	row := sqlTx.QueryRowContext(ctx, q, q, eventID, ticketName, updateTicket.Description,
		updateTicket.AvailableCount, updateTicket.Cost, updateTicket.LinkedRole)

	var oldAvailableCount int64
	if err := row.Scan(&oldAvailableCount); err != nil {
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
