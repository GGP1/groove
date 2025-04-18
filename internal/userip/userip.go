// Package userip provides functions for extracting a user IP address from a
// request and associating it with a Context.
package userip

import (
	"context"
	"net"
	"net/http"
	"strings"
)

const (
	xRealIP       = "X-Real-Ip"
	xForwardedFor = "X-Forwarded-For"
	forwarded     = "Forwarded"
	cloudflareIP  = "Cf-Connecting-Ip"
)

// userIPKey is the context key for the user IP address.
//
// Check if there is an allocation going on and change to const
// and type int with a value of 0 if not.
var userIPKey key

// The key type is unexported to prevent collisions with context keys defined in
// other packages.
type key struct{}

// Get returns the user IP. If it's retrieved from the request it sets it in the request's context.
func Get(ctx context.Context, r *http.Request) string {
	if userIP, ok := r.Context().Value(userIPKey).(string); ok {
		return userIP
	}

	ip := fromRequest(r)
	// Add ip to the request context
	// Currently userip is called once only so it's not worth to store it
	// *r = *r.WithContext(context.WithValue(ctx, userIPKey, ip))

	return ip
}

// fromRequest extracts the user IP from the request and returns it.
func fromRequest(r *http.Request) string {
	ip := r.RemoteAddr
	if strings.Contains(ip, ":") {
		host, _, err := net.SplitHostPort(ip)
		if err == nil {
			return host
		}
	}

	if realIP := getHeader(r, xRealIP); realIP != "" {
		// X-Real-IP: <ip>
		return realIP
	}

	if xff := getHeader(r, xForwardedFor); xff != "" {
		// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/X-Forwarded-For
		// X-Forwarded-For: <client>, <proxy1>, <proxy2>
		idx := strings.Index(xff, ",")
		return xff[:idx]
	}

	// https://support.cloudflare.com/hc/en-us/articles/206776727-What-is-True-Client-IP
	if cf := getHeader(r, cloudflareIP); cf != "" {
		idx := strings.Index(cf, ",")
		return cf[:idx]
	}

	if f := getHeader(r, forwarded); f != "" {
		return parseForwardedHeader(f)
	}

	return ip
}

func getHeader(r *http.Request, key string) string {
	v := r.Header[key]
	if len(v) == 0 {
		return ""
	}
	return v[0]
}

// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Forwarded
// Forwarded: by=<identifier>;for=<identifier>;host=<host>;proto=<http|https>
func parseForwardedHeader(value string) string {
	parts := strings.Split(value, ";")

	for _, part := range parts {
		kv := strings.Split(part, "=")

		if len(kv) == 2 {
			k := strings.ToLower(strings.TrimSpace(kv[0]))
			if k == "for" {
				return strings.TrimSpace(kv[1])
			}
		}
	}

	return ""
}
