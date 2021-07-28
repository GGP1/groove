package sanitize

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalize(t *testing.T) {
	cases := []struct {
		in       string
		expected string
	}{
		{in: "Dança", expected: "Danca"},
		{in: "Çomer", expected: "Comer"},
		{in: "úser", expected: "user"},
		{in: "ïd", expected: "id"},
		{in: "nÀmệ", expected: "nAme"},
	}

	for i, tc := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			got := Normalize(tc.in)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestUserInput(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		query := "searching for values"
		err := UserInput(query)
		assert.NoError(t, err)
	})
	t.Run("Invalid", func(t *testing.T) {
		query := "'; DROP TABLE users; --"
		err := UserInput(query)
		assert.Error(t, err)
	})
}

func BenchmarkNormalize(b *testing.B) {
	str := "BénçhmẬrkstrïng" // Maybe it requires a little more research to estimate a good average input
	for i := 0; i < b.N; i++ {
		str = Normalize(str)
	}
}
