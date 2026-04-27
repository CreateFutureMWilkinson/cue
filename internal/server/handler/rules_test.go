package handler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/server/handler"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// mockRulesManager implements handler.RulesManager for testing.
type mockRulesManager struct {
	// ListRules
	rules    []*repository.RoutingRule
	listErr  error
	listCall string // "all", "source_type", "source_account"

	// GetRule
	rule   *repository.RoutingRule
	getErr error

	// SaveRule
	saveErr   error
	saveInput *repository.RoutingRule

	// DeleteRule
	deleteErr error

	// ReorderRule
	reorderErr      error
	reorderPriority int

	// ToggleRule
	toggleErr     error
	toggleEnabled bool

	// Captured args
	capturedID            uuid.UUID
	capturedSourceType    string
	capturedSourceAccount uuid.UUID
}

func (m *mockRulesManager) ListRules(_ context.Context) ([]*repository.RoutingRule, error) {
	m.listCall = "all"
	return m.rules, m.listErr
}

func (m *mockRulesManager) ListRulesBySourceType(_ context.Context, sourceType string) ([]*repository.RoutingRule, error) {
	m.listCall = "source_type"
	m.capturedSourceType = sourceType
	return m.rules, m.listErr
}

func (m *mockRulesManager) ListRulesBySourceAccount(_ context.Context, accountID uuid.UUID) ([]*repository.RoutingRule, error) {
	m.listCall = "source_account"
	m.capturedSourceAccount = accountID
	return m.rules, m.listErr
}

func (m *mockRulesManager) GetRule(_ context.Context, id uuid.UUID) (*repository.RoutingRule, error) {
	m.capturedID = id
	return m.rule, m.getErr
}

func (m *mockRulesManager) SaveRule(_ context.Context, rule *repository.RoutingRule) error {
	m.saveInput = rule
	return m.saveErr
}

func (m *mockRulesManager) DeleteRule(_ context.Context, id uuid.UUID) error {
	m.capturedID = id
	return m.deleteErr
}

func (m *mockRulesManager) ReorderRule(_ context.Context, id uuid.UUID, newPriority int) error {
	m.capturedID = id
	m.reorderPriority = newPriority
	return m.reorderErr
}

func (m *mockRulesManager) ToggleRule(_ context.Context, id uuid.UUID, enabled bool) error {
	m.capturedID = id
	m.toggleEnabled = enabled
	return m.toggleErr
}

// RulesHandlerSuite tests the routing rules handler endpoints.
type RulesHandlerSuite struct {
	suite.Suite
}

func TestRulesHandler(t *testing.T) {
	suite.Run(t, new(RulesHandlerSuite))
}

// --- List tests ---

func (s *RulesHandlerSuite) TestListRules_ReturnsAllRules() {
	now := time.Now().UTC().Truncate(time.Second)
	id1 := uuid.New()
	acctID := uuid.New()

	mock := &mockRulesManager{
		rules: []*repository.RoutingRule{
			{
				ID:             id1,
				Name:           "Block spam",
				Priority:       10,
				SourceType:     "email",
				SourceAccount:  &acctID,
				ChannelPattern: "inbox",
				ContentPattern: "buy now",
				MessageType:    "",
				Action:         "ignored",
				Enabled:        true,
				CreatedAt:      now,
				UpdatedAt:      now,
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rules", nil)
	rec := httptest.NewRecorder()

	handler.ListRulesHandler(mock)(rec, req)

	s.Equal(http.StatusOK, rec.Code)

	var body map[string]any
	err := json.NewDecoder(rec.Body).Decode(&body)
	s.Require().NoError(err)

	rules, ok := body["rules"].([]any)
	s.Require().True(ok)
	s.Len(rules, 1)
	s.Equal(float64(1), body["count"])

	rule := rules[0].(map[string]any)
	s.Equal(id1.String(), rule["id"])
	s.Equal("Block spam", rule["name"])
	s.Equal(float64(10), rule["priority"])
	s.Equal("email", rule["source_type"])
	s.Equal(acctID.String(), rule["source_account"])
	s.Equal("inbox", rule["channel_pattern"])
	s.Equal("buy now", rule["content_pattern"])
	s.Equal("ignored", rule["action"])
	s.Equal(true, rule["enabled"])

	s.Equal("all", mock.listCall)
}

func (s *RulesHandlerSuite) TestListRules_FilterBySourceType() {
	mock := &mockRulesManager{
		rules: []*repository.RoutingRule{},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rules?source_type=slack", nil)
	rec := httptest.NewRecorder()

	handler.ListRulesHandler(mock)(rec, req)

	s.Equal(http.StatusOK, rec.Code)
	s.Equal("source_type", mock.listCall)
	s.Equal("slack", mock.capturedSourceType)
}

func (s *RulesHandlerSuite) TestListRules_FilterBySourceAccount() {
	acctID := uuid.New()
	mock := &mockRulesManager{
		rules: []*repository.RoutingRule{},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rules?source_account="+acctID.String(), nil)
	rec := httptest.NewRecorder()

	handler.ListRulesHandler(mock)(rec, req)

	s.Equal(http.StatusOK, rec.Code)
	s.Equal("source_account", mock.listCall)
	s.Equal(acctID, mock.capturedSourceAccount)
}

// --- Get tests ---

func (s *RulesHandlerSuite) TestGetRule_Success() {
	now := time.Now().UTC().Truncate(time.Second)
	id1 := uuid.New()

	mock := &mockRulesManager{
		rule: &repository.RoutingRule{
			ID:         id1,
			Name:       "Channel join alert",
			Priority:   1,
			SourceType: "slack",
			Action:     "notified",
			Enabled:    true,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rules/"+id1.String(), nil)
	req.SetPathValue("id", id1.String())
	rec := httptest.NewRecorder()

	handler.GetRuleHandler(mock)(rec, req)

	s.Equal(http.StatusOK, rec.Code)

	var body map[string]any
	err := json.NewDecoder(rec.Body).Decode(&body)
	s.Require().NoError(err)

	s.Equal(id1.String(), body["id"])
	s.Equal("Channel join alert", body["name"])
	s.Equal(float64(1), body["priority"])
	s.Equal("slack", body["source_type"])
	s.Equal("notified", body["action"])
	s.Equal(true, body["enabled"])
}

func (s *RulesHandlerSuite) TestGetRule_NotFound() {
	mock := &mockRulesManager{
		getErr: repository.ErrNotFound,
	}

	unknownID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/rules/"+unknownID.String(), nil)
	req.SetPathValue("id", unknownID.String())
	rec := httptest.NewRecorder()

	handler.GetRuleHandler(mock)(rec, req)

	s.Equal(http.StatusNotFound, rec.Code)
}

func (s *RulesHandlerSuite) TestGetRule_InvalidUUID() {
	mock := &mockRulesManager{}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rules/not-a-uuid", nil)
	req.SetPathValue("id", "not-a-uuid")
	rec := httptest.NewRecorder()

	handler.GetRuleHandler(mock)(rec, req)

	s.Equal(http.StatusBadRequest, rec.Code)
}

// --- Create tests ---

func (s *RulesHandlerSuite) TestCreateRule_Success() {
	mock := &mockRulesManager{}

	body := `{"name":"New Rule","source_type":"slack","channel_pattern":"^general$","content_pattern":"","message_type":"","action":"notified"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/rules", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.CreateRuleHandler(mock)(rec, req)

	s.Equal(http.StatusCreated, rec.Code)

	// Verify the handler called SaveRule with a populated rule.
	s.Require().NotNil(mock.saveInput)
	s.Equal("New Rule", mock.saveInput.Name)
	s.Equal("slack", mock.saveInput.SourceType)
	s.Equal("^general$", mock.saveInput.ChannelPattern)
	s.Equal("notified", mock.saveInput.Action)
	// Should have generated a UUID.
	s.NotEqual(uuid.Nil, mock.saveInput.ID)

	// Response should include the created rule.
	var respBody map[string]any
	err := json.NewDecoder(rec.Body).Decode(&respBody)
	s.Require().NoError(err)
	s.Equal(mock.saveInput.ID.String(), respBody["id"])
	s.Equal("New Rule", respBody["name"])
}

func (s *RulesHandlerSuite) TestCreateRule_WithSourceAccount() {
	acctID := uuid.New()
	mock := &mockRulesManager{}

	body := fmt.Sprintf(`{"name":"Account Rule","source_type":"email","source_account":"%s","channel_pattern":"","content_pattern":"urgent","message_type":"","action":"notified"}`, acctID.String())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/rules", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.CreateRuleHandler(mock)(rec, req)

	s.Equal(http.StatusCreated, rec.Code)
	s.Require().NotNil(mock.saveInput)
	s.Require().NotNil(mock.saveInput.SourceAccount)
	s.Equal(acctID, *mock.saveInput.SourceAccount)
}

func (s *RulesHandlerSuite) TestCreateRule_BadJSON() {
	mock := &mockRulesManager{}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/rules", strings.NewReader("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.CreateRuleHandler(mock)(rec, req)

	s.Equal(http.StatusBadRequest, rec.Code)
}

func (s *RulesHandlerSuite) TestCreateRule_ValidationError() {
	mock := &mockRulesManager{
		saveErr: repository.ErrInvalidRoutingRule,
	}

	// Invalid source_type will be caught by Validate() in the handler or by SaveRule.
	body := `{"name":"Bad Rule","source_type":"ftp","action":"notified"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/rules", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.CreateRuleHandler(mock)(rec, req)

	s.Equal(http.StatusBadRequest, rec.Code)
}

// --- Update tests ---

func (s *RulesHandlerSuite) TestUpdateRule_Success() {
	id1 := uuid.New()
	mock := &mockRulesManager{}

	body := `{"name":"Updated Rule","source_type":"email","channel_pattern":"inbox","content_pattern":"deadline","message_type":"","action":"notified"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/rules/"+id1.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", id1.String())
	rec := httptest.NewRecorder()

	handler.UpdateRuleHandler(mock)(rec, req)

	s.Equal(http.StatusOK, rec.Code)

	s.Require().NotNil(mock.saveInput)
	s.Equal(id1, mock.saveInput.ID)
	s.Equal("Updated Rule", mock.saveInput.Name)
	s.Equal("email", mock.saveInput.SourceType)
	s.Equal("notified", mock.saveInput.Action)
}

func (s *RulesHandlerSuite) TestUpdateRule_NotFound() {
	mock := &mockRulesManager{
		saveErr: fmt.Errorf("routing rule %s: %w", uuid.Nil, repository.ErrNotFound),
	}

	unknownID := uuid.New()
	body := `{"name":"Ghost","source_type":"slack","action":"ignored"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/rules/"+unknownID.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", unknownID.String())
	rec := httptest.NewRecorder()

	handler.UpdateRuleHandler(mock)(rec, req)

	s.Equal(http.StatusNotFound, rec.Code)
}

func (s *RulesHandlerSuite) TestUpdateRule_ValidationError() {
	id1 := uuid.New()
	mock := &mockRulesManager{
		saveErr: repository.ErrInvalidRoutingRule,
	}

	body := `{"name":"Bad","source_type":"ftp","action":"notified"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/rules/"+id1.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", id1.String())
	rec := httptest.NewRecorder()

	handler.UpdateRuleHandler(mock)(rec, req)

	s.Equal(http.StatusBadRequest, rec.Code)
}

// --- Patch tests ---

func (s *RulesHandlerSuite) TestPatchRule_ReorderPriority() {
	id1 := uuid.New()
	mock := &mockRulesManager{}

	body := `{"priority": 5}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/rules/"+id1.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", id1.String())
	rec := httptest.NewRecorder()

	handler.PatchRuleHandler(mock)(rec, req)

	s.Equal(http.StatusNoContent, rec.Code)
	s.Equal(id1, mock.capturedID)
	s.Equal(5, mock.reorderPriority)
}

func (s *RulesHandlerSuite) TestPatchRule_ToggleEnabled() {
	id1 := uuid.New()
	mock := &mockRulesManager{}

	body := `{"enabled": false}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/rules/"+id1.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", id1.String())
	rec := httptest.NewRecorder()

	handler.PatchRuleHandler(mock)(rec, req)

	s.Equal(http.StatusNoContent, rec.Code)
	s.Equal(id1, mock.capturedID)
	s.Equal(false, mock.toggleEnabled)
}

func (s *RulesHandlerSuite) TestPatchRule_InvalidUUID() {
	mock := &mockRulesManager{}

	body := `{"priority": 5}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/rules/not-a-uuid", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "not-a-uuid")
	rec := httptest.NewRecorder()

	handler.PatchRuleHandler(mock)(rec, req)

	s.Equal(http.StatusBadRequest, rec.Code)
}

func (s *RulesHandlerSuite) TestPatchRule_EmptyBody() {
	id1 := uuid.New()
	mock := &mockRulesManager{}

	body := `{}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/rules/"+id1.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", id1.String())
	rec := httptest.NewRecorder()

	handler.PatchRuleHandler(mock)(rec, req)

	s.Equal(http.StatusBadRequest, rec.Code)
}

// --- Delete tests ---

func (s *RulesHandlerSuite) TestDeleteRule_Success() {
	id1 := uuid.New()
	mock := &mockRulesManager{}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/rules/"+id1.String(), nil)
	req.SetPathValue("id", id1.String())
	rec := httptest.NewRecorder()

	handler.DeleteRuleHandler(mock)(rec, req)

	s.Equal(http.StatusNoContent, rec.Code)
	s.Equal(id1, mock.capturedID)
}

func (s *RulesHandlerSuite) TestDeleteRule_UnknownID_Idempotent() {
	unknownID := uuid.New()
	mock := &mockRulesManager{} // no error = idempotent delete

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/rules/"+unknownID.String(), nil)
	req.SetPathValue("id", unknownID.String())
	rec := httptest.NewRecorder()

	handler.DeleteRuleHandler(mock)(rec, req)

	s.Equal(http.StatusNoContent, rec.Code)
}

func (s *RulesHandlerSuite) TestDeleteRule_InvalidUUID() {
	mock := &mockRulesManager{}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/rules/not-a-uuid", nil)
	req.SetPathValue("id", "not-a-uuid")
	rec := httptest.NewRecorder()

	handler.DeleteRuleHandler(mock)(rec, req)

	s.Equal(http.StatusBadRequest, rec.Code)
}
