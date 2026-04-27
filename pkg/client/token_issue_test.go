package client_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/pkg/client"
)

// TokenIssueSuite covers the first-client auto-token flow: on a 401 with a
// nested {"error":{"code":"TOKEN_ISSUED"},"token":"..."} body, the SDK must
// store the token and retry the original request exactly once. Other 401
// shapes and malformed TOKEN_ISSUED bodies must fall through to *APIError
// without a retry, and the retry must be capped at one attempt even if the
// server keeps replying TOKEN_ISSUED.
type TokenIssueSuite struct {
	suite.Suite
}

func TestTokenIssue(t *testing.T) {
	suite.Run(t, new(TokenIssueSuite))
}

// TestFirstClientAutoTokenStoresAndRetries verifies that when the server
// replies 401 with a nested TOKEN_ISSUED body on the first attempt and the
// client retries with the auto-issued token, the SDK stores the token on
// the APIClient (observable via Token()) and the second request succeeds.
func (s *TokenIssueSuite) TestFirstClientAutoTokenStoresAndRetries() {
	const issuedToken = "abc123deadbeef"
	var callCount atomic.Int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("/api/v1/auth/tokens", r.URL.Path)
		n := callCount.Add(1)

		if n == 1 {
			// First call: unauthenticated — server auto-issues a token.
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{
					"code":    "TOKEN_ISSUED",
					"message": "First client — token auto-issued",
				},
				"token": issuedToken,
			})
			return
		}

		// Subsequent calls require the auto-issued token.
		if r.Header.Get("Authorization") != "Bearer "+issuedToken {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "missing token on retry"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer ts.Close()

	c := client.New(ts.URL)
	auth := client.NewAuthClient(c)

	tokens, err := auth.ListTokens(context.Background())
	s.Require().NoError(err, "ListTokens should succeed after auto-token retry")
	s.NotNil(tokens)
	s.Equal(issuedToken, c.Token(), "client should have stored the auto-issued token")
	s.Equal(int32(2), callCount.Load(), "server should see exactly 2 requests (original + 1 retry)")
}

// TestRegular401WithoutTokenIssuedReturnsAPIError verifies that a plain 401
// response (flat body, no TOKEN_ISSUED code) surfaces as *APIError and does
// NOT overwrite an existing token on the client.
func (s *TokenIssueSuite) TestRegular401WithoutTokenIssuedReturnsAPIError() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "bad token"})
	}))
	defer ts.Close()

	c := client.New(ts.URL)
	c.SetToken("existing")
	auth := client.NewAuthClient(c)

	tokens, err := auth.ListTokens(context.Background())
	s.Nil(tokens)
	s.Require().Error(err)

	var apiErr *client.APIError
	s.Require().True(errors.As(err, &apiErr), "err should be retrievable as *APIError via errors.As")
	s.Equal(client.ErrCodeUnauthorized, apiErr.Code)
	s.Equal(http.StatusUnauthorized, apiErr.StatusCode)
	s.Equal("existing", c.Token(), "client token must not be overwritten by a plain 401")
}

// TestTokenIssuedRetryOnlyOnce verifies that if the server keeps issuing
// TOKEN_ISSUED responses, the SDK retries at most once (original + 1 retry)
// and ultimately returns an error rather than looping indefinitely.
func (s *TokenIssueSuite) TestTokenIssuedRetryOnlyOnce() {
	var callCount atomic.Int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code":    "TOKEN_ISSUED",
				"message": "First client — token auto-issued",
			},
			"token": "rolling-token",
		})
	}))
	defer ts.Close()

	c := client.New(ts.URL)
	auth := client.NewAuthClient(c)

	_, err := auth.ListTokens(context.Background())
	s.Require().Error(err, "SDK must return an error rather than loop forever when the server keeps issuing tokens")
	s.Equal(int32(2), callCount.Load(), "handler must see exactly 2 calls: original + one retry (no infinite loop)")
}

// TestTokenIssuedMalformedBodyFallsThroughToAPIError verifies that a 401
// whose body claims TOKEN_ISSUED but is missing the `token` field is treated
// as a plain 401 (APIError with ErrCodeUnauthorized) — no retry, no token
// stored.
func (s *TokenIssueSuite) TestTokenIssuedMalformedBodyFallsThroughToAPIError() {
	var callCount atomic.Int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		// Nested error code but no token field.
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code": "TOKEN_ISSUED",
			},
		})
	}))
	defer ts.Close()

	c := client.New(ts.URL)
	auth := client.NewAuthClient(c)

	_, err := auth.ListTokens(context.Background())
	s.Require().Error(err)

	var apiErr *client.APIError
	s.Require().True(errors.As(err, &apiErr), "malformed TOKEN_ISSUED should surface as *APIError")
	s.Equal(client.ErrCodeUnauthorized, apiErr.Code)
	s.Equal(http.StatusUnauthorized, apiErr.StatusCode)
	s.Equal("", c.Token(), "no token should be stored when the TOKEN_ISSUED body is malformed")
	s.Equal(int32(1), callCount.Load(), "malformed TOKEN_ISSUED must not trigger a retry")
}
