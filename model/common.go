package model

import (
	"errors"
	"time"
)

// ListUser represents a user in the context of an event.
//
// Use pointers to distinguish default values.
//
// Put here as it's a type used in all the services.
type ListUser struct {
	ID              string     `json:"id,omitempty"`
	Name            string     `json:"name,omitempty"`
	Username        string     `json:"username,omitempty"`
	Email           string     `json:"email,omitempty"`
	BirthDate       *time.Time `json:"birth_date,omitempty" db:"birth_date"`
	Description     string     `json:"description,omitempty"`
	Private         *bool      `json:"private,omitempty"`
	Type            UserType   `json:"type,omitempty"`
	VerifiedEmail   *bool      `json:"verified_email,omitempty" db:"verified_email"`
	ProfileImageURL string     `json:"profile_image_url,omitempty" db:"profile_image_url"`
	Invitations     uint8      `json:"invitations,omitempty"`
	FriendsCount    *uint64    `json:"friends_count,omitempty"`
	CreatedAt       *time.Time `json:"created_at,omitempty" db:"created_at"`
	UpdatedAt       *time.Time `json:"updated_at,omitempty" db:"updated_at"`
}

// UserID is commonly used to receive a user id in a body request.
type UserID struct {
	UserID string `json:"user_id,omitempty"`
}

// User type
const (
	Standard UserType = iota + 1
	Organization
)

var errInvalidUserType = errors.New("invalid type")

// UserType represents a user type.
type UserType uint8

// Validate verifies the correctness of the type.
func (u UserType) Validate() error {
	if u != Standard && u != Organization {
		return errInvalidUserType
	}
	return nil
}

// StringToUserType returns a UserType given a string.
func StringToUserType(s string) (UserType, error) {
	if s == "1" {
		return Standard, nil
	} else if s == "2" {
		return Organization, nil
	}
	return 0, errInvalidUserType
}
