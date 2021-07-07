package auth

import (
	"errors"

	"github.com/GGP1/groove/internal/email"

	"github.com/google/uuid"
)

// userSession is used when logging a user in.
type userSession struct {
	ID            uuid.UUID `json:"id,omitempty"`
	Email         string    `json:"email,omitempty"`
	Password      string    `json:"password,omitempty"`
	Premium       bool      `json:"premium,omitempty"`
	VerifiedEmail bool      `json:"verified_email,omitempty" db:"verified_email"`
}

// userLogin is used to decode the input received on a login attempt.
type userLogin struct {
	Email    string `json:"email,omitempty"`
	Password string `json:"password,omitempty"`
}

func (u userLogin) Valid() error {
	if u.Email == "" {
		return errors.New("email required")
	}
	if len(u.Email) < 3 || len(u.Email) > 254 || !email.IsValid(u.Email) {
		return errors.New("invalid email")
	}
	if u.Password == "" {
		return errors.New("password required")
	}
	if len(u.Password) < 8 {
		return errors.New("password length must be atleast 8 characters long")
	}

	return nil
}
