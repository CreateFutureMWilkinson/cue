package vector

import (
	"context"

	"github.com/google/uuid"
)

// EmbeddingFunc generates a vector embedding for the given text.
type EmbeddingFunc func(ctx context.Context, text string) ([]float32, error)

// SimilarResult represents a similarity search result.
type SimilarResult struct {
	MessageID uuid.UUID
	Score     float32
}

// VectorEmbedder stores vector embeddings for messages.
type VectorEmbedder interface {
	StoreEmbedding(ctx context.Context, messageID uuid.UUID, content string) (*uuid.UUID, error)
}

// VectorQuerier searches stored embeddings by similarity.
type VectorQuerier interface {
	QuerySimilar(ctx context.Context, queryText string, topN int) ([]SimilarResult, error)
}
