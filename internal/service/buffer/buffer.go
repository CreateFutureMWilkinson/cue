package buffer

import (
	"context"

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
	return nil, nil
}

// GetBufferedMessages returns all buffered messages sorted oldest-first.
func (bs *BufferService) GetBufferedMessages(ctx context.Context) ([]*repository.Message, error) {
	return nil, nil
}

// CountBuffered returns the number of buffered messages.
func (bs *BufferService) CountBuffered(ctx context.Context) (int, error) {
	return 0, nil
}

// SaveRating applies a user rating (0-10) and optional feedback to a buffered message.
func (bs *BufferService) SaveRating(ctx context.Context, messageID uuid.UUID, rating int, feedback *string) error {
	return nil
}

// DeleteMessage marks a buffered message as resolved without rating.
func (bs *BufferService) DeleteMessage(ctx context.Context, messageID uuid.UUID) error {
	return nil
}
