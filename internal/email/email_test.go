package email

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValid(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		ok := IsValid("testing_email_regexp@test.com")
		assert.True(t, ok)
	})

	t.Run("Invalid", func(t *testing.T) {
		emails := []string{"testing_email_regexptest.com", "@test.com", "%·$·@#2t.com", "invalid_email@test"}

		for i, email := range emails {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				ok := IsValid(email)
				assert.False(t, ok)
			})
		}
	})
}

func BenchmarkIsValid(b *testing.B) {
	email := "testing_email_regexp@test.com"
	for i := 0; i < b.N; i++ {
		IsValid(email)
	}
}
