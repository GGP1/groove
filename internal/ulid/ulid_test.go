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

func TestValidate(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		id := NewString()
		err := Validate(id)
		assert.NoError(t, err)
	})

	t.Run("Invalid size", func(t *testing.T) {
		err := Validate("123")
		assert.Error(t, err)
	})

	t.Run("Invalid first character", func(t *testing.T) {
		err := Validate("81FATYTQYMDPJFEJTGC4SHXA27")
		assert.Error(t, err)
	})

	t.Run("Invalid characters", func(t *testing.T) {
		err := Validate("01FATYTQYMDPJFEJTGC4SHXA2I")
		assert.Error(t, err)
	})
}

func TestValidateN(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		ids := []string{NewString(), NewString(), NewString()}
		err := ValidateN(ids...)
		assert.NoError(t, err)
	})

	t.Run("Invalid", func(t *testing.T) {
		ids := []string{NewString(), "123", NewString()}
		err := ValidateN(ids...)
		assert.Error(t, err)
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

func BenchmarkValidate(b *testing.B) {
	id := NewString()
	b.SetBytes(EncodedSize)
	for i := 0; i < b.N; i++ {
		Validate(id)
	}
}
