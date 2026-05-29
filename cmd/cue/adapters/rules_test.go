package adapters_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/cmd/cue/adapters"
	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/pkg/client"
)

// RulesAdapterSuite covers the routing rule CRUD round-trip and the
// queue-depth probe via /api/v1/messages.
type RulesAdapterSuite struct {
	suite.Suite
}

func TestRulesAdapter(t *testing.T) {
	suite.Run(t, new(RulesAdapterSuite))
}

// AC: list / upsert (create + update) / delete all flow through the
// SDK against a real httptest server.
func (s *RulesAdapterSuite) TestRulesRoundTrip() {
	created := uuid.New()
	srcAccount := uuid.New()
	srcAccountStr := srcAccount.String()
	var deleteCalls int
	var lastUpdate client.UpdateRuleRequest

	ruleDTO := func(id uuid.UUID, priority int, enabled bool) map[string]any {
		return map[string]any{
			"id":              id.String(),
			"name":            "test rule",
			"priority":        priority,
			"source_type":     "slack",
			"source_account":  srcAccountStr,
			"channel_pattern": "#general",
			"content_pattern": "deadline",
			"message_type":    "",
			"action":          "notified",
			"enabled":         enabled,
			"created_at":      "2026-04-27T09:00:00Z",
			"updated_at":      "2026-04-27T09:00:00Z",
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/rules", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			body := map[string]any{
				"rules": []any{ruleDTO(created, 10, true)},
			}
			s.Require().NoError(json.NewEncoder(w).Encode(body))
		case http.MethodPost:
			s.Require().NoError(json.NewEncoder(w).Encode(ruleDTO(created, 100, true)))
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/v1/rules/"+created.String(), func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			body, err := io.ReadAll(r.Body)
			s.Require().NoError(err)
			s.Require().NoError(json.Unmarshal(body, &lastUpdate))
			w.Header().Set("Content-Type", "application/json")
			s.Require().NoError(json.NewEncoder(w).Encode(ruleDTO(created, 5, false)))
		case http.MethodDelete:
			deleteCalls++
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	api := client.New(ts.URL)
	api.SetToken("test-token")
	a := adapters.NewRulesAdapter(client.NewRulesClient(api))
	ctx := context.Background()

	// ListRules.
	rules, err := a.ListRules(ctx)
	s.Require().NoError(err)
	s.Require().Len(rules, 1)
	got := rules[0]
	s.Equal(created, got.ID)
	s.Require().NotNil(got.SourceAccount)
	s.Equal(srcAccount, *got.SourceAccount)
	s.True(got.Enabled)
	s.False(got.CreatedAt.IsZero())

	// UpsertRule on a fresh rule (ID=Nil) hits POST.
	fresh := &repository.RoutingRule{
		Name:           "test rule",
		SourceType:     "slack",
		SourceAccount:  &srcAccount,
		ChannelPattern: "#general",
		ContentPattern: "deadline",
		Action:         "notified",
		Enabled:        true,
	}
	s.Require().NoError(a.UpsertRule(ctx, fresh))
	s.Equal(created, fresh.ID, "Upsert must populate ID from the server response")
	s.Equal(100, fresh.Priority)

	// UpsertRule on an existing rule hits PUT and the wire body
	// carries the priority/enabled the caller set.
	existing := &repository.RoutingRule{
		ID:             created,
		Name:           "test rule",
		Priority:       5,
		SourceType:     "slack",
		SourceAccount:  &srcAccount,
		ChannelPattern: "#general",
		ContentPattern: "deadline",
		Action:         "notified",
		Enabled:        false,
	}
	s.Require().NoError(a.UpsertRule(ctx, existing))
	s.Equal(5, lastUpdate.Priority)
	s.False(lastUpdate.Enabled)

	// DeleteRule.
	s.Require().NoError(a.DeleteRule(ctx, created))
	s.Equal(1, deleteCalls)
}

// AC: PendingCount uses /api/v1/messages with status=Pending and
// returns the server's reported total.
func (s *RulesAdapterSuite) TestQueueDepthPendingCount() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/messages", func(w http.ResponseWriter, r *http.Request) {
		s.Equal("Pending", r.URL.Query().Get("status"))
		w.Header().Set("Content-Type", "application/json")
		s.Require().NoError(json.NewEncoder(w).Encode(map[string]any{
			"messages": []any{},
			"total":    42,
		}))
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	api := client.New(ts.URL)
	api.SetToken("test-token")
	q := adapters.NewQueueDepthAdapter(client.NewMessageClient(api))

	count, err := q.PendingCount(context.Background())
	s.Require().NoError(err)
	s.Equal(42, count)
}

// AC: server-side queue write methods return a clear "unavailable"
// error so callers cannot accidentally rely on them client-side.
func (s *RulesAdapterSuite) TestQueueDepthRejectsWriteMethods() {
	q := adapters.NewQueueDepthAdapter(nil)
	ctx := context.Background()

	s.Require().Error(q.Enqueue(ctx, uuid.New()))
	_, err := q.DequeueOldest(ctx)
	s.Require().Error(err)
	s.Require().Error(q.MarkDone(ctx, uuid.New()))
	s.Require().Error(q.MarkFailed(ctx, uuid.New()))
	s.Require().Error(q.PurgeCompleted(ctx))
	s.Require().Error(q.PurgeAll(ctx))
	_, err = q.ResetProcessing(ctx)
	s.Require().Error(err)
}
