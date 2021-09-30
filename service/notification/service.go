package notification

import (
	"context"
	"database/sql"
	"net"
	"time"

	"github.com/GGP1/groove/config"
	"github.com/GGP1/groove/internal/bufferpool"
	"github.com/GGP1/groove/internal/httperr"
	"github.com/GGP1/groove/internal/log"
	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/roles"
	"github.com/GGP1/groove/internal/scan"
	"github.com/GGP1/groove/internal/sqltx"
	"github.com/GGP1/groove/internal/ulid"
	"github.com/GGP1/groove/model"
	"github.com/GGP1/groove/service/auth"
	"github.com/GGP1/groove/service/event/role"
	"github.com/GGP1/groove/storage/dgraph"
	"github.com/GGP1/groove/storage/postgres"

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
// Here a LSM tree would do better than a B+ one. But it all depends if the users
// take actions on notifications or not (if not they will be retrieved multiple times).

// Service represents the notification service.
type Service interface {
	Answer(ctx context.Context, id, authUserID string, accepted bool) error
	Create(ctx context.Context, session auth.Session, notification CreateNotification) error
	Delete(ctx context.Context, notificationID string) error
	GetFromUser(ctx context.Context, userID string, params params.Query) ([]Notification, error)
	GetFromUserCount(ctx context.Context, userID string) (int, error)
	Send(ctx context.Context, message *messaging.Message)
	SendMulticast(ctx context.Context, notification *messaging.MulticastMessage)
	SendMany(ctx context.Context, messages []*messaging.Message)
	SuscribeToTopic(ctx context.Context, session auth.Session, topic topic) error
	UnsuscribeFromTopic(ctx context.Context, session auth.Session, topic topic) error
}

// https://github.com/firebase/firebase-admin-go/blob/717412a1698e23b42f1c0e454274277c7d743050/snippets/messaging.go
type service struct {
	db         *sql.DB
	dc         *dgo.Dgraph
	fcm        *messaging.Client
	metrics    metrics
	maxRetries int

	authService auth.Service
	roleService role.Service
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
	sqlTx := sqltx.FromContext(ctx)

	rows, err := sqlTx.QueryContext(ctx, "SELECT * FROM notifications WHERE id=$1", id)
	if err != nil {
		return errors.Wrap(err, "querying notification")
	}

	var notification Notification
	if err := scan.Row(&notification, rows); err != nil {
		return errors.Wrap(err, "scanning notification")
	}

	if notification.ReceiverID != authUserID {
		return httperr.New("access denied", httperr.Forbidden)
	}

	if !accepted {
		if _, err := sqlTx.ExecContext(ctx, "DELETE FROM notifications WHERE id=$1", id); err != nil {
			return errors.Wrap(err, "deleting notification")
		}
		return nil
	}

	switch notification.Type {
	case Invitation: // Add receiver as an attendant of the event
		return s.roleService.SetReservedRole(ctx, notification.EventID.String, notification.ReceiverID, roles.Attendant)
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
	case Proposal: // TODO: Increment counter
	}

	return nil
}

// Create creates a new notification.
func (s service) Create(ctx context.Context, session auth.Session, notification CreateNotification) error {
	s.metrics.incMethodCalls("Create")
	sqlTx := sqltx.FromContext(ctx)

	if notification.SenderID == notification.ReceiverID {
		return httperr.New("cannot perform this action on your account", httperr.Forbidden)
	}

	q := `INSERT INTO notifications (id, sender_id, receiver_id, event_id, content, type) VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := sqlTx.ExecContext(ctx, q, ulid.NewString(), notification.SenderID,
		notification.ReceiverID, notification.EventID, notification.Content, notification.Type)
	if err != nil {
		return errors.Wrap(err, "creating notification")
	}

	tokens := s.authService.TokensFromID(ctx, notification.ReceiverID)
	switch notification.Type {
	case FriendRequest:
		s.SendMulticast(ctx, NewFriendRequest(session, tokens))
	case Invitation:
		if err := notification.Validate(); err != nil {
			return err
		}
		// TODO: is it necessary to use dgraph edges for this? maybe create an edge between the users to identify
		// who invited whom
		err := dgraph.AddEventEdge(ctx, s.dc, *notification.EventID, dgraph.Invited, notification.ReceiverID)
		if err != nil {
			return err
		}
		err = s.roleService.SetReservedRole(ctx, *notification.EventID, notification.ReceiverID, roles.Viewer)
		if err != nil {
			return err
		}
		s.SendMulticast(ctx, NewInvitation(session, tokens))
	case Proposal:
		s.SendMulticast(ctx, NewProposal(session, tokens))
	case Mention:
		s.SendMulticast(ctx, NewMention(session, tokens))
	}

	return nil
}

// Delete removes a notification from the database
func (s service) Delete(ctx context.Context, notificationID string) error {
	s.metrics.incMethodCalls("Delete")
	sqlTx := sqltx.FromContext(ctx)

	if _, err := sqlTx.ExecContext(ctx, "DELETE FROM notifications WHERE id=$1", notificationID); err != nil {
		return errors.Wrap(err, "deleting notification")
	}

	return nil
}

// GetFromUser returns all the user's notifications.
//
// Pagination is not enabled in this method.
func (s service) GetFromUser(ctx context.Context, userID string, params params.Query) ([]Notification, error) {
	s.metrics.incMethodCalls("GetFromUser")
	sqlTx := sqltx.FromContext(ctx)

	// Update-then-select has better performance and produces less garbage
	// than update-returning when querying already seen notifications.
	q1 := "UPDATE notifications SET seen=true WHERE receiver_id=$1 AND seen=false"
	if _, err := sqlTx.ExecContext(ctx, q1, userID); err != nil {
		return nil, errors.Wrap(err, "updating seen statuses")
	}

	buf := bufferpool.Get()
	buf.WriteString("SELECT ")
	postgres.WriteFields(buf, model.Notification, params.Fields)
	buf.WriteString(" FROM notifications WHERE receiver_id=$1")
	rows, err := sqlTx.QueryContext(ctx, buf.String(), userID)
	if err != nil {
		return nil, errors.Wrap(err, "querying notifications")
	}
	bufferpool.Put(buf)

	var notifications []Notification
	if err := scan.Rows(&notifications, rows); err != nil {
		return nil, errors.Wrap(err, "scanning notifications")
	}

	return notifications, nil
}

// GetFromUserCount returns the user's number of unseen notifications.
func (s service) GetFromUserCount(ctx context.Context, userID string) (int, error) {
	s.metrics.incMethodCalls("GetFromUserCount")

	rows, err := s.db.QueryContext(ctx, "SELECT COUNT(*) FROM notifications WHERE receiver_id=$1 AND seen=false", userID)
	if err != nil {
		return 0, errors.Wrap(err, "querying notifications count")
	}

	var count int
	if err := scan.Row(&count, rows); err != nil {
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
