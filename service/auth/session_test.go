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

	parts := strings.Split(token, separatorStr)
	assert.Equal(t, parts[0], id)
	assert.Equal(t, parts[1], username)
	assert.Equal(t, parts[2], deviceToken)
	assert.Equal(t, parts[3], strconv.Itoa(int(typ)))
}

func TestUnparseSessionToken(t *testing.T) {
	cases := []struct {
		desc        string
		id          string
		username    string
		deviceToken string
		userType    model.UserType
		pass        bool
	}{
		{
			desc:        "Valid",
			id:          ulid.NewString(),
			username:    "username",
			deviceToken: "device_token",
			userType:    model.Business,
			pass:        true,
		},
		{
			desc:        "Invalid id",
			id:          "fehfuzloes",
			username:    "username",
			deviceToken: "device_token",
			userType:    model.Personal,
			pass:        false,
		},
		{
			desc:        "Invalid username",
			id:          ulid.NewString(),
			username:    "us/e/r/n/ame",
			deviceToken: "device_token",
			userType:    model.Personal,
			pass:        false,
		},
		{
			desc:        "Invalid length",
			id:          ulid.NewString(),
			username:    "76858810357044703314725763675081543890951673178550067181005282605223920506896937851805633135494218278849905411316972878553179396129112121509787996953257891070102827088434465853246797772932049554425851266831149804794777141397949601396529197467668991093474837220210377470858757297978007061693077653502482642920654757848315473951273287921077338549303662343532935921885013445957270342720139646808533609119285555945013924583809914594060947967344367922771084930614347100818686589389540000958300817375677292",
			deviceToken: "",
			userType:    model.Personal,
			pass:        false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			token := parseSessionToken(tc.id, tc.username, tc.deviceToken, tc.userType)
			session, err := unparseSessionToken(token)
			if tc.pass {
				assert.NoError(t, err)
				assert.Equal(t, session.ID, tc.id)
				assert.Equal(t, session.Username, tc.username)
				assert.Equal(t, session.DeviceToken, tc.deviceToken)
				assert.Equal(t, session.Type, tc.userType)
			} else {
				assert.Error(t, err)
			}
		})
	}
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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetSession(ctx, r)
	}
}

func BenchmarkParseSessionToken(b *testing.B) {
	id := ulid.NewString()
	username := "username"
	deviceToken := "TSnuFRAAsXDknfcMbn7GZITJx5EMWyNfzCNuR1BdPymmgxcDm58Inzqw5x2v58lA"
	typ := model.UserType(2)

	b.ResetTimer()
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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		unparseSessionToken(token)
	}
}
