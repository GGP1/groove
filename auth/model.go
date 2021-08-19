package auth

import (
	"errors"

	"github.com/oklog/ulid/v2"
)

// userSession is used when logging a user in.
type userSession struct {
	ID            ulid.ULID `json:"id"`
	Email         string    `json:"email"`
	Username      string    `json:"username"`
	Password      string    `json:"-"`
	Premium       bool      `json:"premium"`
	Private       bool      `json:"private"`
	VerifiedEmail bool      `json:"verified_email" db:"verified_email"`
}

// userLogin is used to decode the input received on a login attempt.
type userLogin struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

func (u userLogin) Valid() error {
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
