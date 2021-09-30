package userip

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFromRequest(t *testing.T) {
	cases := []struct {
		desc     string
		key      string
		value    string
		expected string
	}{
		{
			desc:     "X-Real-Ip",
			key:      xRealIP,
			value:    "127.0.0.1",
			expected: "127.0.0.1",
		},
		{
			desc:     "X-Forwarded-For",
			key:      xForwardedFor,
			value:    "127.0.0.1, 133.0.25.2",
			expected: "127.0.0.1",
		},
		{
			desc:     "Cf-Connecting-Ip",
			key:      cloudflareIP,
			value:    "127.0.0.1, 133.0.25.2",
			expected: "127.0.0.1",
		},
		{
			desc:     "Forwarded",
			key:      forwarded,
			value:    "for=127.0.0.1;host=0.0.0.0;proto=https",
			expected: "127.0.0.1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			r.RemoteAddr = ""

			r.Header.Set(tc.key, tc.value)

			got := fromRequest(r)
			assert.Equal(t, tc.expected, got)
		})
	}

	t.Run("Remote address", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = "127.0.0.1:8080"

		got := fromRequest(r)
		assert.Equal(t, "127.0.0.1", got)
	})
}

func TestGet(t *testing.T) {
	ctx := context.Background()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "127.0.0.1:7070"
	expected := "127.0.0.1"

	got := Get(ctx, r)
	assert.Equal(t, expected, got)

	ip, ok := r.Context().Value(userIPKey).(string)
	assert.True(t, ok)
	assert.Equal(t, expected, ip)
}
