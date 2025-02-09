package romap_test

import (
	"testing"

	"github.com/GGP1/groove/internal/romap"

	"github.com/stretchr/testify/assert"
)

func TestRomap(t *testing.T) {
	key := "test"
	value := 10
	mp := map[string]int{
		key: value,
	}
	roMap := romap.New(mp)

	t.Run("Map", func(t *testing.T) {
		assert.Equal(t, mp, roMap.Map())
	})
	t.Run("Keys", func(t *testing.T) {
		gotKeys := roMap.Keys()
		expectedKeys := []string{key}
		assert.Equal(t, expectedKeys, gotKeys)
	})
	t.Run("Exists", func(t *testing.T) {
		assert.True(t, roMap.Exists(key))
		assert.False(t, roMap.Exists("non-existent"))
	})
	t.Run("Get", func(t *testing.T) {
		got, ok := roMap.Get(key)
		assert.True(t, ok)
		assert.Equal(t, value, got)
		var expectedType int
		assert.IsType(t, expectedType, value)
	})
}
