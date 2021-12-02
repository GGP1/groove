package auth

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/GGP1/groove/internal/bufferpool"
	"github.com/GGP1/groove/internal/cookie"
	"github.com/GGP1/groove/internal/httperr"
	"github.com/GGP1/groove/internal/ulid"
	"github.com/GGP1/groove/internal/validate"
	"github.com/GGP1/groove/model"
)

var (
	errLoginToAccess                  = httperr.Unauthorized("log in to access")
	errCorruptedSession               = httperr.Forbidden("corrupted session")
	sessionKey          sessionCtxKey = struct{}{}
)

const (
	idLen     = ulid.EncodedSize // ULID string length
	saltLen   = 16
	separator = "/"
)

// Session contains the information about the user session.
type Session struct {
	ID       string
	Username string
	// TODO PRODUCTION: the cookie will be sent over https, meaning that it's infeasible that someone will get access to them, however
	// if the cookie gets stolen on the client-side then the attacker could use replay attacks to send requests to the server,
	// getting access to that user's account. If the client (browser or application) can't be secured maybe the best approach would be
	// to use a nonce (instead of a salt) that's incremented everytime the user makes a request. It would require one redis call more and replacing the cookie with
	// the new value each time but it mitigates the attack.
	DeviceToken string
	Type        model.UserType
}

type sessionCtxKey struct{}

// GetSession returns the user session information.
//
// The first time it fetches the info from cookies and sets it in the request's context.
func GetSession(ctx context.Context, r *http.Request) (Session, error) {
	session, ok := r.Context().Value(sessionKey).(Session)
	if !ok {
		sessionToken, err := cookie.GetValue(r, cookie.Session)
		if err != nil {
			return Session{}, errLoginToAccess
		}

		sess, err := unparseSessionToken(sessionToken)
		if err != nil {
			return Session{}, err
		}

		// Add Session struct to the request context
		*r = *r.WithContext(context.WithValue(ctx, sessionKey, sess))

		return sess, nil
	}

	return session, nil
}

func parseSessionToken(id, username, deviceToken string, typ model.UserType) string {
	buf := bufferpool.Get()
	buf.WriteString(id)
	buf.WriteString(separator)
	buf.WriteString(username)
	buf.WriteString(separator)
	buf.WriteString(deviceToken)
	buf.WriteString(separator)
	buf.WriteString(strconv.Itoa(int(typ)))
	token := buf.String()
	bufferpool.Put(buf)

	return token
}

func unparseSessionToken(token string) (Session, error) {
	parts := strings.Split(token, separator)
	if len(parts) != 4 {
		return Session{}, errCorruptedSession
	}

	id := parts[0]
	if err := validate.ULID(id); err != nil {
		return Session{}, errCorruptedSession
	}
	typ, err := model.StringToUserType(parts[3])
	if err != nil {
		return Session{}, errCorruptedSession
	}
	return Session{
		ID:          id,
		Username:    parts[1],
		DeviceToken: parts[2],
		Type:        typ,
	}, nil
}
