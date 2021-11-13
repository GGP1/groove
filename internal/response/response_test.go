package response_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GGP1/groove/internal/httperr"
	"github.com/GGP1/groove/internal/response"
	"github.com/GGP1/groove/test"

	"github.com/stretchr/testify/assert"
)

func TestEncodedJSON(t *testing.T) {
	expected := []byte("test")
	rec := httptest.NewRecorder()
	response.EncodedJSON(rec, []byte("test"))

	var buf bytes.Buffer
	_, err := buf.ReadFrom(rec.Body)
	assert.NoError(t, err)

	assert.Equal(t, expected, buf.Bytes())
}

func TestError(t *testing.T) {
	t.Run("Standard error", func(t *testing.T) {
		expectedHeaderCT := "application/json; charset=UTF-8"
		expectedStatus := 404
		expectedText := "{\"status\":404,\"error\":\"test\"}\n"

		rec := httptest.NewRecorder()
		response.Error(rec, http.StatusNotFound, errors.New("test"))

		assert.Equal(t, expectedHeaderCT, rec.Header().Get("Content-Type"))
		assert.Equal(t, expectedStatus, rec.Code)

		var buf bytes.Buffer
		_, err := buf.ReadFrom(rec.Body)
		assert.NoError(t, err, "Failed reading response body")

		assert.Equal(t, expectedText, buf.String())
	})

	t.Run("Custom error", func(t *testing.T) {
		expectedHeaderCT := "application/json; charset=UTF-8"
		expectedStatus := 403
		expectedText := "{\"status\":403,\"error\":\"test\"}\n"

		rec := httptest.NewRecorder()
		response.Error(rec, http.StatusInternalServerError, httperr.Forbidden("test"))

		assert.Equal(t, expectedHeaderCT, rec.Header().Get("Content-Type"))
		assert.Equal(t, expectedStatus, rec.Code)

		var buf bytes.Buffer
		_, err := buf.ReadFrom(rec.Body)
		assert.NoError(t, err, "Failed reading response body")

		assert.Equal(t, expectedText, buf.String())
	})
}

func TestJSON(t *testing.T) {
	expectedHeader := "application/json; charset=UTF-8"
	expectedStatus := 201
	expectedText := "\"test\"\n"

	rec := httptest.NewRecorder()
	response.JSON(rec, http.StatusCreated, "test")

	assert.Equal(t, expectedHeader, rec.Header().Get("Content-Type"))
	assert.Equal(t, expectedStatus, rec.Code)

	var buf bytes.Buffer
	_, err := buf.ReadFrom(rec.Body)
	assert.NoError(t, err, "Failed reading response body")

	assert.Equal(t, expectedText, buf.String())
}

func TestJSONAndCache(t *testing.T) {
	// Requires dockertest to initialize memcached.
	mc := test.StartMemcached(t)
	expectedHeader := "application/json; charset=UTF-8"
	expectedStatus := 200
	expectedRes := "\"test\"\n"
	key := "test_cache"
	value := "test"

	rec := httptest.NewRecorder()
	response.JSONAndCache(mc, rec, key, value)

	assert.Equal(t, expectedHeader, rec.Header().Get("Content-Type"))

	assert.Equal(t, expectedStatus, rec.Code)

	var resContent bytes.Buffer
	_, err := resContent.ReadFrom(rec.Body)
	assert.NoError(t, err, "Failed reading response body")
	assert.Equal(t, expectedRes, resContent.String())

	v, err := mc.Get(key)
	assert.NoError(t, err)

	var cacheContent bytes.Buffer
	err = json.NewEncoder(&cacheContent).Encode(value)
	assert.NoError(t, err)
	assert.Equal(t, cacheContent.Bytes(), v)
}

func TestJSONText(t *testing.T) {
	expectedHeader := "application/json; charset=UTF-8"
	expectedStatus := 200
	expectedRes := "{\"status\":200,\"message\":\"test\"}\n"

	rec := httptest.NewRecorder()
	response.JSONMessage(rec, http.StatusOK, "test")

	assert.Equal(t, expectedHeader, rec.Header().Get("Content-Type"))
	assert.Equal(t, expectedStatus, rec.Code)

	var buf bytes.Buffer
	_, err := buf.ReadFrom(rec.Body)
	assert.NoError(t, err, "Failed reading response body")

	assert.Equal(t, expectedRes, buf.String())
}

func TestNoContent(t *testing.T) {
	expectedStatus := 204
	rec := httptest.NewRecorder()

	response.NoContent(rec)
	assert.Equal(t, expectedStatus, rec.Code)
}

var benchMessage = struct {
	Name      string
	Username  string
	BirthDate time.Time
	Host      bool
}{
	Name:      "Benchmark Test",
	Username:  "__benchmark__",
	BirthDate: time.Unix(1515151, 15),
	Host:      false,
}

func BenchmarkEncodedJSON(b *testing.B) {
	rec := httptest.NewRecorder()
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(benchMessage)
	assert.NoError(b, err)

	for i := 0; i < b.N; i++ {
		response.EncodedJSON(rec, buf.Bytes())
	}
}

func BenchmarkJSON(b *testing.B) {
	rec := httptest.NewRecorder()
	for i := 0; i < b.N; i++ {
		response.JSON(rec, 200, benchMessage)
	}
}

func BenchmarkNoContent(b *testing.B) {
	rec := httptest.NewRecorder()
	for i := 0; i < b.N; i++ {
		response.NoContent(rec)
	}
}
