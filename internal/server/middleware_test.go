package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/server"
)

type MiddlewareSuite struct {
	suite.Suite
}

func TestMiddleware(t *testing.T) {
	suite.Run(t, new(MiddlewareSuite))
}

// ---------------------------------------------------------------------------
// RecoveryMiddleware
// ---------------------------------------------------------------------------

func (s *MiddlewareSuite) TestRecoveryMiddlewareReturns500OnPanic() {
	panicking := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	handler := server.RecoveryMiddleware(panicking)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	// Should not panic — middleware should catch it.
	s.NotPanics(func() {
		handler.ServeHTTP(rec, req)
	})

	s.Equal(http.StatusInternalServerError, rec.Code,
		"RecoveryMiddleware must return 500 on handler panic")
}

func (s *MiddlewareSuite) TestRecoveryMiddlewarePassesThroughNormal() {
	normal := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := server.RecoveryMiddleware(normal)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	handler.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code,
		"RecoveryMiddleware must pass through normal responses")
}

// ---------------------------------------------------------------------------
// RequestIDMiddleware
// ---------------------------------------------------------------------------

func (s *MiddlewareSuite) TestRequestIDMiddlewareSetsHeader() {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := server.RequestIDMiddleware(inner)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	handler.ServeHTTP(rec, req)

	requestID := rec.Header().Get("X-Request-ID")
	s.NotEmpty(requestID, "RequestIDMiddleware must set X-Request-ID response header")
}

func (s *MiddlewareSuite) TestRequestIDMiddlewareUniquePerRequest() {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := server.RequestIDMiddleware(inner)

	rec1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec1, req1)

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec2, req2)

	id1 := rec1.Header().Get("X-Request-ID")
	id2 := rec2.Header().Get("X-Request-ID")
	s.NotEqual(id1, id2, "each request must get a unique X-Request-ID")
}

// ---------------------------------------------------------------------------
// CORSMiddleware
// ---------------------------------------------------------------------------

func (s *MiddlewareSuite) TestCORSMiddlewareSetsHeaders() {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := server.CORSMiddleware(inner)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	handler.ServeHTTP(rec, req)

	s.NotEmpty(rec.Header().Get("Access-Control-Allow-Origin"),
		"CORSMiddleware must set Access-Control-Allow-Origin")
}

func (s *MiddlewareSuite) TestCORSMiddlewareHandlesPreflight() {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := server.CORSMiddleware(inner)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")

	handler.ServeHTTP(rec, req)

	s.Contains([]int{http.StatusOK, http.StatusNoContent}, rec.Code,
		"CORS preflight should return 200 or 204")
	s.NotEmpty(rec.Header().Get("Access-Control-Allow-Methods"),
		"CORS preflight must set Access-Control-Allow-Methods")
}

// ---------------------------------------------------------------------------
// ContentTypeMiddleware
// ---------------------------------------------------------------------------

func (s *MiddlewareSuite) TestContentTypeMiddlewareRejectsNonJSON() {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := server.ContentTypeMiddleware(inner)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "text/plain")

	handler.ServeHTTP(rec, req)

	s.Equal(http.StatusUnsupportedMediaType, rec.Code,
		"ContentTypeMiddleware must reject non-JSON POST bodies with 415")
}

func (s *MiddlewareSuite) TestContentTypeMiddlewareAllowsJSON() {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := server.ContentTypeMiddleware(inner)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"key":"val"}`))
	req.Header.Set("Content-Type", "application/json")

	handler.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code,
		"ContentTypeMiddleware must allow application/json bodies")
}

func (s *MiddlewareSuite) TestContentTypeMiddlewareSkipsGET() {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := server.ContentTypeMiddleware(inner)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	handler.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code,
		"ContentTypeMiddleware must not check Content-Type for GET requests")
}

// ---------------------------------------------------------------------------
// LoggingMiddleware
// ---------------------------------------------------------------------------

func (s *MiddlewareSuite) TestLoggingMiddlewarePassesThrough() {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})

	handler := server.LoggingMiddleware(inner)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	handler.ServeHTTP(rec, req)

	s.Equal(http.StatusTeapot, rec.Code,
		"LoggingMiddleware must pass through the downstream status code")
}
