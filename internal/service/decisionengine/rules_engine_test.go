package decisionengine_test

import (
	"testing"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/service/decisionengine"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// --- Test Helpers ---

func newRule(opts ...func(*repository.RoutingRule)) *repository.RoutingRule {
	now := time.Now()
	rule := &repository.RoutingRule{
		ID:        uuid.New(),
		Priority:  0,
		Source:    "slack",
		Field:     "channel",
		Negate:    false,
		Pattern:   "^general$",
		Action:    "notified",
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	for _, opt := range opts {
		opt(rule)
	}
	return rule
}

func newMessage(opts ...func(*repository.Message)) *repository.Message {
	msg := &repository.Message{
		Source:  "slack",
		Channel: "general",
	}
	for _, opt := range opts {
		opt(msg)
	}
	return msg
}

// --- Suite ---

type RulesEngineSuite struct {
	suite.Suite
}

func TestRulesEngine(t *testing.T) {
	suite.Run(t, new(RulesEngineSuite))
}

func (s *RulesEngineSuite) TestNewRulesEngineFiltersDisabledRules() {
	rules := []*repository.RoutingRule{
		newRule(func(r *repository.RoutingRule) {
			r.Pattern = "^random$" // won't match our test message
		}),
		newRule(func(r *repository.RoutingRule) {
			r.Priority = 1
			r.Pattern = "^alerts$" // won't match our test message
		}),
		newRule(func(r *repository.RoutingRule) {
			r.Priority = 2
			r.Field = "sender"
			r.Pattern = "^boss$" // would match, but disabled
			r.Enabled = false
		}),
	}

	engine := decisionengine.NewRulesEngine(rules)
	s.NotNil(engine)

	// A message matching the disabled rule's pattern should NOT match.
	msg := newMessage(func(m *repository.Message) {
		m.Sender = "boss"
		m.Channel = "unmatched" // won't match enabled rules
	})
	action, matched := engine.Evaluate(msg)
	s.Equal("queue", action, "disabled rule should not match; expected 'queue'")
	s.Nil(matched, "disabled rule should not produce a matched rule")
}

func (s *RulesEngineSuite) TestNewRulesEngineSkipsInvalidPatterns() {
	rules := []*repository.RoutingRule{
		newRule(func(r *repository.RoutingRule) {
			r.Field = "content"
			r.Pattern = "^hello$"
		}),
		newRule(func(r *repository.RoutingRule) {
			r.Priority = 1
			r.Field = "content"
			r.Pattern = "[invalid" // invalid regex — should be silently excluded
			r.Action = "ignored"
		}),
	}

	engine := decisionengine.NewRulesEngine(rules)
	s.NotNil(engine)

	// The valid rule should still work.
	msg := newMessage(func(m *repository.Message) {
		m.RawContent = "hello"
	})
	action, matched := engine.Evaluate(msg)
	s.Equal("notified", action)
	if s.NotNil(matched) {
		s.Equal("^hello$", matched.Pattern)
	}
}

func (s *RulesEngineSuite) TestNewRulesEngineSortsByPriority() {
	// Pass rules in REVERSE priority order to verify sorting.
	rules := []*repository.RoutingRule{
		newRule(func(r *repository.RoutingRule) {
			r.Priority = 1 // higher number = lower priority
			r.Pattern = "^alerts$"
			r.Action = "ignored"
		}),
		newRule(func(r *repository.RoutingRule) {
			r.Priority = 0 // lower number = higher priority — should win
			r.Pattern = "^alerts$"
		}),
	}

	engine := decisionengine.NewRulesEngine(rules)
	s.NotNil(engine)

	msg := newMessage(func(m *repository.Message) {
		m.Channel = "alerts"
	})
	action, matched := engine.Evaluate(msg)
	s.Equal("notified", action, "priority 0 rule should win over priority 1")
	if s.NotNil(matched) {
		s.Equal(0, matched.Priority)
	}
}

func (s *RulesEngineSuite) TestNewRulesEngineEmptyRules() {
	// Nil rules
	engine := decisionengine.NewRulesEngine(nil)
	s.NotNil(engine)

	msg := newMessage(func(m *repository.Message) {
		m.RawContent = "hello world"
	})
	action, matched := engine.Evaluate(msg)
	s.Equal("queue", action)
	s.Nil(matched)

	// Empty slice
	engine2 := decisionengine.NewRulesEngine([]*repository.RoutingRule{})
	s.NotNil(engine2)

	action2, matched2 := engine2.Evaluate(msg)
	s.Equal("queue", action2)
	s.Nil(matched2)
}

// --- Evaluate Behavior Tests ---

func (s *RulesEngineSuite) TestEvaluateSourceScopeMismatch() {
	rules := []*repository.RoutingRule{
		newRule(), // defaults to "slack" source
	}

	engine := decisionengine.NewRulesEngine(rules)

	msg := newMessage(func(m *repository.Message) {
		m.Source = "email" // different source
	})
	action, matched := engine.Evaluate(msg)
	s.Equal("queue", action, "source mismatch should fall through to queue")
	s.Nil(matched, "source mismatch should not produce a matched rule")
}

func (s *RulesEngineSuite) TestEvaluateFieldExtraction() {
	tests := []struct {
		name       string
		field      string
		pattern    string
		msg        *repository.Message
		wantAction string
	}{
		{
			name:    "sender field",
			field:   "sender",
			pattern: "^alice$",
			msg: newMessage(func(m *repository.Message) {
				m.Sender = "alice"
			}),
			wantAction: "notified",
		},
		{
			name:    "channel field",
			field:   "channel",
			pattern: "^alerts$",
			msg: newMessage(func(m *repository.Message) {
				m.Channel = "alerts"
			}),
			wantAction: "notified",
		},
		{
			name:    "content field",
			field:   "content",
			pattern: "urgent",
			msg: newMessage(func(m *repository.Message) {
				m.RawContent = "this is urgent please respond"
			}),
			wantAction: "notified",
		},
		{
			name:    "message_type field",
			field:   "message_type",
			pattern: "^channel_join$",
			msg: newMessage(func(m *repository.Message) {
				m.MessageType = "channel_join"
			}),
			wantAction: "notified",
		},
		{
			name:    "subject field extracts first line",
			field:   "subject",
			pattern: "^Important Subject$",
			msg: newMessage(func(m *repository.Message) {
				m.Source = "email"
				m.RawContent = "Important Subject\nBody text goes here"
			}),
			wantAction: "notified",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			rules := []*repository.RoutingRule{
				newRule(func(r *repository.RoutingRule) {
					r.Source = tc.msg.Source
					r.Field = tc.field
					r.Pattern = tc.pattern
				}),
			}
			engine := decisionengine.NewRulesEngine(rules)
			action, matched := engine.Evaluate(tc.msg)
			s.Equal(tc.wantAction, action)
			s.NotNil(matched)
		})
	}
}

func (s *RulesEngineSuite) TestEvaluateSubjectExtractionNoNewline() {
	rules := []*repository.RoutingRule{
		newRule(func(r *repository.RoutingRule) {
			r.Source = "email"
			r.Field = "subject"
			r.Pattern = "^Single line content$"
		}),
	}

	engine := decisionengine.NewRulesEngine(rules)

	msg := newMessage(func(m *repository.Message) {
		m.Source = "email"
		m.RawContent = "Single line content"
	})
	action, matched := engine.Evaluate(msg)
	s.Equal("notified", action, "subject should be entire RawContent when no newline present")
	s.NotNil(matched)
}

func (s *RulesEngineSuite) TestEvaluateNegation() {
	rules := []*repository.RoutingRule{
		newRule(func(r *repository.RoutingRule) {
			r.Field = "sender"
			r.Negate = true
			r.Pattern = "^spam$"
		}),
	}

	engine := decisionengine.NewRulesEngine(rules)

	// "boss" does NOT match "^spam$", negation inverts → true → should match rule
	msgBoss := newMessage(func(m *repository.Message) {
		m.Sender = "boss"
	})
	action, matched := engine.Evaluate(msgBoss)
	s.Equal("notified", action, "negated rule should match when pattern does NOT match field")
	s.NotNil(matched)

	// "spam" matches "^spam$", negation inverts → false → should NOT match rule
	msgSpam := newMessage(func(m *repository.Message) {
		m.Sender = "spam"
	})
	action2, matched2 := engine.Evaluate(msgSpam)
	s.Equal("queue", action2, "negated rule should not match when pattern matches field")
	s.Nil(matched2)
}

func (s *RulesEngineSuite) TestEvaluateFirstMatchWins() {
	rules := []*repository.RoutingRule{
		newRule(func(r *repository.RoutingRule) {
			r.Priority = 2
			r.Pattern = "^alerts$"
			r.Action = "ignored"
		}),
		newRule(func(r *repository.RoutingRule) {
			r.Priority = 0
			r.Pattern = "^alerts$"
			r.Action = "notified"
		}),
		newRule(func(r *repository.RoutingRule) {
			r.Priority = 1
			r.Pattern = "^alerts$"
			r.Action = "ignored"
		}),
	}

	engine := decisionengine.NewRulesEngine(rules)

	msg := newMessage(func(m *repository.Message) {
		m.Channel = "alerts"
	})
	action, matched := engine.Evaluate(msg)
	s.Equal("notified", action, "priority 0 rule should win (first match after sorting)")
	if s.NotNil(matched) {
		s.Equal(0, matched.Priority, "matched rule should be priority 0")
	}
}
