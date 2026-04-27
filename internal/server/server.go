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
	"github.com/CreateFutureMWilkinson/cue/internal/server/handler"
)

// Deps holds optional dependencies injected into the server.
// Nil fields disable the corresponding API surface.
type Deps struct {
	Messages handler.MessageQuerier
	// Buffer is the feedback buffer service for rating/dismissing buffered messages.
	// If nil, buffer API endpoints are not registered.
	Buffer handler.BufferRater
	// Todos is the todo/task service for CRUD operations.
	// If nil, task API endpoints are not registered.
	Todos handler.TodoServicer
	// EffectiveEstimate computes the effective estimate for a todo.
	// Required when Todos is non-nil.
	EffectiveEstimate handler.EffectiveEstimateFunc
	// Schedules is the schedule store for planner CRUD operations.
	// If nil, planner API endpoints are not registered.
	Schedules handler.ScheduleStore
	// ScheduleGenerator generates schedule options from calendar events.
	// Required when Schedules is non-nil.
	ScheduleGenerator handler.ScheduleGenerator
	// Calendar fetches calendar events for schedule generation.
	// Required when Schedules is non-nil.
	Calendar handler.CalendarFetcher
	// Hub is the WebSocket event broadcaster. If nil, New creates its own hub.
	// Inject a shared hub when you need external components (orchestrator, queue processor)
	// to publish events to the same WebSocket clients that connect to this server.
	Hub *Hub
}

// Server is the headless HTTP/WebSocket entry point for Cue.
// It wires repositories, services, and watchers and exposes them
// over HTTP handlers and a WebSocket event broadcaster.
type Server struct {
	cfg       config.ServerConfig
	deps      Deps
	hub       *Hub
	wsManager *handler.Manager
	mux       *http.ServeMux
	handler   http.Handler
	server    *http.Server

	mu       sync.Mutex
	listener net.Listener
}

// New creates a Server from the given config and optional dependencies.
// It does not start listening.
func New(cfg config.ServerConfig, deps ...Deps) (*Server, error) {
	mux := http.NewServeMux()

	var d Deps
	if len(deps) > 0 {
		d = deps[0]
	}

	hub := d.Hub
	if hub == nil {
		hub = NewHub()
	}

	s := &Server{
		cfg:       cfg,
		deps:      d,
		hub:       hub,
		wsManager: handler.NewManager(newHubPublisher(hub)),
		mux:       mux,
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
	s.mux.Handle("GET /api/v1/websocket/events", s.wsManager.Handler())
	s.mux.Handle("GET /api/v1/events", handler.EventsHandler(newHubPublisher(s.hub)))

	if s.deps.Messages != nil {
		s.mux.Handle("GET /api/v1/notifications", handler.ListNotificationsHandler(s.deps.Messages))
		s.mux.Handle("GET /api/v1/notifications/{id}", handler.GetNotificationHandler(s.deps.Messages))
		s.mux.Handle("POST /api/v1/notifications/{id}/resolve", handler.ResolveNotificationHandler(s.deps.Messages))
		s.mux.Handle("POST /api/v1/notifications/{id}/dismiss", handler.DismissNotificationHandler(s.deps.Messages))
		s.mux.Handle("GET /api/v1/messages", handler.ListMessagesHandler(s.deps.Messages))
		s.mux.Handle("GET /api/v1/messages/{id}", handler.GetMessageHandler(s.deps.Messages))

		s.mux.Handle("GET /api/v1/buffer", handler.ListBufferedHandler(s.deps.Messages))
		s.mux.Handle("GET /api/v1/buffer/stats", handler.BufferStatsHandler(s.deps.Messages))
		s.mux.Handle("GET /api/v1/buffer/{id}", handler.GetBufferedHandler(s.deps.Messages))

		if s.deps.Buffer != nil {
			s.mux.Handle("POST /api/v1/buffer/{id}/rate", handler.RateBufferedHandler(s.deps.Messages, s.deps.Buffer))
			s.mux.Handle("DELETE /api/v1/buffer/{id}", handler.DeleteBufferedHandler(s.deps.Messages, s.deps.Buffer))
		}
	}

	if s.deps.Todos != nil {
		s.mux.Handle("GET /api/v1/tasks", handler.ListTasksHandler(s.deps.Todos, s.deps.EffectiveEstimate))
		s.mux.Handle("POST /api/v1/tasks", handler.CreateTaskHandler(s.deps.Todos, s.deps.EffectiveEstimate))
		s.mux.Handle("GET /api/v1/tasks/{id}", handler.GetTaskHandler(s.deps.Todos, s.deps.EffectiveEstimate))
		s.mux.Handle("PUT /api/v1/tasks/{id}", handler.UpdateTaskHandler(s.deps.Todos, s.deps.EffectiveEstimate))
		s.mux.Handle("DELETE /api/v1/tasks/{id}", handler.DeleteTaskHandler(s.deps.Todos))
	}

	if s.deps.Schedules != nil {
		// Register /active routes before /{date} so ServeMux does not
		// match "active" as a {date} parameter.
		s.mux.Handle("GET /api/v1/planner/active", handler.ActiveDateHandler(handler.GetScheduleHandler(s.deps.Schedules)))
		s.mux.Handle("DELETE /api/v1/planner/active", handler.ActiveDateHandler(handler.DeleteScheduleHandler(s.deps.Schedules)))
		s.mux.Handle("GET /api/v1/planner/{date}", handler.GetScheduleHandler(s.deps.Schedules))
		s.mux.Handle("PUT /api/v1/planner/{date}", handler.PutScheduleHandler(s.deps.Schedules))
		s.mux.Handle("DELETE /api/v1/planner/{date}", handler.DeleteScheduleHandler(s.deps.Schedules))
		if s.deps.ScheduleGenerator != nil && s.deps.Calendar != nil {
			s.mux.Handle("POST /api/v1/planner/generate", handler.GenerateSchedulesHandler(s.deps.ScheduleGenerator, s.deps.Calendar))
		}
	}
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

// Shutdown performs an ordered shutdown: closes live WebSocket connections
// first (since net/http.Shutdown does not close hijacked connections),
// then stops accepting connections and drains in-flight requests.
func (s *Server) Shutdown(ctx context.Context) error {
	s.wsManager.CloseAll()
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
