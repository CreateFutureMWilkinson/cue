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

// MessageHandlerSuite tests the message handler endpoints.
type MessageHandlerSuite struct {
	suite.Suite
}

func TestMessageHandler(t *testing.T) {
	suite.Run(t, new(MessageHandlerSuite))
}

func (s *MessageHandlerSuite) TestListMessagesWithFilters() {
	now := time.Now().UTC().Truncate(time.Second)

	msg := &repository.Message{
		ID:              uuid.New(),
		Source:          "slack",
		SourceAccount:   "T12345",
		Sender:          "alice",
		Channel:         "#general",
		RawContent:      "Please review the PR",
		ImportanceScore: 7.0,
		ConfidenceScore: 0.5,
		Status:          "Buffered",
		CreatedAt:       now.Add(-10 * time.Minute),
	}

	mock := &mockMessageQuerier{
		messages: []*repository.Message{msg},
		total:    1,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages?status=Buffered&source=slack&channel=%23general&limit=25&offset=10", nil)
	rec := httptest.NewRecorder()

	handler.ListMessagesHandler(mock)(rec, req)

	s.Equal(http.StatusOK, rec.Code, "expected 200 OK")

	var body map[string]any
	err := json.NewDecoder(rec.Body).Decode(&body)
	s.Require().NoError(err, "response body should be valid JSON")

	messages, ok := body["messages"].([]any)
	s.Require().True(ok, "response should have a 'messages' array")
	s.Len(messages, 1, "should return 1 message")

	total, ok := body["total"].(float64)
	s.Require().True(ok, "response should have a 'total' field")
	s.Equal(float64(1), total, "total should be 1")

	// Verify the first message item includes a status field.
	item, ok := messages[0].(map[string]any)
	s.Require().True(ok, "message item should be an object")
	s.Equal("Buffered", item["status"], "message item should include status")

	// Verify the mock was called with the correct filter.
	s.Equal("Buffered", mock.capturedFilter.Status)
	s.Equal("slack", mock.capturedFilter.Source)
	s.Equal("#general", mock.capturedFilter.Channel)
	s.Equal(25, mock.capturedFilter.Limit)
	s.Equal(10, mock.capturedFilter.Offset)
}

func (s *MessageHandlerSuite) TestGetMessageReturnsFullDetail() {
	now := time.Now().UTC().Truncate(time.Second)

	msg := &repository.Message{
		ID:              uuid.New(),
		Source:          "email",
		SourceAccount:   "work@example.com",
		Sender:          "boss@example.com",
		Channel:         "INBOX",
		MessageID:       "email-msg-42",
		RawContent:      "Quarterly review is overdue",
		ImportanceScore: 8.5,
		ConfidenceScore: 0.9,
		Reasoning:       "Deadline-related with urgency",
		Status:          "Buffered",
		CreatedAt:       now.Add(-30 * time.Minute),
		UpdatedAt:       now.Add(-25 * time.Minute),
	}

	mock := &mockMessageQuerier{
		queryByIDMessage: msg,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages/"+msg.ID.String(), nil)
	req.SetPathValue("id", msg.ID.String())
	rec := httptest.NewRecorder()

	handler.GetMessageHandler(mock)(rec, req)

	s.Equal(http.StatusOK, rec.Code, "expected 200 OK")

	var body map[string]any
	err := json.NewDecoder(rec.Body).Decode(&body)
	s.Require().NoError(err, "response body should be valid JSON")

	s.Equal(msg.ID.String(), body["id"])
	s.Equal("email", body["source"])
	s.Equal("Buffered", body["status"])
	s.Equal("Quarterly review is overdue", body["content"])
	s.Equal("Deadline-related with urgency", body["reasoning"])
}

func (s *MessageHandlerSuite) TestGetMessageNotFound() {
	mock := &mockMessageQuerier{
		queryByIDErr: repository.ErrNotFound,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages/"+uuid.New().String(), nil)
	req.SetPathValue("id", uuid.New().String())
	rec := httptest.NewRecorder()

	handler.GetMessageHandler(mock)(rec, req)

	s.Equal(http.StatusNotFound, rec.Code, "expected 404 Not Found")
}
