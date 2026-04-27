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
	// Rules is the routing-rules manager for CRUD operations.
	// If nil, routing-rule API endpoints are not registered.
	Rules handler.RulesManager
	// Services is the service configuration manager for CRUD operations.
	// If nil, service configuration API endpoints are not registered.
	Services handler.ServiceManager
	// Hub is the WebSocket event broadcaster. If nil, New creates its own hub.
	// Inject a shared hub when you need external components (orchestrator, queue processor)
	// to publish events to the same WebSocket clients that connect to this server.
	Hub *Hub
	// AuthTokens is the auth token repository for TOFU authentication.
	// If nil and auth is enabled, the middleware cannot function.
	AuthTokens AuthTokenLookup
	// AuthTokenManager provides CRUD operations for token management endpoints.
	// If nil, token management endpoints are not registered.
	AuthTokenManager handler.AuthTokenManager
}

// Server is the headless HTTP/WebSocket entry point for Cue.
// It wires repositories, services, and watchers and exposes them
// over HTTP handlers and a WebSocket event broadcaster.
type Server struct {
	cfg          config.ServerConfig
	deps         Deps
	hub          *Hub
	ticker       *Ticker
	pairingStore *pairingStoreAdapter
	wsManager    *handler.Manager
	mux          *http.ServeMux
	handler      http.Handler
	server       *http.Server

	mu       sync.Mutex
	listener net.Listener
}

// @title           Cue API
// @version         1.0
// @description     Cue is a local-first, privacy-centric productivity assistant.
// @description     This REST API exposes messages, notifications, feedback buffer,
// @description     tasks, day planner, routing rules, service configuration, timer,
// @description     auth/pairing, and event replay. The WebSocket event channel at
// @description     /api/v1/websocket/events is documented separately in
// @description     docs/api/websocket.md.
// @externalDocs.description  WebSocket event channel reference
// @externalDocs.url          https://github.com/CreateFutureMWilkinson/cue/src/branch/main/docs/api/websocket.md
//
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

	pub := newHubPublisher(hub)
	wsManager := handler.NewManager(pub)

	// Set up WebSocket token validation when auth is enabled.
	if cfg.AuthEnabled && d.AuthTokens != nil {
		wsManager.SetTokenValidator(func(token string) bool {
			h := hashToken(token)
			t, err := d.AuthTokens.LookupByHash(context.Background(), h)
			return err == nil && !t.Revoked
		})
	}

	var ps *pairingStoreAdapter
	if cfg.AuthEnabled {
		ps = newPairingStoreAdapter(NewPairingStore())
	}

	s := &Server{
		cfg:          cfg,
		deps:         d,
		hub:          hub,
		pairingStore: ps,
		wsManager:    wsManager,
		mux:          mux,
	}

	s.registerRoutes()

	middlewares := []func(http.Handler) http.Handler{
		ContentTypeMiddleware,
		CORSMiddleware,
		RequestIDMiddleware,
		LoggingMiddleware,
		RecoveryMiddleware,
	}
	if d.AuthTokens != nil {
		middlewares = append([]func(http.Handler) http.Handler{
			AuthMiddleware(d.AuthTokens, cfg.AuthEnabled),
		}, middlewares...)
	}
	s.handler = chain(mux, middlewares...)

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

	s.registerDocsRoutes()

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

	if s.deps.Rules != nil {
		s.mux.Handle("GET /api/v1/rules", handler.ListRulesHandler(s.deps.Rules))
		s.mux.Handle("GET /api/v1/rules/{id}", handler.GetRuleHandler(s.deps.Rules))
		s.mux.Handle("POST /api/v1/rules", handler.CreateRuleHandler(s.deps.Rules))
		s.mux.Handle("PUT /api/v1/rules/{id}", handler.UpdateRuleHandler(s.deps.Rules))
		s.mux.Handle("PATCH /api/v1/rules/{id}", handler.PatchRuleHandler(s.deps.Rules))
		s.mux.Handle("DELETE /api/v1/rules/{id}", handler.DeleteRuleHandler(s.deps.Rules))
	}

	if s.deps.Services != nil {
		// Slack
		s.mux.Handle("GET /api/v1/services/slack", handler.ListSlackAccountsHandler(s.deps.Services))
		s.mux.Handle("GET /api/v1/services/slack/{id}", handler.GetSlackAccountHandler(s.deps.Services))
		s.mux.Handle("POST /api/v1/services/slack", handler.CreateSlackAccountHandler(s.deps.Services))
		s.mux.Handle("PUT /api/v1/services/slack/{id}", handler.UpdateSlackAccountHandler(s.deps.Services))
		s.mux.Handle("DELETE /api/v1/services/slack/{id}", handler.DeleteSlackAccountHandler(s.deps.Services))
		s.mux.Handle("POST /api/v1/services/slack/{id}/toggle", handler.ToggleSlackAccountHandler(s.deps.Services))
		// Email
		s.mux.Handle("GET /api/v1/services/email", handler.ListEmailAccountsHandler(s.deps.Services))
		s.mux.Handle("GET /api/v1/services/email/{id}", handler.GetEmailAccountHandler(s.deps.Services))
		s.mux.Handle("POST /api/v1/services/email", handler.CreateEmailAccountHandler(s.deps.Services))
		s.mux.Handle("PUT /api/v1/services/email/{id}", handler.UpdateEmailAccountHandler(s.deps.Services))
		s.mux.Handle("DELETE /api/v1/services/email/{id}", handler.DeleteEmailAccountHandler(s.deps.Services))
		s.mux.Handle("POST /api/v1/services/email/{id}/toggle", handler.ToggleEmailAccountHandler(s.deps.Services))
		// Calendar
		s.mux.Handle("GET /api/v1/services/calendar", handler.ListCalendarAccountsHandler(s.deps.Services))
		s.mux.Handle("GET /api/v1/services/calendar/{id}", handler.GetCalendarAccountHandler(s.deps.Services))
		s.mux.Handle("POST /api/v1/services/calendar", handler.CreateCalendarAccountHandler(s.deps.Services))
		s.mux.Handle("PUT /api/v1/services/calendar/{id}", handler.UpdateCalendarAccountHandler(s.deps.Services))
		s.mux.Handle("DELETE /api/v1/services/calendar/{id}", handler.DeleteCalendarAccountHandler(s.deps.Services))
		s.mux.Handle("POST /api/v1/services/calendar/{id}/toggle", handler.ToggleCalendarAccountHandler(s.deps.Services))
		// Status
		s.mux.Handle("GET /api/v1/services/status", handler.ServiceStatusHandler(s.deps.Services))
	}

	if s.pairingStore != nil {
		pub := newHubPublisher(s.hub)
		s.mux.Handle("POST /api/v1/auth/pair", handler.InitiatePairingHandler(s.pairingStore, pub))
		s.mux.Handle("GET /api/v1/auth/pair/{id}", handler.PollPairingHandler(s.pairingStore))
		s.mux.Handle("POST /api/v1/auth/pair/{id}/approve", handler.ApprovePairingHandler(s.pairingStore, s.deps.AuthTokenManager, pub))
		s.mux.Handle("POST /api/v1/auth/pair/{id}/deny", handler.DenyPairingHandler(s.pairingStore, pub))
	}

	if s.deps.AuthTokenManager != nil {
		s.mux.Handle("GET /api/v1/auth/tokens", handler.ListTokensHandler(s.deps.AuthTokenManager))
		s.mux.Handle("PUT /api/v1/auth/tokens/{id}", handler.UpdateTokenLabelHandler(s.deps.AuthTokenManager))
		s.mux.Handle("DELETE /api/v1/auth/tokens/{id}", handler.RevokeTokenHandler(s.deps.AuthTokenManager))
	}

	if s.deps.Schedules != nil {
		// Timer endpoint (read-only, no onChange needed).
		s.mux.Handle("GET /api/v1/timer", handler.GetTimerHandler(s.deps.Schedules, wallClockTimerAdapter{}))

		// Build onChange callbacks for schedule mutations.
		// The closure captures s and checks s.ticker at call time,
		// so the ticker can be attached after server construction.
		onChange := []func(){func() {
			if s.ticker != nil {
				s.ticker.NotifyScheduleChanged(context.Background())
			}
		}}

		// Register /active routes before /{date} so ServeMux does not
		// match "active" as a {date} parameter.
		s.mux.Handle("GET /api/v1/planner/active", handler.ActiveDateHandler(handler.GetScheduleHandler(s.deps.Schedules)))
		s.mux.Handle("DELETE /api/v1/planner/active", handler.ActiveDateHandler(handler.DeleteScheduleHandler(s.deps.Schedules, onChange...)))
		s.mux.Handle("GET /api/v1/planner/{date}", handler.GetScheduleHandler(s.deps.Schedules))
		s.mux.Handle("PUT /api/v1/planner/{date}", handler.PutScheduleHandler(s.deps.Schedules, onChange...))
		s.mux.Handle("DELETE /api/v1/planner/{date}", handler.DeleteScheduleHandler(s.deps.Schedules, onChange...))
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

// wallClockTimerAdapter implements handler.TimerClock using the real system clock.
type wallClockTimerAdapter struct{}

func (wallClockTimerAdapter) Now() time.Time { return time.Now() }

// SetTicker attaches a Ticker to the server for schedule-change notifications.
func (s *Server) SetTicker(t *Ticker) { s.ticker = t }

// chain wraps h with the given middleware. The first middleware in the
// list runs first (outermost).
func chain(h http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}
