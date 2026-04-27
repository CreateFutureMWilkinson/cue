package client_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/pkg/client"
)

// AuthSuite covers the AuthClient adapter over /api/v1/auth/* endpoints.
type AuthSuite struct {
	suite.Suite
}

func TestAuth(t *testing.T) {
	suite.Run(t, new(AuthSuite))
}

// testRequestID is a deterministic UUID used across suite tests to verify
// that the adapter interpolates the request ID into the URL path.
var testRequestID = uuid.MustParse("11111111-1111-1111-1111-111111111111")

// TestInitiatePairingPostsAndReturnsSession verifies that InitiatePairing
// issues POST /api/v1/auth/pair with the label payload and parses the
// 202 Accepted response into a *PairSession.
func (s *AuthSuite) TestInitiatePairingPostsAndReturnsSession() {
	returnedID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPost, r.Method)
		s.Equal("/api/v1/auth/pair", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		s.Require().NoError(err)
		var payload map[string]string
		s.Require().NoError(json.Unmarshal(body, &payload))
		s.Equal("desktop", payload["label"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"request_id": returnedID.String(),
			"code":       "472859",
		})
	}))
	defer ts.Close()

	auth := client.NewAuthClient(client.New(ts.URL))
	session, err := auth.InitiatePairing(context.Background(), "desktop")
	s.Require().NoError(err)
	s.Require().NotNil(session)
	s.Equal(returnedID, session.RequestID)
	s.Equal("472859", session.Code)
}

// TestPollPairingGetsAndReturnsResult verifies that PollPairing issues
// GET /api/v1/auth/pair/{id} and parses an approved result with token.
func (s *AuthSuite) TestPollPairingGetsAndReturnsResult() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal("/api/v1/auth/pair/"+testRequestID.String(), r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "approved",
			"token":  "abc123",
		})
	}))
	defer ts.Close()

	auth := client.NewAuthClient(client.New(ts.URL))
	result, err := auth.PollPairing(context.Background(), testRequestID)
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Equal("approved", result.Status)
	s.Equal("abc123", result.Token)
}

// TestPollPairingPending verifies that PollPairing returns an empty Token
// when the server omits it (status="pending").
func (s *AuthSuite) TestPollPairingPending() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal("/api/v1/auth/pair/"+testRequestID.String(), r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "pending",
		})
	}))
	defer ts.Close()

	auth := client.NewAuthClient(client.New(ts.URL))
	result, err := auth.PollPairing(context.Background(), testRequestID)
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Equal("pending", result.Status)
	s.Equal("", result.Token)
}

// TestApprovePairingPosts verifies that ApprovePairing issues
// POST /api/v1/auth/pair/{id}/approve and returns nil on 200.
func (s *AuthSuite) TestApprovePairingPosts() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPost, r.Method)
		s.Equal("/api/v1/auth/pair/"+testRequestID.String()+"/approve", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "approved"})
	}))
	defer ts.Close()

	auth := client.NewAuthClient(client.New(ts.URL))
	err := auth.ApprovePairing(context.Background(), testRequestID)
	s.NoError(err)
}

// TestDenyPairingPosts verifies that DenyPairing issues
// POST /api/v1/auth/pair/{id}/deny and returns nil on 200.
func (s *AuthSuite) TestDenyPairingPosts() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPost, r.Method)
		s.Equal("/api/v1/auth/pair/"+testRequestID.String()+"/deny", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "denied"})
	}))
	defer ts.Close()

	auth := client.NewAuthClient(client.New(ts.URL))
	err := auth.DenyPairing(context.Background(), testRequestID)
	s.NoError(err)
}

// TestListTokensReturnsArray verifies that ListTokens issues
// GET /api/v1/auth/tokens and parses the JSON array response,
// including snake_case `created_at` / `last_seen` fields.
func (s *AuthSuite) TestListTokensReturnsArray() {
	id1 := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	id2 := uuid.MustParse("44444444-4444-4444-4444-444444444444")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal("/api/v1/auth/tokens", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"id":         id1.String(),
				"label":      "desktop",
				"created_at": "2026-04-01T10:00:00Z",
				"last_seen":  "2026-04-24T09:30:00Z",
				"revoked":    false,
			},
			{
				"id":         id2.String(),
				"label":      "phone",
				"created_at": "2026-04-10T12:00:00Z",
				"last_seen":  "2026-04-20T18:15:00Z",
				"revoked":    true,
			},
		})
	}))
	defer ts.Close()

	auth := client.NewAuthClient(client.New(ts.URL))
	tokens, err := auth.ListTokens(context.Background())
	s.Require().NoError(err)
	s.Require().Len(tokens, 2)

	s.Equal(id1, tokens[0].ID)
	s.Equal("desktop", tokens[0].Label)
	s.Equal("2026-04-01T10:00:00Z", tokens[0].CreatedAt)
	s.Equal("2026-04-24T09:30:00Z", tokens[0].LastSeen)
	s.False(tokens[0].Revoked)

	s.Equal(id2, tokens[1].ID)
	s.Equal("phone", tokens[1].Label)
	s.Equal("2026-04-10T12:00:00Z", tokens[1].CreatedAt)
	s.Equal("2026-04-20T18:15:00Z", tokens[1].LastSeen)
	s.True(tokens[1].Revoked)
}

// TestUpdateTokenLabelPuts verifies that UpdateTokenLabel issues
// PUT /api/v1/auth/tokens/{id} with a `{"label": "..."}` body and
// returns nil on 200.
func (s *AuthSuite) TestUpdateTokenLabelPuts() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPut, r.Method)
		s.Equal("/api/v1/auth/tokens/"+testRequestID.String(), r.URL.Path)

		body, err := io.ReadAll(r.Body)
		s.Require().NoError(err)
		var payload map[string]string
		s.Require().NoError(json.Unmarshal(body, &payload))
		s.Equal("phone", payload["label"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
	}))
	defer ts.Close()

	auth := client.NewAuthClient(client.New(ts.URL))
	err := auth.UpdateTokenLabel(context.Background(), testRequestID, "phone")
	s.NoError(err)
}

// TestRevokeTokenDeletes verifies that RevokeToken issues
// DELETE /api/v1/auth/tokens/{id} and returns nil on 200.
func (s *AuthSuite) TestRevokeTokenDeletes() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodDelete, r.Method)
		s.Equal("/api/v1/auth/tokens/"+testRequestID.String(), r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "revoked"})
	}))
	defer ts.Close()

	auth := client.NewAuthClient(client.New(ts.URL))
	err := auth.RevokeToken(context.Background(), testRequestID)
	s.NoError(err)
}

// TestInitiatePairingOnServerErrorReturnsAPIError verifies that a 500
// response from POST /api/v1/auth/pair surfaces as a *APIError with
// Code=ErrCodeServerError, retrievable via errors.As.
func (s *AuthSuite) TestInitiatePairingOnServerErrorReturnsAPIError() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPost, r.Method)
		s.Equal("/api/v1/auth/pair", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "boom"})
	}))
	defer ts.Close()

	auth := client.NewAuthClient(client.New(ts.URL))
	session, err := auth.InitiatePairing(context.Background(), "desktop")
	s.Nil(session)
	s.Require().Error(err)

	var apiErr *client.APIError
	s.Require().True(errors.As(err, &apiErr), "err should be retrievable as *APIError via errors.As")
	s.Equal(client.ErrCodeServerError, apiErr.Code)
	s.Equal(http.StatusInternalServerError, apiErr.StatusCode)
}
