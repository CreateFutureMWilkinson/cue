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

// FeedbackSuite covers the FeedbackClient adapter over /api/v1/buffer.
type FeedbackSuite struct {
	suite.Suite
}

func TestFeedback(t *testing.T) {
	suite.Run(t, new(FeedbackSuite))
}

// testBufferID is a deterministic UUID used across suite tests so path
// interpolation can be asserted directly.
var testBufferID = uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")

// testBufferSourceAccountID is a deterministic UUID used for source_account
// fields in fake server responses so decoding can be asserted.
var testBufferSourceAccountID = uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")

// stringPtr returns a pointer to s. Used to construct optional feedback
// values in rate request bodies.
func stringPtr(s string) *string { return &s }

// TestListBufferedSendsPagination verifies that ListOptions limit and offset
// are encoded as query parameters on /api/v1/buffer.
func (s *FeedbackSuite) TestListBufferedSendsPagination() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal("/api/v1/buffer", r.URL.Path)
		s.Equal("25", r.URL.Query().Get("limit"))
		s.Equal("5", r.URL.Query().Get("offset"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"messages": []any{},
			"total":    0,
			"count":    0,
		})
	}))
	defer ts.Close()

	fc := client.NewFeedbackClient(client.New(ts.URL))
	msgs, total, err := fc.ListBuffered(context.Background(), client.ListOptions{
		Limit:  25,
		Offset: 5,
	})
	s.Require().NoError(err)
	s.Empty(msgs)
	s.Equal(0, total)
}

// TestListBufferedDecodesResponse verifies snake_case JSON fields on the
// buffer list payload decode into the typed BufferedMessage struct.
func (s *FeedbackSuite) TestListBufferedDecodesResponse() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("/api/v1/buffer", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"messages": []map[string]any{
				{
					"id":               testBufferID.String(),
					"source":           "slack",
					"source_account":   testBufferSourceAccountID.String(),
					"sender":           "alice",
					"channel":          "general",
					"content":          "maybe important",
					"importance_score": 7.0,
					"confidence_score": 0.3,
					"reasoning":        "low confidence",
					"created_at":       "2026-04-01T12:00:00Z",
				},
				{
					"id":               uuid.NewString(),
					"source":           "email",
					"source_account":   uuid.NewString(),
					"sender":           "bob@example.com",
					"channel":          "inbox",
					"content":          "another",
					"importance_score": 6.5,
					"confidence_score": 0.5,
					"reasoning":        "borderline",
					"created_at":       "2026-04-02T09:30:00Z",
				},
			},
			"total": 2,
			"count": 2,
		})
	}))
	defer ts.Close()

	fc := client.NewFeedbackClient(client.New(ts.URL))
	msgs, total, err := fc.ListBuffered(context.Background(), client.ListOptions{})
	s.Require().NoError(err)
	s.Equal(2, total)
	s.Require().Len(msgs, 2)

	first := msgs[0]
	s.Equal(testBufferID, first.ID)
	s.Equal("slack", first.Source)
	s.Equal(testBufferSourceAccountID, first.SourceAccount)
	s.Equal("alice", first.Sender)
	s.Equal("general", first.Channel)
	s.Equal("maybe important", first.Content)
	s.InDelta(7.0, first.ImportanceScore, 1e-9)
	s.InDelta(0.3, first.ConfidenceScore, 1e-9)
	s.Equal("low confidence", first.Reasoning)
	s.Equal("2026-04-01T12:00:00Z", first.CreatedAt)
}

// TestGetBufferedReturnsMessage verifies that GetBuffered issues
// GET /api/v1/buffer/{id} and decodes the BufferedMessage payload.
func (s *FeedbackSuite) TestGetBufferedReturnsMessage() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal("/api/v1/buffer/"+testBufferID.String(), r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":               testBufferID.String(),
			"source":           "slack",
			"source_account":   testBufferSourceAccountID.String(),
			"sender":           "alice",
			"channel":          "general",
			"content":          "maybe important",
			"importance_score": 7.0,
			"confidence_score": 0.3,
			"reasoning":        "low confidence",
			"created_at":       "2026-04-01T12:00:00Z",
		})
	}))
	defer ts.Close()

	fc := client.NewFeedbackClient(client.New(ts.URL))
	msg, err := fc.GetBuffered(context.Background(), testBufferID)
	s.Require().NoError(err)
	s.Require().NotNil(msg)
	s.Equal(testBufferID, msg.ID)
	s.Equal("slack", msg.Source)
	s.Equal(testBufferSourceAccountID, msg.SourceAccount)
	s.Equal("alice", msg.Sender)
	s.Equal("general", msg.Channel)
	s.Equal("maybe important", msg.Content)
	s.InDelta(7.0, msg.ImportanceScore, 1e-9)
	s.InDelta(0.3, msg.ConfidenceScore, 1e-9)
	s.Equal("low confidence", msg.Reasoning)
	s.Equal("2026-04-01T12:00:00Z", msg.CreatedAt)
}

// TestBufferStatsReturnsCounts verifies that BufferStats decodes the
// total_buffered and by_source map fields from /api/v1/buffer/stats.
func (s *FeedbackSuite) TestBufferStatsReturnsCounts() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal("/api/v1/buffer/stats", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"total_buffered": 42,
			"by_source": map[string]int{
				"slack": 10,
				"email": 32,
			},
		})
	}))
	defer ts.Close()

	fc := client.NewFeedbackClient(client.New(ts.URL))
	stats, err := fc.BufferStats(context.Background())
	s.Require().NoError(err)
	s.Require().NotNil(stats)
	s.Equal(42, stats.TotalBuffered)
	s.Require().NotNil(stats.BySource)
	s.Equal(10, stats.BySource["slack"])
	s.Equal(32, stats.BySource["email"])
}

// TestBufferStatsPathPrecedenceOverIDPath verifies that the SDK sends the
// literal /api/v1/buffer/stats URL — not /api/v1/buffer/{id}. The server
// registers both routes under the /buffer/ prefix, so the SDK must NOT
// treat "stats" as a path-interpolated id segment.
func (s *FeedbackSuite) TestBufferStatsPathPrecedenceOverIDPath() {
	var capturedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"total_buffered": 0,
			"by_source":      map[string]int{},
		})
	}))
	defer ts.Close()

	fc := client.NewFeedbackClient(client.New(ts.URL))
	_, err := fc.BufferStats(context.Background())
	s.Require().NoError(err)
	s.Equal("/api/v1/buffer/stats", capturedPath,
		"BufferStats must hit the literal stats path, not /buffer/{id}")
}

// TestRateBufferedPostsBodyWithRatingAndFeedback verifies that RateBuffered
// POSTs a body containing the integer rating and optional feedback string.
func (s *FeedbackSuite) TestRateBufferedPostsBodyWithRatingAndFeedback() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPost, r.Method)
		s.Equal("/api/v1/buffer/"+testBufferID.String()+"/rate", r.URL.Path)

		var body struct {
			Rating   int     `json:"rating"`
			Feedback *string `json:"feedback"`
		}
		s.Require().NoError(json.NewDecoder(r.Body).Decode(&body))
		s.Equal(8, body.Rating)
		s.Require().NotNil(body.Feedback)
		s.Equal("useful", *body.Feedback)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "rated",
		})
	}))
	defer ts.Close()

	fc := client.NewFeedbackClient(client.New(ts.URL))
	err := fc.RateBuffered(context.Background(), testBufferID, 8, stringPtr("useful"))
	s.Require().NoError(err)
}

// TestRateBufferedOmitsFeedbackWhenNil verifies that a nil feedback pointer
// sends either an explicit null or an omitted field — decoded server-side
// as a nil *string.
func (s *FeedbackSuite) TestRateBufferedOmitsFeedbackWhenNil() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPost, r.Method)
		s.Equal("/api/v1/buffer/"+testBufferID.String()+"/rate", r.URL.Path)

		var body struct {
			Rating   int     `json:"rating"`
			Feedback *string `json:"feedback"`
		}
		s.Require().NoError(json.NewDecoder(r.Body).Decode(&body))
		s.Equal(5, body.Rating)
		s.Nil(body.Feedback, "feedback must decode to nil when caller passed nil")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "rated",
		})
	}))
	defer ts.Close()

	fc := client.NewFeedbackClient(client.New(ts.URL))
	err := fc.RateBuffered(context.Background(), testBufferID, 5, nil)
	s.Require().NoError(err)
}

// TestRateBufferedConflictReturnsAPIError verifies that a 409 from the
// server surfaces as an *APIError with ErrCodeConflict (e.g. the message
// is no longer in Buffered state).
func (s *FeedbackSuite) TestRateBufferedConflictReturnsAPIError() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("/api/v1/buffer/"+testBufferID.String()+"/rate", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "not buffered",
		})
	}))
	defer ts.Close()

	fc := client.NewFeedbackClient(client.New(ts.URL))
	err := fc.RateBuffered(context.Background(), testBufferID, 8, stringPtr("useful"))
	s.Require().Error(err)

	var apiErr *client.APIError
	s.Require().True(errors.As(err, &apiErr), "expected *APIError, got %T", err)
	s.Equal(client.ErrCodeConflict, apiErr.Code)
	s.Equal(http.StatusConflict, apiErr.StatusCode)
}

// TestDeleteBufferedDeletes204 verifies that DeleteBuffered issues
// DELETE /api/v1/buffer/{id} and tolerates a 204 No Content response with
// an empty body (the doJSON transport must not attempt to decode empty).
func (s *FeedbackSuite) TestDeleteBufferedDeletes204() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodDelete, r.Method)
		s.Equal("/api/v1/buffer/"+testBufferID.String(), r.URL.Path)

		// Drain body to match real client behavior; then respond 204 empty.
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	fc := client.NewFeedbackClient(client.New(ts.URL))
	err := fc.DeleteBuffered(context.Background(), testBufferID)
	s.Require().NoError(err)
}
