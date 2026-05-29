package adapters_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/cmd/cue/adapters"
	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/pkg/client"
)

// MessagesAdapterSuite covers DTO → repository.Message translation
// (QueryByStatus) and the Update → ResolveNotification/DismissNotification
// routing. A single httptest round-trip pins both code paths through
// the SDK.
type MessagesAdapterSuite struct {
	suite.Suite
}

func TestMessagesAdapter(t *testing.T) {
	suite.Run(t, new(MessagesAdapterSuite))
}

// AC: ListMessages → QueryByStatus translates the wire shape into the
// repository model and Update("Resolved") routes to the resolve
// endpoint. Driven through the real SDK against an httptest server.
func (s *MessagesAdapterSuite) TestRoundTrip() {
	id := uuid.New()
	srcAccount := uuid.New()
	var resolveCalls int

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/messages", func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal("Notified", r.URL.Query().Get("status"))
		w.Header().Set("Content-Type", "application/json")
		body := map[string]any{
			"messages": []any{
				map[string]any{
					"id":               id.String(),
					"source":           "slack",
					"source_account":   srcAccount.String(),
					"sender":           "alice",
					"channel":          "#general",
					"content":          "deadline soon",
					"importance_score": 8.0,
					"confidence_score": 0.9,
					"status":           "Notified",
					"created_at":       "2026-04-27T09:00:00Z",
				},
			},
			"total": 1,
		}
		s.Require().NoError(json.NewEncoder(w).Encode(body))
	})
	mux.HandleFunc("/api/v1/notifications/"+id.String()+"/resolve", func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPost, r.Method)
		resolveCalls++
		w.WriteHeader(http.StatusNoContent)
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	api := client.New(ts.URL)
	api.SetToken("test-token")
	mc := client.NewMessageClient(api)
	a := adapters.NewMessagesAdapter(mc)

	ctx := context.Background()

	// QueryByStatus.
	msgs, err := a.QueryByStatus(ctx, "Notified")
	s.Require().NoError(err)
	s.Require().Len(msgs, 1)
	got := msgs[0]
	s.Equal(id, got.ID)
	s.Equal("slack", got.Source)
	s.Equal(srcAccount.String(), got.SourceAccount)
	s.Equal("alice", got.Sender)
	s.Equal("#general", got.Channel)
	s.Equal("deadline soon", got.RawContent)
	s.InDelta(8.0, got.ImportanceScore, 0.001)
	s.InDelta(0.9, got.ConfidenceScore, 0.001)
	s.Equal("Notified", got.Status)
	s.False(got.CreatedAt.IsZero(), "CreatedAt must be parsed from RFC3339")

	// Update("Resolved") → POST /resolve.
	got.Status = "Resolved"
	s.Require().NoError(a.Update(ctx, got))
	s.Equal(1, resolveCalls)
}

// AC: Update with an unsupported status surfaces a clear error so the
// caller cannot accidentally fall through to a no-op.
func (s *MessagesAdapterSuite) TestUpdateRejectsNonTerminalStatus() {
	a := adapters.NewMessagesAdapter(nil)
	err := a.Update(context.Background(), &repository.Message{Status: "Notified"})
	s.Require().Error(err)
	s.Contains(err.Error(), "not a writable transition")
}
