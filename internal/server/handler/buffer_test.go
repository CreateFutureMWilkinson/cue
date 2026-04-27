package handler_test

import (
	"context"
	"encoding/json"
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

// mockBufferRater implements handler.BufferRater for testing.
type mockBufferRater struct {
	saveRatingErr    error
	deleteMessageErr error

	// captured args
	saveRatingID       uuid.UUID
	saveRatingRating   int
	saveRatingFeedback *string

	deleteMessageID uuid.UUID
}

func (m *mockBufferRater) SaveRating(_ context.Context, messageID uuid.UUID, rating int, feedback *string) error {
	m.saveRatingID = messageID
	m.saveRatingRating = rating
	m.saveRatingFeedback = feedback
	return m.saveRatingErr
}

func (m *mockBufferRater) DeleteMessage(_ context.Context, messageID uuid.UUID) error {
	m.deleteMessageID = messageID
	return m.deleteMessageErr
}

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

func (s *BufferHandlerSuite) TestRateBufferedSuccess() {
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

	repo := &mockMessageQuerier{
		queryByIDMessage: msg,
	}
	buf := &mockBufferRater{}

	body := `{"rating": 7, "feedback": "was important"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/buffer/"+msgID.String()+"/rate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", msgID.String())
	rec := httptest.NewRecorder()

	handler.RateBufferedHandler(repo, buf)(rec, req)

	s.Equal(http.StatusOK, rec.Code, "expected 200 OK")

	// Verify SaveRating was called with correct args.
	s.Equal(msgID, buf.saveRatingID, "SaveRating should receive the message ID")
	s.Equal(7, buf.saveRatingRating, "SaveRating should receive rating 7")
	s.Require().NotNil(buf.saveRatingFeedback, "SaveRating should receive non-nil feedback")
	s.Equal("was important", *buf.saveRatingFeedback, "SaveRating should receive the feedback text")

	// Verify response body contains the message ID.
	var respBody map[string]any
	err := json.NewDecoder(rec.Body).Decode(&respBody)
	s.Require().NoError(err, "response body should be valid JSON")
	s.Equal(msgID.String(), respBody["id"], "response should contain the message ID")
}

func (s *BufferHandlerSuite) TestRateBufferedInvalidRatingTooHigh() {
	now := time.Now().UTC().Truncate(time.Second)
	msgID := uuid.New()

	msg := &repository.Message{
		ID:        msgID,
		Status:    "Buffered",
		CreatedAt: now,
		UpdatedAt: now,
	}

	repo := &mockMessageQuerier{
		queryByIDMessage: msg,
	}
	buf := &mockBufferRater{}

	body := `{"rating": 11}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/buffer/"+msgID.String()+"/rate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", msgID.String())
	rec := httptest.NewRecorder()

	handler.RateBufferedHandler(repo, buf)(rec, req)

	s.Equal(http.StatusBadRequest, rec.Code, "expected 400 for rating > 10")
}

func (s *BufferHandlerSuite) TestRateBufferedInvalidRatingNegative() {
	now := time.Now().UTC().Truncate(time.Second)
	msgID := uuid.New()

	msg := &repository.Message{
		ID:        msgID,
		Status:    "Buffered",
		CreatedAt: now,
		UpdatedAt: now,
	}

	repo := &mockMessageQuerier{
		queryByIDMessage: msg,
	}
	buf := &mockBufferRater{}

	body := `{"rating": -1}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/buffer/"+msgID.String()+"/rate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", msgID.String())
	rec := httptest.NewRecorder()

	handler.RateBufferedHandler(repo, buf)(rec, req)

	s.Equal(http.StatusBadRequest, rec.Code, "expected 400 for rating < 0")
}

func (s *BufferHandlerSuite) TestRateBufferedMissingBody() {
	now := time.Now().UTC().Truncate(time.Second)
	msgID := uuid.New()

	msg := &repository.Message{
		ID:        msgID,
		Status:    "Buffered",
		CreatedAt: now,
		UpdatedAt: now,
	}

	repo := &mockMessageQuerier{
		queryByIDMessage: msg,
	}
	buf := &mockBufferRater{}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/buffer/"+msgID.String()+"/rate", nil)
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", msgID.String())
	rec := httptest.NewRecorder()

	handler.RateBufferedHandler(repo, buf)(rec, req)

	s.Equal(http.StatusBadRequest, rec.Code, "expected 400 for missing body")
}

func (s *BufferHandlerSuite) TestRateBufferedNotFound() {
	repo := &mockMessageQuerier{
		queryByIDErr: repository.ErrNotFound,
	}
	buf := &mockBufferRater{}

	unknownID := uuid.New()
	body := `{"rating": 5}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/buffer/"+unknownID.String()+"/rate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", unknownID.String())
	rec := httptest.NewRecorder()

	handler.RateBufferedHandler(repo, buf)(rec, req)

	s.Equal(http.StatusNotFound, rec.Code, "expected 404 Not Found")
}

func (s *BufferHandlerSuite) TestDeleteBufferedSuccess() {
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

	repo := &mockMessageQuerier{
		queryByIDMessage: msg,
	}
	buf := &mockBufferRater{}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/buffer/"+msgID.String(), nil)
	req.SetPathValue("id", msgID.String())
	rec := httptest.NewRecorder()

	handler.DeleteBufferedHandler(repo, buf)(rec, req)

	s.Equal(http.StatusOK, rec.Code, "expected 200 OK")

	// Verify DeleteMessage was called with correct ID.
	s.Equal(msgID, buf.deleteMessageID, "DeleteMessage should receive the message ID")

	// Verify response body contains the message ID.
	var respBody map[string]any
	err := json.NewDecoder(rec.Body).Decode(&respBody)
	s.Require().NoError(err, "response body should be valid JSON")
	s.Equal(msgID.String(), respBody["id"], "response should contain the message ID")
}

func (s *BufferHandlerSuite) TestDeleteBufferedNotFound() {
	repo := &mockMessageQuerier{
		queryByIDErr: repository.ErrNotFound,
	}
	buf := &mockBufferRater{}

	unknownID := uuid.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/buffer/"+unknownID.String(), nil)
	req.SetPathValue("id", unknownID.String())
	rec := httptest.NewRecorder()

	handler.DeleteBufferedHandler(repo, buf)(rec, req)

	s.Equal(http.StatusNotFound, rec.Code, "expected 404 Not Found")
}

func (s *BufferHandlerSuite) TestDeleteBufferedAlreadyResolved() {
	now := time.Now().UTC().Truncate(time.Second)
	resolvedAt := now.Add(-1 * time.Minute)
	msgID := uuid.New()

	msg := &repository.Message{
		ID:         msgID,
		Status:     "Resolved",
		CreatedAt:  now.Add(-10 * time.Minute),
		UpdatedAt:  now.Add(-2 * time.Minute),
		ResolvedAt: &resolvedAt,
	}

	repo := &mockMessageQuerier{
		queryByIDMessage: msg,
	}
	buf := &mockBufferRater{}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/buffer/"+msgID.String(), nil)
	req.SetPathValue("id", msgID.String())
	rec := httptest.NewRecorder()

	handler.DeleteBufferedHandler(repo, buf)(rec, req)

	s.Equal(http.StatusConflict, rec.Code, "expected 409 for already-resolved message")
}

func (s *BufferHandlerSuite) TestRateBufferedAlreadyResolved() {
	now := time.Now().UTC().Truncate(time.Second)
	resolvedAt := now.Add(-1 * time.Minute)
	msgID := uuid.New()

	msg := &repository.Message{
		ID:         msgID,
		Status:     "Resolved",
		CreatedAt:  now.Add(-10 * time.Minute),
		UpdatedAt:  now.Add(-2 * time.Minute),
		ResolvedAt: &resolvedAt,
	}

	repo := &mockMessageQuerier{
		queryByIDMessage: msg,
	}
	buf := &mockBufferRater{}

	body := `{"rating": 5}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/buffer/"+msgID.String()+"/rate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", msgID.String())
	rec := httptest.NewRecorder()

	handler.RateBufferedHandler(repo, buf)(rec, req)

	s.Equal(http.StatusConflict, rec.Code, "expected 409 for already-resolved message")
}
