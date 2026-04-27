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

func (s *BufferHandlerSuite) TestGetBufferedReturnsFullDetail() {
	now := time.Now().UTC().Truncate(time.Second)
	msgID := uuid.New()

	msg := &repository.Message{
		ID:              msgID,
		Source:          "slack",
		SourceAccount:   "T12345",
		Channel:         "#deployments",
		Sender:          "bob",
		MessageID:       "slack-msg-042",
		RawContent:      "Deploy scheduled for tonight",
		ImportanceScore: 7.5,
		ConfidenceScore: 0.6,
		Reasoning:       "Deployment notice with potential impact",
		Status:          "Buffered",
		CreatedAt:       now.Add(-10 * time.Minute),
		UpdatedAt:       now.Add(-9 * time.Minute),
	}

	mock := &mockMessageQuerier{
		queryByIDMessage: msg,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/buffer/"+msgID.String(), nil)
	req.SetPathValue("id", msgID.String())
	rec := httptest.NewRecorder()

	handler.GetBufferedHandler(mock)(rec, req)

	s.Equal(http.StatusOK, rec.Code, "expected 200 OK")

	var body map[string]any
	err := json.NewDecoder(rec.Body).Decode(&body)
	s.Require().NoError(err, "response body should be valid JSON")

	s.Equal(msgID.String(), body["id"])
	s.Equal("slack", body["source"])
	s.Equal("T12345", body["source_account"])
	s.Equal("#deployments", body["channel"])
	s.Equal("bob", body["sender"])
	s.Equal("slack-msg-042", body["message_id"])
	s.Equal("Deploy scheduled for tonight", body["content"])
	s.Equal(7.5, body["importance_score"])
	s.Equal(0.6, body["confidence_score"])
	s.Equal("Deployment notice with potential impact", body["reasoning"])
	s.Equal("Buffered", body["status"])
	s.Equal(msg.CreatedAt.Format(time.RFC3339), body["created_at"])
	s.Equal(msg.UpdatedAt.Format(time.RFC3339), body["updated_at"])
}

func (s *BufferHandlerSuite) TestGetBufferedReturns404ForNonBufferedMessage() {
	now := time.Now().UTC().Truncate(time.Second)
	msgID := uuid.New()

	msg := &repository.Message{
		ID:              msgID,
		Source:          "slack",
		SourceAccount:   "T12345",
		Channel:         "#incidents",
		Sender:          "alice",
		MessageID:       "slack-msg-099",
		RawContent:      "Server is down",
		ImportanceScore: 9.0,
		ConfidenceScore: 0.95,
		Reasoning:       "Production outage",
		Status:          "Notified",
		CreatedAt:       now.Add(-5 * time.Minute),
		UpdatedAt:       now.Add(-4 * time.Minute),
	}

	mock := &mockMessageQuerier{
		queryByIDMessage: msg,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/buffer/"+msgID.String(), nil)
	req.SetPathValue("id", msgID.String())
	rec := httptest.NewRecorder()

	handler.GetBufferedHandler(mock)(rec, req)

	s.Equal(http.StatusNotFound, rec.Code, "expected 404 for non-Buffered message")
}

func (s *BufferHandlerSuite) TestGetBufferedReturns404ForUnknownID() {
	mock := &mockMessageQuerier{
		queryByIDErr: repository.ErrNotFound,
	}

	unknownID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/buffer/"+unknownID.String(), nil)
	req.SetPathValue("id", unknownID.String())
	rec := httptest.NewRecorder()

	handler.GetBufferedHandler(mock)(rec, req)

	s.Equal(http.StatusNotFound, rec.Code, "expected 404 Not Found")
}
