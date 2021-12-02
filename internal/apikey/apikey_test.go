package apikey

import (
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GGP1/groove/internal/ulid"

	"github.com/dgrijalva/jwt-go"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	viper.Set("secrets.apikeys", "124sa5dad23as")

	apiKey, err := New(ulid.NewString())
	assert.NoError(t, err)

	assert.Equal(t, finalKeyLen, len(apiKey))
	assert.Equal(t, apiKey[:len(prefix)], prefix)
	assert.NotEqual(t, -1, strings.Index(apiKey, "."))
}

func TestCheck(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		apiKey, err := New(ulid.NewString())
		assert.NoError(t, err)

		err = Check(apiKey)
		assert.NoError(t, err)
	})

	t.Run("Invalid", func(t *testing.T) {
		apiKey, err := New(ulid.NewString())
		assert.NoError(t, err)

		err = Check("2" + apiKey)
		assert.Error(t, err)
	})
}

func TestFromRequest(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		viper.Set("secrets.apikeys", "124sa5dad23as")
		apiKey, err := New(ulid.NewString())
		assert.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(headerName, apiKey)
		got, err := FromRequest(req)
		assert.NoError(t, err)

		assert.Equal(t, apiKey, got)
	})

	t.Run("Invalid", func(t *testing.T) {
		apiKey, err := New("invalid_ulid")
		assert.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(headerName, apiKey)
		_, err = FromRequest(req)
		assert.Error(t, err)
	})

	t.Run("Non-existent", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		_, err := FromRequest(req)
		assert.Error(t, err)
	})
}

func TestRejectNoneMethod(t *testing.T) {
	salt := make([]byte, saltLen)
	_, err := rand.Read(salt)
	assert.NoError(t, err)

	token := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"key":  ulid.NewString(),
		"salt": salt,
	})

	apiKey, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	assert.NoError(t, err)

	err = Check(prefix + apiKey)
	assert.Error(t, err)
}

func BenchmarkNew(b *testing.B) {
	id := ulid.NewString()
	for i := 0; i < b.N; i++ {
		New(id)
	}
}
