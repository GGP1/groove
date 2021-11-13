package server_test

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/GGP1/groove/config"
	"github.com/GGP1/groove/server"
	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	cfg := config.Server{
		Host: "localhost",
		Port: "7654",
	}

	mux := http.DefaultServeMux
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "test")
	})

	srv := server.New(cfg, mux)
	ctx := context.Background()
	go srv.Run(ctx)
	// Wait for the server to start
	time.Sleep(20 * time.Millisecond)

	url := fmt.Sprintf("http://%s/", net.JoinHostPort(cfg.Host, cfg.Port))
	resp, err := http.Get(url)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	data, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "test\n", string(data))

	err = srv.Shutdown(ctx)
	assert.NoError(t, err)

	err = srv.Close()
	assert.NoError(t, err)
}
