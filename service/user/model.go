package user

import (
	"time"

	"github.com/GGP1/groove/internal/validate"
	"github.com/GGP1/groove/model"

	"github.com/pkg/errors"
)

// User represents a user inside the system.
type User struct {
	Private         *bool             `json:"private,omitempty"`
	CreatedAt       *time.Time        `json:"created_at,omitempty" db:"created_at"`
	IsAdmin         *bool             `json:"is_admin,omitempty" db:"is_admin"`
	VerifiedEmail   *bool             `json:"verified_email,omitempty" db:"verified_email"`
	BirthDate       *time.Time        `json:"birth_date,omitempty" db:"birth_date"`
	UpdatedAt       *time.Time        `json:"updated_at,omitempty" db:"updated_at"`
	ProfileImageURL string            `json:"profile_image_url,omitempty" db:"profile_image_url"`
	Description     string            `json:"description,omitempty"`
	Email           string            `json:"email,omitempty"`
	Username        string            `json:"username,omitempty"`
	Name            string            `json:"name,omitempty"`
	ID              string            `json:"id,omitempty"`
	Type            model.UserType    `json:"type,omitempty"`
	Invitations     model.Invitations `json:"invitations,omitempty"`
}

// CreateUser is the structure used for the creation of a user.
type CreateUser struct {
	ProfileImageURL *string           `json:"profile_image_url,omitempty"`
	BirthDate       *time.Time        `json:"birth_date,omitempty"`
	Type            *model.UserType   `json:"type,omitempty"`
	Name            string            `json:"name,omitempty"`
	Username        string            `json:"username,omitempty"`
	Password        string            `json:"password,omitempty"`
	Description     string            `json:"description,omitempty"`
	Email           string            `json:"email,omitempty"`
	Invitations     model.Invitations `json:"invitations,omitempty"`
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
	if *c.Type == model.Personal {
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

// Statistics contains statistics from a user.
type Statistics struct {
	Blocked         *uint64 `json:"blocked_count,omitempty"`
	BlockedBy       *uint64 `json:"blocked_by_count,omitempty"`
	Friends         *uint64 `json:"friends_count,omitempty"`
	Following       *uint64 `json:"following_count,omitempty"`
	Followers       *uint64 `json:"followers_count,omitempty"`
	Invitations     *uint64 `json:"invitations_count,omitempty"`
	LikedEvents     *uint64 `json:"liked_events_count,omitempty"`
	AttendingEvents *int64  `json:"attending_events_count,omitempty"`
	HostedEvents    *int64  `json:"hosted_events_count,omitempty"`
}

// UpdateUser is the struct used to update users.
//
// Pointers are used to distinguish default values.
type UpdateUser struct {
	Name            *string            `json:"name,omitempty"`
	Username        *string            `json:"username,omitempty"`
	ProfileImageURL *string            `json:"profile_image_url,omitempty" db:"profile_image_url"`
	Private         *bool              `json:"private,omitempty"`
	Invitations     *model.Invitations `json:"invitations,omitempty"`
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
