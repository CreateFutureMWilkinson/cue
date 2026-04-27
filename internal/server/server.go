package server

import (
	"context"
	"errors"
	"net/http"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
)

// ErrNotImplemented is returned by stub functions that have not yet been implemented.
var ErrNotImplemented = errors.New("not implemented")

// Server is the headless HTTP/WebSocket entry point for Cue.
// It wires repositories, services, and watchers and exposes them
// over HTTP handlers and a WebSocket event broadcaster.
type Server struct {
	cfg    config.ServerConfig
	hub    *Hub
	mux    *http.ServeMux
	server *http.Server
}

// New creates a Server from the given config. It does not start listening.
func New(cfg config.ServerConfig) (*Server, error) {
	return nil, ErrNotImplemented
}

// Handler returns the root HTTP handler with all middleware applied.
func (s *Server) Handler() http.Handler {
	return nil
}

// Start begins listening on the configured host:port. It blocks until
// the server is shut down or an error occurs.
func (s *Server) Start() error {
	return ErrNotImplemented
}

// Shutdown performs an ordered shutdown: stops accepting connections,
// drains in-flight requests, and closes resources.
func (s *Server) Shutdown(ctx context.Context) error {
	return ErrNotImplemented
}

// Addr returns the address the server is listening on, or empty string
// if not yet started.
func (s *Server) Addr() string {
	return ""
}
