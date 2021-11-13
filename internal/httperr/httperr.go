// Package httperr provides errors to be returned inside service methods when
// the status should be other than InternalServerError.
package httperr

import (
	"fmt"
	"net/http"
)

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
func New(message string, status int) error {
	return &Err{
		msg:    message,
		status: int(status),
	}
}

// Errorf creates a formatted error.
func Errorf(status int, format string, args ...interface{}) error {
	return &Err{
		msg:    fmt.Sprintf(format, args...),
		status: int(status),
	}
}

// BadRequest returns a custom error that contains a status 400.
func BadRequest(message string) error {
	return &Err{
		msg:    message,
		status: http.StatusBadRequest,
	}
}

// Unauthorized returns a custom error that contains a status 401.
func Unauthorized(message string) error {
	return &Err{
		msg:    message,
		status: http.StatusUnauthorized,
	}
}

// Forbidden returns a custom error that contains a status 400.
func Forbidden(message string) error {
	return &Err{
		msg:    message,
		status: http.StatusForbidden,
	}
}
