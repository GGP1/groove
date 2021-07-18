package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GGP1/groove/internal/cookie"
	"github.com/GGP1/groove/internal/crypt"
	"github.com/GGP1/groove/internal/ulid"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestGetSession(t *testing.T) {
	expectedID := ulid.NewString()
	expectedSalt := []byte("0123456789012345")
	expectedPremium := true
	viper.Set("secrets.encryption", "l'[3 k2F]Q")
	sessionToken := parseSessionToken(expectedID, expectedSalt, expectedPremium)
	ciphertext, err := crypt.Encrypt([]byte(sessionToken))
	assert.NoError(t, err)

	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{
		Name:  cookie.Session,
		Value: hex.EncodeToString(ciphertext),
		Path:  "/",
	})

	sessionInfo, err := GetSession(context.Background(), r)
	assert.NoError(t, err)

	assert.Equal(t, expectedID, sessionInfo.ID)
	assert.Equal(t, string(expectedSalt), sessionInfo.Salt)
	assert.Equal(t, expectedPremium, sessionInfo.Premium)

	value, ok := r.Context().Value(sessionKey).(Session)
	assert.True(t, ok)
	assert.Equal(t, expectedID, value.ID)
	assert.Equal(t, string(expectedSalt), value.Salt)
	assert.Equal(t, expectedPremium, value.Premium)
}

func TestParseSessionData(t *testing.T) {
	id := ulid.NewString()
	salt := make([]byte, saltLen)
	_, err := rand.Read(salt)
	assert.NoError(t, err)
	token := parseSessionToken(id, salt, true)

	assert.Equal(t, token[:len(token)-saltLen-1], id)
	assert.Equal(t, token[len(token)-saltLen-1:len(token)-1], string(salt))
	assert.Equal(t, token[len(token)-1], uint8('t'))
}

func TestUnparseSessionData(t *testing.T) {
	id := ulid.NewString()
	salt := make([]byte, saltLen)
	_, err := rand.Read(salt)
	assert.NoError(t, err)
	premium := true
	token := parseSessionToken(id, salt, premium)

	sessionInfo, err := unparseSessionToken(token)
	assert.NoError(t, err)

	assert.Equal(t, sessionInfo.ID, id)
	assert.Equal(t, sessionInfo.Salt, string(salt))
	assert.Equal(t, sessionInfo.Premium, premium)
}

func BenchmarkGetSession(b *testing.B) {
	ctx := context.Background()
	id := ulid.NewString()
	salt := make([]byte, saltLen)
	_, err := rand.Read(salt)
	assert.NoError(b, err)

	token := parseSessionToken(id, salt, true)
	ciphertext, err := crypt.Encrypt([]byte(token))
	assert.NoError(b, err)

	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{
		Name:  cookie.Session,
		Value: hex.EncodeToString(ciphertext),
		Path:  "/",
	})

	for i := 0; i < b.N; i++ {
		GetSession(ctx, r)
	}
}
