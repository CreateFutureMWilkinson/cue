package decisionengine

import (
	"regexp"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
)

type compiledRule struct {
	rule    *repository.RoutingRule
	pattern *regexp.Regexp
}

// RulesEngine evaluates messages against a set of compiled routing rules.
type RulesEngine struct {
	rules []compiledRule
}

// NewRulesEngine creates a RulesEngine from the given routing rules.
// Disabled rules and rules with invalid regex patterns are silently excluded.
// Rules are evaluated in priority order (lowest priority number first).
func NewRulesEngine(rules []*repository.RoutingRule) *RulesEngine {
	return &RulesEngine{}
}

// Evaluate checks the message against compiled rules in priority order.
// Returns the action string ("notified", "ignored", or "queue") and the matched rule (nil if "queue").
func (e *RulesEngine) Evaluate(msg *repository.Message) (string, *repository.RoutingRule) {
	return "", nil
}
