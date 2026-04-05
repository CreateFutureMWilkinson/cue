package vector

import (
	"context"
	"errors"
	"math"
	"sort"

	"github.com/google/uuid"
)

// EmbeddingFunc generates a vector embedding for the given text.
type EmbeddingFunc func(ctx context.Context, text string) ([]float32, error)

// SimilarResult represents a similarity search result.
type SimilarResult struct {
	MessageID uuid.UUID
	Score     float32
}

type storedVector struct {
	messageID uuid.UUID
	vectorID  uuid.UUID
	embedding []float32
}

// VectorStore stores embeddings and supports similarity search.
type VectorStore struct {
	embeddingFn EmbeddingFunc
	storagePath string
	vectors     []storedVector
}

// NewVectorStore creates a new VectorStore with the given embedding function
// and storage path. Both arguments are required.
func NewVectorStore(embeddingFn EmbeddingFunc, storagePath string) (*VectorStore, error) {
	if embeddingFn == nil {
		return nil, errors.New("vector: embedding function is required")
	}
	if storagePath == "" {
		return nil, errors.New("vector: storage path is required")
	}
	return &VectorStore{
		embeddingFn: embeddingFn,
		storagePath: storagePath,
	}, nil
}

// StoreEmbedding embeds the content and stores the vector, returning the VectorID.
func (vs *VectorStore) StoreEmbedding(ctx context.Context, messageID uuid.UUID, content string) (*uuid.UUID, error) {
	embedding, err := vs.embeddingFn(ctx, content)
	if err != nil {
		return nil, err
	}

	vectorID := uuid.New()
	vs.vectors = append(vs.vectors, storedVector{
		messageID: messageID,
		vectorID:  vectorID,
		embedding: embedding,
	})

	return &vectorID, nil
}

// QuerySimilar finds the topN most similar stored vectors to the query text.
func (vs *VectorStore) QuerySimilar(ctx context.Context, queryText string, topN int) ([]SimilarResult, error) {
	queryEmb, err := vs.embeddingFn(ctx, queryText)
	if err != nil {
		return nil, err
	}

	if len(vs.vectors) == 0 {
		return nil, nil
	}

	results := make([]SimilarResult, len(vs.vectors))
	for i, v := range vs.vectors {
		results[i] = SimilarResult{
			MessageID: v.messageID,
			Score:     cosineSimilarity(queryEmb, v.embedding),
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	n := min(topN, len(results))
	return results[:n], nil
}

func cosineSimilarity(a, b []float32) float32 {
	var dot, magA, magB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		magA += float64(a[i]) * float64(a[i])
		magB += float64(b[i]) * float64(b[i])
	}
	denom := math.Sqrt(magA) * math.Sqrt(magB)
	if denom == 0 {
		return 0
	}
	return float32(dot / denom)
}
