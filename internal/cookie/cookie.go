// Package cookie provides utilities for managing encrypted http cookies.
package cookie

import (
	"encoding/hex"
	"net/http"

	"github.com/GGP1/groove/internal/crypt"

	"github.com/pkg/errors"
)

// Session is the name of the cookie used to store session information.
const Session = "SID"

// Considerations before choosing between standard (Set) and secure (SetSecure) cookies.
// https://tools.ietf.org/html/draft-ietf-httpbis-cookie-same-site-00#section-5.2

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
		Secure:   false, // Only https (set to true when in production)
		HttpOnly: true,  // True means no scripts, http requests only. It does not refer to http(s)
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400 * 15,
	})

	return nil
}

// SetSecure is like Set but with more restrictions and security.
func SetSecure(w http.ResponseWriter, name, value, path string) error {
	ciphertext, err := crypt.Encrypt([]byte(value))
	if err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "__Secure-" + name,
		Value:    hex.EncodeToString(ciphertext),
		Path:     path,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400 * 30,
	})

	return nil
}
