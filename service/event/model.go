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
	_ eventType = iota
	Meeting
	Party
	Tournament
	League
	GrandPrix
	Birthday
	Marriage
	Match
	Trip
	Concert
	Conference
	Marathon
	Hackathon
	Reunion
	Ceremony
	Graduation
	Talk
	Show
	Protest
	Tribute
)

// eventType of an event.
type eventType uint8

// Event represents an event.
//
// Use pointers to distinguish default values.
type Event struct {
	ID             string            `json:"id,omitempty"`
	Name           string            `json:"name,omitempty"`
	Type           eventType         `json:"type,omitempty"`
	Public         *bool             `json:"public,omitempty"`
	StartTime      uint64            `json:"start_time,omitempty" db:"start_time"`
	EndTime        uint64            `json:"end_time,omitempty" db:"end_time"`
	MinAge         uint16            `json:"min_age,omitempty" db:"min_age"`
	TicketCost     *uint64           `json:"ticket_cost,omitempty" db:"ticket_cost"`
	Slots          *uint64           `json:"slots,omitempty"`
	BannedCount    *uint64           `json:"banned_count,omitempty"`
	ConfirmedCount *uint64           `json:"confirmed_count,omitempty"`
	InvitedCount   *uint64           `json:"invited_count,omitempty"`
	LikesCount     *uint64           `json:"likes_count,omitempty"`
	Location       Location          `json:"location,omitempty"`
	Products       []product.Product `json:"products,omitempty"`
	Media          []media.Media     `json:"media,omitempty"`
	Zones          []zone.Zone       `json:"zones,omitempty"`
	CreatedAt      *time.Time        `json:"created_at,omitempty" db:"created_at"`
	UpdatedAt      *time.Time        `json:"updated_at,omitempty" db:"updated_at"`
}

// CreateEvent is the structure used to create an event.
type CreateEvent struct {
	HostID     string    `json:"host_id,omitempty"`
	Name       string    `json:"name,omitempty"`
	Type       eventType `json:"type,omitempty"`
	Public     *bool     `json:"public,omitempty"`
	StartTime  uint64    `json:"start_time,omitempty" db:"start_time"`
	EndTime    uint64    `json:"end_time,omitempty" db:"end_time"`
	MinAge     uint16    `json:"min_age,omitempty" db:"min_age"`
	Slots      uint64    `json:"slots,omitempty"`
	TicketCost uint64    `json:"ticket_cost,omitempty" db:"ticket_cost"`
	Location   Location  `json:"location,omitempty"`
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
	if c.StartTime == 0 {
		return errors.New("start_time required")
	}
	if c.EndTime == 0 {
		return errors.New("end_time required")
	}
	if c.MinAge == 0 {
		return errors.New("min_age required")
	}
	if c.Slots == 0 {
		return errors.New("slots required")
	}
	return c.Location.Validate()
}

// Location represents the place where the event will take place, it could be on-site or virtual.
type Location struct {
	Country  string `json:"country,omitempty"`
	State    string `json:"state,omitempty"`
	ZipCode  string `json:"zip_code,omitempty" db:"zip_code"`
	City     string `json:"city,omitempty"`
	Address  string `json:"address,omitempty"`
	Virtual  *bool  `json:"virtual,omitempty"`
	Platform string `json:"platform,omitempty"`
	URL      string `json:"url,omitempty"`
}

// Validate ..
func (l Location) Validate() error {
	if l.Virtual == nil {
		return errors.New("virtual required")
	}

	if *l.Virtual {
		if l.Platform == "" {
			return errors.New("plarform required")
		}
		if l.URL == "" {
			return errors.New("url required")
		}
	} else {
		if l.Country == "" {
			return errors.New("country required")
		}
		if l.State == "" {
			return errors.New("state required")
		}
		if l.City == "" {
			return errors.New("city required")
		}
		if l.Address == "" {
			return errors.New("address required")
		}
	}
	return nil
}

// UpdateEvent is the struct used to update events.
//
// Use pointers to distinguish default values.
type UpdateEvent struct {
	Name       *string    `json:"name,omitempty"`
	Type       *eventType `json:"type,omitempty"`
	StartTime  *uint64    `json:"start_time,omitempty" db:"start_time"`
	EndTime    *uint64    `json:"end_time,omitempty" db:"end_time"`
	MinAge     *uint16    `json:"min_age,omitempty" db:"min_age"`
	Slots      *uint64    `json:"slots,omitempty"`
	TicketCost *uint64    `json:"ticket_cost,omitempty" db:"ticket_cost"`
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
	FollowersCount  *uint64    `json:"followers_count,omitempty"`
	FollowingCount  *uint64    `json:"following_count,omitempty"`
	CreatedAt       *time.Time `json:"created_at,omitempty" db:"created_at"`
	UpdatedAt       *time.Time `json:"updated_at,omitempty" db:"updated_at"`
}
