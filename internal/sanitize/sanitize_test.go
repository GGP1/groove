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

func TestStrings(t *testing.T) {
	operaHouse := "opera house "
	dance := "dánce"
	groove := " groove "
	expOperaHouse := "opera house"
	expDance := "dance"
	expGroove := "groove"
	Strings(&operaHouse, &dance, &groove)

	assert.Equal(t, expOperaHouse, operaHouse)
	assert.Equal(t, expDance, dance)
	assert.Equal(t, expGroove, groove)
}

func BenchmarkNormalize(b *testing.B) {
	str := "BénçhmẬrkstrïng" // Maybe it requires a little more research to estimate a good average input
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		str = Normalize(str)
	}
}

func BenchmarkDirtyStrings(b *testing.B) {
	operaHouse := "opera house "
	dance := "dánce"
	groove := " groove "
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Strings(&operaHouse, &dance, &groove)
	}
}

func BenchmarkCleanStrings(b *testing.B) {
	operaHouse := "opera house"
	dance := "dance"
	groove := "groove"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Strings(&operaHouse, &dance, &groove)
	}
}
