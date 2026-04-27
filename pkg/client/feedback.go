package client

import (
	"context"

	"github.com/google/uuid"
)

// BufferedMessage mirrors a single item in the /api/v1/buffer list response.
//
// It matches the server's bufferedMessageItem DTO (see
// internal/server/handler/buffer.go) field-for-field, with snake_case JSON
// tags. Unlike MessageDetail, buffer items deliberately omit status (the
// server filters to status="Buffered" implicitly) and updated_at/resolved_at
// (not relevant for pending review).
type BufferedMessage struct {
	ID              uuid.UUID `json:"id"`
	Source          string    `json:"source"`
	SourceAccount   uuid.UUID `json:"source_account"`
	Sender          string    `json:"sender"`
	Channel         string    `json:"channel"`
	Content         string    `json:"content"`
	ImportanceScore float64   `json:"importance_score"`
	ConfidenceScore float64   `json:"confidence_score"`
	Reasoning       string    `json:"reasoning"`
	CreatedAt       string    `json:"created_at"`
}

// BufferStats mirrors the /api/v1/buffer/stats response.
//
// TotalBuffered is the total count of messages currently in Buffered state;
// BySource breaks that total down by source string (e.g. "slack", "email").
type BufferStats struct {
	TotalBuffered int            `json:"total_buffered"`
	BySource      map[string]int `json:"by_source"`
}

// FeedbackClient wraps the /api/v1/buffer routes: listing buffered messages,
// fetching a single buffered message, reading aggregate stats, submitting a
// user rating/feedback, and deleting (dismissing) a buffered message.
//
// Implementations must route stats traffic to the literal /api/v1/buffer/stats
// path and must NOT interpolate it through the /{id} detail route.
type FeedbackClient interface {
	ListBuffered(ctx context.Context, opts ListOptions) ([]BufferedMessage, int, error)
	GetBuffered(ctx context.Context, id uuid.UUID) (*BufferedMessage, error)
	BufferStats(ctx context.Context) (*BufferStats, error)
	RateBuffered(ctx context.Context, id uuid.UUID, rating int, feedback *string) error
	DeleteBuffered(ctx context.Context, id uuid.UUID) error
}

// feedbackAdapter is the concrete FeedbackClient backed by an *APIClient.
type feedbackAdapter struct {
	client *APIClient
}

// NewFeedbackClient returns a FeedbackClient backed by the given APIClient.
func NewFeedbackClient(c *APIClient) FeedbackClient {
	return &feedbackAdapter{client: c}
}

// ListBuffered issues GET /api/v1/buffer with pagination. Returns the
// decoded messages slice and total count.
func (a *feedbackAdapter) ListBuffered(ctx context.Context, opts ListOptions) ([]BufferedMessage, int, error) {
	_ = ctx
	_ = opts
	return nil, 0, ErrNotImplemented
}

// GetBuffered issues GET /api/v1/buffer/{id} for a single buffered message.
func (a *feedbackAdapter) GetBuffered(ctx context.Context, id uuid.UUID) (*BufferedMessage, error) {
	_ = ctx
	_ = id
	return nil, ErrNotImplemented
}

// BufferStats issues GET /api/v1/buffer/stats and returns the decoded stats.
func (a *feedbackAdapter) BufferStats(ctx context.Context) (*BufferStats, error) {
	_ = ctx
	return nil, ErrNotImplemented
}

// RateBuffered issues POST /api/v1/buffer/{id}/rate with the given rating
// and optional feedback. A nil feedback pointer omits the field from the
// request body.
func (a *feedbackAdapter) RateBuffered(ctx context.Context, id uuid.UUID, rating int, feedback *string) error {
	_ = ctx
	_ = id
	_ = rating
	_ = feedback
	return ErrNotImplemented
}

// DeleteBuffered issues DELETE /api/v1/buffer/{id}. The server responds
// 204 No Content on success.
func (a *feedbackAdapter) DeleteBuffered(ctx context.Context, id uuid.UUID) error {
	_ = ctx
	_ = id
	return ErrNotImplemented
}
