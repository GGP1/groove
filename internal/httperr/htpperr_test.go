package httperr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	message := "message error"
	status := BadRequest
	err := New(message, status)

	assert.Equal(t, err.Error(), message)
	assert.Equal(t, err.(*Err).Status(), int(status))
}

func TestErrorf(t *testing.T) {
	status := Forbidden
	expectedMessage := "formatted error"

	err := Errorf(status, "formatted %s", "error")

	assert.Equal(t, err.Error(), expectedMessage)
	assert.Equal(t, err.(*Err).Status(), int(status))
}
