package ulid

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	assert.NotPanics(t, func() {
		New()
	})
}

func TestNewString(t *testing.T) {
	assert.NotPanics(t, func() {
		NewString()
	})
}

func BenchmarkNew(b *testing.B) {
	b.SetBytes(16)
	for i := 0; i < b.N; i++ {
		New()
	}
}

func BenchmarkNewString(b *testing.B) {
	b.SetBytes(EncodedSize)
	for i := 0; i < b.N; i++ {
		NewString()
	}
}
