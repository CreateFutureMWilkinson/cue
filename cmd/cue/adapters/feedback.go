package adapters

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/pkg/client"
)

// FeedbackAdapter satisfies presenter.BufferReviewer on top of the
// SDK's FeedbackClient. It maps the Buffered list / stats / rate /
// delete endpoints onto the four methods the feedback presenter uses.
type FeedbackAdapter struct {
	client client.FeedbackClient
}

// NewFeedbackAdapter wraps the given SDK feedback client.
func NewFeedbackAdapter(c client.FeedbackClient) *FeedbackAdapter {
	return &FeedbackAdapter{client: c}
}

// GetBufferedMessages returns every message currently in the buffered
// queue. The wire shape implicitly filters to Status="Buffered"; the
// adapter stamps that on the repository struct so downstream callers
// can rely on a populated Status field.
func (a *FeedbackAdapter) GetBufferedMessages(ctx context.Context) ([]*repository.Message, error) {
	dtos, _, err := a.client.ListBuffered(ctx, client.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list buffered messages: %w", err)
	}
	out := make([]*repository.Message, 0, len(dtos))
	for i := range dtos {
		out = append(out, bufferedDTOToRepo(dtos[i]))
	}
	return out, nil
}

// CountBuffered returns the server-reported total of buffered
// messages. Exposed as a separate count so callers don't need to
// fetch the full list to render a badge.
func (a *FeedbackAdapter) CountBuffered(ctx context.Context) (int, error) {
	stats, err := a.client.BufferStats(ctx)
	if err != nil {
		return 0, fmt.Errorf("buffer stats: %w", err)
	}
	return stats.TotalBuffered, nil
}

// SaveRating submits a 0–10 user rating with optional free-text
// feedback to the buffered message.
func (a *FeedbackAdapter) SaveRating(ctx context.Context, messageID uuid.UUID, rating int, feedback *string) error {
	if err := a.client.RateBuffered(ctx, messageID, rating, feedback); err != nil {
		return fmt.Errorf("rate buffered %s: %w", messageID, err)
	}
	return nil
}

// DeleteMessage dismisses a buffered message.
func (a *FeedbackAdapter) DeleteMessage(ctx context.Context, messageID uuid.UUID) error {
	if err := a.client.DeleteBuffered(ctx, messageID); err != nil {
		return fmt.Errorf("delete buffered %s: %w", messageID, err)
	}
	return nil
}

// bufferedDTOToRepo translates an SDK BufferedMessage into the
// repository shape. Status is stamped to "Buffered" since the server
// filters this list implicitly.
func bufferedDTOToRepo(m client.BufferedMessage) *repository.Message {
	return &repository.Message{
		ID:              m.ID,
		Source:          m.Source,
		SourceAccount:   uuidString(m.SourceAccount),
		Sender:          m.Sender,
		Channel:         m.Channel,
		RawContent:      m.Content,
		ImportanceScore: m.ImportanceScore,
		ConfidenceScore: m.ConfidenceScore,
		Reasoning:       m.Reasoning,
		Status:          "Buffered",
		CreatedAt:       parseRFC3339OrZero(m.CreatedAt),
	}
}
