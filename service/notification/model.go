package notification

import (
	"database/sql"
	"errors"
	"time"

	"github.com/GGP1/groove/internal/validate"
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
	CreatedAt  time.Time      `json:"created_at,omitempty" db:"created_at"`
	ID         string         `json:"id,omitempty"`
	ReceiverID string         `json:"receiver_id,omitempty" db:"receiver_id"`
	Content    string         `json:"content,omitempty"`
	SenderID   string         `json:"sender_id,omitempty" db:"sender_id"`
	EventID    sql.NullString `json:"event_id" db:"event_id"`
	Type       notifType      `json:"type,omitempty"`
	Seen       bool           `json:"seen,omitempty"`
}

// CreateNotification is the struct used for the creation of notifications.
//
// The values are not validated as the creation is reserved for the developers and not the users.
type CreateNotification struct {
	EventID    *string   `json:"event_id" db:"event_id"`
	SenderID   string    `json:"sender_id,omitempty" db:"sender_id"`
	ReceiverID string    `json:"receiver_id,omitempty" db:"receiver_id"`
	Content    string    `json:"content,omitempty"`
	Type       notifType `json:"type,omitempty"`
}

// Validate ..
func (cn CreateNotification) Validate() error {
	if cn.SenderID == cn.ReceiverID {
		return errors.New("cannot perform this action on your account")
	}
	if err := validate.ULIDs(cn.SenderID, cn.ReceiverID); err != nil {
		return err
	}
	if cn.Type == Invitation && cn.EventID == nil {
		return errors.New("event_id required")
	}
	if len(cn.Content) > 240 {
		return errors.New("invalid content, maximum length is 240 characters")
	}
	return nil
}

// CreateNotificationMany is the struct used for the creation of notifications for many users.
//
// The values are not validated as the creation is reserved for the developers and not the users.
type CreateNotificationMany struct {
	EventID     *string   `json:"event_id" db:"event_id"`
	SenderID    string    `json:"sender_id,omitempty" db:"sender_id"`
	Content     string    `json:"content,omitempty"`
	ReceiverIDs []string  `json:"receiver_ids,omitempty" db:"receiver_id"`
	Type        notifType `json:"type,omitempty"`
}
