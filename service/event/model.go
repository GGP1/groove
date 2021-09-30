package event

import (
	"net/url"
	"time"

	"github.com/GGP1/groove/internal/validate"

	"github.com/pkg/errors"
)

// Event represents an event.
//
// Use pointers to distinguish default values.
type Event struct {
	ID          string     `json:"id,omitempty"`
	Name        string     `json:"name,omitempty"`
	Description string     `json:"description,omitempty"`
	Type        eventType  `json:"type,omitempty"`
	Public      *bool      `json:"public,omitempty"`
	Virtual     *bool      `json:"virtual,omitempty"`
	Location    *Location  `json:"location,omitempty"`
	URL         *string    `json:"url,omitempty"`
	LogoURL     *string    `json:"logo_url,omitempty" db:"logo_url"`
	HeaderURL   *string    `json:"header_url,omitempty" db:"header_url"`
	StartTime   *time.Time `json:"start_time,omitempty" db:"start_time"`
	EndTime     *time.Time `json:"end_time,omitempty" db:"end_time"`
	MinAge      *uint16    `json:"min_age,omitempty" db:"min_age"`
	Slots       *uint64    `json:"slots,omitempty"`
	TicketType  ticketType `json:"ticket_type,omitempty" db:"ticket_type"`
	CreatedAt   *time.Time `json:"created_at,omitempty" db:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty" db:"updated_at"`
}

// Statistics contains statistics from an event.
//
// TODO: consider moving into Event (Stats Statistics)
type Statistics struct {
	Banned  *uint64 `json:"banned_count,omitempty"`
	Members int64   `json:"members_count,omitempty" db:"members_count"`
	Invited *uint64 `json:"invited_count,omitempty"`
	Likes   *uint64 `json:"likes_count,omitempty"`
}

// CreateEvent is the structure used to create an event.
type CreateEvent struct {
	HostID      string     `json:"host_id,omitempty"`
	Name        string     `json:"name,omitempty"`
	Description string     `json:"description,omitempty"`
	Type        eventType  `json:"type,omitempty"`
	TicketType  ticketType `json:"ticket_type,omitempty" db:"ticket_type"`
	Public      *bool      `json:"public,omitempty"`
	Virtual     *bool      `json:"virtual,omitempty"`
	URL         *string    `json:"url,omitempty"`
	LogoURL     *string    `json:"logo_url,omitempty" db:"logo_url"`
	HeaderURL   *string    `json:"header_url,omitempty" db:"header_url"`
	Location    *Location  `json:"location,omitempty"`
	StartTime   time.Time  `json:"start_time,omitempty" db:"start_time"`
	EndTime     time.Time  `json:"end_time,omitempty" db:"end_time"`
	// If the event is completely free (no tickets), ask the user if he wants to specify a slots quantity,
	// else take it from the sum of the available tickets.
	Slots  uint64 `json:"slots,omitempty"`
	MinAge uint16 `json:"min_age,omitempty" db:"min_age"`
}

// Validate verifies if the event received is valid.
func (c CreateEvent) Validate() error {
	if c.Name == "" {
		return errors.New("name required")
	}
	if len(c.Name) < 3 {
		return errors.New("name must contain at least 2 characters")
	}
	if c.Type < Meeting && c.Type > Campsite {
		return errors.New("invalid type")
	}
	if c.Public == nil {
		return errors.New("public required")
	}
	if c.Virtual == nil {
		return errors.New("virtual required")
	}
	if c.Location != nil {
		if len(c.Location.Address) > 480 {
			return errors.New("maximum characters for an address is 480")
		}
		if c.Location.Coordinates.Latitude == 0 {
			return errors.New("invalid latitude")
		}
		if c.Location.Coordinates.Longitude == 0 {
			return errors.New("invalid longitude")
		}
	} else if !*c.Virtual {
		return errors.New("location required")
	}
	if c.URL != nil {
		if _, err := url.ParseRequestURI(*c.URL); err != nil {
			return errors.Wrap(err, "invalid url")
		}
	}
	if c.LogoURL != nil {
		if _, err := url.ParseRequestURI(*c.LogoURL); err != nil {
			return errors.Wrap(err, "invalid logo_url")
		}
	}
	if c.HeaderURL != nil {
		if _, err := url.ParseRequestURI(*c.HeaderURL); err != nil {
			return errors.Wrap(err, "invalid header_url")
		}
	}
	if c.TicketType < Free && c.TicketType > Donation {
		return errors.New("invalid ticket_type")
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
	if c.MinAge < 0 || c.MinAge > 100 {
		return errors.New("min_age must be between 0 and 100")
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
	URL         *string    `json:"url,omitempty"`
	LogoURL     *string    `json:"logo_url,omitempty" db:"logo_url"`
	HeaderURL   *string    `json:"header_url,omitempty" db:"header_url"`
	Location    *Location  `json:"location,omitempty"`
	StartTime   *time.Time `json:"start_time,omitempty" db:"start_time"`
	EndTime     *time.Time `json:"end_time,omitempty" db:"end_time"`
	MinAge      *uint16    `json:"min_age,omitempty" db:"min_age"`
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
	if u.URL != nil {
		if _, err := url.ParseRequestURI(*u.URL); err != nil {
			return errors.New("invalid url")
		}
	}
	if u.LogoURL != nil {
		if _, err := url.ParseRequestURI(*u.LogoURL); err != nil {
			return errors.New("invalid logo_url")
		}
	}
	if u.HeaderURL != nil {
		if _, err := url.ParseRequestURI(*u.HeaderURL); err != nil {
			return errors.New("invalid header_url")
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

// Location represents the location of an event.
type Location struct {
	// Address is a text-based description of a location, typically derived from the coordinates
	Address     string      `json:"address,omitempty"`
	Coordinates Coordinates `json:"coordinates,omitempty"`
}

// Coordinates represents a latitude/longitude pair.
type Coordinates struct {
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
}

// LocationSearch is the structured used to perform location-based searches.
type LocationSearch struct {
	Latitude       float64   `json:"latitude,omitempty"`
	Longitude      float64   `json:"longitude,omitempty"`
	LatitudeDelta  float64   `json:"latitude_delta,omitempty"`
	LongitudeDelta float64   `json:"longitude_delta,omitempty"`
	DiscardIDs     *[]string `json:"discard_ids,omitempty"`
}

// Validate makes sure the query is correct.
func (ls LocationSearch) Validate() error {
	if ls.Latitude < -90 || ls.Latitude > 90 {
		return errors.Errorf("invalid latitude (%f), must be a value between -90 and 90", ls.Latitude)
	}
	if ls.Longitude < -180 || ls.Longitude > 180 {
		return errors.Errorf("invalid longitude (%f), must be a value between -180 and 180", ls.Latitude)
	}
	if ls.LatitudeDelta > 1.2 {
		return errors.Errorf("invalid latitude_delta (%f), maximum value allowed is 1.2", ls.LatitudeDelta)
	}
	if ls.DiscardIDs != nil {
		if err := validate.ULIDs(*ls.DiscardIDs...); err != nil {
			return err
		}
	}
	return nil
}

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
	Seminar
	Attraction
	Gala
	Convention
	Campsite
)

// Event ticket type
const (
	Free ticketType = iota + 1
	Paid
	Mixed    // Includes free and paid ticket options
	Donation // Users decide how much to pay (if anything)
)

// eventType of an event.
type eventType uint8
type ticketType uint8
