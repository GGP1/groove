// Package cookie provides utilities for managing encrypted http cookies.
package cookie

import (
	"encoding/hex"
	"net/http"

	"github.com/GGP1/groove/internal/crypt"

	"github.com/pkg/errors"
)

const (
	// Session is the name of the cookie used to store session information.
	Session = "SID"
	maxAge  = 0x13C680
)

// Considerations before choosing between standard (Set) and secure (SetSecure) cookies.
// https://tools.ietf.org/html/draft-ietf-httpbis-cookie-same-site-00#section-5.2
// Always use either lax or strict modes to avoid being vulnerable to csrf attacks

// Delete a cookie.
func Delete(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
}

// Get deciphers and returns the cookie.
func Get(r *http.Request, name string) (*http.Cookie, error) {
	cookie, err := r.Cookie(name)
	if err != nil {
		return nil, err
	}

	ciphertext, err := hex.DecodeString(cookie.Value)
	if err != nil {
		return nil, errors.Wrap(err, "decoding cookie value")
	}

	plaintext, err := crypt.Decrypt(ciphertext)
	if err != nil {
		return nil, err
	}

	cookie.Value = string(plaintext)

	return cookie, nil
}

// GetValue is like Get but returns only the value.
func GetValue(r *http.Request, name string) (string, error) {
	cookie, err := Get(r, name)
	if err != nil {
		return "", err
	}

	return cookie.Value, nil
}

// IsSet returns whether the cookie is set or not.
func IsSet(r *http.Request, name string) bool {
	c, _ := r.Cookie(name)
	return c != nil
}

// Set a cookie.
func Set(w http.ResponseWriter, name, value, path string) error {
	ciphertext, err := crypt.Encrypt([]byte(value))
	if err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    hex.EncodeToString(ciphertext),
		Path:     path,
		Secure:   false, // Only https (TODO PRODUCTION: set to true)
		HttpOnly: true,  // True means no scripts, http requests only. It does not refer to http(s)
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
	})

	return nil
}

// SetHost sets a cookie that is accepted only if comes from a secure origin.
// It is domain-locked and has the path set to "/".
//
// See: https://developer.mozilla.org/en-US/docs/Web/HTTP/Cookies
func SetHost(w http.ResponseWriter, name, value string) error {
	ciphertext, err := crypt.Encrypt([]byte(value))
	if err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "__Host-" + name,
		Value:    hex.EncodeToString(ciphertext),
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   maxAge * 2,
	})

	return nil
}
