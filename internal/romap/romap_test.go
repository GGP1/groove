package romap_test

import (
	"testing"

	"github.com/GGP1/groove/internal/romap"

	"github.com/stretchr/testify/assert"
)

func TestRomap(t *testing.T) {
	key := "test"
	value := map[string]struct{}{
		"cheers": {},
	}
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
	t.Run("GetStringMapStruct", func(t *testing.T) {
		strStructMp, ok2 := roMap.GetStringMapStruct(key)
		assert.True(t, ok2)
		assert.Equal(t, value, strStructMp)
	})
	t.Run("Panic", func(t *testing.T) {
		assert.Panics(t, func() {
			roMap.GetStringMapStruct(panicKey)
		})
	})
}
