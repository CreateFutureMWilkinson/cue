package decisionengine

import (
	"cmp"
	"log/slog"
	"regexp"
	"slices"
	"strings"

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
	var compiled []compiledRule
	for _, r := range rules {
		if !r.Enabled {
			continue
		}
		re, err := regexp.Compile(r.Pattern)
		if err != nil {
			slog.Warn("skipping routing rule with invalid pattern", "rule_id", r.ID, "pattern", r.Pattern, "error", err)
			continue
		}
		compiled = append(compiled, compiledRule{rule: r, pattern: re})
	}
	slices.SortFunc(compiled, func(a, b compiledRule) int {
		return cmp.Compare(a.rule.Priority, b.rule.Priority)
	})
	return &RulesEngine{rules: compiled}
}

// extractField extracts the specified field value from a message.
func extractField(msg *repository.Message, field string) string {
	switch field {
	case "sender":
		return msg.Sender
	case "subject":
		if idx := strings.Index(msg.RawContent, "\n"); idx >= 0 {
			return msg.RawContent[:idx]
		}
		return msg.RawContent
	case "channel":
		return msg.Channel
	case "content":
		return msg.RawContent
	case "message_type":
		return msg.MessageType
	default:
		return ""
	}
}

// Evaluate checks the message against compiled rules in priority order.
// Returns the action string ("notified", "ignored", or "queue") and the matched rule (nil if "queue").
func (e *RulesEngine) Evaluate(msg *repository.Message) (string, *repository.RoutingRule) {
	for _, cr := range e.rules {
		if cr.rule.Source != msg.Source {
			continue
		}

		fieldValue := extractField(msg, cr.rule.Field)
		matched := cr.pattern.MatchString(fieldValue)
		if cr.rule.Negate {
			matched = !matched
		}

		if matched {
			return cr.rule.Action, cr.rule
		}
	}
	return "queue", nil
}
