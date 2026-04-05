package decisionengine

import (
	"context"
	"fmt"
	"strings"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
)

const (
	StatusNotified = "Notified"
	StatusBuffered = "Buffered"
	StatusIgnored  = "Ignored"
)

// Deterministic rule constants
const (
	ChannelJoinImportanceScore = 9.0
	AtMentionImportanceScore   = 8.0
	FallbackImportanceScore    = 7.0
	HighConfidenceScore        = 1.0
	NoConfidenceScore          = 0.0
)

// RouterConfig contains thresholds for routing decisions.
type RouterConfig struct {
	ImportanceThreshold int
	ConfidenceThreshold float64
}

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

// Router applies deterministic rules and scorer-based logic to route messages
// into Notified, Buffered, or Ignored status based on importance and confidence thresholds.
type Router struct {
	scorer              Scorer
	usernames           []string
	importanceThreshold float64
	confidenceThreshold float64
	advisor             VectorScoreAdvisor
}

// NewRouter creates a new Router with the given scorer, usernames, configuration, and optional advisor.
// Returns an error if scorer is nil or usernames is empty.
func NewRouter(scorer Scorer, usernames []string, cfg RouterConfig, advisor VectorScoreAdvisor) (*Router, error) {
	if scorer == nil {
		return nil, fmt.Errorf("scorer must not be nil")
	}
	if len(usernames) == 0 {
		return nil, fmt.Errorf("usernames must not be empty")
	}
	return &Router{
		scorer:              scorer,
		usernames:           usernames,
		importanceThreshold: float64(cfg.ImportanceThreshold),
		confidenceThreshold: cfg.ConfidenceThreshold,
		advisor:             advisor,
	}, nil
}

// Route evaluates a message using deterministic rules first, then scorer-based logic.
// Returns the message with updated ImportanceScore, ConfidenceScore, Status, and Reasoning fields.
func (r *Router) Route(ctx context.Context, msg *repository.Message) (*repository.Message, error) {
	// Apply deterministic rules first - these short-circuit scorer evaluation
	if r.applyDeterministicRules(msg) {
		return msg, nil
	}

	// Use scorer for non-deterministic messages
	result, err := r.scorer.Score(ctx, msg)
	if err != nil {
		r.applyFallbackScoring(msg, err)
		return msg, nil
	}

	// Apply scorer results and determine status
	msg.ImportanceScore = result.ImportanceScore
	msg.ConfidenceScore = result.ConfidenceScore
	msg.Reasoning = result.Reasoning

	// Apply vector-based score adjustment if advisor is available
	if r.advisor != nil {
		advice, advErr := r.advisor.Advise(ctx, msg.RawContent)
		if advErr == nil && advice != nil && advice.Adjustment != 0 {
			// Adjust score based on historical user feedback for similar messages
			msg.ImportanceScore += advice.Adjustment

			// Clamp final score to valid range [0, 10]
			if msg.ImportanceScore < 0 {
				msg.ImportanceScore = 0
			} else if msg.ImportanceScore > 10 {
				msg.ImportanceScore = 10
			}
		}
	}

	r.assignStatus(msg)
	return msg, nil
}

// RouteBatch routes multiple messages, applying Route logic to each.
// Errors from individual routing calls are ignored to prevent batch failures.
func (r *Router) RouteBatch(ctx context.Context, msgs []*repository.Message) ([]*repository.Message, error) {
	results := make([]*repository.Message, 0, len(msgs))
	for _, msg := range msgs {
		routed, _ := r.Route(ctx, msg)
		results = append(results, routed)
	}
	return results, nil
}

// applyDeterministicRules checks for channel join and @mention patterns.
// Returns true if a deterministic rule was applied, false otherwise.
func (r *Router) applyDeterministicRules(msg *repository.Message) bool {
	// Channel join takes precedence over all other rules
	if msg.MessageType == "channel_join" {
		msg.ImportanceScore = ChannelJoinImportanceScore
		msg.ConfidenceScore = HighConfidenceScore
		msg.Status = StatusNotified
		msg.Reasoning = "User added to new channel"
		return true
	}

	// Check for mention message type (e.g., email To/CC/BCC detection)
	if msg.MessageType == "mention" {
		msg.ImportanceScore = AtMentionImportanceScore
		msg.ConfidenceScore = HighConfidenceScore
		msg.Status = StatusNotified
		msg.Reasoning = "Direct mention of user detected via message type"
		return true
	}

	// Check for @mentions in content
	lower := strings.ToLower(msg.RawContent)
	for _, u := range r.usernames {
		if strings.Contains(lower, "@"+strings.ToLower(u)) {
			msg.ImportanceScore = AtMentionImportanceScore
			msg.ConfidenceScore = HighConfidenceScore
			msg.Status = StatusNotified
			msg.Reasoning = "Direct @mention of user"
			return true
		}
	}

	return false
}

// applyFallbackScoring sets safe default values when scorer fails.
func (r *Router) applyFallbackScoring(msg *repository.Message, err error) {
	msg.ImportanceScore = FallbackImportanceScore
	msg.ConfidenceScore = NoConfidenceScore
	msg.Status = StatusBuffered
	msg.Reasoning = fmt.Sprintf("scorer error: %v", err)
}

// assignStatus determines the final status based on importance and confidence thresholds.
func (r *Router) assignStatus(msg *repository.Message) {
	if msg.ImportanceScore >= r.importanceThreshold && msg.ConfidenceScore >= r.confidenceThreshold {
		msg.Status = StatusNotified
	} else if msg.ImportanceScore >= r.importanceThreshold {
		msg.Status = StatusBuffered
	} else {
		msg.Status = StatusIgnored
	}
}
