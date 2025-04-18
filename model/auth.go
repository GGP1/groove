package model

import (
	"github.com/oklog/ulid/v2"
	"github.com/pkg/errors"
)

// UserSession is used when logging a user in.
type UserSession struct {
	ProfileImageURL *string   `json:"profile_image_url,omitempty" db:"profile_image_url"`
	Email           string    `json:"email"`
	Username        string    `json:"username"`
	Password        string    `json:"-"`
	ID              ulid.ULID `json:"id"`
	VerifiedEmail   bool      `json:"verified_email" db:"verified_email"`
	Type            UserType  `json:"type"`
}

// Login is used to decode the input received on a Login attempt.
type Login struct {
	Username    string `json:"username,omitempty"`
	Password    string `json:"password,omitempty"`
	DeviceToken string `json:"device_token,omitempty"`
}

// Validate verifies the values received are correct.
func (u Login) Validate() error {
	if u.Username == "" {
		return errors.New("username required")
	}
	if len(u.Username) < 3 || len(u.Username) > 254 {
		return errors.New("invalid username")
	}
	if u.Password == "" {
		return errors.New("password required")
	}
	if len(u.Password) < 8 {
		return errors.New("password length must be atleast 8 characters long")
	}

	return nil
}
