package notification

import (
	"text/template"

	"github.com/GGP1/groove/internal/bufferpool"
	"github.com/GGP1/groove/model"
	"github.com/GGP1/groove/service/auth"

	"firebase.google.com/go/v4/messaging"
)

var (
	invitationTmpl    = template.Must(template.New("invitation").Parse("{{ . }} invited you to an event"))
	friendRequestTmpl = template.Must(template.New("friend_request").Parse("{{ . }} wants to be your friend"))
	proposalTmpl      = template.Must(template.New("proposal").Parse("{{ . }} wants to create an event"))
	likeTmpl          = template.Must(template.New("like").Parse("{{ . }} liked your event"))
	mentionTmpl       = template.Must(template.New("mention").Parse("{{ . }} mentioned you in a comment"))

	// contentAvailable allows us to display background notifications on ios.
	contentAvailable = &messaging.APNSConfig{Payload: &messaging.APNSPayload{Aps: &messaging.Aps{ContentAvailable: true}}}
)

// NewMessage creates a new fcm message.
func NewMessage(receiverToken, title, body, imageURL string) *messaging.Message {
	return &messaging.Message{
		Token: receiverToken,
		Notification: &messaging.Notification{
			Title:    title,
			Body:     body,
			ImageURL: imageURL,
		},
		APNS: contentAvailable,
	}
}

// NewTopicMessage is like NewMessage but for a certain topic.
func NewTopicMessage(receiverToken, title, body string, topic model.Topic) *messaging.Message {
	return &messaging.Message{
		Token: receiverToken,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Topic: string(topic),
		APNS:  contentAvailable,
	}
}

// NewInvitation creates a new invitation notification.
func NewInvitation(session auth.Session, tokens []string) *messaging.MulticastMessage {
	return &messaging.MulticastMessage{
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title: "Invitation",
			Body:  InvitationContent(session),
		},
		APNS: contentAvailable,
	}
}

// NewFriendRequest creates a new friend request notification.
func NewFriendRequest(session auth.Session, tokens []string) *messaging.MulticastMessage {
	return &messaging.MulticastMessage{
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title: "Friend request",
			Body:  FriendRequestContent(session),
		},
		APNS: contentAvailable,
	}
}

// NewLike creates a like notification.
func NewLike(session auth.Session, tokens []string) *messaging.MulticastMessage {
	return &messaging.MulticastMessage{
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title: "Like",
			Body:  tmplString(likeTmpl, session.Username),
		},
		APNS: contentAvailable,
	}
}

// NewMention creates a mention notification.
func NewMention(session auth.Session, tokens []string) *messaging.MulticastMessage {
	return &messaging.MulticastMessage{
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title: "Mention",
			Body:  MentionContent(session),
		},
		APNS: contentAvailable,
	}
}

// NewProposal creates a new proposal notification.
func NewProposal(session auth.Session, tokens []string) *messaging.MulticastMessage {
	return &messaging.MulticastMessage{
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title: "Proposal",
			Body:  tmplString(proposalTmpl, session.Username),
		},
		APNS: contentAvailable,
	}
}

// FriendRequestContent returns the content template of a friend request.
func FriendRequestContent(session auth.Session) string {
	return tmplString(friendRequestTmpl, session.Username)
}

// InvitationContent returns the content template of an invitation.
func InvitationContent(session auth.Session) string {
	return tmplString(invitationTmpl, session.Username)
}

// MentionContent returns the content template of an mention.
func MentionContent(session auth.Session) string {
	return tmplString(mentionTmpl, session.Username)
}

func tmplString(t *template.Template, data interface{}) string {
	buf := bufferpool.Get()
	if err := t.Execute(buf, data); err != nil {
		panic(err)
	}
	tmpl := buf.String()
	bufferpool.Put(buf)
	return tmpl
}
