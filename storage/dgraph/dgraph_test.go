package dgraph_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/GGP1/groove/config"
	"github.com/GGP1/groove/storage/dgraph"
	"github.com/GGP1/groove/test"

	"github.com/stretchr/testify/assert"
)

func TestConnect(t *testing.T) {
	pool, resource := test.NewResource(t, "dgraph/standalone", "v21.03.0", nil)

	err := pool.Retry(func() error {
		port, _ := strconv.Atoi(resource.GetPort("9080/tcp"))
		_, closeConn, err := dgraph.Connect(context.Background(), config.Dgraph{
			Host: "localhost",
			Port: port,
		})
		if err != nil {
			return err
		}
		return closeConn()
	})
	assert.NoError(t, err)
}
