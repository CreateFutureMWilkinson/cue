package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
)

// Server is the headless HTTP/WebSocket entry point for Cue.
// It wires repositories, services, and watchers and exposes them
// over HTTP handlers and a WebSocket event broadcaster.
type Server struct {
	cfg     config.ServerConfig
	hub     *Hub
	mux     *http.ServeMux
	handler http.Handler
	server  *http.Server

	mu       sync.Mutex
	listener net.Listener
}

// New creates a Server from the given config. It does not start listening.
func New(cfg config.ServerConfig) (*Server, error) {
	mux := http.NewServeMux()
	hub := NewHub()

	s := &Server{
		cfg: cfg,
		hub: hub,
		mux: mux,
	}

	s.registerRoutes()

	s.handler = chain(mux,
		ContentTypeMiddleware,
		CORSMiddleware,
		RequestIDMiddleware,
		LoggingMiddleware,
		RecoveryMiddleware,
	)

	s.server = &http.Server{
		Handler:      s.handler,
		ReadTimeout:  time.Duration(cfg.ReadTimeoutSeconds) * time.Second,
		WriteTimeout: time.Duration(cfg.WriteTimeoutSeconds) * time.Second,
	}
	return s, nil
}

// registerRoutes mounts the v1 API. Health endpoints are exposed both
// at /health (unversioned, conventional) and under /api/v1/health for
// API consumers.
func (s *Server) registerRoutes() {
	s.mux.Handle("GET /health", HealthHandler())
	s.mux.Handle("GET /health/ready", ReadyHandler())
	s.mux.Handle("GET /api/v1/health", HealthHandler())
	s.mux.Handle("GET /api/v1/health/ready", ReadyHandler())
}

// Hub returns the WebSocket event broadcaster so callers can publish
// events into the server.
func (s *Server) Hub() *Hub { return s.hub }

// Handler returns the root HTTP handler with all middleware applied.
func (s *Server) Handler() http.Handler {
	return s.handler
}

// Start begins listening on the configured host:port. It blocks until
// the server is shut down or an error occurs.
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}

	s.mu.Lock()
	s.listener = ln
	s.mu.Unlock()

	if err := s.server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown performs an ordered shutdown: stops accepting connections,
// drains in-flight requests, and closes resources.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// Addr returns the address the server is listening on, or empty string
// if not yet started.
func (s *Server) Addr() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

// chain wraps h with the given middleware. The first middleware in the
// list runs first (outermost).
func chain(h http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}
