package user

import (
	"net/url"
	"time"

	"github.com/GGP1/groove/internal/validate"
	"github.com/GGP1/groove/model"

	"github.com/pkg/errors"
)

// User represents a user inside the system.
type User struct {
	ID              string         `json:"id,omitempty"`
	Name            string         `json:"name,omitempty"`
	Username        string         `json:"username,omitempty"`
	Email           string         `json:"email,omitempty"`
	BirthDate       *time.Time     `json:"birth_date,omitempty" db:"birth_date"`
	Description     string         `json:"description,omitempty"`
	ProfileImageURL string         `json:"profile_image_url,omitempty" db:"profile_image_url"`
	Private         *bool          `json:"private,omitempty"`
	Type            model.UserType `json:"type,omitempty"`
	VerifiedEmail   *bool          `json:"verified_email,omitempty" db:"verified_email"`
	IsAdmin         *bool          `json:"is_admin,omitempty" db:"is_admin"`
	Invitations     invitations    `json:"invitations,omitempty"`
	CreatedAt       *time.Time     `json:"created_at,omitempty" db:"created_at"`
	UpdatedAt       *time.Time     `json:"updated_at,omitempty" db:"updated_at"`
}

// CreateUser is the structure used for the creation of a user.
type CreateUser struct {
	Name            string          `json:"name,omitempty"`
	Username        string          `json:"username,omitempty"`
	Email           string          `json:"email,omitempty"`
	Password        string          `json:"password,omitempty"`
	BirthDate       *time.Time      `json:"birth_date,omitempty"`
	Type            *model.UserType `json:"type,omitempty"`
	Description     string          `json:"description,omitempty"`
	ProfileImageURL *string         `json:"profile_image_url,omitempty"`
	Invitations     invitations     `json:"invitations,omitempty"`
}

// Validate verifies the user passed is valid.
func (c CreateUser) Validate() error {
	if c.Name == "" {
		return errors.New("name required")
	}
	if c.Username == "" {
		return errors.New("username required")
	}
	if err := validate.Username(c.Username); err != nil {
		return err
	}
	if c.Email == "" {
		return errors.New("email required")
	}
	if err := validate.Email(c.Email); err != nil {
		return err
	}
	if c.Password == "" {
		return errors.New("password required")
	}
	if err := validate.Password(c.Password); err != nil {
		return err
	}
	if c.Type == nil {
		return errors.New("type required")
	}
	if err := c.Type.Validate(); err != nil {
		return err
	}
	if *c.Type == model.Standard {
		if c.BirthDate == nil {
			return errors.New("birth_date required")
		}
	}
	if len(c.Description) > 145 {
		return errors.New("invalid description length, maximum is 144 characters")
	}
	if c.ProfileImageURL != nil {
		if _, err := url.ParseRequestURI(*c.ProfileImageURL); err != nil {
			return errors.Wrap(err, "invalid profile_image_url")
		}
	}
	return nil
}

// Statistics contains statistics from a user.
type Statistics struct {
	Blocked         *uint64 `json:"blocked_count,omitempty"`
	BlockedBy       *uint64 `json:"blocked_by_count,omitempty"`
	Friends         *uint64 `json:"friends_count,omitempty"`
	Followers       *uint64 `json:"followers_count,omitempty"`
	AttendingEvents int64   `json:"attending_events_count,omitempty"`
	HostedEvents    *uint64 `json:"hosted_events_count,omitempty"`
	InvitedEvents   *uint64 `json:"invited_events_count,omitempty"`
}

// UpdateUser is the struct used to update users.
//
// Pointers are used to distinguish default values.
type UpdateUser struct {
	Name        *string      `json:"name,omitempty"`
	Username    *string      `json:"username,omitempty"`
	Private     *bool        `json:"private,omitempty"`
	Invitations *invitations `json:"invitations,omitempty"`
}

// Validate ..
func (u UpdateUser) Validate() error {
	if u == (UpdateUser{}) {
		return errors.New("no values provided")
	}
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

// Invitations settings
const (
	Friends invitations = iota + 1
	Nobody
)

type invitations uint8

func (i invitations) Validate() error {
	switch i {
	case Friends, Nobody:
		return nil
	}

	return errors.Errorf("invalid invitations value: %d", i)
}

// Invite represents an invitation.
type Invite struct {
	EventID string `json:"event_id,omitempty"`
	UserID  string `json:"user_id,omitempty"`
}

// Validate checks the values received are valid.
func (i Invite) Validate() error {
	if i.EventID == "" {
		return errors.New("event_id required")
	}
	if i.UserID == "" {
		return errors.New("user_id required")
	}
	if err := validate.ULID(i.EventID); err != nil {
		return errors.Wrap(err, "invalid event_id")
	}
	if err := validate.ULID(i.UserID); err != nil {
		return errors.Wrap(err, "invalid user_id")
	}
	return nil
}
