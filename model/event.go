package model

import (
	"time"

	"github.com/GGP1/groove/internal/validate"

	"github.com/pkg/errors"
)

// Event represents an event.
//
// Use pointers to distinguish default values.
type Event struct {
	HeaderURL   *string    `json:"header_url,omitempty" db:"header_url"`
	CreatedAt   *time.Time `json:"created_at,omitempty" db:"created_at"`
	Slots       *int64     `json:"slots,omitempty"`
	Public      *bool      `json:"public,omitempty"`
	Virtual     *bool      `json:"virtual,omitempty"`
	Location    *Location  `json:"location,omitempty"`
	MinAge      *uint16    `json:"min_age,omitempty" db:"min_age"`
	StartDate   *time.Time `json:"start_date,omitempty" db:"start_date"`
	EndDate     *time.Time `json:"end_date,omitempty" db:"end_date"`
	URL         *string    `json:"url,omitempty"`
	LogoURL     *string    `json:"logo_url,omitempty" db:"logo_url"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty" db:"updated_at"`
	Cron        string     `json:"cron,omitempty"`
	Description string     `json:"description,omitempty"`
	Name        string     `json:"name,omitempty"`
	ID          string     `json:"id,omitempty"`
	TicketType  ticketType `json:"ticket_type,omitempty" db:"ticket_type"`
	Type        eventType  `json:"type,omitempty"`
}

// EventStatistics contains statistics from an event.
type EventStatistics struct {
	Banned  int64 `json:"banned_count,omitempty" db:"banned_count"`
	Invited int64 `json:"invited_count,omitempty" db:"invited_count"`
	Likes   int64 `json:"likes_count,omitempty" db:"likes_count"`
	Members int64 `json:"members_count,omitempty" db:"members_count"`
}

// CreateEvent is the structure used to create an event.
type CreateEvent struct {
	StartDate   time.Time  `json:"start_date,omitempty" db:"start_date"`
	EndDate     time.Time  `json:"end_date,omitempty" db:"end_date"`
	LogoURL     *string    `json:"logo_url,omitempty" db:"logo_url"`
	URL         *string    `json:"url,omitempty"`
	Public      *bool      `json:"public,omitempty"`
	Virtual     *bool      `json:"virtual,omitempty"`
	HeaderURL   *string    `json:"header_url,omitempty" db:"header_url"`
	Location    *Location  `json:"location,omitempty"`
	HostID      string     `json:"host_id,omitempty"`
	Cron        string     `json:"cron,omitempty"`
	Description string     `json:"description,omitempty"`
	Name        string     `json:"name,omitempty"`
	Slots       int64      `json:"slots,omitempty"`
	MinAge      uint16     `json:"min_age,omitempty" db:"min_age"`
	TicketType  ticketType `json:"ticket_type,omitempty" db:"ticket_type"`
	Type        eventType  `json:"type,omitempty"`
}

// Validate verifies if the event received is valid.
func (c CreateEvent) Validate() error {
	if c.Name == "" {
		return errors.New("name required")
	}
	if c.Public == nil {
		return errors.New("public required")
	}
	if c.Virtual == nil {
		return errors.New("virtual required")
	}
	if c.StartDate.IsZero() {
		return errors.New("start_date required")
	}
	if c.EndDate.IsZero() {
		return errors.New("end_date required")
	}
	if len(c.Name) < 3 {
		return errors.New("name must contain at least 2 characters")
	} else if len(c.Name) > 60 {
		return errors.New("name maximum length is 60 characters")
	}
	if c.Type < Meeting && c.Type > Campsite {
		return errors.New("invalid type")
	}
	if c.Slots < -1 {
		return errors.New("slots must be equal to or higher than -1")
	}
	if c.Location != nil {
		if len(c.Location.Address) > 120 {
			return errors.New("maximum characters for an address is 120")
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
	if len(c.Name) > 60 {
		return errors.New("invalid name, maximum length is 60 characters")
	}
	if len(c.Description) > 200 {
		return errors.New("invalid description, maximum length is 200 characters")
	}
	if c.URL != nil {
		if err := validate.URL(*c.URL); err != nil {
			return errors.Wrap(err, "url")
		}
	}
	if c.LogoURL != nil {
		if err := validate.URL(*c.LogoURL); err != nil {
			return errors.Wrap(err, "logo_url")
		}
	}
	if c.HeaderURL != nil {
		if err := validate.URL(*c.LogoURL); err != nil {
			return errors.Wrap(err, "header_url")
		}
	}
	if c.TicketType < Free && c.TicketType > Donation {
		return errors.New("invalid ticket_type")
	}
	if c.MinAge < 0 || c.MinAge > 100 {
		return errors.New("min_age must be between 0 and 100")
	}
	if err := validate.Cron(c.Cron); err != nil {
		return errors.Wrap(err, "invalid cron")
	}
	if c.EndDate.Before(time.Now()) {
		return errors.New("invalid end_date")
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
	Cron        *string    `json:"cron,omitempty"`
	StartDate   *time.Time `json:"start_date,omitempty" db:"start_date"`
	EndDate     *time.Time `json:"end_date,omitempty" db:"end_date"`
	MinAge      *uint16    `json:"min_age,omitempty" db:"min_age"`
	Slots       *int64     `json:"slots,omitempty"`
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
		if err := validate.URL(*u.URL); err != nil {
			return errors.Wrap(err, "url")
		}
	}
	if u.LogoURL != nil {
		if err := validate.URL(*u.LogoURL); err != nil {
			return errors.Wrap(err, "logo_url")
		}
	}
	if u.HeaderURL != nil {
		if err := validate.URL(*u.LogoURL); err != nil {
			return errors.Wrap(err, "header_url")
		}
	}
	if u.Type != nil {
		if *u.Type == 0 {
			return errors.New("invalid type")
		}
	}
	if u.Cron != nil {
		if err := validate.Cron(*u.Cron); err != nil {
			return errors.Wrap(err, "invalid cron")
		}
	}
	if u.StartDate != nil {
		if u.StartDate.IsZero() {
			return errors.New("invalid start_date")
		}
	}
	if u.EndDate != nil {
		if u.EndDate.IsZero() || u.EndDate.Before(time.Now()) {
			return errors.New("invalid end_date")
		}
	}
	if u.MinAge != nil {
		if *u.MinAge == 0 {
			return errors.New("min_age must be higher than zero")
		}
	}
	if u.Slots != nil {
		if *u.Slots < -1 {
			return errors.New("slots must be equal to or higher than -1")
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

// Validate makes sure the coordinates are within the bounds.
func (c Coordinates) Validate() error {
	if c.Latitude < -90 || c.Latitude > 90 {
		return errors.Errorf("invalid latitude (%f), must be a value between -90 and 90", c.Latitude)
	}
	if c.Longitude < -180 || c.Longitude > 180 {
		return errors.Errorf("invalid longitude (%f), must be a value between -180 and 180", c.Latitude)
	}
	return nil
}

// LocationSearch is the structured used to perform location-based searches.
type LocationSearch struct {
	Latitude       float64 `json:"latitude,omitempty"`
	Longitude      float64 `json:"longitude,omitempty"`
	LatitudeDelta  float64 `json:"latitude_delta,omitempty"`
	LongitudeDelta float64 `json:"longitude_delta,omitempty"`
}

// Validate makes sure the values are correct.
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
