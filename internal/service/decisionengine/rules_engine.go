package decisionengine

import (
	"cmp"
	"log/slog"
	"regexp"
	"slices"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
)

type compiledRule struct {
	rule           *repository.RoutingRule
	channelPattern *regexp.Regexp
	contentPattern *regexp.Regexp
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

		cr := compiledRule{rule: r}
		valid := true

		if r.ChannelPattern != "" {
			re, err := regexp.Compile(r.ChannelPattern)
			if err != nil {
				slog.Warn("skipping routing rule with invalid channel pattern", "rule_id", r.ID, "pattern", r.ChannelPattern, "error", err)
				valid = false
			} else {
				cr.channelPattern = re
			}
		}

		if r.ContentPattern != "" {
			re, err := regexp.Compile(r.ContentPattern)
			if err != nil {
				slog.Warn("skipping routing rule with invalid content pattern", "rule_id", r.ID, "pattern", r.ContentPattern, "error", err)
				valid = false
			} else {
				cr.contentPattern = re
			}
		}

		if valid {
			compiled = append(compiled, cr)
		}
	}
	slices.SortFunc(compiled, func(a, b compiledRule) int {
		return cmp.Compare(a.rule.Priority, b.rule.Priority)
	})
	return &RulesEngine{rules: compiled}
}

// Evaluate checks the message against compiled rules in priority order.
// Returns the action string ("notified", "ignored", or "queue") and the matched rule (nil if "queue").
// All set fields in a rule must match (AND logic). Empty pattern fields match any value.
func (e *RulesEngine) Evaluate(msg *repository.Message) (string, *repository.RoutingRule) {
	for _, cr := range e.rules {
		if cr.rule.SourceType != msg.Source {
			continue
		}

		if cr.rule.SourceAccount != nil && cr.rule.SourceAccount.String() != msg.SourceAccount {
			continue
		}

		if cr.rule.MessageType != "" && cr.rule.MessageType != msg.MessageType {
			continue
		}

		if cr.channelPattern != nil && !cr.channelPattern.MatchString(msg.Channel) {
			continue
		}

		if cr.contentPattern != nil && !cr.contentPattern.MatchString(msg.RawContent) {
			continue
		}

		return cr.rule.Action, cr.rule
	}
	return "queue", nil
}
