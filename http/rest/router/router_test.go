package router_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/GGP1/groove/config"
	"github.com/GGP1/groove/http/rest/router"
	"github.com/GGP1/groove/test"

	"github.com/stretchr/testify/assert"

	_ "github.com/lib/pq"
)

func TestNew(t *testing.T) {
	err := os.Setenv("GROOVE_CONFIG", "") // Use default configuration
	assert.NoError(t, err)
	cfg, err := config.New()
	assert.NoError(t, err)

	db := test.StartPostgres(t)
	dc := test.StartDgraph(t)
	mc := test.StartMemcached(t)
	rdb := test.StartRedis(t)

	srv := httptest.NewServer(router.New(cfg, db, dc, rdb, mc))
	defer srv.Close()

	res, err := srv.Client().Get(srv.URL)
	assert.NoError(t, err)

	assert.Equal(t, res.StatusCode, http.StatusOK)

	h := res.Header
	assert.Equal(t, h.Get("Access-Control-Allow-Origin"), "null")
	assert.Equal(t, h.Get("Access-Control-Allow-Credentials"), "true")
	assert.Equal(t, h.Get("Access-Control-Allow-Headers"),
		"Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, accept, origin, Cache-Control, X-Requested-With")
	assert.Equal(t, h.Get("Access-Control-Allow-Methods"), "POST, GET, PUT, DELETE, HEAD, OPTIONS")
	assert.Equal(t, h.Get("Access-Control-Expose-Headers"), "SID")
	assert.Equal(t, h.Get("Content-Security-Policy"), "default-src 'self'")
	assert.Equal(t, h.Get("Content-Type"), "application/json; charset=UTF-8")
	assert.Equal(t, h.Get("Feature-Policy"), "microphone 'none'; camera 'none'")
	assert.Equal(t, h.Get("Referrer-Policy"), "no-referrer")
	assert.Equal(t, h.Get("Strict-Transport-Security"), "max-age=63072000; includeSubDomains; preload")
	assert.Equal(t, h.Get("X-Content-Type-Options"), "nosniff")
	assert.Equal(t, h.Get("X-Frame-Options"), "DENY")
	assert.Equal(t, h.Get("X-Permitted-Cross-Domain-Policies"), "none")
	assert.Equal(t, h.Get("X-Xss-Protection"), "1; mode=block")
}
