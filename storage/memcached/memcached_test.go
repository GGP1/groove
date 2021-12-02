package memcached_test

import (
	"testing"

	"github.com/GGP1/groove/test"
)

func TestConnect(t *testing.T) {
	_ = test.StartMemcached(t)
}
