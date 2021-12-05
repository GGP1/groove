package notification

import (
	"context"
	"database/sql"
	"net"
	"time"

	"github.com/GGP1/groove/config"
	"github.com/GGP1/groove/internal/httperr"
	"github.com/GGP1/groove/internal/log"
	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/roles"
	"github.com/GGP1/groove/internal/txgroup"
	"github.com/GGP1/groove/internal/ulid"
	"github.com/GGP1/groove/model"
	"github.com/GGP1/groove/service/auth"
	"github.com/GGP1/groove/service/event/role"
	"github.com/GGP1/groove/storage/postgres"
	"github.com/GGP1/sqan"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/dgraph-io/dgo/v210"
	"github.com/dgraph-io/dgo/v210/protos/api"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/api/option"
)

// NOTE: I'll implement this service using postgres to simplify things but
// it's probably not the best database for the job as the notifications
// will be read few times (and deleted once they have been accepted/declined).
// Here a LSM tree would do better than a B/B+ one. But it all depends if the users
// take actions on notifications or not (if not they will be retrieved multiple times).

// Service represents the notification service.
type Service interface {
	Answer(ctx context.Context, id, authUserID string, accepted bool) error
	Create(ctx context.Context, session auth.Session, notification CreateNotification) error
	CreateMany(ctx context.Context, session auth.Session, notification CreateNotificationMany) error
	Delete(ctx context.Context, notificationID string) error
	DeleteInvitation(ctx context.Context, eventID, senderID, receiverID string) error
	GetFromUser(ctx context.Context, userID string, params params.Query) ([]Notification, error)
	GetFromUserCount(ctx context.Context, userID string) (int, error)
	Send(ctx context.Context, message *messaging.Message)
	SendMulticast(ctx context.Context, notification *messaging.MulticastMessage)
	SendMany(ctx context.Context, messages []*messaging.Message)
	SuscribeToTopic(ctx context.Context, session auth.Session, topic topic) error
	UnsuscribeFromTopic(ctx context.Context, session auth.Session, topic topic) error
}

type service struct {
	dc  *dgo.Dgraph
	db  *sql.DB
	fcm *messaging.Client

	roleService role.Service
	authService auth.Service

	metrics    metrics
	maxRetries int
}

// NewService returns a new notification service.
func NewService(
	db *sql.DB,
	dc *dgo.Dgraph,
	config config.Notifications,
	authService auth.Service,
	roleService role.Service,
) Service {
	ctx := context.Background()

	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile(config.CredentialsFile))
	if err != nil {
		log.Fatal("failed creating firebase app", zap.Error(err))
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		log.Fatal("failed creating FCM client", zap.Error(err))
	}

	return service{
		db:          db,
		dc:          dc,
		fcm:         client,
		metrics:     initMetrics(),
		authService: authService,
		roleService: roleService,
		maxRetries:  config.MaxRetries,
	}
}

// Answer handles the accept or decline of a notification.
func (s service) Answer(ctx context.Context, id, authUserID string, accepted bool) error {
	s.metrics.incMethodCalls("Answer")
	sqlTx := txgroup.SQLTx(ctx)

	q := "SELECT sender_id, receiver_id, event_id, type FROM notifications WHERE id=$1"
	rows, err := sqlTx.QueryContext(ctx, q, id)
	if err != nil {
		return errors.Wrap(err, "querying notification")
	}

	var notification Notification
	if err := sqan.Row(&notification, rows); err != nil {
		return errors.Wrap(err, "scanning notification")
	}

	if notification.ReceiverID != authUserID {
		return httperr.Forbidden("access denied")
	}

	if !accepted {
		if err := s.Delete(ctx, id); err != nil {
			return err
		}
		if notification.Type == Invitation {
			return s.roleService.UnsetRole(ctx, *notification.EventID, notification.ReceiverID)
		}
		return nil
	}

	switch notification.Type {
	case Invitation: // Add receiver as an attendant of the event
		return s.roleService.SetReservedRole(ctx, *notification.EventID, notification.ReceiverID, roles.Attendant)
	case FriendRequest: // Execute friendship (add friend edges)
		vars := map[string]string{"$user_id": notification.SenderID, "$friend_id": notification.ReceiverID}
		query := `query q($user_id: string, $friend_id: string) {
		user as var(func: eq(user_id, $user_id))
		friend as var(func: eq(user_id, $friend_id))
	}`
		mu := &api.Mutation{
			Cond: "@if(eq(len(user), 1) AND eq(len(friend), 1))",
			SetNquads: []byte(`uid(user) <friend> uid(friend) .
		uid(friend) <friend> uid(user) .`),
		}
		req := &api.Request{
			Query:     query,
			Vars:      vars,
			Mutations: []*api.Mutation{mu},
			CommitNow: true,
		}
		if _, err := s.dc.NewTxn().Do(ctx, req); err != nil {
			return errors.Wrap(err, "creating friend edges")
		}

		return nil
	case Proposal: // TODO: Handle
	}

	return nil
}

// Create creates a new notification.
func (s service) Create(ctx context.Context, session auth.Session, notification CreateNotification) error {
	s.metrics.incMethodCalls("Create")

	if err := notification.Validate(); err != nil {
		return httperr.Forbidden(err.Error())
	}

	q := `INSERT INTO notifications (id, sender_id, receiver_id, event_id, content, type) VALUES ($1, $2, $3, $4, $5, $6)`
	sqlTx := txgroup.SQLTx(ctx)
	_, err := sqlTx.ExecContext(ctx, q, ulid.NewString(), notification.SenderID,
		notification.ReceiverID, notification.EventID, notification.Content, notification.Type)
	if err != nil {
		return errors.Wrap(err, "creating notification")
	}

	if notification.Type == Invitation {
		err = s.roleService.SetReservedRole(ctx, *notification.EventID, notification.ReceiverID, roles.Viewer)
		if err != nil {
			return err
		}
	}

	tokens := s.authService.TokensFromID(ctx, notification.ReceiverID)
	message := notificationMessage(notification.Type, session, tokens)
	s.SendMulticast(ctx, message)

	return nil
}

// CreateMany is like Create but it creates multiple notifications.
func (s service) CreateMany(ctx context.Context, session auth.Session, notification CreateNotificationMany) error {
	s.metrics.incMethodCalls("CreateMany")
	sqlTx := txgroup.SQLTx(ctx)

	fields := []string{"id", "sender_id", "receiver_id", "event_id", "content", "type"}
	stmt, err := postgres.BulkInsert(ctx, sqlTx, model.Notification.Tablename(), fields...)
	if err != nil {
		return err
	}
	defer stmt.Close()

	tokens := make([]string, 0, len(notification.ReceiverIDs))
	for _, receiverID := range notification.ReceiverIDs {
		_, err := stmt.ExecContext(ctx, ulid.NewString(), notification.SenderID, receiverID,
			notification.EventID, notification.Content, notification.Type)
		if err != nil {
			return err
		}
		userTokens := s.authService.TokensFromID(ctx, receiverID)
		tokens = append(tokens, userTokens...)
	}

	if _, err := stmt.ExecContext(ctx); err != nil {
		return errors.Wrap(err, "flush buffered data")
	}

	if notification.Type == Invitation {
		for _, receiverID := range notification.ReceiverIDs {
			err := s.roleService.SetReservedRole(ctx, *notification.EventID, receiverID, roles.Viewer)
			if err != nil {
				return err
			}
		}
	}

	message := notificationMessage(notification.Type, session, tokens)
	s.SendMulticast(ctx, message)

	return nil
}

// Delete removes a notification from the database
func (s service) Delete(ctx context.Context, notificationID string) error {
	s.metrics.incMethodCalls("Delete")
	sqlTx := txgroup.SQLTx(ctx)

	if _, err := sqlTx.ExecContext(ctx, "DELETE FROM notifications WHERE id=$1", notificationID); err != nil {
		return errors.Wrap(err, "deleting notification")
	}

	return nil
}

// DeleteInvitation deletes an invitation from the event.
func (s service) DeleteInvitation(ctx context.Context, eventID, senderID, receiverID string) error {
	s.metrics.incMethodCalls("DeleteInvitation")

	sqlTx := txgroup.SQLTx(ctx)

	q := "DELETE FROM notifications WHERE event_id=$1 AND sender_id=$2 AND receiver_id=$3 AND type=$4"
	if _, err := sqlTx.ExecContext(ctx, q, eventID, senderID, receiverID, Invitation); err != nil {
		return errors.Wrap(err, "deleting invitation")
	}

	return nil
}

// GetFromUser returns all the user's notifications.
func (s service) GetFromUser(ctx context.Context, userID string, params params.Query) ([]Notification, error) {
	s.metrics.incMethodCalls("GetFromUser")
	sqlTx := txgroup.SQLTx(ctx)

	// Update-then-select has better performance and produces less garbage
	// than update-returning when querying already seen notifications.
	q1 := "UPDATE notifications SET seen=true WHERE receiver_id=$1 AND seen=false"
	if _, err := sqlTx.ExecContext(ctx, q1, userID); err != nil {
		return nil, errors.Wrap(err, "updating seen statuses")
	}

	q2 := postgres.SelectWhere(model.Notification, "receiver_id=$1", "id", params)
	rows, err := sqlTx.QueryContext(ctx, q2, userID)
	if err != nil {
		return nil, errors.Wrap(err, "querying notifications")
	}

	var notifications []Notification
	if err := sqan.Rows(&notifications, rows); err != nil {
		return nil, errors.Wrap(err, "scanning notifications")
	}

	return notifications, nil
}

// GetFromUserCount returns the user's number of unseen notifications.
func (s service) GetFromUserCount(ctx context.Context, userID string) (int, error) {
	s.metrics.incMethodCalls("GetFromUserCount")

	q := "SELECT COUNT(*) FROM notifications WHERE receiver_id=$1 AND seen=false"
	row := s.db.QueryRowContext(ctx, q, userID)

	var count int
	if err := row.Scan(&count); err != nil {
		return 0, errors.Wrap(err, "scanning notifications count")
	}

	return count, nil
}

// Send sends a push notification message.
func (s service) Send(ctx context.Context, message *messaging.Message) {
	if message.Token == "" {
		return
	}
	s.sendNotification(func() error {
		if _, err := s.fcm.Send(context.Background(), message); err != nil {
			return errors.Wrap(err, "sending notification")
		}
		return nil
	})
}

// SendMulticast is like SendNotification but it sends it to multiple users.
func (s service) SendMulticast(ctx context.Context, notification *messaging.MulticastMessage) {
	if len(notification.Tokens) == 0 {
		return
	}
	s.sendNotification(func() error {
		if _, err := s.fcm.SendMulticast(context.Background(), notification); err != nil {
			log.Debug("sending notification", zap.Error(err))
			return errors.Wrap(err, "sending notification")
		}
		return nil
	})
}

// SendMany sends multiples notification messages.
func (s service) SendMany(ctx context.Context, messages []*messaging.Message) {
	s.sendNotification(func() error {
		if _, err := s.fcm.SendAll(context.Background(), messages); err != nil {
			return errors.Wrap(err, "sending notification to all")
		}
		return nil
	})
}

// SuscribeToTopic suscribes the authenticated user to a certain topic.
func (s service) SuscribeToTopic(ctx context.Context, session auth.Session, topic topic) error {
	s.metrics.incMethodCalls("SuscribeToTopic")

	if _, err := s.fcm.SubscribeToTopic(ctx, []string{session.DeviceToken}, string(topic)); err != nil {
		return errors.Wrapf(err, "suscribing to topic %q", topic)
	}
	return nil
}

// UnsuscribeFromTopic unsuscribes the authenticated user from a certain topic.
func (s service) UnsuscribeFromTopic(ctx context.Context, session auth.Session, topic topic) error {
	s.metrics.incMethodCalls("UnsuscribeFromTopic")

	if _, err := s.fcm.UnsubscribeFromTopic(ctx, []string{session.DeviceToken}, string(topic)); err != nil {
		return errors.Wrapf(err, "unsuscribing from topic %q", topic)
	}
	return nil
}

// sendNotification creates a goroutine that attempts to send the notification multiple times.
//
// Functions inside f should create a new context, otherwise the one from the request will be cancelled.
func (s service) sendNotification(f func() error) {
	go func() {
		if err := retry(f, s.maxRetries); err != nil {
			s.metrics.fail.Inc()
			return
		}
		s.metrics.sent.Inc()
	}()
}

func notificationMessage(typ notifType, session auth.Session, tokens []string) *messaging.MulticastMessage {
	switch typ {
	case FriendRequest:
		return NewFriendRequest(session, tokens)
	case Invitation:
		return NewInvitation(session, tokens)
	case Proposal:
		return NewProposal(session, tokens)
	case Mention:
		return NewMention(session, tokens)
	default:
		return nil
	}
}

func notificationContent(typ notifType, session auth.Session) string {
	switch typ {
	case FriendRequest:
		return tmplString(friendRequestTmpl, session.Username)
	case Invitation:
		return tmplString(invitationTmpl, session.Username)
	case Proposal:
		return tmplString(proposalTmpl, session.Username)
	case Mention:
		return tmplString(mentionTmpl, session.Username)
	default:
		return ""
	}
}

const (
	minBackoff = 100 * time.Millisecond
	maxBackoff = 1 * time.Minute
)

func retry(fn func() error, maxRetries int) error {
	var attempt int
	for {
		err := fn()
		if err == nil {
			return nil
		}

		if netErr, ok := err.(net.Error); !ok || !netErr.Temporary() {
			return err
		}

		attempt++
		backoff := minBackoff * time.Duration(attempt*attempt)
		if attempt > maxRetries || backoff > maxBackoff {
			return err
		}

		time.Sleep(backoff)
	}
}
