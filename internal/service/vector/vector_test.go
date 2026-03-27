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

// deterministicEmbedding returns a mock EmbeddingFunc that produces a
// deterministic 3-dimensional vector derived from the input text length.
// Different texts produce different (but reproducible) vectors, which is
// sufficient for testing similarity ordering.
func deterministicEmbedding() vector.EmbeddingFunc {
	return func(_ context.Context, text string) ([]float32, error) {
		n := float32(len(text))
		// Normalise to unit-ish vector so cosine similarity is meaningful.
		mag := float32(math.Sqrt(float64(n*n + (n+1)*(n+1) + (n+2)*(n+2))))
		return []float32{n / mag, (n + 1) / mag, (n + 2) / mag}, nil
	}
}

// failingEmbedding returns an EmbeddingFunc that always errors.
func failingEmbedding() vector.EmbeddingFunc {
	return func(_ context.Context, _ string) ([]float32, error) {
		return nil, errors.New("embedding service unavailable")
	}
}

// ---------------------------------------------------------------------------
// Suite
// ---------------------------------------------------------------------------

type VectorStoreSuite struct {
	suite.Suite
}

func TestVectorStore(t *testing.T) {
	suite.Run(t, new(VectorStoreSuite))
}

// ---------------------------------------------------------------------------
// Constructor validation
// ---------------------------------------------------------------------------

func (s *VectorStoreSuite) TestNewVectorStoreRequiresEmbeddingFunc() {
	tmpDir := s.T().TempDir()

	_, err := vector.NewVectorStore(nil, tmpDir)

	s.Error(err, "NewVectorStore must reject a nil EmbeddingFunc")
}

func (s *VectorStoreSuite) TestNewVectorStoreRequiresStoragePath() {
	_, err := vector.NewVectorStore(deterministicEmbedding(), "")

	s.Error(err, "NewVectorStore must reject an empty storage path")
}

func (s *VectorStoreSuite) TestNewVectorStoreValidInputs() {
	tmpDir := s.T().TempDir()

	vs, err := vector.NewVectorStore(deterministicEmbedding(), tmpDir)

	s.NoError(err)
	s.NotNil(vs)
}

// ---------------------------------------------------------------------------
// StoreEmbedding
// ---------------------------------------------------------------------------

func (s *VectorStoreSuite) TestStoreEmbeddingReturnsVectorID() {
	tmpDir := s.T().TempDir()
	vs, err := vector.NewVectorStore(deterministicEmbedding(), tmpDir)
	s.Require().NoError(err)

	ctx := context.Background()
	msgID := uuid.New()

	vectorID, err := vs.StoreEmbedding(ctx, msgID, "server is on fire")

	s.NoError(err)
	s.NotNil(vectorID, "StoreEmbedding must return a non-nil VectorID")
	s.NotEqual(uuid.Nil, *vectorID)
}

func (s *VectorStoreSuite) TestStoreEmbeddingAssociatesWithMessageID() {
	tmpDir := s.T().TempDir()
	vs, err := vector.NewVectorStore(deterministicEmbedding(), tmpDir)
	s.Require().NoError(err)

	ctx := context.Background()
	msgID := uuid.New()

	_, err = vs.StoreEmbedding(ctx, msgID, "server is on fire")
	s.Require().NoError(err)

	// Query with the same text should return the original message ID.
	results, err := vs.QuerySimilar(ctx, "server is on fire", 1)

	s.NoError(err)
	s.Require().Len(results, 1)
	s.Equal(msgID, results[0].MessageID,
		"query result must reference the original message ID")
}

// ---------------------------------------------------------------------------
// QuerySimilar
// ---------------------------------------------------------------------------

func (s *VectorStoreSuite) TestQuerySimilarReturnsTopN() {
	tmpDir := s.T().TempDir()
	vs, err := vector.NewVectorStore(deterministicEmbedding(), tmpDir)
	s.Require().NoError(err)

	ctx := context.Background()

	// Store 5 messages with varying content.
	for i := 0; i < 5; i++ {
		_, err := vs.StoreEmbedding(ctx, uuid.New(), "message content "+string(rune('A'+i)))
		s.Require().NoError(err)
	}

	results, err := vs.QuerySimilar(ctx, "message content", 3)

	s.NoError(err)
	s.LessOrEqual(len(results), 3,
		"QuerySimilar must return at most topN results")
}

func (s *VectorStoreSuite) TestQuerySimilarReturnsSimilarityScores() {
	tmpDir := s.T().TempDir()
	vs, err := vector.NewVectorStore(deterministicEmbedding(), tmpDir)
	s.Require().NoError(err)

	ctx := context.Background()

	_, err = vs.StoreEmbedding(ctx, uuid.New(), "production outage alert")
	s.Require().NoError(err)

	results, err := vs.QuerySimilar(ctx, "production outage alert", 1)

	s.NoError(err)
	s.Require().NotEmpty(results)
	for _, r := range results {
		s.GreaterOrEqual(r.Score, float32(0.0),
			"similarity score must be >= 0")
		s.LessOrEqual(r.Score, float32(1.0),
			"similarity score must be <= 1")
	}
}

func (s *VectorStoreSuite) TestQuerySimilarEmptyStoreReturnsEmpty() {
	tmpDir := s.T().TempDir()
	vs, err := vector.NewVectorStore(deterministicEmbedding(), tmpDir)
	s.Require().NoError(err)

	ctx := context.Background()

	results, err := vs.QuerySimilar(ctx, "any query", 5)

	s.NoError(err, "querying an empty store must not error")
	s.Empty(results, "querying an empty store must return no results")
}

// ---------------------------------------------------------------------------
// Error handling — embedding failures
// ---------------------------------------------------------------------------

func (s *VectorStoreSuite) TestEmbeddingFuncErrorReturnsError() {
	tmpDir := s.T().TempDir()
	vs, err := vector.NewVectorStore(failingEmbedding(), tmpDir)
	s.Require().NoError(err)

	ctx := context.Background()

	vectorID, err := vs.StoreEmbedding(ctx, uuid.New(), "some content")

	s.Error(err, "StoreEmbedding must surface embedding errors")
	s.Nil(vectorID, "VectorID must be nil when embedding fails")
}

func (s *VectorStoreSuite) TestQuerySimilarWithEmbeddingError() {
	tmpDir := s.T().TempDir()
	vs, err := vector.NewVectorStore(failingEmbedding(), tmpDir)
	s.Require().NoError(err)

	ctx := context.Background()

	results, err := vs.QuerySimilar(ctx, "any query", 5)

	s.Error(err, "QuerySimilar must surface embedding errors")
	s.Nil(results, "results must be nil when embedding fails")
}
