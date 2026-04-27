package server_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/server"
)

type HealthSuite struct {
	suite.Suite
}

func TestHealth(t *testing.T) {
	suite.Run(t, new(HealthSuite))
}

// ---------------------------------------------------------------------------
// Behavior 4: Health endpoints — /health
// ---------------------------------------------------------------------------

func (s *HealthSuite) TestHealthReturns200() {
	handler := server.HealthHandler()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	handler.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code, "GET /health must return 200")
}

func (s *HealthSuite) TestHealthReturnsStatusOKJSON() {
	handler := server.HealthHandler()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	handler.ServeHTTP(rec, req)

	var body map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &body)
	s.Require().NoError(err, "response body must be valid JSON")
	s.Equal("ok", body["status"], `response must contain {"status":"ok"}`)
}

func (s *HealthSuite) TestHealthSetsContentTypeJSON() {
	handler := server.HealthHandler()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	handler.ServeHTTP(rec, req)

	s.Contains(rec.Header().Get("Content-Type"), "application/json",
		"Content-Type must be application/json")
}

// ---------------------------------------------------------------------------
// Behavior 4: Health endpoints — /health/ready
// ---------------------------------------------------------------------------

// mockChecker is a test double for SubsystemChecker.
type mockChecker struct {
	name string
	err  error
}

func (m *mockChecker) Name() string { return m.name }
func (m *mockChecker) Check() error { return m.err }

func (s *HealthSuite) TestReadyReturns200WhenAllHealthy() {
	db := &mockChecker{name: "database", err: nil}
	ollama := &mockChecker{name: "ollama", err: nil}

	handler := server.ReadyHandler(db, ollama)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)

	handler.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code, "GET /health/ready must return 200 when all subsystems healthy")
}

func (s *HealthSuite) TestReadyReturns503WhenSubsystemUnhealthy() {
	db := &mockChecker{name: "database", err: nil}
	ollama := &mockChecker{name: "ollama", err: errors.New("connection refused")}

	handler := server.ReadyHandler(db, ollama)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)

	handler.ServeHTTP(rec, req)

	s.Equal(http.StatusServiceUnavailable, rec.Code,
		"GET /health/ready must return 503 when any subsystem is unhealthy")
}

func (s *HealthSuite) TestReadyResponseIncludesSubsystemDetails() {
	db := &mockChecker{name: "database", err: nil}
	ollama := &mockChecker{name: "ollama", err: errors.New("timeout")}

	handler := server.ReadyHandler(db, ollama)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)

	handler.ServeHTTP(rec, req)

	var body map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &body)
	s.Require().NoError(err, "response body must be valid JSON")

	// Should contain subsystem check results.
	s.Contains(rec.Body.String(), "database",
		"ready response must include subsystem names")
	s.Contains(rec.Body.String(), "ollama",
		"ready response must include subsystem names")
}

func (s *HealthSuite) TestReadyReturns200WithNoCheckers() {
	handler := server.ReadyHandler()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)

	handler.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code,
		"GET /health/ready with no checkers should return 200")
}

func (s *HealthSuite) TestReadySetsContentTypeJSON() {
	handler := server.ReadyHandler()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)

	handler.ServeHTTP(rec, req)

	s.Contains(rec.Header().Get("Content-Type"), "application/json",
		"Content-Type must be application/json")
}
