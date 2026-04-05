package vector

import (
	"context"

	"github.com/google/uuid"
)

// VectorEmbedder stores vector embeddings for messages.
type VectorEmbedder interface {
	StoreEmbedding(ctx context.Context, messageID uuid.UUID, content string) (*uuid.UUID, error)
}

// VectorQuerier searches stored embeddings by similarity.
type VectorQuerier interface {
	QuerySimilar(ctx context.Context, queryText string, topN int) ([]SimilarResult, error)
}
