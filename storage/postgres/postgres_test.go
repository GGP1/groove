package postgres_test

import (
	"testing"

	"github.com/GGP1/groove/test"
)

func TestConnect(t *testing.T) {
	test.StartPostgres(t)
}
