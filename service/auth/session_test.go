package auth

import (
	"context"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/GGP1/groove/internal/cookie"
	"github.com/GGP1/groove/internal/crypt"
	"github.com/GGP1/groove/internal/ulid"
	"github.com/GGP1/groove/model"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestGetSession(t *testing.T) {
	expectedID := ulid.NewString()
	expectedUsername := "username"
	expectedDeviceToken := "0123456789012345"
	expectedType := model.UserType(1)
	viper.Set("secrets.encryption", "l'[3 k2F]Q")
	sessionToken := parseSessionToken(expectedID, expectedUsername, expectedDeviceToken, expectedType)
	ciphertext, err := crypt.Encrypt([]byte(sessionToken))
	assert.NoError(t, err)

	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{
		Name:  cookie.Session,
		Value: hex.EncodeToString(ciphertext),
		Path:  "/",
	})

	session, err := GetSession(context.Background(), r)
	assert.NoError(t, err)

	assert.Equal(t, expectedID, session.ID)
	assert.Equal(t, expectedUsername, session.Username)
	assert.Equal(t, string(expectedDeviceToken), session.DeviceToken)
	assert.Equal(t, expectedType, session.Type)

	value, ok := r.Context().Value(sessionKey).(Session)
	assert.True(t, ok)
	assert.Equal(t, expectedID, value.ID)
	assert.Equal(t, expectedUsername, value.Username)
	assert.Equal(t, string(expectedDeviceToken), value.DeviceToken)
	assert.Equal(t, expectedType, value.Type)
}

func TestParseSessionToken(t *testing.T) {
	id := ulid.NewString()
	username := "username"
	deviceToken := "device_token"
	typ := model.UserType(2)
	token := parseSessionToken(id, username, deviceToken, typ)

	parts := strings.Split(token, separator)
	assert.Equal(t, parts[0], id)
	assert.Equal(t, parts[1], username)
	assert.Equal(t, parts[2], deviceToken)
	assert.Equal(t, parts[3], strconv.Itoa(int(typ)))
}

func TestUnparseSessionToken(t *testing.T) {
	id := ulid.NewString()
	username := "username"
	deviceToken := "device_token"
	typ := model.UserType(2)
	token := parseSessionToken(id, username, deviceToken, typ)

	session, err := unparseSessionToken(token)
	assert.NoError(t, err)

	assert.Equal(t, session.ID, id)
	assert.Equal(t, session.Username, username)
	assert.Equal(t, session.DeviceToken, deviceToken)
	assert.Equal(t, session.Type, typ)
}

func BenchmarkGetSession(b *testing.B) {
	ctx := context.Background()
	id := ulid.NewString()
	username := "username"
	deviceToken := "TSnuFRAAsXDknfcMbn7GZITJx5EMWyNfzCNuR1BdPymmgxcDm58Inzqw5x2v58lA"
	typ := model.UserType(1)
	token := parseSessionToken(id, username, deviceToken, typ)

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

func BenchmarkParseSessionToken(b *testing.B) {
	id := ulid.NewString()
	username := "username"
	deviceToken := "TSnuFRAAsXDknfcMbn7GZITJx5EMWyNfzCNuR1BdPymmgxcDm58Inzqw5x2v58lA"
	typ := model.UserType(2)
	for i := 0; i < b.N; i++ {
		parseSessionToken(id, username, deviceToken, typ)
	}
}

func BenchmarkUnparseSessionToken(b *testing.B) {
	id := ulid.NewString()
	username := "username"
	deviceToken := "TSnuFRAAsXDknfcMbn7GZITJx5EMWyNfzCNuR1BdPymmgxcDm58Inzqw5x2v58lA"
	typ := model.UserType(1)
	token := parseSessionToken(id, username, deviceToken, typ)

	for i := 0; i < b.N; i++ {
		unparseSessionToken(token)
	}
}
