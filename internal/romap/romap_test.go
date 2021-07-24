package romap_test

import (
	"testing"

	"github.com/GGP1/groove/internal/romap"

	"github.com/stretchr/testify/assert"
)

func TestRomap(t *testing.T) {
	key := "test"
	value := []string{"cheers"}
	panicKey := "panic"
	mp := map[string]interface{}{
		key:      value,
		panicKey: 0,
	}
	roMap := romap.New(mp)

	t.Run("Map", func(t *testing.T) {
		assert.Equal(t, mp, roMap.Map())
	})
	t.Run("Keys", func(t *testing.T) {
		assert.Equal(t, []string{key, panicKey}, roMap.Keys())
	})
	t.Run("Exists", func(t *testing.T) {
		assert.True(t, roMap.Exists(key))
		assert.False(t, roMap.Exists("non-existent"))
	})
	t.Run("Get", func(t *testing.T) {
		got, ok := roMap.Get(key)
		assert.True(t, ok)
		assert.Equal(t, value, got)
	})
	t.Run("GetStringSlice", func(t *testing.T) {
		strStructMp, ok := roMap.GetStringSlice(key)
		assert.True(t, ok)
		assert.Equal(t, value, strStructMp)

		t.Run("Nil", func(t *testing.T) {
			nilValue, ok := roMap.GetStringSlice("non-existent")
			assert.False(t, ok)
			assert.Nil(t, nilValue)
		})
	})
	t.Run("Panic", func(t *testing.T) {
		assert.Panics(t, func() {
			roMap.GetStringSlice(panicKey)
		})
	})
}
