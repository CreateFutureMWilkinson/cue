package decisionengine_test

import (
	"context"
	"testing"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/service/decisionengine"
	"github.com/CreateFutureMWilkinson/cue/internal/service/vector"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// mockVectorQuerier implements vector.VectorQuerier for testing.
type mockVectorQuerier struct {
	results []vector.SimilarResult
	err     error
}

func (m *mockVectorQuerier) QuerySimilar(_ context.Context, _ string, _ int) ([]vector.SimilarResult, error) {
	return m.results, m.err
}

// mockMessageQuerier implements decisionengine.MessageQuerier for testing.
type mockMessageQuerier struct {
	messages map[uuid.UUID]*repository.Message
}

func (m *mockMessageQuerier) QueryByID(_ context.Context, id uuid.UUID) (*repository.Message, error) {
	msg, ok := m.messages[id]
	if !ok {
		return nil, nil
	}
	return msg, nil
}

// --- Suite ---

type FewShotProviderSuite struct {
	suite.Suite
}

func TestFewShotProvider(t *testing.T) {
	suite.Run(t, new(FewShotProviderSuite))
}

// --- Tests ---

func (s *FewShotProviderSuite) TestGetExamples_EmptyVectorStore_ReturnsEmptySlice() {
	vq := &mockVectorQuerier{results: []vector.SimilarResult{}}
	mq := &mockMessageQuerier{messages: map[uuid.UUID]*repository.Message{}}

	provider, err := decisionengine.NewFewShotProvider(vq, mq, decisionengine.FewShotProviderConfig{
		SimilarityThreshold: 0.75,
		MaxExamples:         5,
	})
	s.Require().NoError(err)

	examples, err := provider.GetExamples(context.Background(), "test content")
	s.NoError(err, "expected no error when vector store is empty")
	s.Empty(examples, "expected empty slice when vector store has no results")
}
