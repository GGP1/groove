package server

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/GGP1/groove/config"
	"github.com/GGP1/groove/internal/log"

	"github.com/pkg/errors"
	"golang.org/x/crypto/acme/autocert"
)

// Server contains the server configuration.
type Server struct {
	*http.Server

	letsEncrypt     letsEncrypt
	shutdownTimeout time.Duration
}

// letsEncrypt contains the configuration necessary for the Lets Encrypt service.
type letsEncrypt struct {
	enabled   bool
	acceptTOS bool
	cache     string
	hosts     []string
}

// New create and returns a server.
func New(cfg config.Server, router http.Handler) *Server {
	return &Server{
		Server: &http.Server{
			Addr:           net.JoinHostPort(cfg.Host, cfg.Port),
			Handler:        router,
			ReadTimeout:    cfg.Timeout.Read * time.Second,
			WriteTimeout:   cfg.Timeout.Write * time.Second,
			MaxHeaderBytes: 1 << 20,
			TLSConfig: &tls.Config{
				MinVersion:   tls.VersionTLS12,
				Certificates: cfg.TLSCertificates,
			},
		},
		letsEncrypt: letsEncrypt{
			enabled:   cfg.LetsEncrypt.Enabled,
			acceptTOS: cfg.LetsEncrypt.AcceptTOS,
			cache:     cfg.LetsEncrypt.Cache,
			hosts:     cfg.LetsEncrypt.Hosts,
		},

		shutdownTimeout: cfg.Timeout.Shutdown * time.Second,
	}
}

// Run starts the server.
func (srv *Server) Run(ctx context.Context) error {
	serverErr := make(chan error, 1)

	go srv.listenAndServe(serverErr)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		return errors.Wrap(err, "Listen and serve failed")

	case <-interrupt:
		log.Info("Start shutdown")

		// Give outstanding requests a deadline for completion
		ctx, cancel := context.WithTimeout(ctx, srv.shutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			return errors.Wrapf(err, "Graceful shutdown did not complete in %v", srv.shutdownTimeout)
		}

		if err := srv.Close(); err != nil {
			return errors.Wrap(err, "Couldn't stop server gracefully")
		}

		log.Info("Server shutdown gracefully")
		return nil
	}
}

func (srv *Server) listenAndServe(serverErr chan error) {
	if len(srv.TLSConfig.Certificates) < 1 {
		// Do not use TLS
		l, err := net.Listen("tcp", srv.Addr)
		if err != nil {
			serverErr <- err
			return
		}

		log.Sugar().Infof("[HTTP] Listening on http://" + srv.Addr)
		serverErr <- srv.Serve(l)
		return
	}

	if srv.letsEncrypt.enabled {
		// TODO: Test implementation
		certManager := autocert.Manager{
			Prompt:     func(tosURL string) bool { return srv.letsEncrypt.acceptTOS },
			HostPolicy: autocert.HostWhitelist(srv.letsEncrypt.hosts...),
			Cache:      autocert.DirCache(srv.letsEncrypt.cache),
		}

		srv.Handler = certManager.HTTPHandler(srv.Handler)
		srv.TLSConfig.GetCertificate = certManager.GetCertificate
	}

	l, err := tls.Listen("tcp", srv.Addr, srv.TLSConfig)
	if err != nil {
		serverErr <- err
		return
	}

	log.Sugar().Infof("[HTTPS] Listening on https://" + srv.Addr)
	serverErr <- srv.Serve(l)
}
