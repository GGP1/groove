package model

import (
	"time"

	"github.com/pkg/errors"
)

const (
	// Personal user type
	Personal UserType = iota + 1
	// Business user type
	Business

	// Friends invitations setting
	Friends Invitations = iota + 1
	// Nobody invitations setting
	Nobody
)

var errInvalidUserType = errors.New("invalid type")

// ListUser represents a user in the context of an event.
//
// Use pointers to distinguish default values.
//
// Put here as it's a type used in all the services.
type ListUser struct {
	Private         *bool       `json:"private,omitempty"`
	FriendsCount    *uint64     `json:"friends_count,omitempty"`
	VerifiedEmail   *bool       `json:"verified_email,omitempty" db:"verified_email"`
	CreatedAt       *time.Time  `json:"created_at,omitempty" db:"created_at"`
	BirthDate       *time.Time  `json:"birth_date,omitempty" db:"birth_date"`
	UpdatedAt       *time.Time  `json:"updated_at,omitempty" db:"updated_at"`
	ID              string      `json:"id,omitempty"`
	Description     string      `json:"description,omitempty"`
	Email           string      `json:"email,omitempty"`
	Username        string      `json:"username,omitempty"`
	ProfileImageURL string      `json:"profile_image_url,omitempty" db:"profile_image_url"`
	Name            string      `json:"name,omitempty"`
	Invitations     Invitations `json:"invitations,omitempty"`
	Type            UserType    `json:"type,omitempty"`
}

// UserID is commonly used to receive a user id in a body request.
type UserID struct {
	UserID string `json:"user_id,omitempty"`
}

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
