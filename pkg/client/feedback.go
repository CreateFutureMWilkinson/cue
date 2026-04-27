package client

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/google/uuid"
)

const (
	bufferPath      = "/api/v1/buffer"
	bufferStatsPath = "/api/v1/buffer/stats"
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
// decoded messages slice and total count. Limit and offset are only
// emitted as query parameters when > 0.
func (a *feedbackAdapter) ListBuffered(ctx context.Context, opts ListOptions) ([]BufferedMessage, int, error) {
	q := url.Values{}
	if opts.Limit > 0 {
		q.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.Offset > 0 {
		q.Set("offset", strconv.Itoa(opts.Offset))
	}

	path := buildPath(bufferPath, q)

	var out struct {
		Messages []BufferedMessage `json:"messages"`
		Total    int               `json:"total"`
		Count    int               `json:"count"`
	}
	if err := a.client.doJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, 0, err
	}
	return out.Messages, out.Total, nil
}

// GetBuffered issues GET /api/v1/buffer/{id} for a single buffered message.
func (a *feedbackAdapter) GetBuffered(ctx context.Context, id uuid.UUID) (*BufferedMessage, error) {
	var msg BufferedMessage
	if err := a.client.doJSON(ctx, http.MethodGet, bufferPath+"/"+id.String(), nil, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// BufferStats issues GET /api/v1/buffer/stats and returns the decoded stats.
// The path is the literal /api/v1/buffer/stats — it must NOT be interpolated
// through the /{id} detail route.
func (a *feedbackAdapter) BufferStats(ctx context.Context) (*BufferStats, error) {
	var stats BufferStats
	if err := a.client.doJSON(ctx, http.MethodGet, bufferStatsPath, nil, &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}

// RateBuffered issues POST /api/v1/buffer/{id}/rate with the given rating
// and optional feedback. A nil feedback pointer omits the field from the
// request body via json:",omitempty".
func (a *feedbackAdapter) RateBuffered(ctx context.Context, id uuid.UUID, rating int, feedback *string) error {
	body := struct {
		Rating   int     `json:"rating"`
		Feedback *string `json:"feedback,omitempty"`
	}{Rating: rating, Feedback: feedback}
	return a.client.doJSON(ctx, http.MethodPost, bufferPath+"/"+id.String()+"/rate", body, nil)
}

// DeleteBuffered issues DELETE /api/v1/buffer/{id}. The server responds
// 204 No Content on success; doJSON's nil-out path skips body decoding.
func (a *feedbackAdapter) DeleteBuffered(ctx context.Context, id uuid.UUID) error {
	return a.client.doJSON(ctx, http.MethodDelete, bufferPath+"/"+id.String(), nil, nil)
}
