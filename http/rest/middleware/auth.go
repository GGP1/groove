package middleware

import (
	"database/sql"
	"net/http"

	"github.com/GGP1/groove/internal/apikey"
	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/response"
	"github.com/GGP1/groove/service/auth"
	"github.com/GGP1/groove/service/event"
	"github.com/GGP1/groove/service/event/role"
	"github.com/GGP1/groove/service/user"

	"github.com/pkg/errors"
)

var (
	errAccessDenied  = errors.New("access denied")
	errLoginToAccess = errors.New("log in to access")
)

// Auth holds the redis instance used to authenticate users.
type Auth struct {
	db           *sql.DB
	authService  auth.Service
	userService  user.Service
	eventService event.Service
	roleService  role.Service
}

// NewAuth returns a new authentication/authorization middleware.
func NewAuth(db *sql.DB, authSv auth.Service, eventSv event.Service, userSv user.Service, roleSv role.Service) Auth {
	return Auth{
		db:           db,
		authService:  authSv,
		userService:  userSv,
		eventService: eventSv,
		roleService:  roleSv,
	}
}

// AdminsOnly requires the user to be an administrator to proceed.
//
// Hide endpoint existence by returning Not Found on errors.
func (a Auth) AdminsOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, ok := a.authService.AlreadyLoggedIn(ctx, r)
		if !ok {
			http.NotFound(w, r)
			return
		}

		isAdmin, err := a.userService.IsAdmin(ctx, session.ID)
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

// NotBanned lets in to an event only non-banned users.
func (a Auth) NotBanned(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		session, ok := a.authService.AlreadyLoggedIn(ctx, r)
		if !ok {
			response.Error(w, http.StatusUnauthorized, errLoginToAccess)
			return
		}

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		isBanned, err := a.eventService.IsBanned(ctx, eventID, session.ID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}
		if isBanned {
			response.Error(w, http.StatusForbidden, errors.New("you are banned from this event"))
		}
	})
}

// OwnUserOnly ensures that users are attempting to perform an action on their own account.
//
// This function already checks that the user is inside a session.
func (a Auth) OwnUserOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		session, ok := a.authService.AlreadyLoggedIn(ctx, r)
		if !ok {
			response.Error(w, http.StatusUnauthorized, errLoginToAccess)
			return
		}

		userID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if userID != session.ID {
			response.Error(w, http.StatusForbidden, errAccessDenied)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// EventPrivacyFilter makes sure the user can view a private event.
func (a *Auth) EventPrivacyFilter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if err := a.roleService.PrivacyFilter(ctx, r, eventID); err != nil {
			response.Error(w, http.StatusForbidden, err)
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
		if _, ok := a.authService.AlreadyLoggedIn(r.Context(), r); !ok {
			r.Header["Www-Authenticate"] = []string{`Basic realm="restricted", charset="UTF-8"`}
			response.Error(w, http.StatusUnauthorized, errLoginToAccess)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequirePermissions makes sure the user has the permissions necessary to perform a request.
func (a *Auth) RequirePermissions(permKeys ...string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			if _, ok := a.authService.AlreadyLoggedIn(ctx, r); !ok {
				response.Error(w, http.StatusUnauthorized, errLoginToAccess)
				return
			}

			eventID, err := params.IDFromCtx(ctx)
			if err != nil {
				response.Error(w, http.StatusBadRequest, err)
				return
			}

			if err := a.roleService.RequirePermissions(ctx, r, eventID, permKeys...); err != nil {
				response.Error(w, http.StatusForbidden, err)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// UserPrivacyFilter makes sure the user can view a user profile.
func (a *Auth) UserPrivacyFilter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		private, err := a.userService.PrivateProfile(ctx, userID)
		if err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		if private {
			session, ok := a.authService.AlreadyLoggedIn(ctx, r)
			if !ok {
				response.Error(w, http.StatusUnauthorized, errLoginToAccess)
				return
			}

			areFriends, err := a.userService.AreFriends(ctx, session.ID, userID)
			if err != nil {
				response.Error(w, http.StatusForbidden, err)
				return
			}

			if !areFriends {
				response.Error(w, http.StatusForbidden, errAccessDenied)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
