package client_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/pkg/client"
)

// MessageSuite covers the MessageClient adapter over /api/v1/messages and
// /api/v1/notifications.
type MessageSuite struct {
	suite.Suite
}

func TestMessage(t *testing.T) {
	suite.Run(t, new(MessageSuite))
}

// testMessageID is a deterministic UUID used across suite tests so path
// interpolation can be asserted directly.
var testMessageID = uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

// testSourceAccountID is a deterministic UUID used for source_account fields
// in fake server responses so decoding can be asserted.
var testSourceAccountID = uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")

// TestListMessagesSendsFiltersAsQueryParams verifies that ListMessages
// encodes every field of MessageFilter as the expected query parameter.
func (s *MessageSuite) TestListMessagesSendsFiltersAsQueryParams() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal("/api/v1/messages", r.URL.Path)

		q := r.URL.Query()
		s.Equal("Buffered", q.Get("status"))
		s.Equal("slack", q.Get("source"))
		s.Equal("general", q.Get("channel"))
		s.Equal("2026-01-01T00:00:00Z", q.Get("since"))
		s.Equal("25", q.Get("limit"))
		s.Equal("10", q.Get("offset"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"messages": []any{},
			"total":    0,
		})
	}))
	defer ts.Close()

	mc := client.NewMessageClient(client.New(ts.URL))
	msgs, total, err := mc.ListMessages(context.Background(), client.MessageFilter{
		Status:  "Buffered",
		Source:  "slack",
		Channel: "general",
		Since:   "2026-01-01T00:00:00Z",
		Limit:   25,
		Offset:  10,
	})
	s.Require().NoError(err)
	s.Empty(msgs)
	s.Equal(0, total)
}

// TestListMessagesDecodesResponse verifies snake_case JSON fields on the
// message list payload decode into the typed Message struct.
func (s *MessageSuite) TestListMessagesDecodesResponse() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("/api/v1/messages", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"messages": []map[string]any{
				{
					"id":               testMessageID.String(),
					"source":           "slack",
					"source_account":   testSourceAccountID.String(),
					"sender":           "alice",
					"channel":          "general",
					"content":          "hello world",
					"importance_score": 8.5,
					"confidence_score": 0.92,
					"status":           "Notified",
					"created_at":       "2026-04-01T12:00:00Z",
				},
				{
					"id":               uuid.NewString(),
					"source":           "email",
					"source_account":   uuid.NewString(),
					"sender":           "bob@example.com",
					"channel":          "inbox",
					"content":          "another",
					"importance_score": 5.0,
					"confidence_score": 0.4,
					"status":           "Buffered",
					"created_at":       "2026-04-02T09:30:00Z",
				},
			},
			"total": 2,
		})
	}))
	defer ts.Close()

	mc := client.NewMessageClient(client.New(ts.URL))
	msgs, total, err := mc.ListMessages(context.Background(), client.MessageFilter{})
	s.Require().NoError(err)
	s.Equal(2, total)
	s.Require().Len(msgs, 2)

	first := msgs[0]
	s.Equal(testMessageID, first.ID)
	s.Equal("slack", first.Source)
	s.Equal(testSourceAccountID, first.SourceAccount)
	s.Equal("alice", first.Sender)
	s.Equal("general", first.Channel)
	s.Equal("hello world", first.Content)
	s.InDelta(8.5, first.ImportanceScore, 1e-9)
	s.InDelta(0.92, first.ConfidenceScore, 1e-9)
	s.Equal("Notified", first.Status)
	s.Equal("2026-04-01T12:00:00Z", first.CreatedAt)

	s.Equal("Buffered", msgs[1].Status)
}

// TestGetMessageReturnsDetail verifies that GetMessage issues
// GET /api/v1/messages/{id} and decodes the full MessageDetail including
// the optional resolved_at pointer field.
func (s *MessageSuite) TestGetMessageReturnsDetail() {
	resolvedAt := "2026-04-03T15:00:00Z"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal("/api/v1/messages/"+testMessageID.String(), r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":               testMessageID.String(),
			"source":           "slack",
			"source_account":   testSourceAccountID.String(),
			"channel":          "general",
			"sender":           "alice",
			"message_id":       "slack-123",
			"content":          "hello world",
			"importance_score": 9.0,
			"confidence_score": 0.95,
			"reasoning":        "direct mention",
			"status":           "Resolved",
			"created_at":       "2026-04-01T12:00:00Z",
			"updated_at":       "2026-04-03T15:00:00Z",
			"resolved_at":      resolvedAt,
		})
	}))
	defer ts.Close()

	mc := client.NewMessageClient(client.New(ts.URL))
	detail, err := mc.GetMessage(context.Background(), testMessageID)
	s.Require().NoError(err)
	s.Require().NotNil(detail)
	s.Equal(testMessageID, detail.ID)
	s.Equal("slack", detail.Source)
	s.Equal(testSourceAccountID, detail.SourceAccount)
	s.Equal("general", detail.Channel)
	s.Equal("alice", detail.Sender)
	s.Equal("slack-123", detail.MessageID)
	s.Equal("hello world", detail.Content)
	s.InDelta(9.0, detail.ImportanceScore, 1e-9)
	s.InDelta(0.95, detail.ConfidenceScore, 1e-9)
	s.Equal("direct mention", detail.Reasoning)
	s.Equal("Resolved", detail.Status)
	s.Equal("2026-04-01T12:00:00Z", detail.CreatedAt)
	s.Equal("2026-04-03T15:00:00Z", detail.UpdatedAt)
	s.Require().NotNil(detail.ResolvedAt)
	s.Equal(resolvedAt, *detail.ResolvedAt)
}

// TestGetMessageReturnsNotFoundAPIError verifies that a 404 from the server
// surfaces as an *APIError with ErrCodeNotFound.
func (s *MessageSuite) TestGetMessageReturnsNotFoundAPIError() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("/api/v1/messages/"+testMessageID.String(), r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "not found",
		})
	}))
	defer ts.Close()

	mc := client.NewMessageClient(client.New(ts.URL))
	detail, err := mc.GetMessage(context.Background(), testMessageID)
	s.Require().Error(err)
	s.Nil(detail)

	var apiErr *client.APIError
	s.Require().True(errors.As(err, &apiErr), "expected *APIError, got %T", err)
	s.Equal(client.ErrCodeNotFound, apiErr.Code)
	s.Equal(http.StatusNotFound, apiErr.StatusCode)
}

// TestListNotificationsSendsPagination verifies that ListOptions limit and
// offset are encoded as query parameters on /api/v1/notifications.
func (s *MessageSuite) TestListNotificationsSendsPagination() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal("/api/v1/notifications", r.URL.Path)
		s.Equal("25", r.URL.Query().Get("limit"))
		s.Equal("50", r.URL.Query().Get("offset"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"notifications": []any{},
			"total":         0,
		})
	}))
	defer ts.Close()

	mc := client.NewMessageClient(client.New(ts.URL))
	notifs, total, err := mc.ListNotifications(context.Background(), client.ListOptions{
		Limit:  25,
		Offset: 50,
	})
	s.Require().NoError(err)
	s.Empty(notifs)
	s.Equal(0, total)
}

// TestListNotificationsDecodesResponse verifies the decoded notification
// summary has the expected scalar fields (no Status field by design).
func (s *MessageSuite) TestListNotificationsDecodesResponse() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("/api/v1/notifications", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"notifications": []map[string]any{
				{
					"id":               testMessageID.String(),
					"source":           "slack",
					"source_account":   testSourceAccountID.String(),
					"sender":           "alice",
					"channel":          "general",
					"content":          "ping",
					"importance_score": 8.0,
					"confidence_score": 0.9,
					"created_at":       "2026-04-01T12:00:00Z",
				},
			},
			"total": 1,
		})
	}))
	defer ts.Close()

	mc := client.NewMessageClient(client.New(ts.URL))
	notifs, total, err := mc.ListNotifications(context.Background(), client.ListOptions{})
	s.Require().NoError(err)
	s.Equal(1, total)
	s.Require().Len(notifs, 1)

	n := notifs[0]
	s.Equal(testMessageID, n.ID)
	s.Equal("slack", n.Source)
	s.Equal(testSourceAccountID, n.SourceAccount)
	s.Equal("alice", n.Sender)
	s.Equal("general", n.Channel)
	s.Equal("ping", n.Content)
	s.InDelta(8.0, n.ImportanceScore, 1e-9)
	s.InDelta(0.9, n.ConfidenceScore, 1e-9)
	s.Equal("2026-04-01T12:00:00Z", n.CreatedAt)
}

// TestGetNotificationReturnsDetail verifies that GetNotification issues
// GET /api/v1/notifications/{id} and decodes the same MessageDetail shape
// as GetMessage.
func (s *MessageSuite) TestGetNotificationReturnsDetail() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal("/api/v1/notifications/"+testMessageID.String(), r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":               testMessageID.String(),
			"source":           "email",
			"source_account":   testSourceAccountID.String(),
			"channel":          "inbox",
			"sender":           "bob@example.com",
			"message_id":       "email-42",
			"content":          "server down",
			"importance_score": 9.5,
			"confidence_score": 0.88,
			"reasoning":        "urgent outage",
			"status":           "Notified",
			"created_at":       "2026-04-01T12:00:00Z",
			"updated_at":       "2026-04-01T12:00:00Z",
		})
	}))
	defer ts.Close()

	mc := client.NewMessageClient(client.New(ts.URL))
	detail, err := mc.GetNotification(context.Background(), testMessageID)
	s.Require().NoError(err)
	s.Require().NotNil(detail)
	s.Equal(testMessageID, detail.ID)
	s.Equal("email", detail.Source)
	s.Equal("bob@example.com", detail.Sender)
	s.Equal("email-42", detail.MessageID)
	s.Equal("urgent outage", detail.Reasoning)
	s.Equal("Notified", detail.Status)
	s.Nil(detail.ResolvedAt, "resolved_at omitted in payload should decode to nil")
}

// TestResolveNotificationPosts verifies that ResolveNotification issues
// POST /api/v1/notifications/{id}/resolve with no body and expects 200.
func (s *MessageSuite) TestResolveNotificationPosts() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPost, r.Method)
		s.Equal("/api/v1/notifications/"+testMessageID.String()+"/resolve", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "resolved",
		})
	}))
	defer ts.Close()

	mc := client.NewMessageClient(client.New(ts.URL))
	err := mc.ResolveNotification(context.Background(), testMessageID)
	s.Require().NoError(err)
}

// TestResolveNotificationAlreadyResolvedConflict verifies that a 409 from
// the server surfaces as an *APIError with ErrCodeConflict.
func (s *MessageSuite) TestResolveNotificationAlreadyResolvedConflict() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("/api/v1/notifications/"+testMessageID.String()+"/resolve", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "already resolved",
		})
	}))
	defer ts.Close()

	mc := client.NewMessageClient(client.New(ts.URL))
	err := mc.ResolveNotification(context.Background(), testMessageID)
	s.Require().Error(err)

	var apiErr *client.APIError
	s.Require().True(errors.As(err, &apiErr), "expected *APIError, got %T", err)
	s.Equal(client.ErrCodeConflict, apiErr.Code)
	s.Equal(http.StatusConflict, apiErr.StatusCode)
}

// TestDismissNotificationPosts verifies that DismissNotification issues
// POST /api/v1/notifications/{id}/dismiss with no body and expects 200.
func (s *MessageSuite) TestDismissNotificationPosts() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPost, r.Method)
		s.Equal("/api/v1/notifications/"+testMessageID.String()+"/dismiss", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "dismissed",
		})
	}))
	defer ts.Close()

	mc := client.NewMessageClient(client.New(ts.URL))
	err := mc.DismissNotification(context.Background(), testMessageID)
	s.Require().NoError(err)
}
