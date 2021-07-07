package event

import (
	"strings"
	"time"

	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/permissions"

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
	ID             string     `json:"id,omitempty"`
	Name           string     `json:"name,omitempty"`
	Type           eventType  `json:"type,omitempty"`
	Public         *bool      `json:"public,omitempty"`
	Virtual        *bool      `json:"virtual,omitempty"`
	StartTime      int64      `json:"start_time,omitempty" db:"start_time"`
	EndTime        int64      `json:"end_time,omitempty" db:"end_time"`
	MinAge         uint16     `json:"min_age,omitempty" db:"min_age"`
	TicketCost     *uint64    `json:"ticket_cost,omitempty" db:"ticket_cost"`
	Slots          *uint64    `json:"slots,omitempty"`
	BannedCount    *uint64    `json:"banned_count,omitempty"`
	ConfirmedCount *uint64    `json:"confirmed_count,omitempty"`
	InvitedCount   *uint64    `json:"invited_count,omitempty"`
	LikesCount     *uint64    `json:"likes_count,omitempty"`
	Location       Location   `json:"location,omitempty"`
	Products       []Product  `json:"products,omitempty"`
	Reports        []Report   `json:"reports,omitempty"`
	Media          []Media    `json:"media,omitempty"`
	CreatedAt      *time.Time `json:"created_at,omitempty" db:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at,omitempty" db:"updated_at"`
}

// Node represents an event inside Dgraph.
type Node struct {
	Bans      []string `json:"bans,omitempty"`
	Confirmed []string `json:"confirmed,omitempty"`
	LikedBy   []string `json:"invited_by,omitempty"`
	Invited   []string `json:"invited,omitempty"`
}

// CreateEvent ..
type CreateEvent struct {
	CreatorID  string    `json:"creator_id,omitempty"`
	Name       string    `json:"name,omitempty"`
	Type       eventType `json:"type,omitempty"`
	Public     *bool     `json:"public,omitempty"`
	Virtual    *bool     `json:"virtual,omitempty"`
	StartTime  int64     `json:"start_time,omitempty" db:"start_time"`
	EndTime    int64     `json:"end_time,omitempty" db:"end_time"`
	MinAge     uint16    `json:"min_age,omitempty" db:"min_age"`
	Slots      uint64    `json:"slots,omitempty"`
	TicketCost uint64    `json:"ticket_cost,omitempty" db:"ticket_cost"`
	Location   Location  `json:"location,omitempty"`
}

// Validate verifies if the event received is valid.
func (c CreateEvent) Validate() error {
	if c.CreatorID == "" {
		return errors.New("creator_id required")
	}
	if err := params.ValidateUUID(c.CreatorID); err != nil {
		return err
	}
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

// UpdateEvent is the struct used to update events.
//
// Use pointers to distinguish default values.
type UpdateEvent struct {
	Name       *string    `json:"name,omitempty"`
	Type       *eventType `json:"type,omitempty"`
	StartTime  *int64     `json:"start_time,omitempty" db:"start_time"`
	EndTime    *int64     `json:"end_time,omitempty" db:"end_time"`
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

// Role represents a set of permissions inside the event.
type Role struct {
	Name           string              `json:"name,omitempty"`
	PermissionKeys map[string]struct{} `json:"permission_keys,omitempty"`
}

// Validate ..
func (r Role) Validate() error {
	if r.Name == "" {
		return errors.New("name required")
	}
	if len(r.Name) > 20 {
		return errors.New("invalid name length, maximum is 20")
	}
	if len(r.PermissionKeys) == 0 {
		return errors.New("permissions_keys required")
	}
	for pk := range r.PermissionKeys {
		if strings.Contains(pk, permissions.Separator) {
			return errors.Errorf("permission key [%q] cannot contain character %q", pk, permissions.Separator)
		}
	}
	return nil
}

// Permission represents a privilege inside an event.
type Permission struct {
	Name        string `json:"name,omitempty"`
	Key         string `json:"key,omitempty"`
	Description string `json:"description,omitempty"`
}

// Validate ..
func (p Permission) Validate() error {
	if p.Key == "" {
		return errors.New("key required")
	}
	if p.Key == permissions.All {
		return errors.New("invalid key")
	}
	if len(p.Key) > 20 {
		return errors.New("invalid key length, maximum is 20")
	}
	if strings.Contains(p.Key, permissions.Separator) {
		return errors.Errorf("permission key cannot contain character %q", permissions.Separator)
	}
	if p.Name == "" {
		return errors.New("name required")
	}
	if len(p.Name) > 20 {
		return errors.New("invalid name length, maximum is 20")
	}
	if len(p.Description) > 20 {
		return errors.New("invalid description length, maximum is 50")
	}
	return nil
}

// Location represents the place where the event will take place, it could be on-site or virtual.
type Location struct {
	Country   string `json:"country,omitempty"`
	State     string `json:"state,omitempty"`
	ZipCode   string `json:"zip_code,omitempty" db:"zip_code"`
	City      string `json:"city,omitempty"`
	Address   string `json:"address,omitempty"`
	Virtual   *bool  `json:"virtual,omitempty"`
	Platform  string `json:"platform,omitempty"`   // Field for virtual events
	InviteURL string `json:"invite_url,omitempty"` // Field for virtual events
}

// Validate ..
func (l Location) Validate() error {
	if l.Virtual == nil {
		return errors.New("virtual required")
	}

	if !*l.Virtual {
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

// Media reprensents images, videos and audio.
type Media struct {
	ID      string `json:"id,omitempty"`
	EventID string `json:"event_id,omitempty" db:"event_id"`
	URL     string `json:"url,omitempty"`
}

// Validate ..
func (m Media) Validate() error {
	if err := params.ValidateUUID(m.EventID); err != nil {
		return errors.Wrap(err, "invalid event_id")
	}
	if m.URL == "" {
		return errors.New("url required")
	}
	return nil
}

// Product represents a market commodity.
//
// Amounts to be provided in a currency’s smallest unit.
type Product struct {
	ID          string `json:"id,omitempty"`
	EventID     string `json:"event_id" db:"event_id"`
	Stock       uint   `json:"stock"`
	Brand       string `json:"brand"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Discount    int64  `json:"discount"`
	Taxes       int64  `json:"taxes"`
	Subtotal    int64  `json:"subtotal"`
	Total       int64  `json:"total"`
}

// Validate ..
func (p Product) Validate() error {
	if err := params.ValidateUUID(p.EventID); err != nil {
		return errors.Wrap(err, "invalid event_id")
	}
	if p.Discount < 0 {
		return errors.New("invalid discount, minimum is 0")
	}
	if p.Taxes < 0 {
		return errors.New("invalid taxes, minimum is 0")
	}
	if p.Total < 0 {
		return errors.New("invalid total, minimum is 0")
	}
	return nil
}

// Report represents a report made by a user on an event/user
type Report struct {
	EventID string `json:"event_id,omitempty" db:"event_id"` // TODO: make the report for other users as well?
	UserID  string `json:"user_id,omitempty" db:"user_id"`
	Type    string `json:"type,omitempty"`
	Details string `json:"details,omitempty"`
}
