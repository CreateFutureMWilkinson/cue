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

func (s *NotificationHandlerSuite) TestListNotificationsTrimsContentAndAddsSubjectAndWebURL() {
	now := time.Now().UTC().Truncate(time.Second)

	longSlack := strings.Repeat("a", 500)
	slackMsg := &repository.Message{
		ID:              uuid.New(),
		Source:          "slack",
		SourceAccount:   "T1",
		Sender:          "alice",
		Channel:         "general",
		RawContent:      longSlack,
		WebURL:          "https://acme.slack.com",
		ImportanceScore: 9,
		ConfidenceScore: 0.95,
		Status:          "Notified",
		CreatedAt:       now.Add(-1 * time.Minute),
	}
	emailMsg := &repository.Message{
		ID:              uuid.New(),
		Source:          "email",
		SourceAccount:   "user@example.com",
		Sender:          "boss@example.com",
		Channel:         "INBOX",
		Subject:         "Q4 deadline",
		RawContent:      "Subject line then a long body that the UI should not have to receive in the notification pane at all.",
		WebURL:          "https://mail.example.com/u/0/inbox",
		ImportanceScore: 8,
		ConfidenceScore: 0.9,
		Status:          "Notified",
		CreatedAt:       now.Add(-2 * time.Minute),
	}

	mock := &mockMessageQuerier{messages: []*repository.Message{slackMsg, emailMsg}, total: 2}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
	rec := httptest.NewRecorder()
	handler.ListNotificationsHandler(mock)(rec, req)

	s.Require().Equal(http.StatusOK, rec.Code)

	var body struct {
		Notifications []map[string]any `json:"notifications"`
		Total         int              `json:"total"`
	}
	s.Require().NoError(json.NewDecoder(rec.Body).Decode(&body))
	s.Require().Len(body.Notifications, 2)

	slackItem := body.Notifications[0]
	emailItem := body.Notifications[1]

	// Slack: content trimmed to 280 chars, web_url present.
	slackContent, _ := slackItem["content"].(string)
	s.LessOrEqual(len(slackContent), 280, "slack content must be trimmed to <=280 chars on the wire")
	s.Equal("https://acme.slack.com", slackItem["web_url"], "slack item should carry account web_url")
	s.Equal("", slackItem["subject"], "slack item subject should be empty string")

	// Email: subject populated, content empty (no body sent), web_url present.
	s.Equal("Q4 deadline", emailItem["subject"], "email item should carry the subject")
	s.Equal("", emailItem["content"], "email item content must be empty — subject is the visible field")
	s.Equal("https://mail.example.com/u/0/inbox", emailItem["web_url"])
}

func (s *NotificationHandlerSuite) TestGetNotificationReturnsFullDetail() {
	now := time.Now().UTC().Truncate(time.Second)
	msgID := uuid.New()

	msg := &repository.Message{
		ID:              msgID,
		Source:          "slack",
		SourceAccount:   "T12345",
		Channel:         "#incidents",
		Sender:          "alice",
		MessageID:       "slack-msg-001",
		RawContent:      "Server is down in us-east-1",
		ImportanceScore: 9.0,
		ConfidenceScore: 0.95,
		Reasoning:       "Production outage affecting customers",
		Status:          "Notified",
		CreatedAt:       now.Add(-5 * time.Minute),
		UpdatedAt:       now.Add(-4 * time.Minute),
	}

	mock := &mockMessageQuerier{
		queryByIDMessage: msg,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/"+msgID.String(), nil)
	req.SetPathValue("id", msgID.String())
	rec := httptest.NewRecorder()

	handler.GetNotificationHandler(mock)(rec, req)

	s.Equal(http.StatusOK, rec.Code, "expected 200 OK")

	var body map[string]any
	err := json.NewDecoder(rec.Body).Decode(&body)
	s.Require().NoError(err, "response body should be valid JSON")

	s.Equal(msgID.String(), body["id"])
	s.Equal("slack", body["source"])
	s.Equal("T12345", body["source_account"])
	s.Equal("#incidents", body["channel"])
	s.Equal("alice", body["sender"])
	s.Equal("slack-msg-001", body["message_id"])
	s.Equal("Server is down in us-east-1", body["content"])
	s.Equal(9.0, body["importance_score"])
	s.Equal(0.95, body["confidence_score"])
	s.Equal("Production outage affecting customers", body["reasoning"])
	s.Equal("Notified", body["status"])
	s.Equal(msg.CreatedAt.Format(time.RFC3339), body["created_at"])
	s.Equal(msg.UpdatedAt.Format(time.RFC3339), body["updated_at"])
}

func (s *NotificationHandlerSuite) TestGetNotificationNotFound() {
	mock := &mockMessageQuerier{
		queryByIDErr: repository.ErrNotFound,
	}

	unknownID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/"+unknownID.String(), nil)
	req.SetPathValue("id", unknownID.String())
	rec := httptest.NewRecorder()

	handler.GetNotificationHandler(mock)(rec, req)

	s.Equal(http.StatusNotFound, rec.Code, "expected 404 Not Found")
}

func (s *NotificationHandlerSuite) TestResolveNotificationSuccess() {
	now := time.Now().UTC().Truncate(time.Second)
	msgID := uuid.New()

	msg := &repository.Message{
		ID:              msgID,
		Source:          "slack",
		SourceAccount:   "T12345",
		Channel:         "#incidents",
		Sender:          "alice",
		MessageID:       "slack-msg-001",
		RawContent:      "Server is down in us-east-1",
		ImportanceScore: 9.0,
		ConfidenceScore: 0.95,
		Reasoning:       "Production outage affecting customers",
		Status:          "Notified",
		CreatedAt:       now.Add(-5 * time.Minute),
		UpdatedAt:       now.Add(-4 * time.Minute),
	}

	mock := &mockMessageQuerier{
		queryByIDMessage: msg,
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/"+msgID.String()+"/resolve", nil)
	req.SetPathValue("id", msgID.String())
	rec := httptest.NewRecorder()

	handler.ResolveNotificationHandler(mock)(rec, req)

	s.Equal(http.StatusOK, rec.Code, "expected 200 OK")
	s.Require().Len(mock.updateCalls, 1, "expected exactly one Update call")

	updated := mock.updateCalls[0]
	s.Equal("Resolved", updated.Status, "status should be Resolved")
	s.NotNil(updated.ResolvedAt, "ResolvedAt should be set")
}

func (s *NotificationHandlerSuite) TestResolveNotificationNotFound() {
	mock := &mockMessageQuerier{
		queryByIDErr: repository.ErrNotFound,
	}

	unknownID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/"+unknownID.String()+"/resolve", nil)
	req.SetPathValue("id", unknownID.String())
	rec := httptest.NewRecorder()

	handler.ResolveNotificationHandler(mock)(rec, req)

	s.Equal(http.StatusNotFound, rec.Code, "expected 404 Not Found")
}

func (s *NotificationHandlerSuite) TestResolveNotificationAlreadyResolved() {
	now := time.Now().UTC().Truncate(time.Second)
	msgID := uuid.New()
	resolvedAt := now.Add(-1 * time.Minute)

	msg := &repository.Message{
		ID:              msgID,
		Source:          "slack",
		SourceAccount:   "T12345",
		Channel:         "#incidents",
		Sender:          "alice",
		MessageID:       "slack-msg-001",
		RawContent:      "Server is down in us-east-1",
		ImportanceScore: 9.0,
		ConfidenceScore: 0.95,
		Status:          "Resolved",
		CreatedAt:       now.Add(-5 * time.Minute),
		UpdatedAt:       now.Add(-4 * time.Minute),
		ResolvedAt:      &resolvedAt,
	}

	mock := &mockMessageQuerier{
		queryByIDMessage: msg,
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/"+msgID.String()+"/resolve", nil)
	req.SetPathValue("id", msgID.String())
	rec := httptest.NewRecorder()

	handler.ResolveNotificationHandler(mock)(rec, req)

	s.Equal(http.StatusConflict, rec.Code, "expected 409 Conflict")
}

func (s *NotificationHandlerSuite) TestDismissNotificationSuccess() {
	now := time.Now().UTC().Truncate(time.Second)
	msgID := uuid.New()

	msg := &repository.Message{
		ID:              msgID,
		Source:          "slack",
		SourceAccount:   "T12345",
		Channel:         "#incidents",
		Sender:          "alice",
		MessageID:       "slack-msg-001",
		RawContent:      "Server is down in us-east-1",
		ImportanceScore: 9.0,
		ConfidenceScore: 0.95,
		Reasoning:       "Production outage affecting customers",
		Status:          "Notified",
		CreatedAt:       now.Add(-5 * time.Minute),
		UpdatedAt:       now.Add(-4 * time.Minute),
	}

	mock := &mockMessageQuerier{
		queryByIDMessage: msg,
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/"+msgID.String()+"/dismiss", nil)
	req.SetPathValue("id", msgID.String())
	rec := httptest.NewRecorder()

	handler.DismissNotificationHandler(mock)(rec, req)

	s.Equal(http.StatusOK, rec.Code, "expected 200 OK")
	s.Require().Len(mock.updateCalls, 1, "expected exactly one Update call")

	updated := mock.updateCalls[0]
	s.Equal("Ignored", updated.Status, "status should be Ignored")
}

func (s *NotificationHandlerSuite) TestDismissNotificationNotFound() {
	mock := &mockMessageQuerier{
		queryByIDErr: repository.ErrNotFound,
	}

	unknownID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/"+unknownID.String()+"/dismiss", nil)
	req.SetPathValue("id", unknownID.String())
	rec := httptest.NewRecorder()

	handler.DismissNotificationHandler(mock)(rec, req)

	s.Equal(http.StatusNotFound, rec.Code, "expected 404 Not Found")
}
