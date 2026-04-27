package server

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const requestIDHeader = "X-Request-ID"

// RecoveryMiddleware catches panics in downstream handlers and returns
// a 500 Internal Server Error instead of crashing the process.
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				// #nosec G706 -- path is sanitized via sanitizeForLog (CR/LF stripped, length-bounded).
				slog.Error("handler panic",
					"panic", rec,
					"path", sanitizeForLog(r.URL.Path),
				)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// RequestIDMiddleware injects a unique X-Request-ID header into every
// response. If the request already carries one it is propagated;
// otherwise a fresh random ID is generated.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(requestIDHeader)
		if id == "" {
			id = newRequestID()
		}
		w.Header().Set(requestIDHeader, id)
		next.ServeHTTP(w, r)
	})
}

// sanitizeForLog strips CR and LF so attacker-controlled request data
// cannot forge log lines (CWE-117). Long values are also truncated to
// keep log entries readable.
func sanitizeForLog(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", "")
	const maxLen = 256
	if len(s) > maxLen {
		s = s[:maxLen]
	}
	return s
}

func newRequestID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// CORSMiddleware adds permissive Cross-Origin Resource Sharing headers
// so local web UIs can call the API. Preflight OPTIONS requests are
// answered directly with 204.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ContentTypeMiddleware enforces application/json Content-Type on
// request bodies for POST/PUT/PATCH methods. Other methods pass
// through unchecked.
func ContentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch:
			ct := r.Header.Get("Content-Type")
			if ct != "" && !isJSONContentType(ct) {
				http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func isJSONContentType(ct string) bool {
	// Tolerate parameters like "application/json; charset=utf-8".
	for i := 0; i < len(ct); i++ {
		if ct[i] == ';' {
			ct = ct[:i]
			break
		}
	}
	for len(ct) > 0 && ct[len(ct)-1] == ' ' {
		ct = ct[:len(ct)-1]
	}
	return ct == "application/json"
}

// LoggingMiddleware logs each request's method, path, status, and
// duration. It wraps the ResponseWriter to capture the status code.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		// #nosec G706 -- method and path are sanitized via sanitizeForLog (CR/LF stripped, length-bounded).
		slog.Info("http request",
			"method", sanitizeForLog(r.Method),
			"path", sanitizeForLog(r.URL.Path),
			"status", sw.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (s *statusWriter) WriteHeader(code int) {
	if s.wroteHeader {
		return
	}
	s.status = code
	s.wroteHeader = true
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusWriter) Write(b []byte) (int, error) {
	if !s.wroteHeader {
		s.wroteHeader = true
	}
	return s.ResponseWriter.Write(b)
}

// Unwrap returns the underlying ResponseWriter so that middleware wrappers
// like statusWriter do not hide interfaces (e.g. http.Hijacker) required
// by WebSocket upgrade handlers. The coder/websocket library uses
// http.NewResponseController which calls Unwrap to discover the real writer.
func (s *statusWriter) Unwrap() http.ResponseWriter { return s.ResponseWriter }
