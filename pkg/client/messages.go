package client

import (
	"context"

	"github.com/google/uuid"
)

// Message is a full message list row from /api/v1/messages.
//
// It includes the explicit Status field (in contrast to NotificationSummary,
// which is implicitly status="Notified") and covers the fields needed to
// render a list item without a follow-up detail fetch.
type Message struct {
	ID              uuid.UUID `json:"id"`
	Source          string    `json:"source"`
	SourceAccount   uuid.UUID `json:"source_account"`
	Sender          string    `json:"sender"`
	Channel         string    `json:"channel"`
	Content         string    `json:"content"`
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
	SourceAccount   uuid.UUID `json:"source_account"`
	Sender          string    `json:"sender"`
	Channel         string    `json:"channel"`
	Content         string    `json:"content"`
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
	SourceAccount   uuid.UUID `json:"source_account"`
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

// ListMessages issues GET /api/v1/messages with the provided filter encoded
// as query parameters. Returns the messages slice, total count, or an error.
func (a *messageAdapter) ListMessages(_ context.Context, _ MessageFilter) ([]Message, int, error) {
	return nil, 0, ErrNotImplemented
}

// GetMessage issues GET /api/v1/messages/{id} and returns the full detail.
func (a *messageAdapter) GetMessage(_ context.Context, _ uuid.UUID) (*MessageDetail, error) {
	return nil, ErrNotImplemented
}

// ListNotifications issues GET /api/v1/notifications with pagination.
// The server filters to status="Notified" automatically.
func (a *messageAdapter) ListNotifications(_ context.Context, _ ListOptions) ([]NotificationSummary, int, error) {
	return nil, 0, ErrNotImplemented
}

// GetNotification issues GET /api/v1/notifications/{id}. Returns the same
// MessageDetail shape as GetMessage.
func (a *messageAdapter) GetNotification(_ context.Context, _ uuid.UUID) (*MessageDetail, error) {
	return nil, ErrNotImplemented
}

// ResolveNotification issues POST /api/v1/notifications/{id}/resolve.
// Returns a *APIError with ErrCodeConflict when the notification has
// already been resolved.
func (a *messageAdapter) ResolveNotification(_ context.Context, _ uuid.UUID) error {
	return ErrNotImplemented
}

// DismissNotification issues POST /api/v1/notifications/{id}/dismiss.
func (a *messageAdapter) DismissNotification(_ context.Context, _ uuid.UUID) error {
	return ErrNotImplemented
}
