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
	Adjustment    float64
	SimilarCount  int
	AvgUserRating float64
	TopSimilarity float32
}

// VectorAdvisorConfig contains configuration for the vector score advisor.
type VectorAdvisorConfig struct {
	SimilarityThreshold float64
	TopN                int
	DampingFactor       float64
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
		weightedRatingSum     float64
		weightedImportanceSum float64
		similaritySum         float64
		count                 int
		maxSimilarity         float32
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

		sim := float64(r.Score)
		weightedRatingSum += float64(*msg.UserRating) * sim
		weightedImportanceSum += msg.ImportanceScore * sim
		similaritySum += sim
		count++

		if r.Score > maxSimilarity {
			maxSimilarity = r.Score
		}
	}

	if count == 0 {
		return &ScoreAdvice{}, nil
	}

	weightedAvgRating := weightedRatingSum / similaritySum
	weightedAvgIS := weightedImportanceSum / similaritySum

	rawAdj := weightedAvgRating - weightedAvgIS

	// Clamp to [-2.0, +2.0]
	if rawAdj > 2.0 {
		rawAdj = 2.0
	}
	if rawAdj < -2.0 {
		rawAdj = -2.0
	}

	// Apply damping factor
	adj := rawAdj * a.cfg.DampingFactor

	return &ScoreAdvice{
		Adjustment:    adj,
		SimilarCount:  count,
		AvgUserRating: weightedAvgRating,
		TopSimilarity: maxSimilarity,
	}, nil
}
