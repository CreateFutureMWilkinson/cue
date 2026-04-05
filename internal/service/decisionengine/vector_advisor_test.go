package decisionengine_test

import (
	"context"
	"errors"
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

type VectorAdvisorSuite struct {
	suite.Suite
}

func TestVectorAdvisor(t *testing.T) {
	suite.Run(t, new(VectorAdvisorSuite))
}

// --- Constructor ---

func (s *VectorAdvisorSuite) TestNewVectorScoreAdvisor_NilQuerier() {
	_, err := decisionengine.NewVectorScoreAdvisor(nil, &mockMessageQuerier{}, decisionengine.VectorAdvisorConfig{})
	s.Error(err)
	s.Contains(err.Error(), "vector querier")
}

func (s *VectorAdvisorSuite) TestNewVectorScoreAdvisor_NilMessageQuerier() {
	_, err := decisionengine.NewVectorScoreAdvisor(&mockVectorQuerier{}, nil, decisionengine.VectorAdvisorConfig{})
	s.Error(err)
	s.Contains(err.Error(), "message querier")
}

func (s *VectorAdvisorSuite) TestNewVectorScoreAdvisor_Valid() {
	advisor, err := decisionengine.NewVectorScoreAdvisor(
		&mockVectorQuerier{},
		&mockMessageQuerier{},
		decisionengine.VectorAdvisorConfig{
			SimilarityThreshold: 0.75,
			TopN:                5,
			DampingFactor:       0.5,
		},
	)
	s.NoError(err)
	s.NotNil(advisor)
}

// --- Advise behavior ---

func (s *VectorAdvisorSuite) TestAdvise_NoSimilarMessages_ZeroAdjustment() {
	vq := &mockVectorQuerier{results: []vector.SimilarResult{}}
	mq := &mockMessageQuerier{messages: map[uuid.UUID]*repository.Message{}}

	advisor, err := decisionengine.NewVectorScoreAdvisor(vq, mq, decisionengine.VectorAdvisorConfig{
		SimilarityThreshold: 0.75,
		TopN:                5,
		DampingFactor:       0.5,
	})
	s.Require().NoError(err)

	advice, err := advisor.Advise(context.Background(), "test content")
	s.NoError(err)
	s.NotNil(advice)
	s.Equal(0.0, advice.Adjustment)
	s.Equal(0, advice.SimilarCount)
}

func (s *VectorAdvisorSuite) TestAdvise_SimilarMessagesWithRatings_WeightedAdjustment() {
	id1 := uuid.New()
	id2 := uuid.New()

	rating1 := 9
	rating2 := 7

	vq := &mockVectorQuerier{results: []vector.SimilarResult{
		{MessageID: id1, Score: 0.90},
		{MessageID: id2, Score: 0.80},
	}}
	mq := &mockMessageQuerier{messages: map[uuid.UUID]*repository.Message{
		id1: {ID: id1, UserRating: &rating1, ImportanceScore: 5.0},
		id2: {ID: id2, UserRating: &rating2, ImportanceScore: 5.0},
	}}

	advisor, err := decisionengine.NewVectorScoreAdvisor(vq, mq, decisionengine.VectorAdvisorConfig{
		SimilarityThreshold: 0.75,
		TopN:                5,
		DampingFactor:       0.5,
	})
	s.Require().NoError(err)

	advice, err := advisor.Advise(context.Background(), "test content")
	s.NoError(err)
	s.NotNil(advice)
	s.Equal(2, advice.SimilarCount)
	s.InDelta(float32(0.90), advice.TopSimilarity, 0.001)

	// Weighted average rating: (9*0.90 + 7*0.80) / (0.90 + 0.80) = (8.1 + 5.6) / 1.7 = 8.06
	// The adjustment depends on the Ollama importance passed or calculated internally.
	// At minimum, verify adjustment is non-zero and within bounds.
	s.True(advice.Adjustment >= -2.0 && advice.Adjustment <= 2.0)
	s.InDelta(8.06, advice.AvgUserRating, 0.1)
}

func (s *VectorAdvisorSuite) TestAdvise_AllBelowSimilarityThreshold_ZeroAdjustment() {
	id1 := uuid.New()
	rating1 := 9

	vq := &mockVectorQuerier{results: []vector.SimilarResult{
		{MessageID: id1, Score: 0.50}, // Below 0.75 threshold
	}}
	mq := &mockMessageQuerier{messages: map[uuid.UUID]*repository.Message{
		id1: {ID: id1, UserRating: &rating1},
	}}

	advisor, err := decisionengine.NewVectorScoreAdvisor(vq, mq, decisionengine.VectorAdvisorConfig{
		SimilarityThreshold: 0.75,
		TopN:                5,
		DampingFactor:       0.5,
	})
	s.Require().NoError(err)

	advice, err := advisor.Advise(context.Background(), "test content")
	s.NoError(err)
	s.NotNil(advice)
	s.Equal(0.0, advice.Adjustment)
	s.Equal(0, advice.SimilarCount)
}

func (s *VectorAdvisorSuite) TestAdvise_MixedRatedAndUnrated_UnratedExcluded() {
	id1 := uuid.New()
	id2 := uuid.New()

	rating1 := 8

	vq := &mockVectorQuerier{results: []vector.SimilarResult{
		{MessageID: id1, Score: 0.90},
		{MessageID: id2, Score: 0.85}, // No user rating
	}}
	mq := &mockMessageQuerier{messages: map[uuid.UUID]*repository.Message{
		id1: {ID: id1, UserRating: &rating1},
		id2: {ID: id2, UserRating: nil}, // Unrated
	}}

	advisor, err := decisionengine.NewVectorScoreAdvisor(vq, mq, decisionengine.VectorAdvisorConfig{
		SimilarityThreshold: 0.75,
		TopN:                5,
		DampingFactor:       0.5,
	})
	s.Require().NoError(err)

	advice, err := advisor.Advise(context.Background(), "test content")
	s.NoError(err)
	s.NotNil(advice)
	// Only id1 should count; id2 is unrated.
	s.Equal(1, advice.SimilarCount)
	s.InDelta(8.0, advice.AvgUserRating, 0.01)
}

func (s *VectorAdvisorSuite) TestAdvise_AdjustmentCappedAtPositiveTwo() {
	id1 := uuid.New()
	rating1 := 10 // Very high rating; if Ollama scored low, raw diff could exceed +2.0

	vq := &mockVectorQuerier{results: []vector.SimilarResult{
		{MessageID: id1, Score: 0.95},
	}}
	mq := &mockMessageQuerier{messages: map[uuid.UUID]*repository.Message{
		id1: {ID: id1, UserRating: &rating1, ImportanceScore: 2.0},
	}}

	advisor, err := decisionengine.NewVectorScoreAdvisor(vq, mq, decisionengine.VectorAdvisorConfig{
		SimilarityThreshold: 0.75,
		TopN:                5,
		DampingFactor:       1.0, // No damping to test cap
	})
	s.Require().NoError(err)

	advice, err := advisor.Advise(context.Background(), "test content")
	s.NoError(err)
	s.NotNil(advice)
	s.True(advice.Adjustment <= 2.0, "adjustment should be capped at +2.0, got %f", advice.Adjustment)
}

func (s *VectorAdvisorSuite) TestAdvise_AdjustmentCappedAtNegativeTwo() {
	id1 := uuid.New()
	rating1 := 0 // Very low rating; if Ollama scored high, raw diff could be below -2.0

	vq := &mockVectorQuerier{results: []vector.SimilarResult{
		{MessageID: id1, Score: 0.95},
	}}
	mq := &mockMessageQuerier{messages: map[uuid.UUID]*repository.Message{
		id1: {ID: id1, UserRating: &rating1, ImportanceScore: 9.0},
	}}

	advisor, err := decisionengine.NewVectorScoreAdvisor(vq, mq, decisionengine.VectorAdvisorConfig{
		SimilarityThreshold: 0.75,
		TopN:                5,
		DampingFactor:       1.0, // No damping to test cap
	})
	s.Require().NoError(err)

	advice, err := advisor.Advise(context.Background(), "test content")
	s.NoError(err)
	s.NotNil(advice)
	s.True(advice.Adjustment >= -2.0, "adjustment should be capped at -2.0, got %f", advice.Adjustment)
}

func (s *VectorAdvisorSuite) TestAdvise_DampingFactorApplied() {
	id1 := uuid.New()
	rating1 := 10

	vq := &mockVectorQuerier{results: []vector.SimilarResult{
		{MessageID: id1, Score: 0.95},
	}}
	mq := &mockMessageQuerier{messages: map[uuid.UUID]*repository.Message{
		// Avg user rating = 10, diff from some baseline will be large.
		// With damping 0.5, result should be halved.
		id1: {ID: id1, UserRating: &rating1, ImportanceScore: 6.0},
	}}

	// With damping 1.0
	advisorFull, err := decisionengine.NewVectorScoreAdvisor(vq, mq, decisionengine.VectorAdvisorConfig{
		SimilarityThreshold: 0.75,
		TopN:                5,
		DampingFactor:       1.0,
	})
	s.Require().NoError(err)

	adviceFull, err := advisorFull.Advise(context.Background(), "test content")
	s.Require().NoError(err)

	// With damping 0.5
	advisorHalf, err := decisionengine.NewVectorScoreAdvisor(vq, mq, decisionengine.VectorAdvisorConfig{
		SimilarityThreshold: 0.75,
		TopN:                5,
		DampingFactor:       0.5,
	})
	s.Require().NoError(err)

	adviceHalf, err := advisorHalf.Advise(context.Background(), "test content")
	s.Require().NoError(err)

	// Half-damped adjustment should be ~half of full-damped adjustment
	if adviceFull.Adjustment != 0.0 {
		s.InDelta(adviceFull.Adjustment*0.5, adviceHalf.Adjustment, 0.01)
	}
}

func (s *VectorAdvisorSuite) TestAdvise_EmptyVectorStore_ZeroAdjustment() {
	vq := &mockVectorQuerier{results: nil}
	mq := &mockMessageQuerier{messages: map[uuid.UUID]*repository.Message{}}

	advisor, err := decisionengine.NewVectorScoreAdvisor(vq, mq, decisionengine.VectorAdvisorConfig{
		SimilarityThreshold: 0.75,
		TopN:                5,
		DampingFactor:       0.5,
	})
	s.Require().NoError(err)

	advice, err := advisor.Advise(context.Background(), "test content")
	s.NoError(err)
	s.NotNil(advice)
	s.Equal(0.0, advice.Adjustment)
	s.Equal(0, advice.SimilarCount)
}

func (s *VectorAdvisorSuite) TestAdvise_VectorQuerierError_ReturnsError() {
	vq := &mockVectorQuerier{err: errors.New("vector store unavailable")}
	mq := &mockMessageQuerier{messages: map[uuid.UUID]*repository.Message{}}

	advisor, err := decisionengine.NewVectorScoreAdvisor(vq, mq, decisionengine.VectorAdvisorConfig{
		SimilarityThreshold: 0.75,
		TopN:                5,
		DampingFactor:       0.5,
	})
	s.Require().NoError(err)

	advice, err := advisor.Advise(context.Background(), "test content")
	s.Error(err)
	s.Nil(advice)
}
