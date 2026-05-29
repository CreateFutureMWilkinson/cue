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
	"github.com/CreateFutureMWilkinson/cue/pkg/client"
)

// FeedbackAdapterSuite covers the four BufferReviewer methods through
// a single httptest server that handles the four buffer endpoints.
type FeedbackAdapterSuite struct {
	suite.Suite
}

func TestFeedbackAdapter(t *testing.T) {
	suite.Run(t, new(FeedbackAdapterSuite))
}

func (s *FeedbackAdapterSuite) TestRoundTrip() {
	id := uuid.New()
	srcAccount := uuid.New()
	var rateCalls, deleteCalls int
	var lastRating int
	var lastFeedback string

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/buffer", func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		body := map[string]any{
			"messages": []any{
				map[string]any{
					"id":               id.String(),
					"source":           "email",
					"source_account":   srcAccount.String(),
					"sender":           "alice@example.com",
					"channel":          "INBOX",
					"content":          "please review",
					"importance_score": 6.5,
					"confidence_score": 0.6,
					"reasoning":        "below confidence threshold",
					"created_at":       "2026-04-27T08:00:00Z",
				},
			},
			"total": 1,
		}
		s.Require().NoError(json.NewEncoder(w).Encode(body))
	})
	mux.HandleFunc("/api/v1/buffer/stats", func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		body := map[string]any{
			"total_buffered": 17,
			"by_source":      map[string]int{"slack": 9, "email": 8},
		}
		s.Require().NoError(json.NewEncoder(w).Encode(body))
	})
	mux.HandleFunc("/api/v1/buffer/"+id.String()+"/rate", func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPost, r.Method)
		body, err := io.ReadAll(r.Body)
		s.Require().NoError(err)
		var payload struct {
			Rating   int    `json:"rating"`
			Feedback string `json:"feedback"`
		}
		s.Require().NoError(json.Unmarshal(body, &payload))
		lastRating = payload.Rating
		lastFeedback = payload.Feedback
		rateCalls++
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/api/v1/buffer/"+id.String(), func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodDelete, r.Method)
		deleteCalls++
		w.WriteHeader(http.StatusNoContent)
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	api := client.New(ts.URL)
	api.SetToken("test-token")
	a := adapters.NewFeedbackAdapter(client.NewFeedbackClient(api))
	ctx := context.Background()

	// GetBufferedMessages.
	msgs, err := a.GetBufferedMessages(ctx)
	s.Require().NoError(err)
	s.Require().Len(msgs, 1)
	got := msgs[0]
	s.Equal(id, got.ID)
	s.Equal("email", got.Source)
	s.Equal(srcAccount.String(), got.SourceAccount)
	s.Equal("Buffered", got.Status, "adapter must stamp Buffered status")
	s.Equal("below confidence threshold", got.Reasoning)
	s.False(got.CreatedAt.IsZero())

	// CountBuffered.
	count, err := a.CountBuffered(ctx)
	s.Require().NoError(err)
	s.Equal(17, count)

	// SaveRating.
	feedback := "thanks!"
	s.Require().NoError(a.SaveRating(ctx, id, 8, &feedback))
	s.Equal(1, rateCalls)
	s.Equal(8, lastRating)
	s.Equal("thanks!", lastFeedback)

	// DeleteMessage.
	s.Require().NoError(a.DeleteMessage(ctx, id))
	s.Equal(1, deleteCalls)
}
