package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/server"
	"github.com/CreateFutureMWilkinson/cue/internal/server/handler"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// mockEventBroadcaster implements handler.EventBroadcaster for testing.
type mockEventBroadcaster struct {
	broadcasts [][]byte
}

func (m *mockEventBroadcaster) Broadcast(data []byte) error {
	m.broadcasts = append(m.broadcasts, data)
	return nil
}

// mockAuthTokenCreator implements handler.AuthTokenCreator for testing.
type mockAuthTokenCreator struct {
	created []*repository.AuthToken
}

func (m *mockAuthTokenCreator) Create(_ context.Context, token *repository.AuthToken) error {
	m.created = append(m.created, token)
	return nil
}

// pairingStoreAdapter wraps server.PairingStore to implement handler.PairingStorer.
// The handler package defines its own PairingRequest type, so we convert between them.
type pairingStoreAdapter struct {
	store *server.PairingStore
}

func (a *pairingStoreAdapter) Create(label string) *handler.PairingRequest {
	req := a.store.Create(label)
	return &handler.PairingRequest{
		ID:        req.ID,
		Label:     req.Label,
		Code:      req.Code,
		ExpiresAt: req.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
		Status:    req.Status,
	}
}

func (a *pairingStoreAdapter) Get(id uuid.UUID) (*handler.PairingRequest, bool) {
	req, ok := a.store.Get(id)
	if !ok {
		return nil, false
	}
	return &handler.PairingRequest{
		ID:        req.ID,
		Label:     req.Label,
		Code:      req.Code,
		ExpiresAt: req.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
		Status:    req.Status,
		Token:     req.Token,
	}, true
}

func (a *pairingStoreAdapter) Approve(id uuid.UUID, token string) error {
	return a.store.Approve(id, token)
}

func (a *pairingStoreAdapter) Deny(id uuid.UUID) error {
	return a.store.Deny(id)
}

// AuthHandlerSuite tests the pairing HTTP handlers.
type AuthHandlerSuite struct {
	suite.Suite
	store     *server.PairingStore
	adapter   *pairingStoreAdapter
	hub       *mockEventBroadcaster
	tokenRepo *mockAuthTokenCreator
}

func TestAuthHandler(t *testing.T) {
	suite.Run(t, new(AuthHandlerSuite))
}

func (s *AuthHandlerSuite) SetupTest() {
	s.store = server.NewPairingStore()
	s.adapter = &pairingStoreAdapter{store: s.store}
	s.hub = &mockEventBroadcaster{}
	s.tokenRepo = &mockAuthTokenCreator{}
}

// TestInitiatePairingReturns202 verifies POST /api/v1/auth/pair returns 202
// with request_id and code in the JSON response.
func (s *AuthHandlerSuite) TestInitiatePairingReturns202() {
	h := handler.InitiatePairingHandler(s.adapter, s.hub)

	body := `{"label": "phone"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/pair", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	s.Equal(http.StatusAccepted, rec.Code, "expected 202 Accepted")

	var resp map[string]any
	err := json.NewDecoder(rec.Body).Decode(&resp)
	s.Require().NoError(err)

	s.Contains(resp, "request_id", "response must include request_id")
	s.Contains(resp, "code", "response must include code")

	// code should be a 6-digit string
	code, ok := resp["code"].(string)
	s.True(ok, "code should be a string")
	s.Regexp(`^\d{6}$`, code)

	// Hub should have received a broadcast
	s.NotEmpty(s.hub.broadcasts, "expected a pairing_request event broadcast")
}

// TestPollPairingReturnsPending verifies GET /api/v1/auth/pair/{id}
// returns the pending status for a freshly created pairing request.
func (s *AuthHandlerSuite) TestPollPairingReturnsPending() {
	// Create a pairing request via the store directly.
	created := s.store.Create("laptop")

	h := handler.PollPairingHandler(s.adapter)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/pair/"+created.ID.String(), nil)
	req.SetPathValue("id", created.ID.String())
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)

	var resp map[string]any
	err := json.NewDecoder(rec.Body).Decode(&resp)
	s.Require().NoError(err)

	s.Equal("pending", resp["status"])
	// Token should not be present for pending requests.
	s.Empty(resp["token"])
}

// TestApprovePairingIssuesToken verifies the full approve flow:
// create pairing, approve it, then poll to see "approved" with a token.
func (s *AuthHandlerSuite) TestApprovePairingIssuesToken() {
	created := s.store.Create("desktop")

	// Approve the pairing request.
	approveHandler := handler.ApprovePairingHandler(s.adapter, s.tokenRepo, s.hub)

	approveReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/pair/"+created.ID.String()+"/approve", nil)
	approveReq.SetPathValue("id", created.ID.String())
	approveRec := httptest.NewRecorder()

	approveHandler.ServeHTTP(approveRec, approveReq)

	s.Equal(http.StatusOK, approveRec.Code, "expected 200 OK for approve")

	var approveResp map[string]any
	err := json.NewDecoder(approveRec.Body).Decode(&approveResp)
	s.Require().NoError(err)
	s.Equal("approved", approveResp["status"])

	// The token repo should have received a Create call.
	s.NotEmpty(s.tokenRepo.created, "expected token to be persisted in repo")

	// Poll to verify the pairing request is now approved with a token.
	pollHandler := handler.PollPairingHandler(s.adapter)

	pollReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/pair/"+created.ID.String(), nil)
	pollReq.SetPathValue("id", created.ID.String())
	pollRec := httptest.NewRecorder()

	pollHandler.ServeHTTP(pollRec, pollReq)

	s.Equal(http.StatusOK, pollRec.Code)

	var pollResp map[string]any
	err = json.NewDecoder(pollRec.Body).Decode(&pollResp)
	s.Require().NoError(err)

	s.Equal("approved", pollResp["status"])
	token, ok := pollResp["token"].(string)
	s.True(ok, "token should be a string")
	s.NotEmpty(token, "approved pairing must include a bearer token")

	// Hub should have received a pairing_resolved broadcast.
	s.NotEmpty(s.hub.broadcasts, "expected a pairing_resolved event broadcast")
}

// TestDenyPairing verifies that denying a pairing request sets status to "denied".
func (s *AuthHandlerSuite) TestDenyPairing() {
	created := s.store.Create("watch")

	denyHandler := handler.DenyPairingHandler(s.adapter, s.hub)

	denyReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/pair/"+created.ID.String()+"/deny", nil)
	denyReq.SetPathValue("id", created.ID.String())
	denyRec := httptest.NewRecorder()

	denyHandler.ServeHTTP(denyRec, denyReq)

	s.Equal(http.StatusOK, denyRec.Code)

	var denyResp map[string]any
	err := json.NewDecoder(denyRec.Body).Decode(&denyResp)
	s.Require().NoError(err)
	s.Equal("denied", denyResp["status"])

	// Poll to verify the pairing request is now denied.
	pollHandler := handler.PollPairingHandler(s.adapter)

	pollReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/pair/"+created.ID.String(), nil)
	pollReq.SetPathValue("id", created.ID.String())
	pollRec := httptest.NewRecorder()

	pollHandler.ServeHTTP(pollRec, pollReq)

	s.Equal(http.StatusOK, pollRec.Code)

	var pollResp map[string]any
	err = json.NewDecoder(pollRec.Body).Decode(&pollResp)
	s.Require().NoError(err)
	s.Equal("denied", pollResp["status"])

	// Hub should have received a pairing_resolved broadcast.
	s.NotEmpty(s.hub.broadcasts, "expected a pairing_resolved event broadcast")
}
