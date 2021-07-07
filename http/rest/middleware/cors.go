package middleware

import "net/http"

// Header keys must be canonical: the first letter and any letter
// following a hyphen must be upper case; the rest must be lowercase.
// For example, the canonical key for "accept-encoding" is "Accept-Encoding".
const (
	allowOrigin      = "Access-Control-Allow-Origin"
	allowCredentials = "Access-Control-Allow-Credentials"
	allowHeaders     = "Access-Control-Allow-Headers"
	allowMethods     = "Access-Control-Allow-Methods"
	exposeHeaders    = "Access-Control-Expose-Headers"
)

// Cors sets origin, credentials, headers and methods allowed.
func Cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Save a few bytes allocated by w.Header().Set() to convert header keys to a canonical format
		header := w.Header()
		header[allowOrigin] = []string{"null"}
		header[allowCredentials] = []string{"true"}
		header[allowHeaders] = []string{"Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, accept, origin, Cache-Control, X-Requested-With"}
		header[allowMethods] = []string{"POST, GET, PUT, DELETE, HEAD, OPTIONS"}
		header[exposeHeaders] = []string{"SID"}

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
