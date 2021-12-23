package model

import (
	"time"

	"github.com/GGP1/groove/internal/validate"

	"github.com/pkg/errors"
)

var errInvalidUserType = errors.New("invalid user type")

// User represents a user inside the system.
type User struct {
	Private         *bool       `json:"private,omitempty"`
	CreatedAt       *time.Time  `json:"created_at,omitempty" db:"created_at"`
	IsAdmin         *bool       `json:"is_admin,omitempty" db:"is_admin"`
	VerifiedEmail   *bool       `json:"verified_email,omitempty" db:"verified_email"`
	BirthDate       *time.Time  `json:"birth_date,omitempty" db:"birth_date"`
	UpdatedAt       *time.Time  `json:"updated_at,omitempty" db:"updated_at"`
	ProfileImageURL *string     `json:"profile_image_url,omitempty" db:"profile_image_url"`
	Description     string      `json:"description,omitempty"`
	Email           string      `json:"email,omitempty"`
	Username        string      `json:"username,omitempty"`
	Name            string      `json:"name,omitempty"`
	ID              string      `json:"id,omitempty"`
	Type            UserType    `json:"type,omitempty"`
	Invitations     Invitations `json:"invitations,omitempty"`
}

// CreateUser is the structure used for the creation of a user.
type CreateUser struct {
	ProfileImageURL *string     `json:"profile_image_url,omitempty"`
	BirthDate       *time.Time  `json:"birth_date,omitempty"`
	Type            *UserType   `json:"type,omitempty"`
	Name            string      `json:"name,omitempty"`
	Username        string      `json:"username,omitempty"`
	Password        string      `json:"password,omitempty"`
	Description     string      `json:"description,omitempty"`
	Email           string      `json:"email,omitempty"`
	Invitations     Invitations `json:"invitations,omitempty"`
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
	if *c.Type == Personal {
		if c.BirthDate == nil {
			return errors.New("birth_date required")
		}
	}
	if len(c.Name) > 40 {
		return errors.New("invalid name, maximum length is 40 characters")
	}
	if len(c.Email) > 120 {
		return errors.New("invalid email, maximum length is 120 characters")
	}
	if len(c.Description) > 200 {
		return errors.New("invalid description length, maximum is 200 characters")
	}
	if c.ProfileImageURL != nil {
		if err := validate.URL(*c.ProfileImageURL); err != nil {
			return errors.Wrap(err, "profile_image_url")
		}
	}
	return nil
}

// UserStatistics contains statistics from a user.
type UserStatistics struct {
	Banned          *int64 `json:"banned_events_count,omitempty" db:"banned_events_count"`
	Blocked         *int64 `json:"blocked_count,omitempty" db:"blocked_count"`
	BlockedBy       *int64 `json:"blocked_by_count,omitempty" db:"blocked_by_count"`
	Friends         *int64 `json:"friends_count,omitempty" db:"friends_count"`
	Following       *int64 `json:"following_count,omitempty" db:"following_count"`
	Followers       *int64 `json:"followers_count,omitempty" db:"followers_count"`
	Invitations     *int64 `json:"invitations_count,omitempty" db:"invitations_count"`
	LikedEvents     *int64 `json:"liked_events_count,omitempty" db:"liked_events_count"`
	AttendingEvents *int64 `json:"attending_events_count,omitempty" db:"attending_events_count"`
	HostedEvents    *int64 `json:"hosted_events_count,omitempty" db:"hosted_events_count"`
}

// UpdateUser is the struct used to update users.
//
// Pointers are used to distinguish default values.
type UpdateUser struct {
	Name            *string      `json:"name,omitempty"`
	Username        *string      `json:"username,omitempty"`
	ProfileImageURL *string      `json:"profile_image_url,omitempty" db:"profile_image_url"`
	Private         *bool        `json:"private,omitempty"`
	Invitations     *Invitations `json:"invitations,omitempty"`
}

// Validate ..
func (u UpdateUser) Validate() error {
	if u == (UpdateUser{}) {
		return errors.New("no values provided")
	}
	if u.Name != nil {
		if len(*u.Name) > 40 {
			return errors.New("invalid name, maximum length is 40 characters")
		}
	}
	if u.Username != nil {
		if len(*u.Username) > 24 {
			return errors.New("invalid username length, must be lower than 24 characters")
		}
	}
	if u.ProfileImageURL != nil {
		if err := validate.URL(*u.ProfileImageURL); err != nil {
			return errors.Wrap(err, "profile_image_url")
		}
	}
	if u.Invitations != nil {
		if err := u.Invitations.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// Invite represents an invitation.
type Invite struct {
	EventID string   `json:"event_id,omitempty"`
	UserIDs []string `json:"user_ids,omitempty"`
}

// Validate checks the values received are valid.
func (i Invite) Validate() error {
	if i.EventID == "" {
		return errors.New("event_id required")
	}
	if len(i.UserIDs) == 0 {
		return errors.New("user_ids required")
	}
	if err := validate.ULID(i.EventID); err != nil {
		return errors.Wrap(err, "event_id")
	}
	if err := validate.ULIDs(i.UserIDs...); err != nil {
		return errors.Wrap(err, "user_ids")
	}
	return nil
}

const (
	// Personal user type
	Personal UserType = iota + 1
	// Business user type
	Business
)

// UserType represents a user type.
type UserType uint8

// Validate verifies the correctness of the type.
func (u UserType) Validate() error {
	if u != Personal && u != Business {
		return errInvalidUserType
	}
	return nil
}

// StringToUserType returns a UserType given a string.
func StringToUserType(s string) (UserType, error) {
	if s == "1" {
		return Personal, nil
	} else if s == "2" {
		return Business, nil
	}
	return 0, errInvalidUserType
}

const (
	// Friends invitations setting
	Friends Invitations = iota + 1
	// Nobody invitations setting
	Nobody
)

// Invitations represents a user's invitations settings
type Invitations uint8

// Validate verifies the invitations number is correct.
func (i Invitations) Validate() error {
	switch i {
	case Friends, Nobody:
		return nil
	}

	return errors.Errorf("invalid invitations value: %d", i)
}
