package vector

import (
	"context"
	"errors"
	"fmt"

	chromem "github.com/rengensheng/chromem-go"

	"github.com/google/uuid"
)

// ChromemVectorStore implements vector storage backed by chromem-go,
// a flat-file persistent vector database with BERT embeddings.
type ChromemVectorStore struct {
	db                 *chromem.DB
	collection         *chromem.Collection
	embeddingFn        EmbeddingFunc
	embeddingModelName string
}

// NewChromemVectorStore creates a new ChromemVectorStore with persistent storage
// at storagePath using the provided embedding function.
func NewChromemVectorStore(storagePath string, embeddingFn EmbeddingFunc, embeddingModelName string) (*ChromemVectorStore, error) {
	if storagePath == "" {
		return nil, errors.New("chromem vector store: storage path is required")
	}
	if embeddingFn == nil {
		return nil, errors.New("chromem vector store: embedding function is required")
	}

	db, err := chromem.NewPersistentDB(storagePath, false)
	if err != nil {
		return nil, fmt.Errorf("chromem vector store: failed to create database: %w", err)
	}

	// Cast our EmbeddingFunc to chromem.EmbeddingFunc (same signature).
	chromemEmbFn := chromem.EmbeddingFunc(embeddingFn)

	collection, err := db.GetOrCreateCollection("cue-embeddings", nil, chromemEmbFn)
	if err != nil {
		return nil, fmt.Errorf("chromem vector store: failed to create collection: %w", err)
	}

	return &ChromemVectorStore{
		db:                 db,
		collection:         collection,
		embeddingFn:        embeddingFn,
		embeddingModelName: embeddingModelName,
	}, nil
}

// StoreEmbedding generates an embedding for the content and stores it in the
// vector database, associating it with the given messageID. Returns the
// generated vector UUID.
func (s *ChromemVectorStore) StoreEmbedding(ctx context.Context, messageID uuid.UUID, content string) (*uuid.UUID, error) {
	vectorID := uuid.New()

	err := s.collection.AddDocument(ctx, chromem.Document{
		ID:      vectorID.String(),
		Content: content,
		Metadata: map[string]string{
			"message_id": messageID.String(),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("chromem vector store: failed to store embedding: %w", err)
	}

	return &vectorID, nil
}

// QuerySimilar finds the topN most similar stored vectors to the query text.
// Returns nil, nil if the store is empty. Propagates embedding errors.
func (s *ChromemVectorStore) QuerySimilar(ctx context.Context, queryText string, topN int) ([]SimilarResult, error) {
	// Validate embedding function works before checking count,
	// so that embedding errors are always propagated.
	_, err := s.embeddingFn(ctx, queryText)
	if err != nil {
		return nil, fmt.Errorf("chromem vector store: embedding failed: %w", err)
	}

	if s.collection.Count() == 0 {
		return nil, nil
	}

	results, err := s.collection.Query(ctx, queryText, topN, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("chromem vector store: query failed: %w", err)
	}

	similarResults := make([]SimilarResult, 0, len(results))
	for _, r := range results {
		msgIDStr, ok := r.Metadata["message_id"]
		if !ok {
			continue
		}
		msgID, err := uuid.Parse(msgIDStr)
		if err != nil {
			continue
		}
		similarResults = append(similarResults, SimilarResult{
			MessageID: msgID,
			Score:     r.Similarity,
		})
	}

	if len(similarResults) == 0 {
		return nil, nil
	}

	return similarResults, nil
}

// Close releases resources held by the store. chromem-go does not require
// explicit cleanup, so this always returns nil.
func (s *ChromemVectorStore) Close() error {
	return nil
}
