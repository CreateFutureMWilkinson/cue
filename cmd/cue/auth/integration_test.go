package auth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/cmd/cue/auth"
	"github.com/CreateFutureMWilkinson/cue/pkg/client"
)

// IntegrationSuite covers the DefaultProbe + end-to-end Bootstrap path
// against a real httptest server.
type IntegrationSuite struct {
	suite.Suite
}

func TestIntegration(t *testing.T) {
	suite.Run(t, new(IntegrationSuite))
}

// C1: DefaultProbe issues GET /api/v1/auth/tokens.
func (s *IntegrationSuite) TestDefaultProbeHitsListTokensEndpoint() {
	var seen struct {
		method string
		path   string
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen.method = r.Method
		seen.path = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]any{})
	}))
	defer ts.Close()

	sdk := client.New(ts.URL)
	sdk.SetToken("any-token")
	s.Require().NoError(auth.DefaultProbe(context.Background(), sdk))
	s.Equal(http.MethodGet, seen.method)
	s.Equal("/api/v1/auth/tokens", seen.path)
}

// C2: Bootstrap with DefaultProbe against a fresh-server httptest performs
// the TOKEN_ISSUED auto-issue, persists the token, and a second Bootstrap
// call short-circuits via the on-disk token (no server calls).
func (s *IntegrationSuite) TestBootstrapEndToEndAutoIssueAndShortCircuit() {
	const issuedToken = "deadbeefcafef00d"
	var requestCount int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		s.Equal("/api/v1/auth/tokens", r.URL.Path)

		if r.Header.Get("Authorization") == "" {
			// First-client probe: emit TOKEN_ISSUED so the SDK's
			// auto-retry installs the token and replays the request.
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{
					"code":    "TOKEN_ISSUED",
					"message": "first client; token issued",
				},
				"token": issuedToken,
			})
			return
		}

		s.Equal("Bearer "+issuedToken, r.Header.Get("Authorization"),
			"retry must carry the freshly-issued bearer (request #%d)", count)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]any{})
	}))
	defer ts.Close()

	tokenPath := filepath.Join(s.T().TempDir(), "client-token")
	store := auth.NewFileStore(tokenPath)
	sdk := client.New(ts.URL)

	// First Bootstrap: should drive the TOKEN_ISSUED auto-issue path.
	s.Require().NoError(auth.Bootstrap(context.Background(), store, sdk, auth.DefaultProbe))
	s.Equal(issuedToken, sdk.Token(), "SDK should hold the auto-issued token")

	persisted, err := store.Load(context.Background())
	s.Require().NoError(err)
	s.Equal(issuedToken, persisted, "FileStore must contain the auto-issued token")

	firstRunCalls := atomic.LoadInt32(&requestCount)
	s.Equal(int32(2), firstRunCalls, "first run = probe (401) + retry (200)")

	// Second Bootstrap: should short-circuit on the on-disk token, no server traffic.
	sdk2 := client.New(ts.URL)
	s.Require().NoError(auth.Bootstrap(context.Background(), store, sdk2, auth.DefaultProbe))
	s.Equal(issuedToken, sdk2.Token())
	s.Equal(firstRunCalls, atomic.LoadInt32(&requestCount),
		"second Bootstrap must not hit the server when token is on disk")
}
