package decisionengine

import (
	"context"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
)

const (
	StatusNotified = "Notified"
	StatusBuffered = "Buffered"
	StatusIgnored  = "Ignored"
	StatusImported = "Imported"
)

// Deterministic rule constants
const (
	ChannelJoinImportanceScore = 9.0
	AtMentionImportanceScore   = 8.0
	FallbackImportanceScore    = 7.0
	HighConfidenceScore        = 1.0
	NoConfidenceScore          = 0.0
)

// ScorerResult contains the scoring output from an LLM or scoring system.
type ScorerResult struct {
	ImportanceScore float64 // 0-10 importance rating
	ConfidenceScore float64 // 0.0-1.0 confidence in the rating
	Reasoning       string  // Human-readable explanation
}

// Scorer evaluates message content and returns importance/confidence scores.
type Scorer interface {
	Score(ctx context.Context, msg *repository.Message) (*ScorerResult, error)
}
