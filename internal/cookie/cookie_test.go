package cookie

import (
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GGP1/groove/internal/crypt"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestDelete(t *testing.T) {
	name := "test-delete"

	w := httptest.NewRecorder()
	http.SetCookie(w, &http.Cookie{
		Name:  name,
		Value: "groove",
		Path:  "/",
	})

	Delete(w, name)
	// The recorder does not delete the cookies but stores another header with the same name
	// but different values, check if this last still holds the value
	cookies := w.Result().Cookies()
	assert.Equal(t, "", cookies[len(cookies)-1].Value)
}

func TestGet(t *testing.T) {
	expected := "groove"
	ciphertext, err := crypt.Encrypt([]byte(expected))
	assert.NoError(t, err)

	name := "test-get"
	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{
		Name:  name,
		Value: hex.EncodeToString(ciphertext),
		Path:  "/",
	})

	got, err := Get(r, name)
	assert.NoError(t, err)

	assert.Equal(t, expected, got.Value)
}

func TestGetValue(t *testing.T) {
	expected := "groove"
	ciphertext, err := crypt.Encrypt([]byte(expected))
	assert.NoError(t, err)

	name := "test-get"
	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{
		Name:  name,
		Value: hex.EncodeToString(ciphertext),
		Path:  "/",
	})

	got, err := GetValue(r, name)
	assert.NoError(t, err)

	assert.Equal(t, expected, got)
}

func TestGetErrors(t *testing.T) {
	t.Run("Cookie isn't set", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		_, err := Get(r, "invalid")
		assert.Error(t, err)
	})

	t.Run("Invalid hex value", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		r.AddCookie(&http.Cookie{
			Name:  "test",
			Value: "fail",
			Path:  "/",
		})

		_, err := Get(r, "test")
		assert.Error(t, err)
	})
}

func TestIsSet(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)

	t.Run("Not set", func(t *testing.T) {
		assert.False(t, IsSet(r, "not-set"))
	})

	t.Run("Set", func(t *testing.T) {
		r.AddCookie(&http.Cookie{
			Name:  "set",
			Value: "test",
			Path:  "/",
		})

		assert.True(t, IsSet(r, "set"))
	})
}

func TestSet(t *testing.T) {
	viper.Reset()
	viper.Set("secrets.encryption", "test")

	w := httptest.NewRecorder()
	name := "test-set"
	value := "groove"
	path := "/"
	err := Set(w, name, value, path)
	assert.NoError(t, err)

	c := w.Result().Cookies()[0]

	assert.Equal(t, c.Name, name)
	assert.Equal(t, c.Path, path)
	assert.Equal(t, c.MaxAge, maxAge)
	assert.Equal(t, c.HttpOnly, true)
	assert.Equal(t, c.Secure, false)
	assert.Equal(t, http.SameSiteLaxMode, c.SameSite)
}

func TestSetHost(t *testing.T) {
	viper.Reset()
	viper.Set("secrets.encryption", "test")

	w := httptest.NewRecorder()
	name := "test-set"
	value := "groove"
	err := SetHost(w, name, value)
	assert.NoError(t, err)

	c := w.Result().Cookies()[0]

	assert.Equal(t, c.Name, "__Host-"+name)
	assert.Equal(t, c.Path, "/")
	assert.Equal(t, c.MaxAge, maxAge*2)
	assert.True(t, c.Secure)
	assert.True(t, c.HttpOnly)
	assert.Equal(t, http.SameSiteStrictMode, c.SameSite)
}

func BenchmarkGet(b *testing.B) {
	ciphertext, err := crypt.Encrypt([]byte("test"))
	assert.NoError(b, err)
	name := "bench"
	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{
		Name:  name,
		Value: hex.EncodeToString(ciphertext),
		Path:  "/",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Get(r, name)
	}
}

func BenchmarkSet(b *testing.B) {
	w := httptest.NewRecorder()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Set(w, "bench", "@*s6%C>USkyaip8~ I7/P_!jAl&JZ45W", "/")
	}
}

func BenchmarkSetHost(b *testing.B) {
	w := httptest.NewRecorder()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SetHost(w, "bench", "@*s6%C>USkyaip8~ I7/P_!jAl&JZ45W")
	}

}
