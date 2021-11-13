package httperr

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	message := "message error"
	status := http.StatusBadRequest
	err := New(message, status)

	assert.Equal(t, err.Error(), message)
	assert.Equal(t, err.(*Err).Status(), int(status))
}

func TestErrorf(t *testing.T) {
	status := http.StatusForbidden
	expectedMessage := "formatted error"

	err := Errorf(status, "formatted %s", "error")

	assert.Equal(t, err.Error(), expectedMessage)
	assert.Equal(t, err.(*Err).Status(), int(status))
}
