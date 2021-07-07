package auth

import (
	"encoding/json"
	"net/http"

	"github.com/GGP1/groove/internal/response"
	"github.com/GGP1/groove/internal/sanitize"

	"github.com/pkg/errors"
)

// BasicAuth provides basic authentication.
func BasicAuth(s Session) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if _, ok := s.AlreadyLoggedIn(ctx, r); ok {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		username, password, ok := r.BasicAuth()
		if !ok {
			r.Header["Www-Authenticate"] = []string{`Basic realm="restricted", charset="UTF-8"`}
			response.Error(w, http.StatusBadRequest, errors.New("Authorization header not found"))
			return
		}

		if err := s.Login(ctx, w, r, username, password); err != nil {
			r.Header["Www-Authenticate"] = []string{`Basic realm="restricted", charset="UTF-8"`}
			response.Error(w, http.StatusForbidden, err)
			return
		}

		response.JSONMessage(w, http.StatusOK, "logged in")
	}
}

// Login takes a user credentials and authenticates it.
func Login(s Session) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if _, ok := s.AlreadyLoggedIn(ctx, r); ok {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		var user userLogin
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		if err := user.Valid(); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		email := sanitize.Normalize(user.Email)
		password := sanitize.Normalize(user.Password)

		if err := s.Login(ctx, w, r, email, password); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		response.JSONMessage(w, http.StatusOK, "logged in")
	}
}

// Logout logs the user out from the session and removes cookies.
func Logout(s Session) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// This will be executed only if the user is already logged in.
		if err := s.Logout(r.Context(), w, r); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSONMessage(w, http.StatusOK, "logged out")
	}
}
