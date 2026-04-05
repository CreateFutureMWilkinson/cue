package decisionengine

import (
	"context"
	"fmt"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/service/vector"

	"github.com/google/uuid"
)

// ScoreAdvice contains the vector-based score adjustment recommendation.
type ScoreAdvice struct {
	Adjustment    float64 // Score adjustment to apply [-2.0, +2.0]
	SimilarCount  int     // Number of similar messages found
	AvgUserRating float64 // Weighted average user rating of similar messages
	TopSimilarity float32 // Highest similarity score among matches
}

// VectorAdvisorConfig contains configuration for the vector score advisor.
type VectorAdvisorConfig struct {
	SimilarityThreshold float64 // Minimum similarity score to consider [0.0, 1.0]
	TopN                int     // Maximum number of similar messages to fetch
	DampingFactor       float64 // Factor to reduce adjustment impact [0.0, 1.0]
}

// VectorScoreAdvisor provides score adjustment advice based on similar historical messages.
type VectorScoreAdvisor interface {
	Advise(ctx context.Context, content string) (*ScoreAdvice, error)
}

// MessageQuerier retrieves messages by ID for vector advisor lookups.
type MessageQuerier interface {
	QueryByID(ctx context.Context, id uuid.UUID) (*repository.Message, error)
}

// vectorScoreAdvisor implements VectorScoreAdvisor using vector similarity search.
type vectorScoreAdvisor struct {
	querier    vector.VectorQuerier
	msgQuerier MessageQuerier
	cfg        VectorAdvisorConfig
}

// NewVectorScoreAdvisor creates a new vector score advisor.
// Returns an error if querier or msgQuerier is nil.
func NewVectorScoreAdvisor(querier vector.VectorQuerier, msgQuerier MessageQuerier, cfg VectorAdvisorConfig) (*vectorScoreAdvisor, error) {
	if querier == nil {
		return nil, fmt.Errorf("vector querier must not be nil")
	}
	if msgQuerier == nil {
		return nil, fmt.Errorf("message querier must not be nil")
	}
	return &vectorScoreAdvisor{
		querier:    querier,
		msgQuerier: msgQuerier,
		cfg:        cfg,
	}, nil
}

// Advise computes a score adjustment based on similar historical messages and their user ratings.
func (a *vectorScoreAdvisor) Advise(ctx context.Context, content string) (*ScoreAdvice, error) {
	results, err := a.querier.QuerySimilar(ctx, content, a.cfg.TopN)
	if err != nil {
		return nil, err
	}

	var (
		totalWeightedRating     float64
		totalWeightedImportance float64
		totalSimilarity         float64
		matchCount              int
		maxSimilarity           float32
	)

	for _, r := range results {
		if float64(r.Score) < a.cfg.SimilarityThreshold {
			continue
		}

		msg, err := a.msgQuerier.QueryByID(ctx, r.MessageID)
		if err != nil {
			continue
		}
		if msg == nil || msg.UserRating == nil {
			continue
		}

		similarity := float64(r.Score)
		totalWeightedRating += float64(*msg.UserRating) * similarity
		totalWeightedImportance += msg.ImportanceScore * similarity
		totalSimilarity += similarity
		matchCount++

		if r.Score > maxSimilarity {
			maxSimilarity = r.Score
		}
	}

	if matchCount == 0 {
		return &ScoreAdvice{}, nil
	}

	avgUserRating := totalWeightedRating / totalSimilarity
	avgImportanceScore := totalWeightedImportance / totalSimilarity

	// Calculate raw adjustment: user rating vs initial scoring
	rawAdjustment := avgUserRating - avgImportanceScore

	// Clamp adjustment to valid range [-2.0, +2.0]
	if rawAdjustment > 2.0 {
		rawAdjustment = 2.0
	} else if rawAdjustment < -2.0 {
		rawAdjustment = -2.0
	}

	// Apply damping factor to moderate adjustment impact
	dampedAdjustment := rawAdjustment * a.cfg.DampingFactor

	return &ScoreAdvice{
		Adjustment:    dampedAdjustment,
		SimilarCount:  matchCount,
		AvgUserRating: avgUserRating,
		TopSimilarity: maxSimilarity,
	}, nil
}
