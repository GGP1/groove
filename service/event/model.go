package event

import (
	"time"

	"github.com/GGP1/groove/service/event/media"
	"github.com/GGP1/groove/service/event/product"
	"github.com/GGP1/groove/service/event/zone"

	"github.com/pkg/errors"
)

// Event type
const (
	Meeting eventType = iota + 1
	Party
	Conference
	Talk
	Show
	Class
	Birthday
	Reunion
	Match
	League
	Tournament
	Trip
	Protest
	GrandPrix
	Marriage
	Concert
	Marathon
	Hackathon
	Ceremony
	Graduation
	Tribute
	Anniversary
)

// eventType of an event.
type eventType uint8

// Event represents an event.
//
// Use pointers to distinguish default values.
type Event struct {
	ID          string            `json:"id,omitempty"`
	Name        string            `json:"name,omitempty"`
	Description string            `json:"description,omitempty"`
	Type        eventType         `json:"type,omitempty"`
	Public      *bool             `json:"public,omitempty"`
	Virtual     *bool             `json:"virtual,omitempty"`
	URL         *string           `json:"url,omitempty"`
	StartTime   time.Time         `json:"start_time,omitempty" db:"start_time"`
	EndTime     time.Time         `json:"end_time,omitempty" db:"end_time"`
	MinAge      uint16            `json:"min_age,omitempty" db:"min_age"`
	TicketCost  *uint64           `json:"ticket_cost,omitempty" db:"ticket_cost"`
	Slots       *uint64           `json:"slots,omitempty"`
	Products    []product.Product `json:"products,omitempty"`
	Media       []media.Media     `json:"media,omitempty"`
	Zones       []zone.Zone       `json:"zones,omitempty"`
	CreatedAt   *time.Time        `json:"created_at,omitempty" db:"created_at"`
	UpdatedAt   *time.Time        `json:"updated_at,omitempty" db:"updated_at"`
}

// Statistics contains statistics from an event.
type Statistics struct {
	Banned    *uint64 `json:"banned_count,omitempty"`
	Confirmed *uint64 `json:"confirmed_count,omitempty"`
	Invited   *uint64 `json:"invited_count,omitempty"`
	Likes     *uint64 `json:"likes_count,omitempty"`
}

// CreateEvent is the structure used to create an event.
type CreateEvent struct {
	HostID      string    `json:"host_id,omitempty"`
	Name        string    `json:"name,omitempty"`
	Description string    `json:"description,omitempty"`
	Type        eventType `json:"type,omitempty"`
	Public      *bool     `json:"public,omitempty"`
	Virtual     *bool     `json:"virtual,omitempty"`
	URL         *string   `json:"url,omitempty"`
	LocationID  *int      `json:"location_id,omitempty"`
	StartTime   time.Time `json:"start_time,omitempty" db:"start_time"`
	EndTime     time.Time `json:"end_time,omitempty" db:"end_time"`
	MinAge      uint16    `json:"min_age,omitempty" db:"min_age"`
	Slots       uint64    `json:"slots,omitempty"`
	TicketCost  uint64    `json:"ticket_cost,omitempty" db:"ticket_cost"`
}

// Validate verifies if the event received is valid.
func (c CreateEvent) Validate() error {
	if c.Name == "" {
		return errors.New("name required")
	}
	if c.Type == 0 {
		return errors.New("type required")
	}
	if c.Public == nil {
		return errors.New("public required")
	}
	if c.Virtual == nil {
		return errors.New("virtual required")
	}
	if c.LocationID == nil {
		return errors.New("location_id required")
	}
	if c.StartTime.IsZero() {
		return errors.New("start_time required")
	}
	if c.StartTime.Before(time.Now()) {
		return errors.New("start_time must be sometime in the future")
	}
	if c.EndTime.IsZero() {
		return errors.New("end_time required")
	}
	if c.EndTime.Before(c.StartTime) {
		return errors.New("end_time must be after start_time")
	}
	if c.MinAge == 0 {
		return errors.New("min_age required")
	}
	if c.Slots == 0 {
		return errors.New("slots required")
	}
	return nil
}

// UpdateEvent is the struct used to update events.
//
// Use pointers to distinguish default values.
type UpdateEvent struct {
	Name        *string    `json:"name,omitempty"`
	Description *string    `json:"description,omitempty"`
	Type        *eventType `json:"type,omitempty"`
	StartTime   *time.Time `json:"start_time,omitempty" db:"start_time"`
	EndTime     *time.Time `json:"end_time,omitempty" db:"end_time"`
	MinAge      *uint16    `json:"min_age,omitempty" db:"min_age"`
	TicketCost  *uint64    `json:"ticket_cost,omitempty" db:"ticket_cost"`
	Slots       *uint64    `json:"slots,omitempty"`
}

// Validate verifies the values inside the struct are valid.
func (u UpdateEvent) Validate() error {
	if u == (UpdateEvent{}) {
		return errors.New("no values provided")
	}
	if u.Name != nil {
		if *u.Name == "" {
			return errors.New("invalid name")
		}
	}
	if u.Type != nil {
		if *u.Type == 0 {
			return errors.New("invalid type")
		}
	}
	if u.StartTime != nil || u.EndTime != nil {
		if u.StartTime == nil || u.EndTime == nil {
			return errors.New("both start_time and end_time must be modified together")
		}
		if u.StartTime.IsZero() {
			return errors.New("invalid start_time")
		}
		if u.StartTime.Before(time.Now()) {
			return errors.New("start_time must be sometime in the future")
		}
		if u.EndTime.IsZero() {
			return errors.New("invalid end_time")
		}
		if u.EndTime.Before(*u.StartTime) {
			return errors.New("end_time must be after start_time")
		}
	}
	if u.MinAge != nil {
		if *u.MinAge == 0 {
			return errors.New("min_age must be higher than zero")
		}
	}
	if u.Slots != nil {
		if *u.Slots == 0 {
			return errors.New("slots must be higher than zero")
		}
	}
	return nil
}

// User represents a user in the context of an event.
//
// Use pointers to distinguish default values.
type User struct {
	ID              string     `json:"id,omitempty"`
	Name            string     `json:"name,omitempty"`
	Username        string     `json:"username,omitempty"`
	Email           string     `json:"email,omitempty"`
	BirthDate       *time.Time `json:"birth_date,omitempty" db:"birth_date"`
	Description     string     `json:"description,omitempty"`
	Premium         *bool      `json:"premium,omitempty"`
	Private         *bool      `json:"private,omitempty"`
	VerifiedEmail   *bool      `json:"verified_email,omitempty" db:"verified_email"`
	ProfileImageURL string     `json:"profile_image_url,omitempty" db:"profile_image_url"`
	FriendsCount    *uint64    `json:"friends_count,omitempty"`
	CreatedAt       *time.Time `json:"created_at,omitempty" db:"created_at"`
	UpdatedAt       *time.Time `json:"updated_at,omitempty" db:"updated_at"`
}
