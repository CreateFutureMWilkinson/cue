package client

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/google/uuid"
)

const (
	messagesPath      = "/api/v1/messages"
	notificationsPath = "/api/v1/notifications"
)

// Message is a full message list row from /api/v1/messages.
//
// It includes the explicit Status field (in contrast to NotificationSummary,
// which is implicitly status="Notified") and covers the fields needed to
// render a list item without a follow-up detail fetch.
type Message struct {
	ID              uuid.UUID `json:"id"`
	Source          string    `json:"source"`
	SourceAccount   string    `json:"source_account"`
	Sender          string    `json:"sender"`
	Channel         string    `json:"channel"`
	Subject         string    `json:"subject"`
	Content         string    `json:"content"`
	WebURL          string    `json:"web_url"`
	ImportanceScore float64   `json:"importance_score"`
	ConfidenceScore float64   `json:"confidence_score"`
	Status          string    `json:"status"`
	CreatedAt       string    `json:"created_at"`
}

// NotificationSummary is a row from /api/v1/notifications. It deliberately
// lacks the Status field because the server filters to status="Notified"
// implicitly; carrying a redundant field on the wire would be noise.
type NotificationSummary struct {
	ID              uuid.UUID `json:"id"`
	Source          string    `json:"source"`
	SourceAccount   string    `json:"source_account"`
	Sender          string    `json:"sender"`
	Channel         string    `json:"channel"`
	Subject         string    `json:"subject"`
	Content         string    `json:"content"`
	WebURL          string    `json:"web_url"`
	ImportanceScore float64   `json:"importance_score"`
	ConfidenceScore float64   `json:"confidence_score"`
	CreatedAt       string    `json:"created_at"`
}

// MessageDetail is the full shape returned by GET /messages/{id} and
// GET /notifications/{id}. ResolvedAt is a pointer because the server
// omits it for messages that have not yet been resolved.
type MessageDetail struct {
	ID              uuid.UUID `json:"id"`
	Source          string    `json:"source"`
	SourceAccount   string    `json:"source_account"`
	Channel         string    `json:"channel"`
	Sender          string    `json:"sender"`
	MessageID       string    `json:"message_id"`
	Content         string    `json:"content"`
	ImportanceScore float64   `json:"importance_score"`
	ConfidenceScore float64   `json:"confidence_score"`
	Reasoning       string    `json:"reasoning"`
	Status          string    `json:"status"`
	CreatedAt       string    `json:"created_at"`
	UpdatedAt       string    `json:"updated_at"`
	ResolvedAt      *string   `json:"resolved_at,omitempty"`
}

// MessageFilter captures the optional query parameters accepted by
// GET /api/v1/messages. Empty string and zero int fields are omitted
// from the outgoing query string.
type MessageFilter struct {
	Status  string
	Source  string
	Channel string
	Since   string // RFC3339
	Limit   int
	Offset  int
}

// ListOptions captures pagination for notification list endpoints.
// Zero-valued fields are omitted from the outgoing query string.
type ListOptions struct {
	Limit  int
	Offset int
}

// MessageClient wraps /api/v1/messages and /api/v1/notifications.
//
// The two resources share the detail shape (MessageDetail) because a
// notification is just a message with status="Notified"; list shapes
// differ because /notifications omits the redundant Status field.
type MessageClient interface {
	ListMessages(ctx context.Context, filter MessageFilter) ([]Message, int, error)
	GetMessage(ctx context.Context, id uuid.UUID) (*MessageDetail, error)
	ListNotifications(ctx context.Context, opts ListOptions) ([]NotificationSummary, int, error)
	GetNotification(ctx context.Context, id uuid.UUID) (*MessageDetail, error)
	ResolveNotification(ctx context.Context, id uuid.UUID) error
	DismissNotification(ctx context.Context, id uuid.UUID) error
}

// messageAdapter is the concrete MessageClient backed by an *APIClient.
type messageAdapter struct {
	client *APIClient
}

// NewMessageClient returns a MessageClient backed by the given APIClient.
func NewMessageClient(c *APIClient) MessageClient {
	return &messageAdapter{client: c}
}

// buildPath appends the encoded query string to the base path if non-empty.
func buildPath(base string, query url.Values) string {
	if encoded := query.Encode(); encoded != "" {
		return base + "?" + encoded
	}
	return base
}

// ListMessages issues GET /api/v1/messages with the provided filter encoded
// as query parameters. Returns the messages slice, total count, or an error.
func (a *messageAdapter) ListMessages(ctx context.Context, filter MessageFilter) ([]Message, int, error) {
	q := url.Values{}
	if filter.Status != "" {
		q.Set("status", filter.Status)
	}
	if filter.Source != "" {
		q.Set("source", filter.Source)
	}
	if filter.Channel != "" {
		q.Set("channel", filter.Channel)
	}
	if filter.Since != "" {
		q.Set("since", filter.Since)
	}
	if filter.Limit > 0 {
		q.Set("limit", strconv.Itoa(filter.Limit))
	}
	if filter.Offset > 0 {
		q.Set("offset", strconv.Itoa(filter.Offset))
	}

	path := buildPath(messagesPath, q)

	var out struct {
		Messages []Message `json:"messages"`
		Total    int       `json:"total"`
	}
	if err := a.client.doJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, 0, err
	}
	return out.Messages, out.Total, nil
}

// GetMessage issues GET /api/v1/messages/{id} and returns the full detail.
func (a *messageAdapter) GetMessage(ctx context.Context, id uuid.UUID) (*MessageDetail, error) {
	var detail MessageDetail
	if err := a.client.doJSON(ctx, http.MethodGet, messagesPath+"/"+id.String(), nil, &detail); err != nil {
		return nil, err
	}
	return &detail, nil
}

// ListNotifications issues GET /api/v1/notifications with pagination.
// The server filters to status="Notified" automatically.
func (a *messageAdapter) ListNotifications(ctx context.Context, opts ListOptions) ([]NotificationSummary, int, error) {
	q := url.Values{}
	if opts.Limit > 0 {
		q.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.Offset > 0 {
		q.Set("offset", strconv.Itoa(opts.Offset))
	}

	path := buildPath(notificationsPath, q)

	var out struct {
		Notifications []NotificationSummary `json:"notifications"`
		Total         int                   `json:"total"`
	}
	if err := a.client.doJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, 0, err
	}
	return out.Notifications, out.Total, nil
}

// GetNotification issues GET /api/v1/notifications/{id}. Returns the same
// MessageDetail shape as GetMessage.
func (a *messageAdapter) GetNotification(ctx context.Context, id uuid.UUID) (*MessageDetail, error) {
	var detail MessageDetail
	if err := a.client.doJSON(ctx, http.MethodGet, notificationsPath+"/"+id.String(), nil, &detail); err != nil {
		return nil, err
	}
	return &detail, nil
}

// ResolveNotification issues POST /api/v1/notifications/{id}/resolve.
// Returns a *APIError with ErrCodeConflict when the notification has
// already been resolved.
func (a *messageAdapter) ResolveNotification(ctx context.Context, id uuid.UUID) error {
	return a.client.doJSON(ctx, http.MethodPost, notificationsPath+"/"+id.String()+"/resolve", nil, nil)
}

// DismissNotification issues POST /api/v1/notifications/{id}/dismiss.
func (a *messageAdapter) DismissNotification(ctx context.Context, id uuid.UUID) error {
	return a.client.doJSON(ctx, http.MethodPost, notificationsPath+"/"+id.String()+"/dismiss", nil, nil)
}
