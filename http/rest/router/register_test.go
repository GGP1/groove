package router

import (
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
)

func TestRegisterEndpoints(t *testing.T) {
	// Just make sure the endpoints do not overlap (radix sort is used) and cause a panic
	r := register{
		router: &Router{
			Router: httprouter.New(),
		},
	}
	assert.NotPanics(t, func() {
		r.All()
	})
}
