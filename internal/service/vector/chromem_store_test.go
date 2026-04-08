package vector_test

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/CreateFutureMWilkinson/cue/internal/service/vector"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// chromemDeterministicEmbedding returns a mock EmbeddingFunc that produces a
// deterministic 3-dimensional vector derived from the input text length.
// Different texts produce different (but reproducible) vectors, which is
// sufficient for testing similarity ordering.
func chromemDeterministicEmbedding() vector.EmbeddingFunc {
	return func(_ context.Context, text string) ([]float32, error) {
		n := float32(len(text))
		mag := float32(math.Sqrt(float64(n*n + (n+1)*(n+1) + (n+2)*(n+2))))
		return []float32{n / mag, (n + 1) / mag, (n + 2) / mag}, nil
	}
}

// chromemFailingEmbedding returns an EmbeddingFunc that always errors.
func chromemFailingEmbedding() vector.EmbeddingFunc {
	return func(_ context.Context, _ string) ([]float32, error) {
		return nil, errors.New("embedding service unavailable")
	}
}

// ---------------------------------------------------------------------------
// Suite
// ---------------------------------------------------------------------------

type ChromemVectorStoreSuite struct {
	suite.Suite
}

func TestChromemVectorStore(t *testing.T) {
	suite.Run(t, new(ChromemVectorStoreSuite))
}

// ---------------------------------------------------------------------------
// Constructor validation
// ---------------------------------------------------------------------------

func (s *ChromemVectorStoreSuite) TestConstructorRejectsEmptyStoragePath() {
	_, err := vector.NewChromemVectorStore("", chromemDeterministicEmbedding(), "test-model")

	s.Error(err, "NewChromemVectorStore must reject an empty storage path")
}

func (s *ChromemVectorStoreSuite) TestConstructorRejectsNilEmbeddingFunc() {
	tmpDir := s.T().TempDir()

	_, err := vector.NewChromemVectorStore(tmpDir, nil, "test-model")

	s.Error(err, "NewChromemVectorStore must reject a nil embedding function")
}

func (s *ChromemVectorStoreSuite) TestConstructorWithValidInputsSucceeds() {
	tmpDir := s.T().TempDir()

	store, err := vector.NewChromemVectorStore(tmpDir, chromemDeterministicEmbedding(), "test-model")

	s.NoError(err)
	s.NotNil(store)
}

// ---------------------------------------------------------------------------
// StoreEmbedding
// ---------------------------------------------------------------------------

func (s *ChromemVectorStoreSuite) TestStoreEmbeddingReturnsValidUUID() {
	tmpDir := s.T().TempDir()
	store, err := vector.NewChromemVectorStore(tmpDir, chromemDeterministicEmbedding(), "test-model")
	s.Require().NoError(err)

	ctx := context.Background()
	msgID := uuid.New()

	vectorID, err := store.StoreEmbedding(ctx, msgID, "server is on fire")

	s.NoError(err)
	s.NotNil(vectorID, "StoreEmbedding must return a non-nil VectorID")
	s.NotEqual(uuid.Nil, *vectorID)
}

func (s *ChromemVectorStoreSuite) TestStoreEmbeddingAssociatesWithCorrectMessageID() {
	tmpDir := s.T().TempDir()
	store, err := vector.NewChromemVectorStore(tmpDir, chromemDeterministicEmbedding(), "test-model")
	s.Require().NoError(err)

	ctx := context.Background()
	msgID := uuid.New()

	_, err = store.StoreEmbedding(ctx, msgID, "server is on fire")
	s.Require().NoError(err)

	// Query with the same text should return the original message ID.
	results, err := store.QuerySimilar(ctx, "server is on fire", 1)

	s.NoError(err)
	s.Require().Len(results, 1)
	s.Equal(msgID, results[0].MessageID,
		"query result must reference the original message ID")
}

// ---------------------------------------------------------------------------
// QuerySimilar
// ---------------------------------------------------------------------------

func (s *ChromemVectorStoreSuite) TestQuerySimilarReturnsSortedBySimilarity() {
	tmpDir := s.T().TempDir()
	store, err := vector.NewChromemVectorStore(tmpDir, chromemDeterministicEmbedding(), "test-model")
	s.Require().NoError(err)

	ctx := context.Background()

	// Store texts of very different lengths to get different embeddings.
	_, err = store.StoreEmbedding(ctx, uuid.New(), "short")
	s.Require().NoError(err)
	_, err = store.StoreEmbedding(ctx, uuid.New(), "a medium length text here")
	s.Require().NoError(err)
	_, err = store.StoreEmbedding(ctx, uuid.New(), "a somewhat longer text for testing similarity ordering in the vector store")
	s.Require().NoError(err)

	results, err := store.QuerySimilar(ctx, "a medium length text here", 3)

	s.NoError(err)
	s.Require().NotEmpty(results)

	// Results must be sorted by descending similarity score.
	for i := 1; i < len(results); i++ {
		s.GreaterOrEqual(results[i-1].Score, results[i].Score,
			"results must be sorted by descending similarity score")
	}

	// The best match should be the exact same text.
	s.InDelta(float64(1.0), float64(results[0].Score), 0.01,
		"exact text match should have score near 1.0")
}

func (s *ChromemVectorStoreSuite) TestQuerySimilarEmptyStoreReturnsNil() {
	tmpDir := s.T().TempDir()
	store, err := vector.NewChromemVectorStore(tmpDir, chromemDeterministicEmbedding(), "test-model")
	s.Require().NoError(err)

	ctx := context.Background()

	results, err := store.QuerySimilar(ctx, "any query", 5)

	s.NoError(err, "querying an empty store must not error")
	s.Nil(results, "querying an empty store must return nil")
}

// ---------------------------------------------------------------------------
// Round-trip: store then query
// ---------------------------------------------------------------------------

func (s *ChromemVectorStoreSuite) TestRoundTripStoreAndQuery() {
	tmpDir := s.T().TempDir()
	store, err := vector.NewChromemVectorStore(tmpDir, chromemDeterministicEmbedding(), "test-model")
	s.Require().NoError(err)

	ctx := context.Background()
	msgID := uuid.New()

	_, err = store.StoreEmbedding(ctx, msgID, "production outage alert")
	s.Require().NoError(err)

	// Store a second, less similar message.
	otherID := uuid.New()
	_, err = store.StoreEmbedding(ctx, otherID, "x")
	s.Require().NoError(err)

	results, err := store.QuerySimilar(ctx, "production outage alert", 2)

	s.NoError(err)
	s.Require().NotEmpty(results)
	s.Equal(msgID, results[0].MessageID,
		"the most similar result should be the exact match")
}

// ---------------------------------------------------------------------------
// Persistence across close/reopen
// ---------------------------------------------------------------------------

func (s *ChromemVectorStoreSuite) TestStoragePersistsAcrossCloseReopen() {
	tmpDir := s.T().TempDir()
	embFn := chromemDeterministicEmbedding()
	ctx := context.Background()
	msgID := uuid.New()

	// Phase 1: create store, store an embedding, close it.
	store1, err := vector.NewChromemVectorStore(tmpDir, embFn, "test-model")
	s.Require().NoError(err)

	_, err = store1.StoreEmbedding(ctx, msgID, "deadline tomorrow")
	s.Require().NoError(err)

	err = store1.Close()
	s.Require().NoError(err)

	// Phase 2: reopen store at same path, query for previously stored data.
	store2, err := vector.NewChromemVectorStore(tmpDir, embFn, "test-model")
	s.Require().NoError(err)

	results, err := store2.QuerySimilar(ctx, "deadline tomorrow", 1)

	s.NoError(err)
	s.Require().Len(results, 1, "reopened store must find previously stored embedding")
	s.Equal(msgID, results[0].MessageID,
		"reopened store must return the correct message ID")
}

// ---------------------------------------------------------------------------
// Embedding function error propagation
// ---------------------------------------------------------------------------

func (s *ChromemVectorStoreSuite) TestStoreEmbeddingPropagatesEmbeddingError() {
	tmpDir := s.T().TempDir()
	store, err := vector.NewChromemVectorStore(tmpDir, chromemFailingEmbedding(), "test-model")
	s.Require().NoError(err)

	ctx := context.Background()

	vectorID, err := store.StoreEmbedding(ctx, uuid.New(), "some content")

	s.Error(err, "StoreEmbedding must surface embedding errors")
	s.Nil(vectorID, "VectorID must be nil when embedding fails")
}

func (s *ChromemVectorStoreSuite) TestQuerySimilarPropagatesEmbeddingError() {
	tmpDir := s.T().TempDir()
	store, err := vector.NewChromemVectorStore(tmpDir, chromemFailingEmbedding(), "test-model")
	s.Require().NoError(err)

	ctx := context.Background()

	results, err := store.QuerySimilar(ctx, "any query", 5)

	s.Error(err, "QuerySimilar must surface embedding errors")
	s.Nil(results, "results must be nil when embedding fails")
}

// ---------------------------------------------------------------------------
// Close
// ---------------------------------------------------------------------------

func (s *ChromemVectorStoreSuite) TestCloseReturnsNoError() {
	tmpDir := s.T().TempDir()
	store, err := vector.NewChromemVectorStore(tmpDir, chromemDeterministicEmbedding(), "test-model")
	s.Require().NoError(err)

	err = store.Close()

	s.NoError(err, "Close must not return an error on a healthy store")
}

// ---------------------------------------------------------------------------
// Embedding model metadata filtering
// ---------------------------------------------------------------------------

func (s *ChromemVectorStoreSuite) TestQuerySimilarFiltersResultsByEmbeddingModel() {
	tmpDir := s.T().TempDir()
	embFn := chromemDeterministicEmbedding()
	ctx := context.Background()

	// Phase 1: create store with model-A, store an embedding.
	storeA, err := vector.NewChromemVectorStore(tmpDir, embFn, "model-A")
	s.Require().NoError(err)

	_, err = storeA.StoreEmbedding(ctx, uuid.New(), "critical production outage")
	s.Require().NoError(err)

	err = storeA.Close()
	s.Require().NoError(err)

	// Phase 2: reopen the same storage path with a DIFFERENT model name.
	storeB, err := vector.NewChromemVectorStore(tmpDir, embFn, "model-B")
	s.Require().NoError(err)

	// QuerySimilar on model-B store should return 0 results because the
	// stored embedding was created with model-A, and model-B should filter
	// it out via metadata mismatch.
	results, err := storeB.QuerySimilar(ctx, "critical production outage", 5)

	s.NoError(err)
	s.Nil(results,
		"QuerySimilar must filter out results stored with a different embedding model")
}
