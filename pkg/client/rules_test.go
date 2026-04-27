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

// RulesSuite covers the RulesClient adapter over /api/v1/rules.
type RulesSuite struct {
	suite.Suite
}

func TestRules(t *testing.T) {
	suite.Run(t, new(RulesSuite))
}

// testRuleID is a deterministic UUID used across suite tests so path
// interpolation can be asserted directly.
var testRuleID = uuid.MustParse("11111111-2222-3333-4444-555555555555")

// testRuleSourceAccount is a deterministic UUID used as the value of the
// pointer-typed SourceAccount field in a rule item so decode correctness
// for the non-nil case can be asserted.
var testRuleSourceAccount = uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")

// boolPtr returns a pointer to b. Used to construct the optional
// *bool fields on PatchRuleRequest. (stringPtr lives in
// feedback_test.go; intPtr lives in tasks_test.go; both reused here.)
func boolPtr(b bool) *bool { return &b }

// TestListRulesSendsFilters verifies that both RuleFilter.SourceType and
// RuleFilter.SourceAccount are emitted as query parameters on
// GET /api/v1/rules.
func (s *RulesSuite) TestListRulesSendsFilters() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal("/api/v1/rules", r.URL.Path)

		q := r.URL.Query()
		s.Equal("slack", q.Get("source_type"))
		s.Equal(testRuleSourceAccount.String(), q.Get("source_account"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"rules": []any{},
			"count": 0,
		})
	}))
	defer ts.Close()

	rc := client.NewRulesClient(client.New(ts.URL))
	rules, err := rc.ListRules(context.Background(), client.RuleFilter{
		SourceType:    "slack",
		SourceAccount: testRuleSourceAccount.String(),
	})
	s.Require().NoError(err)
	s.Empty(rules)
}

// TestListRulesDecodesResponse verifies that snake_case JSON fields on
// the rule list payload decode correctly. Includes one rule with a
// non-nil SourceAccount (pointer populated) and one with SourceAccount
// omitted (pointer should decode to nil).
func (s *RulesSuite) TestListRulesDecodesResponse() {
	secondID := uuid.MustParse("99999999-9999-9999-9999-999999999999")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("/api/v1/rules", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"rules": []map[string]any{
				{
					"id":              testRuleID.String(),
					"name":            "Slack general",
					"priority":        10,
					"source_type":     "slack",
					"source_account":  testRuleSourceAccount.String(),
					"channel_pattern": "general.*",
					"content_pattern": ".*",
					"message_type":    "channel_join",
					"action":          "notify",
					"enabled":         true,
					"created_at":      "2026-04-01T10:00:00Z",
					"updated_at":      "2026-04-01T10:00:00Z",
				},
				{
					"id":       secondID.String(),
					"name":     "Global mention",
					"priority": 20,
					// source_account omitted intentionally — pointer must decode to nil.
					"source_type":     "email",
					"channel_pattern": ".*",
					"content_pattern": "@me",
					"message_type":    "mention",
					"action":          "buffer",
					"enabled":         false,
					"created_at":      "2026-04-02T10:00:00Z",
					"updated_at":      "2026-04-02T10:00:00Z",
				},
			},
			"count": 2,
		})
	}))
	defer ts.Close()

	rc := client.NewRulesClient(client.New(ts.URL))
	rules, err := rc.ListRules(context.Background(), client.RuleFilter{})
	s.Require().NoError(err)
	s.Require().Len(rules, 2)

	first := rules[0]
	s.Equal(testRuleID, first.ID)
	s.Equal("Slack general", first.Name)
	s.Equal(10, first.Priority)
	s.Equal("slack", first.SourceType)
	s.Require().NotNil(first.SourceAccount, "source_account must decode to non-nil when server sends a UUID")
	s.Equal(testRuleSourceAccount.String(), *first.SourceAccount)
	s.Equal("general.*", first.ChannelPattern)
	s.Equal(".*", first.ContentPattern)
	s.Equal("channel_join", first.MessageType)
	s.Equal("notify", first.Action)
	s.True(first.Enabled)
	s.Equal("2026-04-01T10:00:00Z", first.CreatedAt)
	s.Equal("2026-04-01T10:00:00Z", first.UpdatedAt)

	second := rules[1]
	s.Equal(secondID, second.ID)
	s.Nil(second.SourceAccount, "source_account must decode to nil when server omits the field")
	s.False(second.Enabled)
}

// TestGetRuleReturnsRule verifies that GetRule issues
// GET /api/v1/rules/{id} and decodes the ruleItem payload.
func (s *RulesSuite) TestGetRuleReturnsRule() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal("/api/v1/rules/"+testRuleID.String(), r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":              testRuleID.String(),
			"name":            "Fetched rule",
			"priority":        5,
			"source_type":     "slack",
			"source_account":  testRuleSourceAccount.String(),
			"channel_pattern": "ops.*",
			"content_pattern": "incident",
			"message_type":    "text",
			"action":          "notify",
			"enabled":         true,
			"created_at":      "2026-03-15T09:00:00Z",
			"updated_at":      "2026-04-01T09:00:00Z",
		})
	}))
	defer ts.Close()

	rc := client.NewRulesClient(client.New(ts.URL))
	rule, err := rc.GetRule(context.Background(), testRuleID)
	s.Require().NoError(err)
	s.Require().NotNil(rule)
	s.Equal(testRuleID, rule.ID)
	s.Equal("Fetched rule", rule.Name)
	s.Equal(5, rule.Priority)
	s.Equal("slack", rule.SourceType)
	s.Require().NotNil(rule.SourceAccount)
	s.Equal(testRuleSourceAccount.String(), *rule.SourceAccount)
	s.Equal("ops.*", rule.ChannelPattern)
	s.Equal("incident", rule.ContentPattern)
	s.Equal("text", rule.MessageType)
	s.Equal("notify", rule.Action)
	s.True(rule.Enabled)
	s.Equal("2026-03-15T09:00:00Z", rule.CreatedAt)
	s.Equal("2026-04-01T09:00:00Z", rule.UpdatedAt)
}

// TestCreateRulePostsBodyAndReturnsRule verifies that CreateRule POSTs
// a JSON body matching CreateRuleRequest and decodes the server-
// populated ruleItem response (which includes id, priority default,
// and enabled=true set by the server).
func (s *RulesSuite) TestCreateRulePostsBodyAndReturnsRule() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPost, r.Method)
		s.Equal("/api/v1/rules", r.URL.Path)

		var body struct {
			Name           string  `json:"name"`
			SourceType     string  `json:"source_type"`
			SourceAccount  *string `json:"source_account,omitempty"`
			ChannelPattern string  `json:"channel_pattern"`
			ContentPattern string  `json:"content_pattern"`
			MessageType    string  `json:"message_type"`
			Action         string  `json:"action"`
		}
		s.Require().NoError(json.NewDecoder(r.Body).Decode(&body))
		s.Equal("New rule", body.Name)
		s.Equal("slack", body.SourceType)
		s.Require().NotNil(body.SourceAccount)
		s.Equal(testRuleSourceAccount.String(), *body.SourceAccount)
		s.Equal("alerts.*", body.ChannelPattern)
		s.Equal("critical", body.ContentPattern)
		s.Equal("text", body.MessageType)
		s.Equal("notify", body.Action)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":              testRuleID.String(),
			"name":            "New rule",
			"priority":        100,
			"source_type":     "slack",
			"source_account":  testRuleSourceAccount.String(),
			"channel_pattern": "alerts.*",
			"content_pattern": "critical",
			"message_type":    "text",
			"action":          "notify",
			"enabled":         true,
			"created_at":      "2026-04-24T12:00:00Z",
			"updated_at":      "2026-04-24T12:00:00Z",
		})
	}))
	defer ts.Close()

	rc := client.NewRulesClient(client.New(ts.URL))
	accountStr := testRuleSourceAccount.String()
	rule, err := rc.CreateRule(context.Background(), client.CreateRuleRequest{
		Name:           "New rule",
		SourceType:     "slack",
		SourceAccount:  &accountStr,
		ChannelPattern: "alerts.*",
		ContentPattern: "critical",
		MessageType:    "text",
		Action:         "notify",
	})
	s.Require().NoError(err)
	s.Require().NotNil(rule)
	s.Equal(testRuleID, rule.ID)
	s.Equal("New rule", rule.Name)
	s.Equal(100, rule.Priority)
	s.True(rule.Enabled, "server auto-populates enabled=true on create")
}

// TestUpdateRulePutsFullBody verifies that UpdateRule sends a full PUT
// body including Priority and Enabled (distinct from CreateRule which
// does not carry these). The server response is decoded as the updated
// RoutingRule.
func (s *RulesSuite) TestUpdateRulePutsFullBody() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPut, r.Method)
		s.Equal("/api/v1/rules/"+testRuleID.String(), r.URL.Path)

		raw, err := io.ReadAll(r.Body)
		s.Require().NoError(err)
		var body map[string]json.RawMessage
		s.Require().NoError(json.Unmarshal(raw, &body))

		// Full-update body MUST include priority and enabled (the two
		// fields that distinguish UpdateRuleRequest from CreateRuleRequest).
		s.Contains(body, "priority")
		s.Contains(body, "enabled")
		s.Contains(body, "name")
		s.Contains(body, "source_type")
		s.Contains(body, "channel_pattern")
		s.Contains(body, "content_pattern")
		s.Contains(body, "message_type")
		s.Contains(body, "action")

		var priority int
		s.Require().NoError(json.Unmarshal(body["priority"], &priority))
		s.Equal(25, priority)

		var enabled bool
		s.Require().NoError(json.Unmarshal(body["enabled"], &enabled))
		s.False(enabled)

		var name string
		s.Require().NoError(json.Unmarshal(body["name"], &name))
		s.Equal("Updated rule", name)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":              testRuleID.String(),
			"name":            "Updated rule",
			"priority":        25,
			"source_type":     "slack",
			"channel_pattern": "ops.*",
			"content_pattern": ".*",
			"message_type":    "text",
			"action":          "buffer",
			"enabled":         false,
			"created_at":      "2026-03-01T10:00:00Z",
			"updated_at":      "2026-04-24T12:00:00Z",
		})
	}))
	defer ts.Close()

	rc := client.NewRulesClient(client.New(ts.URL))
	rule, err := rc.UpdateRule(context.Background(), testRuleID, client.UpdateRuleRequest{
		Name:           "Updated rule",
		Priority:       25,
		SourceType:     "slack",
		ChannelPattern: "ops.*",
		ContentPattern: ".*",
		MessageType:    "text",
		Action:         "buffer",
		Enabled:        false,
	})
	s.Require().NoError(err)
	s.Require().NotNil(rule)
	s.Equal(testRuleID, rule.ID)
	s.Equal("Updated rule", rule.Name)
	s.Equal(25, rule.Priority)
	s.False(rule.Enabled)
}

// TestPatchRuleSendsPartialBody verifies that a PatchRuleRequest with
// only Priority set serializes to a body containing priority and NOT
// enabled (omitempty on the nil pointer). Server responds 204.
func (s *RulesSuite) TestPatchRuleSendsPartialBody() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPatch, r.Method)
		s.Equal("/api/v1/rules/"+testRuleID.String(), r.URL.Path)

		raw, err := io.ReadAll(r.Body)
		s.Require().NoError(err)
		var body map[string]json.RawMessage
		s.Require().NoError(json.Unmarshal(raw, &body))

		s.Contains(body, "priority")
		s.NotContains(body, "enabled", "enabled must be omitted when PatchRuleRequest.Enabled is nil")

		var priority int
		s.Require().NoError(json.Unmarshal(body["priority"], &priority))
		s.Equal(50, priority)

		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	rc := client.NewRulesClient(client.New(ts.URL))
	err := rc.PatchRule(context.Background(), testRuleID, client.PatchRuleRequest{
		Priority: intPtr(50),
	})
	s.Require().NoError(err)
}

// TestPatchRuleBothFieldsSet verifies that when both Priority and
// Enabled are non-nil, both are emitted on the PATCH body and the
// client tolerates the 204 response.
func (s *RulesSuite) TestPatchRuleBothFieldsSet() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPatch, r.Method)
		s.Equal("/api/v1/rules/"+testRuleID.String(), r.URL.Path)

		raw, err := io.ReadAll(r.Body)
		s.Require().NoError(err)
		var body map[string]json.RawMessage
		s.Require().NoError(json.Unmarshal(raw, &body))

		s.Contains(body, "priority")
		s.Contains(body, "enabled")

		var priority int
		s.Require().NoError(json.Unmarshal(body["priority"], &priority))
		s.Equal(5, priority)

		var enabled bool
		s.Require().NoError(json.Unmarshal(body["enabled"], &enabled))
		s.False(enabled)

		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	rc := client.NewRulesClient(client.New(ts.URL))
	err := rc.PatchRule(context.Background(), testRuleID, client.PatchRuleRequest{
		Priority: intPtr(5),
		Enabled:  boolPtr(false),
	})
	s.Require().NoError(err)
}

// TestDeleteRuleReturns204 verifies that DeleteRule issues
// DELETE /api/v1/rules/{id} and tolerates a 204 No Content response.
func (s *RulesSuite) TestDeleteRuleReturns204() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodDelete, r.Method)
		s.Equal("/api/v1/rules/"+testRuleID.String(), r.URL.Path)

		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	rc := client.NewRulesClient(client.New(ts.URL))
	err := rc.DeleteRule(context.Background(), testRuleID)
	s.Require().NoError(err)
}

// TestGetRuleNotFoundReturnsAPIError verifies that a 404 from the
// server surfaces as an *APIError with ErrCodeNotFound.
func (s *RulesSuite) TestGetRuleNotFoundReturnsAPIError() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal("/api/v1/rules/"+testRuleID.String(), r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "rule not found",
		})
	}))
	defer ts.Close()

	rc := client.NewRulesClient(client.New(ts.URL))
	rule, err := rc.GetRule(context.Background(), testRuleID)
	s.Require().Error(err)
	s.Nil(rule)

	var apiErr *client.APIError
	s.Require().True(errors.As(err, &apiErr), "expected *APIError, got %T", err)
	s.Equal(client.ErrCodeNotFound, apiErr.Code)
	s.Equal(http.StatusNotFound, apiErr.StatusCode)
}
