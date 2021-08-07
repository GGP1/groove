package auth

import (
	"context"
	"net/http"

	"github.com/GGP1/groove/internal/bufferpool"
	"github.com/GGP1/groove/internal/cookie"
	"github.com/GGP1/groove/internal/ulid"
	"github.com/GGP1/groove/internal/validate"

	"github.com/pkg/errors"
)

var (
	errCorruptedSession               = errors.New("corrupted session")
	sessionKey          sessionCtxKey = struct{}{}
)

const (
	idLen   = ulid.EncodedSize // ULID string length
	saltLen = 16
)

// Session contains the information about the user session.
type Session struct {
	ID string
	// TODO: the cookie will be sent over https, meaning that it's infeasible that someone will get access to them, however
	// if the cookie gets stolen on the client-side then the attacker could use replay attacks to send requests to the server,
	// getting access to that user's account. If the client (browser or application) can't be secured maybe the best approach would be
	// to use a nonce (instead of a salt) that's incremented everytime the user makes a request. It would require one redis call more and replacing the cookie with
	// the new value each time but it mitigates the attack.
	Salt    string
	Premium bool
}

type sessionCtxKey struct{}

// GetSession returns the user session information.
//
// The first time it fetches the info from cookies and sets it in the request's context.
func GetSession(ctx context.Context, r *http.Request) (Session, error) {
	session, ok := ctx.Value(sessionKey).(Session)
	if !ok {
		sessionToken, err := cookie.GetValue(r, cookie.Session)
		if err != nil {
			return Session{}, errors.New("login to access")
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

func parseSessionToken(id string, salt []byte, premium bool) string {
	prem := byte('f')
	if premium {
		prem = 't'
	}

	buf := bufferpool.Get()
	buf.Grow(idLen + saltLen + 1)
	buf.WriteString(id)
	buf.Write(salt)
	buf.WriteByte(prem)
	token := buf.String()
	bufferpool.Put(buf)

	return token
}

func unparseSessionToken(token string) (Session, error) {
	// sessionID = ulid(26)+salt(saltLen)+premium(1)
	if len(token) != idLen+saltLen+1 {
		return Session{}, errCorruptedSession
	}
	id := token[:idLen]
	if err := validate.ULID(id); err != nil {
		return Session{}, errCorruptedSession
	}
	last := len(token) - 1
	salt := token[idLen:last]
	premium := token[last]

	return Session{
		ID:      id,
		Salt:    salt,
		Premium: premium == 't',
	}, nil
}
