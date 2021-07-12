package middleware

import (
	"database/sql"
	"net/http"

	"github.com/GGP1/groove/auth"
	"github.com/GGP1/groove/internal/apikey"
	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/response"
	"github.com/GGP1/groove/service/event"
	"github.com/GGP1/groove/service/user"

	"github.com/dgraph-io/dgo/v210"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

var (
	errAccessDenied  = errors.New("access denied")
	errLoginToAccess = errors.New("log in to access")
)

// Auth holds the redis instance used to authenticate users.
type Auth struct {
	db      *sqlx.DB
	dc      *dgo.Dgraph
	session auth.Session
	// TODO: do not make calls to the database here as we will require to open another transaction later anyways
	eventService event.Service
	userService  user.Service
}

// NewAuth returns a new authentication/authorization middleware.
func NewAuth(db *sqlx.DB, dc *dgo.Dgraph, session auth.Session, eventSv event.Service, userSv user.Service) Auth {
	return Auth{
		db:           db,
		dc:           dc,
		session:      session,
		eventService: eventSv,
		userService:  userSv,
	}
}

// AdminsOnly requires the user to be an administrator to proceed.
func (a Auth) AdminsOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Hide endpoint existence by returning Not Found on every error
		ctx := r.Context()
		sessionInfo, ok := a.session.AlreadyLoggedIn(ctx, r)
		if !ok {
			http.NotFound(w, r)
			return
		}

		tx, err := a.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer tx.Commit()

		isAdmin, err := a.userService.IsAdmin(ctx, tx, sessionInfo.ID)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if !isAdmin {
			http.NotFound(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// OwnUserOnly ensures that users are attempting to perform an action on their own account.
//
// This function already checks that the user is inside a session.
func (a Auth) OwnUserOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		sessionInfo, ok := a.session.AlreadyLoggedIn(ctx, r)
		if !ok {
			response.Error(w, http.StatusUnauthorized, errLoginToAccess)
			return
		}

		id, err := params.UUIDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if id != sessionInfo.ID {
			response.Error(w, http.StatusForbidden, errAccessDenied)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequireAPIKey makes sure the client has a valid API key.
func (a *Auth) RequireAPIKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := apikey.FromRequest(r); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequireLogin makes sure the user is logged in, returns an error otherwise.
func (a *Auth) RequireLogin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := a.session.AlreadyLoggedIn(r.Context(), r); !ok {
			r.Header["Www-Authenticate"] = []string{`Basic realm="restricted", charset="UTF-8"`}
			response.Error(w, http.StatusUnauthorized, errLoginToAccess)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequirePremium requires the user to be premium to continue.
func (a *Auth) RequirePremium(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		sessionInfo, ok := a.session.AlreadyLoggedIn(ctx, r)
		if ok {
			response.Error(w, http.StatusUnauthorized, errLoginToAccess)
			return
		}

		if !sessionInfo.Premium {
			response.Error(w, http.StatusForbidden, errAccessDenied)
			return
		}

		next.ServeHTTP(w, r)
	})
}
