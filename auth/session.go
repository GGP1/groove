package auth

import (
	"context"
	"net/http"

	"github.com/GGP1/groove/internal/bufferpool"
	"github.com/GGP1/groove/internal/cookie"
	"github.com/GGP1/groove/internal/params"

	"github.com/pkg/errors"
)

var (
	errCorruptedSession               = errors.New("corrupted session")
	sessionKey          sessionCtxKey = struct{}{}
)

const saltLen = 16

// SessionInfo contains the information about the user session.
type SessionInfo struct {
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

// GetSessionInfo returns the user session information.
//
// The first time it fetches the info from cookies and sets it in the request's context.
func GetSessionInfo(ctx context.Context, r *http.Request) (SessionInfo, error) {
	sessionInfo, ok := ctx.Value(sessionKey).(SessionInfo)
	if !ok {
		sessionID, err := cookie.GetValue(r, cookie.Session)
		if err != nil {
			return SessionInfo{}, errors.New("login to access")
		}

		sInfo, err := unparseSessionInfo(sessionID)
		if err != nil {
			return SessionInfo{}, err
		}

		// Add SessionInfo struct to the request context
		*r = *r.WithContext(context.WithValue(ctx, sessionKey, sInfo))

		return sInfo, nil
	}

	return sessionInfo, nil
}

func parseSessionInfo(id string, salt []byte, premium bool) string {
	prem := byte('f')
	if premium {
		prem = 't'
	}
	buf := bufferpool.Get()
	defer bufferpool.Put(buf)
	buf.WriteString(id)
	buf.Write(salt)
	buf.WriteByte(prem)
	return buf.String()
}

func unparseSessionInfo(sessionID string) (SessionInfo, error) {
	// sessionID = uuid(36)+salt(saltLen)+premium(1)
	if len(sessionID) != 36+saltLen+1 {
		return SessionInfo{}, errCorruptedSession
	}
	id := sessionID[:len(sessionID)-saltLen-1]
	if err := params.ValidateUUID(id); err != nil {
		return SessionInfo{}, errCorruptedSession
	}
	salt := sessionID[len(sessionID)-saltLen-1 : len(sessionID)-1]
	premium := sessionID[len(sessionID)-1]

	return SessionInfo{
		ID:      id,
		Salt:    salt,
		Premium: premium == 't',
	}, nil
}
