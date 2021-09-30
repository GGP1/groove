// Package httperr provides errors to be returned inside service methods when
// the status should be other than InternalServerError.
package httperr

import (
	"fmt"
	"net/http"
)

// TODO: maybe it's better to use error types and use status based on those types (type errBanned -> Forbidden, type errLogin -> Unauthorized)
// Status code
const (
	BadRequest   status = http.StatusBadRequest
	Unauthorized status = http.StatusUnauthorized
	Forbidden    status = http.StatusForbidden
)

type status int

// Err represents an error.
type Err struct {
	msg    string
	status int
}

func (e *Err) Error() string {
	return e.msg
}

// Status returns the error's status code.
func (e *Err) Status() int {
	return e.status
}

// New returns a new error containing an status code.
func New(message string, status status) error {
	return &Err{
		msg:    message,
		status: int(status),
	}
}

// Errorf creates a formatted error.
func Errorf(status status, format string, args ...interface{}) error {
	return &Err{
		msg:    fmt.Sprintf(format, args...),
		status: int(status),
	}
}
