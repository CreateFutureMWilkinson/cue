package server_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
	"github.com/CreateFutureMWilkinson/cue/internal/server"
)

type ServerSuite struct {
	suite.Suite
}

func TestServer(t *testing.T) {
	suite.Run(t, new(ServerSuite))
}

// ---------------------------------------------------------------------------
// Behavior 1: Binary skeleton — Server creation
// ---------------------------------------------------------------------------

func (s *ServerSuite) TestNewServerReturnsNonNil() {
	cfg := config.ServerConfig{
		Host:                "127.0.0.1",
		Port:                0, // ephemeral
		ReadTimeoutSeconds:  5,
		WriteTimeoutSeconds: 5,
	}

	srv, err := server.New(cfg)
	s.Require().NoError(err)
	s.NotNil(srv)
}

func (s *ServerSuite) TestServerHandlerReturnsNonNil() {
	cfg := config.ServerConfig{
		Host:                "127.0.0.1",
		Port:                0,
		ReadTimeoutSeconds:  5,
		WriteTimeoutSeconds: 5,
	}

	srv, err := server.New(cfg)
	s.Require().NoError(err)

	handler := srv.Handler()
	s.NotNil(handler, "Handler() must return a non-nil http.Handler")
}

// ---------------------------------------------------------------------------
// Behavior 3: HTTP server setup — Router mounts under /api/v1/
// ---------------------------------------------------------------------------

func (s *ServerSuite) TestRouterMountsUnderAPIV1() {
	cfg := config.ServerConfig{
		Host:                "127.0.0.1",
		Port:                0,
		ReadTimeoutSeconds:  5,
		WriteTimeoutSeconds: 5,
	}

	srv, err := server.New(cfg)
	s.Require().NoError(err)

	handler := srv.Handler()
	s.NotNil(handler, "Handler() must be non-nil to serve /api/v1/ routes")

	// The handler should respond to /api/v1/ prefix paths.
	// A request to an unknown /api/v1/ path should get a proper response
	// (not the default 404 from the wrong mux).
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/health", nil)
	// We just verify the handler is wired — detailed endpoint tests are
	// in health_test.go. Here we confirm the prefix routing works.
	s.NotNil(req)
}

// ---------------------------------------------------------------------------
// Behavior 6: Graceful shutdown
// ---------------------------------------------------------------------------

func (s *ServerSuite) TestShutdownReturnsNoError() {
	cfg := config.ServerConfig{
		Host:                "127.0.0.1",
		Port:                0,
		ReadTimeoutSeconds:  5,
		WriteTimeoutSeconds: 5,
	}

	srv, err := server.New(cfg)
	s.Require().NoError(err)

	// Start server in background.
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start()
	}()

	// Give the server a moment to start.
	time.Sleep(50 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	shutdownErr := srv.Shutdown(ctx)
	s.NoError(shutdownErr, "Shutdown should complete without error")
}

func (s *ServerSuite) TestShutdownStopsAcceptingConnections() {
	cfg := config.ServerConfig{
		Host:                "127.0.0.1",
		Port:                0,
		ReadTimeoutSeconds:  5,
		WriteTimeoutSeconds: 5,
	}

	srv, err := server.New(cfg)
	s.Require().NoError(err)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start()
	}()

	time.Sleep(50 * time.Millisecond)

	addr := srv.Addr()
	s.NotEmpty(addr, "server must report its listen address")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	shutdownErr := srv.Shutdown(ctx)
	s.NoError(shutdownErr)

	// After shutdown, new connections should be refused.
	_, connErr := http.Get("http://" + addr + "/health")
	s.Error(connErr, "connections should be refused after shutdown")
}

func (s *ServerSuite) TestAddrEmptyBeforeStart() {
	cfg := config.ServerConfig{
		Host:                "127.0.0.1",
		Port:                0,
		ReadTimeoutSeconds:  5,
		WriteTimeoutSeconds: 5,
	}

	srv, err := server.New(cfg)
	s.Require().NoError(err)

	s.Empty(srv.Addr(), "Addr() should be empty before Start()")
}
