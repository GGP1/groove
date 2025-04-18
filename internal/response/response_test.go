package response_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GGP1/groove/internal/httperr"
	"github.com/GGP1/groove/internal/response"
	"github.com/GGP1/groove/internal/ulid"
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
		expectedText := "{\"error\":\"test\",\"status\":404}\n"

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
		expectedText := "{\"error\":\"test\",\"status\":403}\n"

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
	// Requires dockertest to initialize redis.
	rdb := test.StartRedis(t)
	expectedHeader := "application/json; charset=UTF-8"
	expectedStatus := 200
	expectedRes := "\"test\"\n"
	key := "test_cache"
	value := "test"

	rec := httptest.NewRecorder()
	response.JSONAndCache(rdb, rec, key, value)

	assert.Equal(t, expectedHeader, rec.Header().Get("Content-Type"))

	assert.Equal(t, expectedStatus, rec.Code)

	var resContent bytes.Buffer
	_, err := resContent.ReadFrom(rec.Body)
	assert.NoError(t, err, "Failed reading response body")
	assert.Equal(t, expectedRes, resContent.String())

	v, err := rdb.Get(context.Background(), key).Bytes()
	assert.NoError(t, err)

	var cacheContent bytes.Buffer
	err = json.NewEncoder(&cacheContent).Encode(value)
	assert.NoError(t, err)
	assert.Equal(t, cacheContent.Bytes(), v)
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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		response.EncodedJSON(rec, buf.Bytes())
	}
}

func BenchmarkJSON(b *testing.B) {
	rec := httptest.NewRecorder()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		response.JSON(rec, 200, benchMessage)
	}
}

func BenchmarkJSONCount(b *testing.B) {
	rec := httptest.NewRecorder()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		response.JSONCount(rec, http.StatusOK, "test_count", 5)
	}
}

func BenchmarkJSONCursor(b *testing.B) {
	cursor := ulid.NewString()
	b.ResetTimer()
	rec := httptest.NewRecorder()
	for i := 0; i < b.N; i++ {
		response.JSONCursor(rec, cursor, "tests", benchMessage)
	}
}

func BenchmarkNoContent(b *testing.B) {
	rec := httptest.NewRecorder()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		response.NoContent(rec)
	}
}
