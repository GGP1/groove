package response

import (
	"encoding/json"
	"net/http"

	"github.com/GGP1/groove/internal/bufferpool"

	"github.com/bradfitz/gomemcache/memcache"
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

type countResponse struct {
	Status int     `json:"status,omitempty"`
	Count  *uint64 `json:"count,omitempty"`
}

type errResponse struct {
	Status int    `json:"status"`
	Err    string `json:"error"`
}

type msgResponse struct {
	Status  int         `json:"status"`
	Message interface{} `json:"message"`
}

// EncodedJSON writes a response from a buffer with json encoded content.
//
// The status is predefined as 200 (OK).
func EncodedJSON(w http.ResponseWriter, buf []byte) {
	w.Header()[contentType] = []string{"application/json; charset=UTF-8"}
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(buf); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Error is the function used to send error resposes.
func Error(w http.ResponseWriter, status int, err error) {
	JSON(w, status, errResponse{
		Status: status,
		Err:    err.Error(),
	})
}

// JSON is the function used to send JSON responses.
func JSON(w http.ResponseWriter, status int, v interface{}) {
	buf := bufferpool.Get()

	if err := json.NewEncoder(buf).Encode(v); err != nil {
		bufferpool.Put(buf)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// X-Content-Type-Options is already set in the secure middleware
	// Save a few bytes allocated by w.Header().Set() to convert header keys to a canonical format
	w.Header()[contentType] = []string{"application/json; charset=UTF-8"}
	w.WriteHeader(status)

	if _, err := w.Write(buf.Bytes()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	bufferpool.Put(buf)
}

// JSONAndCache works just like json but saves the encoding of v to the cache before writing the response.
//
// The status should always be 200 (OK). Usually, only single users and events will be cached.
func JSONAndCache(mc *memcache.Client, w http.ResponseWriter, key string, v interface{}) {
	buf := bufferpool.Get()

	if err := json.NewEncoder(buf).Encode(v); err != nil {
		bufferpool.Put(buf)
		Error(w, http.StatusInternalServerError, err)
		return
	}

	value := buf.Bytes()
	bufferpool.Put(buf)

	if err := mc.Set(&memcache.Item{Key: key, Value: value}); err != nil {
		Error(w, http.StatusInternalServerError, err)
		return
	}

	w.Header()[contentType] = []string{"application/json; charset=UTF-8"}
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(value); err != nil {
		Error(w, http.StatusInternalServerError, err)
	}
}

// JSONCount sends a json encoded response with the status and a count.
func JSONCount(w http.ResponseWriter, status int, count *uint64) {
	JSON(w, status, countResponse{
		Status: status,
		Count:  count,
	})
}

// JSONMessage is the function used to send JSON formatted message responses.
func JSONMessage(w http.ResponseWriter, status int, message interface{}) {
	JSON(w, status, msgResponse{
		Status:  status,
		Message: message,
	})
}
