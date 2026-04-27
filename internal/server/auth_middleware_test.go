package server_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/server"
)

// ---------------------------------------------------------------------------
// Mock AuthTokenLookup
// ---------------------------------------------------------------------------

type mockAuthTokenLookup struct {
	LookupByHashFn   func(ctx context.Context, hash string) (*repository.AuthToken, error)
	CountActiveFn    func(ctx context.Context) (int, error)
	CreateFn         func(ctx context.Context, token *repository.AuthToken) error
	UpdateLastSeenFn func(ctx context.Context, id uuid.UUID, t time.Time) error
}

func (m *mockAuthTokenLookup) LookupByHash(ctx context.Context, hash string) (*repository.AuthToken, error) {
	if m.LookupByHashFn != nil {
		return m.LookupByHashFn(ctx, hash)
	}
	return nil, nil
}

func (m *mockAuthTokenLookup) CountActive(ctx context.Context) (int, error) {
	if m.CountActiveFn != nil {
		return m.CountActiveFn(ctx)
	}
	return 0, nil
}

func (m *mockAuthTokenLookup) Create(ctx context.Context, token *repository.AuthToken) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, token)
	}
	return nil
}

func (m *mockAuthTokenLookup) UpdateLastSeen(ctx context.Context, id uuid.UUID, t time.Time) error {
	if m.UpdateLastSeenFn != nil {
		return m.UpdateLastSeenFn(ctx, id, t)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Suite
// ---------------------------------------------------------------------------

type AuthMiddlewareSuite struct {
	suite.Suite
}

func TestAuthMiddleware(t *testing.T) {
	suite.Run(t, new(AuthMiddlewareSuite))
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func (s *AuthMiddlewareSuite) TestValidTokenPassesThrough() {
	rawToken := "test-bearer-token"
	hash := sha256.Sum256([]byte(rawToken))
	expectedHash := hex.EncodeToString(hash[:])

	tokenID := uuid.New()
	mock := &mockAuthTokenLookup{
		LookupByHashFn: func(_ context.Context, h string) (*repository.AuthToken, error) {
			s.Equal(expectedHash, h, "middleware must hash the bearer token with SHA-256")
			return &repository.AuthToken{
				ID:        tokenID,
				Label:     "test-token",
				TokenHash: expectedHash,
				CreatedAt: time.Now().Add(-time.Hour),
				LastSeen:  time.Now().Add(-time.Minute),
				Revoked:   false,
			}, nil
		},
		UpdateLastSeenFn: func(_ context.Context, id uuid.UUID, _ time.Time) error {
			s.Equal(tokenID, id, "middleware must update last-seen for the matched token")
			return nil
		},
	}

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	handler := server.AuthMiddleware(mock, true)(inner)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	req.Header.Set("Authorization", "Bearer "+rawToken)

	handler.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code,
		"valid non-revoked token must pass through to the inner handler")
	s.Equal("ok", rec.Body.String(),
		"inner handler body must be returned unmodified")
}

// --- B5: missing/invalid/revoked token returns 401 ---

func (s *AuthMiddlewareSuite) TestMissingAuthHeaderReturns401() {
	mock := &mockAuthTokenLookup{
		CountActiveFn: func(_ context.Context) (int, error) { return 1, nil },
	}
	handler := server.AuthMiddleware(mock, true)(okHandler())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages", nil)
	handler.ServeHTTP(rec, req)

	s.Equal(http.StatusUnauthorized, rec.Code)
}

func (s *AuthMiddlewareSuite) TestMalformedAuthHeaderReturns401() {
	mock := &mockAuthTokenLookup{
		CountActiveFn: func(_ context.Context) (int, error) { return 1, nil },
	}
	handler := server.AuthMiddleware(mock, true)(okHandler())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages", nil)
	req.Header.Set("Authorization", "NotBearer token")
	handler.ServeHTTP(rec, req)

	s.Equal(http.StatusUnauthorized, rec.Code)
}

func (s *AuthMiddlewareSuite) TestInvalidTokenReturns401() {
	mock := &mockAuthTokenLookup{
		LookupByHashFn: func(_ context.Context, _ string) (*repository.AuthToken, error) {
			return nil, repository.ErrNotFound
		},
		CountActiveFn: func(_ context.Context) (int, error) { return 1, nil },
	}
	handler := server.AuthMiddleware(mock, true)(okHandler())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages", nil)
	req.Header.Set("Authorization", "Bearer bad-token")
	handler.ServeHTTP(rec, req)

	s.Equal(http.StatusUnauthorized, rec.Code)
}

func (s *AuthMiddlewareSuite) TestRevokedTokenReturns401() {
	rawToken := "revoked-token"
	hash := sha256.Sum256([]byte(rawToken))
	expectedHash := hex.EncodeToString(hash[:])

	mock := &mockAuthTokenLookup{
		LookupByHashFn: func(_ context.Context, h string) (*repository.AuthToken, error) {
			if h == expectedHash {
				return &repository.AuthToken{
					ID:      uuid.New(),
					Revoked: true,
				}, nil
			}
			return nil, repository.ErrNotFound
		},
		CountActiveFn: func(_ context.Context) (int, error) { return 1, nil },
	}
	handler := server.AuthMiddleware(mock, true)(okHandler())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages", nil)
	req.Header.Set("Authorization", "Bearer "+rawToken)
	handler.ServeHTTP(rec, req)

	s.Equal(http.StatusUnauthorized, rec.Code)
}

// --- B6: exempt routes bypass auth ---

func (s *AuthMiddlewareSuite) TestExemptRoutesPassWithoutAuth() {
	mock := &mockAuthTokenLookup{
		CountActiveFn: func(_ context.Context) (int, error) { return 1, nil },
	}
	handler := server.AuthMiddleware(mock, true)(okHandler())

	exemptPaths := []string{
		"/health",
		"/health/ready",
		"/api/v1/health",
		"/api/v1/health/ready",
		"/api/v1/auth/pair",
	}

	for _, path := range exemptPaths {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		// No Authorization header
		handler.ServeHTTP(rec, req)
		s.Equal(http.StatusOK, rec.Code, "exempt route %s should pass without auth", path)
	}
}

func (s *AuthMiddlewareSuite) TestExemptPairSubpathsPassWithoutAuth() {
	mock := &mockAuthTokenLookup{
		CountActiveFn: func(_ context.Context) (int, error) { return 1, nil },
	}
	handler := server.AuthMiddleware(mock, true)(okHandler())

	// Pair poll endpoint is also unauthenticated
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/pair/some-uuid", nil)
	handler.ServeHTTP(rec, req)
	s.Equal(http.StatusOK, rec.Code, "pair poll endpoint should pass without auth")
}

// --- B7: first-client auto-issue ---

func (s *AuthMiddlewareSuite) TestFirstClientAutoIssuesToken() {
	var createdToken *repository.AuthToken

	mock := &mockAuthTokenLookup{
		CountActiveFn: func(_ context.Context) (int, error) { return 0, nil },
		CreateFn: func(_ context.Context, token *repository.AuthToken) error {
			createdToken = token
			return nil
		},
		LookupByHashFn: func(_ context.Context, _ string) (*repository.AuthToken, error) {
			return nil, repository.ErrNotFound
		},
	}
	handler := server.AuthMiddleware(mock, true)(okHandler())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages", nil)
	// No Authorization header — first client
	handler.ServeHTTP(rec, req)

	s.Equal(http.StatusUnauthorized, rec.Code, "first-client flow returns 401 with token")
	s.NotNil(createdToken, "a token should have been created")
	s.NotEmpty(createdToken.TokenHash, "created token should have a hash")

	// Response body should contain TOKEN_ISSUED code and a token field
	var resp map[string]any
	s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &resp))
	errObj, ok := resp["error"].(map[string]any)
	s.Require().True(ok)
	s.Equal("TOKEN_ISSUED", errObj["code"])
	tokenStr, ok := resp["token"].(string)
	s.True(ok && tokenStr != "", "response must contain plaintext token")
}

func (s *AuthMiddlewareSuite) TestFirstClientNotTriggeredWhenActiveTokensExist() {
	mock := &mockAuthTokenLookup{
		CountActiveFn: func(_ context.Context) (int, error) { return 1, nil },
		LookupByHashFn: func(_ context.Context, _ string) (*repository.AuthToken, error) {
			return nil, repository.ErrNotFound
		},
	}
	handler := server.AuthMiddleware(mock, true)(okHandler())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages", nil)
	req.Header.Set("Authorization", "Bearer some-invalid-token")
	handler.ServeHTTP(rec, req)

	s.Equal(http.StatusUnauthorized, rec.Code, "should return regular 401, not auto-issue")
}

// --- B8: last_seen throttle ---

func (s *AuthMiddlewareSuite) TestLastSeenThrottleOncePerMinute() {
	rawToken := "throttle-token"
	hash := sha256.Sum256([]byte(rawToken))
	expectedHash := hex.EncodeToString(hash[:])
	tokenID := uuid.New()

	updateCount := 0
	mock := &mockAuthTokenLookup{
		LookupByHashFn: func(_ context.Context, h string) (*repository.AuthToken, error) {
			if h == expectedHash {
				return &repository.AuthToken{
					ID:        tokenID,
					TokenHash: expectedHash,
					LastSeen:  time.Now().Add(-30 * time.Second), // last seen 30s ago
					Revoked:   false,
				}, nil
			}
			return nil, repository.ErrNotFound
		},
		UpdateLastSeenFn: func(_ context.Context, _ uuid.UUID, _ time.Time) error {
			updateCount++
			return nil
		},
	}
	handler := server.AuthMiddleware(mock, true)(okHandler())

	// Send 3 rapid requests — should only trigger 1 UpdateLastSeen
	for range 3 {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/messages", nil)
		req.Header.Set("Authorization", "Bearer "+rawToken)
		handler.ServeHTTP(rec, req)
		s.Equal(http.StatusOK, rec.Code)
	}

	// Give goroutines a moment to complete
	time.Sleep(50 * time.Millisecond)

	// First request triggers update (>30s since last seen), subsequent ones are throttled
	s.Equal(1, updateCount, "UpdateLastSeen should fire once, then throttle for subsequent requests within the minute")
}

// --- B9: dev mode (auth disabled) ---

func (s *AuthMiddlewareSuite) TestDevModePassesAllRequests() {
	mock := &mockAuthTokenLookup{} // no functions set — will panic if called
	handler := server.AuthMiddleware(mock, false)(okHandler())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages", nil)
	// No Authorization header
	handler.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code, "dev mode should pass all requests without auth")
}

// --- helpers ---

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
}
