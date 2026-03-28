package buffer

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/google/uuid"
)

// MessageRepository is the subset of repository.MessageRepository needed by
// the buffer service.
type MessageRepository interface {
	QueryByStatus(ctx context.Context, status string) ([]*repository.Message, error)
	Update(ctx context.Context, msg *repository.Message) error
}

// VectorEmbedder stores vector embeddings for messages. May be nil.
type VectorEmbedder interface {
	StoreEmbedding(ctx context.Context, messageID uuid.UUID, content string) (*uuid.UUID, error)
}

// BufferService manages the feedback buffer of messages awaiting user review.
type BufferService struct {
	repo     MessageRepository
	embedder VectorEmbedder
}

// NewBufferService creates a new BufferService. repo is required; embedder may be nil.
func NewBufferService(repo MessageRepository, embedder VectorEmbedder) (*BufferService, error) {
	if repo == nil {
		return nil, fmt.Errorf("repo must not be nil")
	}
	return &BufferService{
		repo:     repo,
		embedder: embedder,
	}, nil
}

// GetBufferedMessages returns all buffered messages sorted oldest-first.
func (bs *BufferService) GetBufferedMessages(ctx context.Context) ([]*repository.Message, error) {
	msgs, err := bs.repo.QueryByStatus(ctx, "Buffered")
	if err != nil {
		return nil, fmt.Errorf("querying buffered messages: %w", err)
	}
	slices.SortFunc(msgs, func(a, b *repository.Message) int {
		return a.CreatedAt.Compare(b.CreatedAt)
	})
	return msgs, nil
}

// CountBuffered returns the number of buffered messages.
func (bs *BufferService) CountBuffered(ctx context.Context) (int, error) {
	msgs, err := bs.repo.QueryByStatus(ctx, "Buffered")
	if err != nil {
		return 0, fmt.Errorf("counting buffered messages: %w", err)
	}
	return len(msgs), nil
}

// SaveRating applies a user rating (0-10) and optional feedback to a buffered message.
func (bs *BufferService) SaveRating(ctx context.Context, messageID uuid.UUID, rating int, feedback *string) error {
	if rating < 0 || rating > 10 {
		return fmt.Errorf("rating must be 0-10, got %d", rating)
	}

	msgs, err := bs.repo.QueryByStatus(ctx, "Buffered")
	if err != nil {
		return fmt.Errorf("querying buffered messages: %w", err)
	}

	var msg *repository.Message
	for _, m := range msgs {
		if m.ID == messageID {
			msg = m
			break
		}
	}
	if msg == nil {
		return fmt.Errorf("message %s not found in buffer", messageID)
	}

	now := time.Now()
	msg.Status = "Resolved"
	msg.UserRating = &rating
	msg.UserFeedback = feedback
	msg.ResolvedAt = &now
	msg.UpdatedAt = now

	if err := bs.repo.Update(ctx, msg); err != nil {
		return fmt.Errorf("updating message rating: %w", err)
	}

	if bs.embedder != nil {
		vectorID, embedErr := bs.embedder.StoreEmbedding(ctx, msg.ID, msg.RawContent)
		if embedErr == nil && vectorID != nil {
			msg.VectorID = vectorID
			_ = bs.repo.Update(ctx, msg)
		}
	}

	return nil
}

// DeleteMessage marks a buffered message as resolved without rating.
func (bs *BufferService) DeleteMessage(ctx context.Context, messageID uuid.UUID) error {
	msgs, err := bs.repo.QueryByStatus(ctx, "Buffered")
	if err != nil {
		return fmt.Errorf("querying buffered messages: %w", err)
	}

	var msg *repository.Message
	for _, m := range msgs {
		if m.ID == messageID {
			msg = m
			break
		}
	}
	if msg == nil {
		return fmt.Errorf("message %s not found in buffer", messageID)
	}

	now := time.Now()
	msg.Status = "Resolved"
	msg.ResolvedAt = &now
	msg.UpdatedAt = now

	if err := bs.repo.Update(ctx, msg); err != nil {
		return fmt.Errorf("deleting message: %w", err)
	}

	return nil
}
