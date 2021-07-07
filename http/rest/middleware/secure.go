package middleware

import "net/http"

// Header keys must be canonical: the first letter and any letter
// following a hyphen must be upper case; the rest must be lowercase.
// For example, the canonical key for "accept-encoding" is "Accept-Encoding".
// Save a few bytes allocated by w.Header().Set() to convert header keys to a canonical format.
const (
	xssProtection                 = "X-Xss-Protection"
	strictTransportSecurity       = "Strict-Transport-Security"
	xFrameOpts                    = "X-Frame-Options"
	xContentTypeOpts              = "X-Content-Type-Options"
	contentSecurityPolicy         = "Content-Security-Policy"
	xPermittedCrossDomainPolicies = "X-Permitted-Cross-Domain-Policies"
	referrerPolicy                = "Referrer-Policy"
	featurePolicy                 = "Feature-Policy"
)

// Secure adds security headers to the http connection.
func Secure(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()
		// X-XSS-Protection: stops a page from loading when it detects XSS attacks
		header[xssProtection] = []string{"1; mode=block"}
		// HTTP Strict Transport Security:
		// lets a web site tell browsers that it should only be accessed using HTTPS, instead of using HTTP
		header[strictTransportSecurity] = []string{"max-age=63072000; includeSubDomains; preload"}
		// X-Frame-Options:
		// indicate whether or not a browser should be allowed to render a page in a <frame>, <iframe>, <embed> or <object>
		header[xFrameOpts] = []string{"DENY"}
		// X-Content-Type-Options:
		// is a marker used by the server to indicate that the MIME types advertised in the Content-Type headers
		// should not be changed and be followed
		header[xContentTypeOpts] = []string{"nosniff"}
		// Content Security Policy: allows web site administrators to control resources the user agent is allowed to load for a given page
		header[contentSecurityPolicy] = []string{"default-src 'self'"}
		// X-Permitted-Cross-Domain-Policies: allow other systems to access the domain
		header[xPermittedCrossDomainPolicies] = []string{"none"}
		// Referrer-Policy: sets the parameter for amount of information sent along with Referer Header while making a request
		header[referrerPolicy] = []string{"no-referrer"}
		// Feature-Policy: provides a mechanism to allow and deny the use of browser features in its own frame,
		// and in content within any <iframe> elements in the document
		header[featurePolicy] = []string{"microphone 'none'; camera 'none'"}

		next.ServeHTTP(w, r)
	})
}
