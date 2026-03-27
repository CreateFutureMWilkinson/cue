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

type RouterConfig struct {
	ImportanceThreshold int
	ConfidenceThreshold float64
}

type ScorerResult struct {
	ImportanceScore float64
	ConfidenceScore float64
	Reasoning       string
}

type Scorer interface {
	Score(ctx context.Context, msg *repository.Message) (*ScorerResult, error)
}

type Router struct {
	scorer              Scorer
	usernames           []string
	importanceThreshold float64
	confidenceThreshold float64
}

func NewRouter(scorer Scorer, usernames []string, cfg RouterConfig) (*Router, error) {
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
	}, nil
}

func (r *Router) Route(ctx context.Context, msg *repository.Message) (*repository.Message, error) {
	if msg.MessageType == "channel_join" {
		msg.ImportanceScore = 9.0
		msg.ConfidenceScore = 1.0
		msg.Status = StatusNotified
		msg.Reasoning = "User added to new channel"
		return msg, nil
	}

	lower := strings.ToLower(msg.RawContent)
	for _, u := range r.usernames {
		if strings.Contains(lower, "@"+strings.ToLower(u)) {
			msg.ImportanceScore = 8.0
			msg.ConfidenceScore = 1.0
			msg.Status = StatusNotified
			msg.Reasoning = "Direct @mention of user"
			return msg, nil
		}
	}

	result, err := r.scorer.Score(ctx, msg)
	if err != nil {
		msg.ImportanceScore = 7.0
		msg.ConfidenceScore = 0.0
		msg.Status = StatusBuffered
		msg.Reasoning = fmt.Sprintf("scorer error: %v", err)
		return msg, nil
	}

	msg.ImportanceScore = result.ImportanceScore
	msg.ConfidenceScore = result.ConfidenceScore
	msg.Reasoning = result.Reasoning

	if msg.ImportanceScore >= r.importanceThreshold && msg.ConfidenceScore >= r.confidenceThreshold {
		msg.Status = StatusNotified
	} else if msg.ImportanceScore >= r.importanceThreshold {
		msg.Status = StatusBuffered
	} else {
		msg.Status = StatusIgnored
	}

	return msg, nil
}

func (r *Router) RouteBatch(ctx context.Context, msgs []*repository.Message) ([]*repository.Message, error) {
	results := make([]*repository.Message, 0, len(msgs))
	for _, msg := range msgs {
		routed, _ := r.Route(ctx, msg)
		results = append(results, routed)
	}
	return results, nil
}
