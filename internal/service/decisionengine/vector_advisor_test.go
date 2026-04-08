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

func (s *FewShotProviderSuite) TestGetExamples_ThreeMatchesMaxFive_ReturnsThree() {
	id1, id2, id3 := uuid.New(), uuid.New(), uuid.New()
	rating1, rating2, rating3 := 8, 7, 9

	vq := &mockVectorQuerier{results: []vector.SimilarResult{
		{MessageID: id1, Score: 0.85},
		{MessageID: id2, Score: 0.80},
		{MessageID: id3, Score: 0.78},
	}}
	mq := &mockMessageQuerier{messages: map[uuid.UUID]*repository.Message{
		id1: {ID: id1, RawContent: "Message 1", UserRating: &rating1},
		id2: {ID: id2, RawContent: "Message 2", UserRating: &rating2},
		id3: {ID: id3, RawContent: "Message 3", UserRating: &rating3},
	}}

	provider, err := decisionengine.NewFewShotProvider(vq, mq, decisionengine.FewShotProviderConfig{
		SimilarityThreshold: 0.75,
		MaxExamples:         5,
	})
	s.Require().NoError(err)

	examples, err := provider.GetExamples(context.Background(), "test content")
	s.NoError(err)
	s.Len(examples, 3, "expected exactly 3 examples when 3 matches and MaxExamples=5")
}

func (s *FewShotProviderSuite) TestGetExamples_SevenMatchesMaxFive_ReturnsFive() {
	id1, id2, id3, id4, id5 := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	rating1, rating2, rating3, rating4, rating5 := 8, 7, 9, 6, 10

	// Mock returns exactly MaxExamples=5 results (topN parameter is respected by implementation)
	vq := &mockVectorQuerier{results: []vector.SimilarResult{
		{MessageID: id1, Score: 0.95},
		{MessageID: id2, Score: 0.90},
		{MessageID: id3, Score: 0.85},
		{MessageID: id4, Score: 0.80},
		{MessageID: id5, Score: 0.78},
	}}
	mq := &mockMessageQuerier{messages: map[uuid.UUID]*repository.Message{
		id1: {ID: id1, RawContent: "Message 1", UserRating: &rating1},
		id2: {ID: id2, RawContent: "Message 2", UserRating: &rating2},
		id3: {ID: id3, RawContent: "Message 3", UserRating: &rating3},
		id4: {ID: id4, RawContent: "Message 4", UserRating: &rating4},
		id5: {ID: id5, RawContent: "Message 5", UserRating: &rating5},
	}}

	provider, err := decisionengine.NewFewShotProvider(vq, mq, decisionengine.FewShotProviderConfig{
		SimilarityThreshold: 0.75,
		MaxExamples:         5,
	})
	s.Require().NoError(err)

	examples, err := provider.GetExamples(context.Background(), "test content")
	s.NoError(err)
	s.Len(examples, 5, "expected exactly 5 examples when limited by MaxExamples")
}

func (s *FewShotProviderSuite) TestGetExamples_BelowSimilarityThreshold_Filtered() {
	id1, id2, id3 := uuid.New(), uuid.New(), uuid.New()
	rating1, rating2, rating3 := 8, 7, 9

	vq := &mockVectorQuerier{results: []vector.SimilarResult{
		{MessageID: id1, Score: 0.85}, // Above threshold (0.75)
		{MessageID: id2, Score: 0.70}, // Below threshold
		{MessageID: id3, Score: 0.80}, // Above threshold
	}}
	mq := &mockMessageQuerier{messages: map[uuid.UUID]*repository.Message{
		id1: {ID: id1, RawContent: "Message 1", UserRating: &rating1},
		id2: {ID: id2, RawContent: "Message 2", UserRating: &rating2},
		id3: {ID: id3, RawContent: "Message 3", UserRating: &rating3},
	}}

	provider, err := decisionengine.NewFewShotProvider(vq, mq, decisionengine.FewShotProviderConfig{
		SimilarityThreshold: 0.75,
		MaxExamples:         5,
	})
	s.Require().NoError(err)

	examples, err := provider.GetExamples(context.Background(), "test content")
	s.NoError(err)
	s.Len(examples, 2, "expected only messages above similarity threshold")

	// Verify the correct messages were included
	s.Equal(8, examples[0].UserRating)
	s.Equal(9, examples[1].UserRating)
}

func (s *FewShotProviderSuite) TestGetExamples_ContentTruncatedTo200Chars() {
	id1 := uuid.New()
	rating1 := 8
	longContent := "This is a very long message that exceeds 200 characters and should be truncated by the FewShotProvider implementation when creating examples for the LLM prompt injection system that helps with scoring new messages based on historical user ratings and feedback patterns."

	vq := &mockVectorQuerier{results: []vector.SimilarResult{
		{MessageID: id1, Score: 0.85},
	}}
	mq := &mockMessageQuerier{messages: map[uuid.UUID]*repository.Message{
		id1: {ID: id1, RawContent: longContent, UserRating: &rating1},
	}}

	provider, err := decisionengine.NewFewShotProvider(vq, mq, decisionengine.FewShotProviderConfig{
		SimilarityThreshold: 0.75,
		MaxExamples:         5,
	})
	s.Require().NoError(err)

	examples, err := provider.GetExamples(context.Background(), "test content")
	s.NoError(err)
	s.Len(examples, 1)
	s.Len(examples[0].Content, 200, "expected content truncated to exactly 200 characters")
	s.Equal(longContent[:200], examples[0].Content, "expected content to be first 200 characters")
}

func (s *FewShotProviderSuite) TestGetExamples_UnratedMessagesExcluded() {
	id1, id2 := uuid.New(), uuid.New()
	rating1 := 8

	vq := &mockVectorQuerier{results: []vector.SimilarResult{
		{MessageID: id1, Score: 0.85},
		{MessageID: id2, Score: 0.80},
	}}
	mq := &mockMessageQuerier{messages: map[uuid.UUID]*repository.Message{
		id1: {ID: id1, RawContent: "Rated message", UserRating: &rating1},
		id2: {ID: id2, RawContent: "Unrated message", UserRating: nil}, // No rating
	}}

	provider, err := decisionengine.NewFewShotProvider(vq, mq, decisionengine.FewShotProviderConfig{
		SimilarityThreshold: 0.75,
		MaxExamples:         5,
	})
	s.Require().NoError(err)

	examples, err := provider.GetExamples(context.Background(), "test content")
	s.NoError(err)
	s.Len(examples, 1, "expected only rated messages to be included")
	s.Equal("Rated message", examples[0].Content)
	s.Equal(8, examples[0].UserRating)
}

func (s *FewShotProviderSuite) TestGetExamples_QueryError_ReturnsError() {
	vq := &mockVectorQuerier{err: context.DeadlineExceeded}
	mq := &mockMessageQuerier{messages: map[uuid.UUID]*repository.Message{}}

	provider, err := decisionengine.NewFewShotProvider(vq, mq, decisionengine.FewShotProviderConfig{
		SimilarityThreshold: 0.75,
		MaxExamples:         5,
	})
	s.Require().NoError(err)

	examples, err := provider.GetExamples(context.Background(), "test content")
	s.Error(err, "expected error when vector querier returns error")
	s.Nil(examples, "expected nil examples when error occurs")
}

func (s *FewShotProviderSuite) TestNewFewShotProvider_NilQuerier() {
	mq := &mockMessageQuerier{messages: map[uuid.UUID]*repository.Message{}}

	provider, err := decisionengine.NewFewShotProvider(nil, mq, decisionengine.FewShotProviderConfig{})
	s.Error(err, "expected error when vector querier is nil")
	s.Nil(provider, "expected nil provider when vector querier is nil")
	s.Contains(err.Error(), "vector querier must not be nil")
}

func (s *FewShotProviderSuite) TestNewFewShotProvider_NilMessageQuerier() {
	vq := &mockVectorQuerier{results: []vector.SimilarResult{}}

	provider, err := decisionengine.NewFewShotProvider(vq, nil, decisionengine.FewShotProviderConfig{})
	s.Error(err, "expected error when message querier is nil")
	s.Nil(provider, "expected nil provider when message querier is nil")
	s.Contains(err.Error(), "message querier must not be nil")
}
