package ticket

import (
	"errors"

	"github.com/GGP1/groove/internal/validate"
)

// Use sockets to update the availability of the tickets in real time to the users

// Ticket represents an event's ticket.
type Ticket struct {
	Name           string  `json:"name,omitempty"`
	AvailableCount *uint64 `json:"available_count,omitempty" db:"available_count"`
	Cost           *uint64 `json:"cost,omitempty"` // 100 = 1 USD
	// LinkedRole is the role assigned to the user when buying the ticket, default is "attendant"
	LinkedRole string `json:"linked_role,omitempty" db:"linked_role"`
}

// Validate verifies if the ticket values are valid.
func (t Ticket) Validate() error {
	if t.Name == "" {
		return errors.New("name required")
	}
	if t.AvailableCount == nil {
		return errors.New("available_count required")
	} else if *t.AvailableCount < 0 {
		return errors.New("available_count must be higher than 0")
	}
	if t.Cost == nil {
		return errors.New("cost required")
	}
	if t.LinkedRole != "" {
		if err := validate.RoleName(t.LinkedRole); err != nil {
			return err
		}
	}
	return nil
}

// UpdateTicket is the structure used to update a ticket.
type UpdateTicket struct {
	AvailableCount *int64  `json:"available,omitempty" db:"available_count"`
	Cost           *int64  `json:"cost,omitempty"`
	LinkedRole     *string `json:"linked_role,omitempty" db:"linked_role"`
}

// Validate verifies if the ticket values for update are valid.
func (u UpdateTicket) Validate() error {
	if u.LinkedRole != nil {
		if *u.LinkedRole == "" {
			return errors.New("linked_role cannot be empty")
		}
	}
	if u.AvailableCount != nil && *u.AvailableCount < 0 {
		return errors.New("available_count must be higher than 0")
	}
	return nil
}
