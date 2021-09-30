package notification

import (
	"database/sql"
	"errors"
	"time"
)

// Notification type.
const (
	Invitation notifType = iota + 1
	FriendRequest
	Proposal
	Mention
)

type notifType uint8
type topic string

// Notification represents a notification inside the application.
type Notification struct {
	ID         string         `json:"id,omitempty"`
	SenderID   string         `json:"sender_id,omitempty" db:"sender_id"`
	ReceiverID string         `json:"receiver_id,omitempty" db:"receiver_id"`
	EventID    sql.NullString `json:"event_id" db:"event_id"`
	Type       notifType      `json:"type,omitempty"`
	Content    string         `json:"content,omitempty"` // May contain a custom message
	Seen       bool           `json:"seen,omitempty"`
	CreatedAt  time.Time      `json:"created_at,omitempty" db:"created_at"`
}

// CreateNotification is the struct used for the creation of notifications.
//
// The values are not validated as the creation is reserved for the developers and not the users.
type CreateNotification struct {
	SenderID   string    `json:"sender_id,omitempty" db:"sender_id"`
	ReceiverID string    `json:"receiver_id,omitempty" db:"receiver_id"`
	EventID    *string   `json:"event_id" db:"event_id"` // Only used in invitations
	Type       notifType `json:"type,omitempty"`
	Content    string    `json:"content,omitempty"`
}

// Validate ..
func (cn CreateNotification) Validate() error {
	if cn.Type == Invitation && cn.EventID == nil {
		return errors.New("event_id required")
	}
	return nil
}
