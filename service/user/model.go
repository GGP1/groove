package user

import (
	"time"

	"github.com/GGP1/groove/internal/email"

	"github.com/pkg/errors"
)

// User ..
type User struct {
	ID              string    `json:"id,omitempty"`
	Name            string    `json:"name,omitempty"`
	Username        string    `json:"username,omitempty"`
	Email           string    `json:"email,omitempty"`
	Password        string    `json:"password,omitempty"`
	BirthDate       time.Time `json:"birth_date,omitempty" db:"birth_date"`
	Description     string    `json:"description,omitempty"`
	ProfileImageURL string    `json:"profile_image_url,omitempty" db:"profile_image_url"`
	Premium         *bool     `json:"premium,omitempty"`
	Private         *bool     `json:"private,omitempty"`
	VerifiedEmail   *bool     `json:"verified_email,omitempty"`
	IsAdmin         *bool     `json:"is_admin,omitempty" db:"is_admin"`
	Invitations     string    `json:"invitations,omitempty"`
	Location        Location  `json:"location,omitempty"`
	FriendsCount    uint64    `json:"friends_count,omitempty"`
	Reports         []Report  `json:"reports,omitempty"`
	Payment         Payment   `json:"payment,omitempty"`
	CreatedAt       time.Time `json:"created_at,omitempty" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at,omitempty" db:"updated_at"`
}

// CreateUser ..
type CreateUser struct {
	Name            string     `json:"name,omitempty"`
	Username        string     `json:"username,omitempty"`
	Email           string     `json:"email,omitempty"`
	Password        string     `json:"password,omitempty"`
	BirthDate       *time.Time `json:"birth_date,omitempty"`
	Location        Location   `json:"location,omitempty"`
	Description     string     `json:"description,omitempty"`
	ProfileImageURL string     `json:"profile_image_url,omitempty"`
}

// Validate verifies the user passed is valid.
func (c CreateUser) Validate() error {
	if c.Name == "" {
		return errors.New("name required")
	}
	if c.Username == "" {
		return errors.New("username required")
	}
	if len(c.Username) > 24 {
		return errors.New("invalid username length, must be lower than 24 characters")
	}
	if c.Email == "" {
		return errors.New("email required")
	}
	if len(c.Email) < 3 || len(c.Email) > 254 || !email.IsValid(c.Email) {
		return errors.New("invalid email")
	}
	if c.Password == "" {
		return errors.New("password required")
	}
	if c.BirthDate == nil {
		return errors.New("birth_date required")
	}
	if len(c.Description) > 144 {
		return errors.New("invalid description length, must be lower than 144 characters")
	}
	return c.Location.Validate()
}

// ListUser contains information about the user to be provided in profiles.
//
// Use pointers to distinguish default values.
type ListUser struct {
	ID                   string     `json:"id,omitempty"`
	Name                 string     `json:"name,omitempty"`
	Username             string     `json:"username,omitempty"`
	Email                string     `json:"email,omitempty"`
	BirthDate            *time.Time `json:"birth_date,omitempty" db:"birth_date"`
	Description          string     `json:"description,omitempty"`
	Premium              *bool      `json:"premium,omitempty"`
	Private              *bool      `json:"private,omitempty"`
	VerifiedEmail        *bool      `json:"verified_email,omitempty" db:"verified_email"`
	ProfileImageURL      string     `json:"profile_image_url,omitempty" db:"profile_image_url"`
	Invitations          string     `json:"invitations,omitempty"`
	Location             *Location  `json:"location,omitempty"`
	BlockedCount         *uint64    `json:"blocked_count,omitempty"`
	BlockedByCount       *uint64    `json:"blocked_by_count,omitempty"`
	ConfirmedEventsCount *uint64    `json:"confirmed_events_count,omitempty"`
	FriendsCount         *uint64    `json:"friends_count,omitempty"`
	HostedEventsCount    *uint64    `json:"hosting_events_count,omitempty"`
	InvitedEventsCount   *uint64    `json:"invited_events_count,omitempty"`
	CreatedAt            *time.Time `json:"created_at,omitempty" db:"created_at"`
	UpdatedAt            *time.Time `json:"updated_at,omitempty" db:"updated_at"`
}

// UpdateUser is the struct used to update users.
//
// Pointers are used to distinguish default values.
type UpdateUser struct {
	Name        *string      `json:"name,omitempty"`
	Username    *string      `json:"username,omitempty"`
	Private     *bool        `json:"private,omitempty"`
	Invitations *invitations `json:"invitations,omitempty"`
	UpdatedAt   *time.Time   `json:"updated_at,omitempty"`
}

// Validate ..
func (u UpdateUser) Validate() error {
	if u.Username != nil {
		if len(*u.Username) > 24 {
			return errors.New("invalid username length, must be lower than 24 characters")
		}
	}
	if u.Invitations != nil {
		if err := u.Invitations.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// Location represents a location for a user.
type Location struct {
	Country string `json:"country,omitempty"`
	State   string `json:"state,omitempty"`
	City    string `json:"city,omitempty"`
}

// Validate ..
func (l Location) Validate() error {
	// For now it won't be required to provide a location
	return nil
}

// Payment contains the financial providers of the users to make transactions.
type Payment struct {
	Provider string `json:"provider,omitempty"`
}

// Report represents a report on a user.
type Report struct {
	ReportedUsername string `json:"reported_username,omitempty"`
	Comment          string `json:"comment,omitempty"`
}

// Invitations settings
const (
	_ invitations = iota
	Friends
	Nobody
)

type invitations uint8

func (i invitations) String() string {
	switch i {
	case Friends:
		return "friends"
	case Nobody:
		return "nobody"
	default:
		return ""
	}
}

func (i invitations) Validate() error {
	switch i {
	case Friends, Nobody:
		return nil
	}

	return errors.Errorf("invalid invitations value: %d", i)
}
