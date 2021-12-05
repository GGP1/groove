package auth

import (
	"encoding/json"
	"net/http"

	"github.com/GGP1/groove/internal/response"
	"github.com/GGP1/groove/internal/sanitize"

	"github.com/pkg/errors"
)

// Handler handles auth endpoints.
type Handler struct {
	service Service
}

// NewHandler returns an auth handler.
func NewHandler(service Service) Handler {
	return Handler{
		service: service,
	}
}

// BasicAuth provides basic authentication.
func (h *Handler) BasicAuth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if _, ok := h.service.AlreadyLoggedIn(ctx, r); ok {
			response.NoContent(w)
			return
		}

		username, password, ok := r.BasicAuth()
		if !ok {
			r.Header["Www-Authenticate"] = []string{`Basic realm="restricted", charset="UTF-8"`}
			response.Error(w, http.StatusBadRequest, errors.New("Authorization header not found"))
			return
		}

		login := Login{
			Username: sanitize.Normalize(username),
			Password: sanitize.Normalize(password),
		}
		user, err := h.service.Login(ctx, w, r, login)
		if err != nil {
			r.Header["Www-Authenticate"] = []string{`Basic realm="restricted", charset="UTF-8"`}
			response.Error(w, http.StatusForbidden, err)
			return
		}

		response.JSON(w, http.StatusOK, user)
	}
}

// Login takes a user credentials and authenticates it.
func (h *Handler) Login() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if _, ok := h.service.AlreadyLoggedIn(ctx, r); ok {
			response.NoContent(w)
			return
		}

		var login Login
		if err := json.NewDecoder(r.Body).Decode(&login); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		userSession, err := h.service.Login(ctx, w, r, login)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, userSession)
	}
}

// Logout logs the user out from the session and removes cookies.
func (h *Handler) Logout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// This will be executed only if the user is already logged in.
		if err := h.service.Logout(r.Context(), w, r); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.NoContent(w)
	}
}
