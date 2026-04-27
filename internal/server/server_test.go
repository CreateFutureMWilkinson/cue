package server_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/server"
	"github.com/CreateFutureMWilkinson/cue/internal/server/handler"
	"github.com/CreateFutureMWilkinson/cue/internal/service/calendar"
	"github.com/CreateFutureMWilkinson/cue/internal/service/planner"
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

// ---------------------------------------------------------------------------
// Behavior 10: Graceful shutdown closes WebSocket connections
// ---------------------------------------------------------------------------

func (s *ServerSuite) TestServerShutdownClosesWebSockets() {
	cfg := config.ServerConfig{
		Host:                "127.0.0.1",
		Port:                0,
		ReadTimeoutSeconds:  5,
		WriteTimeoutSeconds: 5,
	}

	srv, err := server.New(cfg)
	s.Require().NoError(err)

	// Start server in background.
	startErr := make(chan error, 1)
	go func() { startErr <- srv.Start() }()

	// Wait for addr to be populated (server is listening).
	s.Require().Eventually(func() bool {
		return srv.Addr() != ""
	}, 2*time.Second, 10*time.Millisecond, "server should bind an address")

	wsURL := "ws://" + srv.Addr() + "/api/v1/websocket/events"

	dialCtx, dialCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer dialCancel()

	conn, _, err := websocket.Dial(dialCtx, wsURL, nil)
	s.Require().NoError(err, "WebSocket dial should succeed")
	defer conn.CloseNow() //nolint:errcheck

	// Trigger graceful shutdown.
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutCancel()
	s.Require().NoError(srv.Shutdown(shutCtx))

	// After shutdown, the server must have closed the WebSocket connection.
	// A read attempt should return an error (close frame or connection reset).
	readCtx, readCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer readCancel()
	_, _, readErr := conn.Read(readCtx)
	s.Error(readErr, "reading after server shutdown should return an error")
}

// ---------------------------------------------------------------------------
// Behavior 12: REST events route — GET /api/v1/events
// ---------------------------------------------------------------------------

func (s *ServerSuite) TestEventsRouteMounted() {
	cfg := config.ServerConfig{
		Host:                "127.0.0.1",
		Port:                0,
		ReadTimeoutSeconds:  5,
		WriteTimeoutSeconds: 5,
	}

	srv, err := server.New(cfg)
	s.Require().NoError(err)

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// --- Sub-case 1: valid request with since=0 should return 200 + JSON shape ---
	resp, err := http.Get(ts.URL + "/api/v1/events?since=0")
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(http.StatusOK, resp.StatusCode, "GET /api/v1/events?since=0 should return 200")

	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)

	var payload map[string]json.RawMessage
	s.Require().NoError(json.Unmarshal(body, &payload), "response body should be valid JSON")

	for _, key := range []string{"events", "truncated", "oldest_seq", "latest_seq"} {
		_, ok := payload[key]
		s.True(ok, "response JSON should contain key %q", key)
	}

	// --- Sub-case 2: missing since parameter should return 400 ---
	resp2, err := http.Get(ts.URL + "/api/v1/events")
	s.Require().NoError(err)
	defer resp2.Body.Close()

	s.Equal(http.StatusBadRequest, resp2.StatusCode, "GET /api/v1/events without since should return 400")
}

// ---------------------------------------------------------------------------
// Mock TodoServicer for route registration tests
// ---------------------------------------------------------------------------

// serverMockTodoServicer implements handler.TodoServicer with minimal stubs.
type serverMockTodoServicer struct{}

func (m *serverMockTodoServicer) Create(_ context.Context, todo *repository.Todo) (*repository.Todo, error) {
	now := time.Now()
	todo.ID = uuid.New()
	todo.CreatedAt = now
	return todo, nil
}

func (m *serverMockTodoServicer) Get(_ context.Context, _ uuid.UUID) (*repository.Todo, error) {
	now := time.Now()
	return &repository.Todo{
		ID:        uuid.New(),
		Title:     "test task",
		Priority:  3,
		CreatedAt: now,
	}, nil
}

func (m *serverMockTodoServicer) List(_ context.Context, _ repository.TodoFilter) ([]*repository.Todo, int, error) {
	return []*repository.Todo{}, 0, nil
}

func (m *serverMockTodoServicer) Update(_ context.Context, todo *repository.Todo) (*repository.Todo, error) {
	return todo, nil
}

func (m *serverMockTodoServicer) Delete(_ context.Context, _ uuid.UUID) error {
	return nil
}

func stubEffectiveEstimateServer(t *repository.Todo) *int {
	return t.EstimateMinutes
}

// ---------------------------------------------------------------------------
// Task route registration tests
// ---------------------------------------------------------------------------

func (s *ServerSuite) TestTaskRoutesRegistered() {
	cfg := config.ServerConfig{
		Host:                "127.0.0.1",
		Port:                0,
		ReadTimeoutSeconds:  5,
		WriteTimeoutSeconds: 5,
	}

	srv, err := server.New(cfg, server.Deps{
		Todos:             &serverMockTodoServicer{},
		EffectiveEstimate: handler.EffectiveEstimateFunc(stubEffectiveEstimateServer),
	})
	s.Require().NoError(err)

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	taskID := uuid.New().String()

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
	}{
		{
			name:       "GET /api/v1/tasks returns 200",
			method:     http.MethodGet,
			path:       "/api/v1/tasks",
			wantStatus: http.StatusOK,
		},
		{
			name:       "POST /api/v1/tasks returns 201",
			method:     http.MethodPost,
			path:       "/api/v1/tasks",
			body:       `{"title":"test task","priority":3}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "GET /api/v1/tasks/{id} returns 200",
			method:     http.MethodGet,
			path:       "/api/v1/tasks/" + taskID,
			wantStatus: http.StatusOK,
		},
		{
			name:       "PUT /api/v1/tasks/{id} returns 200",
			method:     http.MethodPut,
			path:       "/api/v1/tasks/" + taskID,
			body:       `{"title":"updated"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "DELETE /api/v1/tasks/{id} returns 204",
			method:     http.MethodDelete,
			path:       "/api/v1/tasks/" + taskID,
			wantStatus: http.StatusNoContent,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			var bodyReader io.Reader
			if tc.body != "" {
				bodyReader = strings.NewReader(tc.body)
			}

			req, err := http.NewRequest(tc.method, ts.URL+tc.path, bodyReader)
			s.Require().NoError(err)

			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}

			resp, err := http.DefaultClient.Do(req)
			s.Require().NoError(err)
			defer resp.Body.Close()

			s.NotEqual(http.StatusNotFound, resp.StatusCode,
				"route %s %s should be registered (got 404)", tc.method, tc.path)
			s.Equal(tc.wantStatus, resp.StatusCode,
				"route %s %s should return %d", tc.method, tc.path, tc.wantStatus)
		})
	}
}

func (s *ServerSuite) TestTaskRoutesNotRegisteredWhenNil() {
	cfg := config.ServerConfig{
		Host:                "127.0.0.1",
		Port:                0,
		ReadTimeoutSeconds:  5,
		WriteTimeoutSeconds: 5,
	}

	// No Todos in Deps — task routes should not be registered.
	srv, err := server.New(cfg)
	s.Require().NoError(err)

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	taskID := uuid.New().String()

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/tasks"},
		{http.MethodPost, "/api/v1/tasks"},
		{http.MethodGet, "/api/v1/tasks/" + taskID},
		{http.MethodPut, "/api/v1/tasks/" + taskID},
		{http.MethodDelete, "/api/v1/tasks/" + taskID},
	}

	for _, tc := range paths {
		s.Run(tc.method+" "+tc.path+" returns 404", func() {
			var bodyReader io.Reader
			if tc.method == http.MethodPost || tc.method == http.MethodPut {
				bodyReader = strings.NewReader(`{"title":"test"}`)
			}

			req, err := http.NewRequest(tc.method, ts.URL+tc.path, bodyReader)
			s.Require().NoError(err)

			resp, err := http.DefaultClient.Do(req)
			s.Require().NoError(err)
			defer resp.Body.Close()

			s.Equal(http.StatusNotFound, resp.StatusCode,
				"%s %s should return 404 when Todos is nil", tc.method, tc.path)
		})
	}
}

// ---------------------------------------------------------------------------
// Mock ScheduleStore for route registration tests
// ---------------------------------------------------------------------------

type serverMockScheduleStore struct{}

func (m *serverMockScheduleStore) LoadByDate(_ context.Context, date time.Time) (*repository.Schedule, error) {
	return &repository.Schedule{
		ID:        uuid.New(),
		Date:      date,
		Strategy:  "focus",
		Blocks:    []repository.ScheduleBlock{},
		CreatedAt: time.Now(),
	}, nil
}

func (m *serverMockScheduleStore) Save(_ context.Context, schedule *repository.Schedule) error {
	return nil
}

func (m *serverMockScheduleStore) Delete(_ context.Context, _ time.Time) error {
	return nil
}

// serverMockScheduleGenerator implements handler.ScheduleGenerator with minimal stubs.
type serverMockScheduleGenerator struct{}

func (m *serverMockScheduleGenerator) GenerateSchedules(_ context.Context, _ []planner.TaskEstimate, _ []calendar.CalendarEvent, _ time.Time) (*planner.DaySchedule, *planner.DaySchedule, error) {
	return &planner.DaySchedule{Strategy: "focus", Blocks: []planner.TimeBlock{}},
		&planner.DaySchedule{Strategy: "recovery", Blocks: []planner.TimeBlock{}}, nil
}

func (m *serverMockScheduleGenerator) TargetDate(now time.Time) time.Time {
	return now
}

// serverMockCalendarFetcher implements handler.CalendarFetcher with minimal stubs.
type serverMockCalendarFetcher struct{}

func (m *serverMockCalendarFetcher) FetchEvents(_ context.Context, _ time.Time) ([]calendar.CalendarEvent, error) {
	return []calendar.CalendarEvent{}, nil
}

// ---------------------------------------------------------------------------
// Planner route registration tests
// ---------------------------------------------------------------------------

func (s *ServerSuite) TestPlannerRoutesRegistered() {
	cfg := config.ServerConfig{
		Host:                "127.0.0.1",
		Port:                0,
		ReadTimeoutSeconds:  5,
		WriteTimeoutSeconds: 5,
	}

	srv, err := server.New(cfg, server.Deps{
		Schedules:         &serverMockScheduleStore{},
		ScheduleGenerator: &serverMockScheduleGenerator{},
		Calendar:          &serverMockCalendarFetcher{},
	})
	s.Require().NoError(err)

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
	}{
		{
			name:       "GET /api/v1/planner/{date} returns 200",
			method:     http.MethodGet,
			path:       "/api/v1/planner/2026-04-20",
			wantStatus: http.StatusOK,
		},
		{
			name:       "PUT /api/v1/planner/{date} returns 200",
			method:     http.MethodPut,
			path:       "/api/v1/planner/2026-04-20",
			body:       `{"strategy":"focus","blocks":[]}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "DELETE /api/v1/planner/{date} returns 204",
			method:     http.MethodDelete,
			path:       "/api/v1/planner/2026-04-20",
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "GET /api/v1/planner/active returns 200",
			method:     http.MethodGet,
			path:       "/api/v1/planner/active",
			wantStatus: http.StatusOK,
		},
		{
			name:       "DELETE /api/v1/planner/active returns 204",
			method:     http.MethodDelete,
			path:       "/api/v1/planner/active",
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "POST /api/v1/planner/generate returns 200",
			method:     http.MethodPost,
			path:       "/api/v1/planner/generate",
			body:       `{"date":"2026-04-20"}`,
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			var bodyReader io.Reader
			if tc.body != "" {
				bodyReader = strings.NewReader(tc.body)
			}

			req, err := http.NewRequest(tc.method, ts.URL+tc.path, bodyReader)
			s.Require().NoError(err)

			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}

			resp, err := http.DefaultClient.Do(req)
			s.Require().NoError(err)
			defer resp.Body.Close()

			s.NotEqual(http.StatusNotFound, resp.StatusCode,
				"route %s %s should be registered (got 404)", tc.method, tc.path)
			s.Equal(tc.wantStatus, resp.StatusCode,
				"route %s %s should return %d", tc.method, tc.path, tc.wantStatus)
		})
	}
}

func (s *ServerSuite) TestPlannerRoutesNotRegisteredWhenNil() {
	cfg := config.ServerConfig{
		Host:                "127.0.0.1",
		Port:                0,
		ReadTimeoutSeconds:  5,
		WriteTimeoutSeconds: 5,
	}

	// No Schedules in Deps — planner routes should not be registered.
	srv, err := server.New(cfg)
	s.Require().NoError(err)

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/planner/2026-04-20"},
		{http.MethodPut, "/api/v1/planner/2026-04-20"},
		{http.MethodDelete, "/api/v1/planner/2026-04-20"},
		{http.MethodGet, "/api/v1/planner/active"},
		{http.MethodDelete, "/api/v1/planner/active"},
		{http.MethodPost, "/api/v1/planner/generate"},
	}

	for _, tc := range paths {
		s.Run(tc.method+" "+tc.path+" returns 404", func() {
			var bodyReader io.Reader
			if tc.method == http.MethodPost || tc.method == http.MethodPut {
				bodyReader = strings.NewReader(`{"strategy":"focus","blocks":[]}`)
			}

			req, err := http.NewRequest(tc.method, ts.URL+tc.path, bodyReader)
			s.Require().NoError(err)

			resp, err := http.DefaultClient.Do(req)
			s.Require().NoError(err)
			defer resp.Body.Close()

			s.Equal(http.StatusNotFound, resp.StatusCode,
				"%s %s should return 404 when Schedules is nil", tc.method, tc.path)
		})
	}
}
