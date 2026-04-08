package decisionengine

import (
	"context"
	"errors"
	"fmt"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/service/vector"

	"github.com/google/uuid"
)

var ErrNotImplemented = errors.New("not implemented")

// FewShotExample is a previously-rated message used as LLM context.
type FewShotExample struct {
	Content    string  // Truncated to 200 chars
	UserRating int     // 0-10
	Similarity float32 // For logging/debugging; not sent to LLM
}

// FewShotProviderConfig holds construction-time configuration.
type FewShotProviderConfig struct {
	SimilarityThreshold float64 // Minimum cosine similarity [0.0, 1.0]
	MaxExamples         int     // Maximum examples to return
}

// FewShotProvider retrieves similar rated messages for prompt injection.
type FewShotProvider interface {
	GetExamples(ctx context.Context, content string) ([]FewShotExample, error)
}

// MessageQuerier retrieves messages by ID for FewShotProvider lookups.
type MessageQuerier interface {
	QueryByID(ctx context.Context, id uuid.UUID) (*repository.Message, error)
}

// fewShotProvider implements FewShotProvider using vector similarity search.
type fewShotProvider struct {
	querier    vector.VectorQuerier
	msgQuerier MessageQuerier
	cfg        FewShotProviderConfig
}

// NewFewShotProvider creates a new FewShotProvider.
// Returns an error if querier or msgQuerier is nil.
func NewFewShotProvider(querier vector.VectorQuerier, msgQuerier MessageQuerier, cfg FewShotProviderConfig) (*fewShotProvider, error) {
	if querier == nil {
		return nil, fmt.Errorf("vector querier must not be nil")
	}
	if msgQuerier == nil {
		return nil, fmt.Errorf("message querier must not be nil")
	}
	return &fewShotProvider{
		querier:    querier,
		msgQuerier: msgQuerier,
		cfg:        cfg,
	}, nil
}

// GetExamples returns similar rated messages for use as few-shot examples.
func (p *fewShotProvider) GetExamples(ctx context.Context, content string) ([]FewShotExample, error) {
	return nil, ErrNotImplemented
}
