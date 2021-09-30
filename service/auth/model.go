package auth

import (
	"errors"

	"github.com/GGP1/groove/model"

	"github.com/oklog/ulid/v2"
)

// userSession is used when logging a user in.
type userSession struct {
	ID              ulid.ULID      `json:"id"`
	Email           string         `json:"email"`
	Username        string         `json:"username"`
	Password        string         `json:"-"`
	VerifiedEmail   bool           `json:"verified_email" db:"verified_email"`
	ProfileImageURL string         `json:"profile_image_url,omitempty" db:"profile_image_url"`
	Type            model.UserType `json:"type,omitempty"`
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
