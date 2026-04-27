package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/server/handler"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// BufferHandlerSuite tests the buffer handler endpoints.
type BufferHandlerSuite struct {
	suite.Suite
}

func TestBufferHandler(t *testing.T) {
	suite.Run(t, new(BufferHandlerSuite))
}

func (s *BufferHandlerSuite) TestListBufferedReturnsBufferedMessages() {
	now := time.Now().UTC().Truncate(time.Second)

	msg1 := &repository.Message{
		ID:              uuid.New(),
		Source:          "slack",
		SourceAccount:   "T12345",
		Sender:          "bob",
		Channel:         "#deployments",
		RawContent:      "Deploy scheduled for tonight",
		ImportanceScore: 7.5,
		ConfidenceScore: 0.6,
		Reasoning:       "Deployment notice with potential impact",
		Status:          "Buffered",
		CreatedAt:       now.Add(-10 * time.Minute),
	}
	msg2 := &repository.Message{
		ID:              uuid.New(),
		Source:          "email",
		SourceAccount:   "work@example.com",
		Sender:          "carol@example.com",
		Channel:         "INBOX",
		RawContent:      "Q2 planning meeting moved",
		ImportanceScore: 7.0,
		ConfidenceScore: 0.5,
		Reasoning:       "Schedule change but low confidence",
		Status:          "Buffered",
		CreatedAt:       now.Add(-5 * time.Minute),
	}

	mock := &mockMessageQuerier{
		messages: []*repository.Message{msg1, msg2},
		total:    8,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/buffer?limit=50&offset=0", nil)
	rec := httptest.NewRecorder()

	handler.ListBufferedHandler(mock)(rec, req)

	s.Equal(http.StatusOK, rec.Code, "expected 200 OK")

	var body map[string]any
	err := json.NewDecoder(rec.Body).Decode(&body)
	s.Require().NoError(err, "response body should be valid JSON")

	messages, ok := body["messages"].([]any)
	s.Require().True(ok, "response should have a 'messages' array")
	s.Len(messages, 2, "should return 2 buffered messages")

	total, ok := body["total"].(float64)
	s.Require().True(ok, "response should have a 'total' field")
	s.Equal(float64(8), total, "total should be 8 (total matching, not page count)")

	count, ok := body["count"].(float64)
	s.Require().True(ok, "response should have a 'count' field")
	s.Equal(float64(2), count, "count should match the number of items in the page")

	// Verify the mock was called with the correct filter.
	s.Equal("Buffered", mock.capturedFilter.Status)
	s.Equal(50, mock.capturedFilter.Limit)
	s.Equal(0, mock.capturedFilter.Offset)
}
