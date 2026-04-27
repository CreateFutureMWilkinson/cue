package handler_test

import (
	"context"
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

// mockMessageQuerier implements handler.MessageQuerier for testing.
type mockMessageQuerier struct {
	messages []*repository.Message
	total    int
	err      error

	queryByIDMessage *repository.Message
	queryByIDErr     error

	updateCalls []*repository.Message
	updateErr   error

	// captured filter from QueryFiltered call
	capturedFilter repository.MessageFilter
}

func (m *mockMessageQuerier) QueryFiltered(_ context.Context, filter repository.MessageFilter) ([]*repository.Message, int, error) {
	m.capturedFilter = filter
	return m.messages, m.total, m.err
}

func (m *mockMessageQuerier) QueryByID(_ context.Context, id uuid.UUID) (*repository.Message, error) {
	return m.queryByIDMessage, m.queryByIDErr
}

func (m *mockMessageQuerier) Update(_ context.Context, msg *repository.Message) error {
	m.updateCalls = append(m.updateCalls, msg)
	return m.updateErr
}

// NotificationHandlerSuite tests the notification handler endpoints.
type NotificationHandlerSuite struct {
	suite.Suite
}

func TestNotificationHandler(t *testing.T) {
	suite.Run(t, new(NotificationHandlerSuite))
}

func (s *NotificationHandlerSuite) TestListNotificationsReturnsNotifiedMessages() {
	now := time.Now().UTC().Truncate(time.Second)

	msg1 := &repository.Message{
		ID:              uuid.New(),
		Source:          "slack",
		SourceAccount:   "T12345",
		Sender:          "alice",
		Channel:         "#incidents",
		RawContent:      "Server is down in us-east-1",
		ImportanceScore: 9.0,
		ConfidenceScore: 0.95,
		Status:          "Notified",
		CreatedAt:       now.Add(-5 * time.Minute),
	}
	msg2 := &repository.Message{
		ID:              uuid.New(),
		Source:          "email",
		SourceAccount:   "work@example.com",
		Sender:          "bob@example.com",
		Channel:         "INBOX",
		RawContent:      "Deadline moved to Friday",
		ImportanceScore: 8.0,
		ConfidenceScore: 0.88,
		Status:          "Notified",
		CreatedAt:       now.Add(-2 * time.Minute),
	}

	mock := &mockMessageQuerier{
		messages: []*repository.Message{msg1, msg2},
		total:    2,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications?limit=50&offset=0", nil)
	rec := httptest.NewRecorder()

	handler.ListNotificationsHandler(mock)(rec, req)

	s.Equal(http.StatusOK, rec.Code, "expected 200 OK")

	var body map[string]any
	err := json.NewDecoder(rec.Body).Decode(&body)
	s.Require().NoError(err, "response body should be valid JSON")

	notifications, ok := body["notifications"].([]any)
	s.Require().True(ok, "response should have a 'notifications' array")
	s.Len(notifications, 2, "should return 2 notifications")

	total, ok := body["total"].(float64)
	s.Require().True(ok, "response should have a 'total' field")
	s.Equal(float64(2), total, "total should be 2")

	// Verify the mock was called with the correct filter.
	s.Equal("Notified", mock.capturedFilter.Status)
	s.Equal(50, mock.capturedFilter.Limit)
	s.Equal(0, mock.capturedFilter.Offset)
}
