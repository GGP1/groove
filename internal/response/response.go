package response

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/GGP1/groove/internal/bufferpool"
	"github.com/GGP1/groove/internal/httperr"
	redi "github.com/GGP1/groove/storage/redis"

	"github.com/go-redis/redis/v8"
)

// Performance optimizations:
//
// • w.Write(buf.Bytes()) is slightly faster than io.Copy(w, buf) as this last consumes some bytes until it reaches buf.WriteTo().
// • JSON Encoder.Encode uses ~80 less B/op, 1 less alloc/op and is ~8 ns/op faster than json.Marshal.
// • Setting headers manually is ~32% faster than using the Set method (which converts keys to MIME type).
// • Defer isn't used, the difference is almost not noticeable.

// Header keys must be canonical: the first letter and any letter
// following a hyphen must be upper case; the rest must be lowercase.
// For example, the canonical key for "accept-encoding" is "Accept-Encoding".
const contentType = "Content-Type"

var applicationJSON = []string{"application/json; charset=UTF-8"}

type errResponse struct {
	Error  string `json:"error"`
	Status int    `json:"status"`
}

// ID contains a unique identifier.
type ID struct {
	ID string `json:"id,omitempty"`
}

// Name contains a unique name.
type Name struct {
	Name string `json:"name,omitempty"`
}

// EncodedJSON writes a response from a buffer with json encoded content.
//
// The status is pre-defined as 200 (OK).
func EncodedJSON(w http.ResponseWriter, buf []byte) {
	w.Header()[contentType] = applicationJSON
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(buf); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Error is the function used to send error resposes.
func Error(w http.ResponseWriter, status int, err error) {
	// If the error contains a specific status, use it instead of the one provided.
	if e, ok := err.(*httperr.Err); ok {
		status = e.Status()
	}
	JSON(w, status, errResponse{
		Status: status,
		Error:  err.Error(),
	})
}

// JSON is the function used to send JSON responses.
func JSON(w http.ResponseWriter, status int, v any) {
	buf := bufferpool.Get()

	if err := json.NewEncoder(buf).Encode(v); err != nil {
		bufferpool.Put(buf)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// X-Content-Type-Options is already set in the secure middleware
	// Save a few bytes allocated by w.Header().Set() to convert header keys to a canonical format
	w.Header()[contentType] = applicationJSON
	w.WriteHeader(status)

	if _, err := w.Write(buf.Bytes()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	bufferpool.Put(buf)
}

// JSONAndCache works just like json but saves the encoding of v to the cache before writing the response.
//
// The status should always be 200 (OK). Usually, only single users and events will be cached.
func JSONAndCache(rdb *redis.Client, w http.ResponseWriter, key string, v any) {
	buf := bufferpool.Get()

	if err := json.NewEncoder(buf).Encode(v); err != nil {
		bufferpool.Put(buf)
		Error(w, http.StatusInternalServerError, err)
		return
	}

	// Copied here once as it's used twice, returning the buffer as soon as possible
	value := buf.Bytes()
	bufferpool.Put(buf)

	if err := rdb.Set(context.Background(), key, value, redi.ItemExpiration).Err(); err != nil {
		Error(w, http.StatusInternalServerError, err)
		return
	}

	w.Header()[contentType] = applicationJSON
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(value); err != nil {
		Error(w, http.StatusInternalServerError, err)
	}
}

// JSONCount sends a json encoded response with the status and a count.
func JSONCount(w http.ResponseWriter, status int, fieldName string, count any) {
	JSON(w, status, map[string]any{
		"status":  status,
		fieldName: count,
	})
}

// JSONCursor sends a json encoded response with the next cursor and items.
func JSONCursor(w http.ResponseWriter, nextCursor, fieldName string, items any) {
	JSON(w, http.StatusOK, map[string]any{
		"next_cursor": nextCursor,
		fieldName:     items,
	})
}

// NoContent writes a response with no content.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
