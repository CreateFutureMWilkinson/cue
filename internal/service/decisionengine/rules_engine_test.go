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
		ID:             uuid.New(),
		Priority:       0,
		SourceType:     "slack",
		ChannelPattern: "^general$",
		Action:         "notified",
		Enabled:        true,
		CreatedAt:      now,
		UpdatedAt:      now,
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
			r.ChannelPattern = "^random$" // won't match our test message
		}),
		newRule(func(r *repository.RoutingRule) {
			r.Priority = 1
			r.ChannelPattern = "^alerts$" // won't match our test message
		}),
		newRule(func(r *repository.RoutingRule) {
			r.Priority = 2
			r.ContentPattern = "^boss$" // would match, but disabled
			r.ChannelPattern = ""
			r.Enabled = false
		}),
	}

	engine := decisionengine.NewRulesEngine(rules)
	s.NotNil(engine)

	// A message matching the disabled rule's pattern should NOT match.
	msg := newMessage(func(m *repository.Message) {
		m.RawContent = "boss"
		m.Channel = "unmatched" // won't match enabled rules
	})
	action, matched := engine.Evaluate(msg)
	s.Equal("queue", action, "disabled rule should not match; expected 'queue'")
	s.Nil(matched, "disabled rule should not produce a matched rule")
}

func (s *RulesEngineSuite) TestNewRulesEngineSkipsInvalidPatterns() {
	rules := []*repository.RoutingRule{
		newRule(func(r *repository.RoutingRule) {
			r.ChannelPattern = ""
			r.ContentPattern = "^hello$"
		}),
		newRule(func(r *repository.RoutingRule) {
			r.Priority = 1
			r.ChannelPattern = ""
			r.ContentPattern = "[invalid" // invalid regex — should be silently excluded
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
		s.Equal("^hello$", matched.ContentPattern)
	}
}

func (s *RulesEngineSuite) TestNewRulesEngineSortsByPriority() {
	// Pass rules in REVERSE priority order to verify sorting.
	rules := []*repository.RoutingRule{
		newRule(func(r *repository.RoutingRule) {
			r.Priority = 1 // higher number = lower priority
			r.ChannelPattern = "^alerts$"
			r.Action = "ignored"
		}),
		newRule(func(r *repository.RoutingRule) {
			r.Priority = 0 // lower number = higher priority — should win
			r.ChannelPattern = "^alerts$"
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

func (s *RulesEngineSuite) TestEvaluateSourceTypeMismatch() {
	rules := []*repository.RoutingRule{
		newRule(), // defaults to "slack" source type
	}

	engine := decisionengine.NewRulesEngine(rules)

	msg := newMessage(func(m *repository.Message) {
		m.Source = "email" // different source
	})
	action, matched := engine.Evaluate(msg)
	s.Equal("queue", action, "source type mismatch should fall through to queue")
	s.Nil(matched, "source type mismatch should not produce a matched rule")
}

func (s *RulesEngineSuite) TestEvaluateChannelPatternMatching() {
	rules := []*repository.RoutingRule{
		newRule(func(r *repository.RoutingRule) {
			r.ChannelPattern = "^alerts$"
		}),
	}

	engine := decisionengine.NewRulesEngine(rules)

	msg := newMessage(func(m *repository.Message) {
		m.Channel = "alerts"
	})
	action, matched := engine.Evaluate(msg)
	s.Equal("notified", action)
	s.NotNil(matched)
}

func (s *RulesEngineSuite) TestEvaluateContentPatternMatching() {
	rules := []*repository.RoutingRule{
		newRule(func(r *repository.RoutingRule) {
			r.ChannelPattern = ""
			r.ContentPattern = "urgent"
		}),
	}

	engine := decisionengine.NewRulesEngine(rules)

	msg := newMessage(func(m *repository.Message) {
		m.RawContent = "this is urgent please respond"
	})
	action, matched := engine.Evaluate(msg)
	s.Equal("notified", action)
	s.NotNil(matched)
}

func (s *RulesEngineSuite) TestEvaluateMessageTypeMatching() {
	rules := []*repository.RoutingRule{
		newRule(func(r *repository.RoutingRule) {
			r.ChannelPattern = ""
			r.MessageType = "channel_join"
		}),
	}

	engine := decisionengine.NewRulesEngine(rules)

	msg := newMessage(func(m *repository.Message) {
		m.MessageType = "channel_join"
	})
	action, matched := engine.Evaluate(msg)
	s.Equal("notified", action)
	s.NotNil(matched)
}

func (s *RulesEngineSuite) TestEvaluateEmptyPatternsMatchAnything() {
	// A rule with no patterns set should match any message of the right source type
	rules := []*repository.RoutingRule{
		newRule(func(r *repository.RoutingRule) {
			r.ChannelPattern = ""
			r.ContentPattern = ""
			r.MessageType = ""
		}),
	}

	engine := decisionengine.NewRulesEngine(rules)

	msg := newMessage(func(m *repository.Message) {
		m.Channel = "anything"
		m.RawContent = "whatever"
		m.MessageType = "normal"
	})
	action, matched := engine.Evaluate(msg)
	s.Equal("notified", action, "empty patterns should match any message")
	s.NotNil(matched)
}

func (s *RulesEngineSuite) TestEvaluateANDLogicAllFieldsMustMatch() {
	// Rule requires both channel and content patterns to match
	rules := []*repository.RoutingRule{
		newRule(func(r *repository.RoutingRule) {
			r.ChannelPattern = "^alerts$"
			r.ContentPattern = "critical"
		}),
	}

	engine := decisionengine.NewRulesEngine(rules)

	// Channel matches but content doesn't → no match
	msg1 := newMessage(func(m *repository.Message) {
		m.Channel = "alerts"
		m.RawContent = "just a normal message"
	})
	action1, matched1 := engine.Evaluate(msg1)
	s.Equal("queue", action1, "partial match should not trigger rule")
	s.Nil(matched1)

	// Both match → match
	msg2 := newMessage(func(m *repository.Message) {
		m.Channel = "alerts"
		m.RawContent = "critical failure detected"
	})
	action2, matched2 := engine.Evaluate(msg2)
	s.Equal("notified", action2, "both patterns matching should trigger rule")
	s.NotNil(matched2)
}

func (s *RulesEngineSuite) TestEvaluateFirstMatchWins() {
	rules := []*repository.RoutingRule{
		newRule(func(r *repository.RoutingRule) {
			r.Priority = 2
			r.ChannelPattern = "^alerts$"
			r.Action = "ignored"
		}),
		newRule(func(r *repository.RoutingRule) {
			r.Priority = 0
			r.ChannelPattern = "^alerts$"
			r.Action = "notified"
		}),
		newRule(func(r *repository.RoutingRule) {
			r.Priority = 1
			r.ChannelPattern = "^alerts$"
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

func (s *RulesEngineSuite) TestEvaluateSourceAccountScoping() {
	accountID := uuid.New()
	rules := []*repository.RoutingRule{
		newRule(func(r *repository.RoutingRule) {
			r.SourceAccount = &accountID
			r.ChannelPattern = ""
			r.ContentPattern = ""
		}),
	}

	engine := decisionengine.NewRulesEngine(rules)

	// Matching account
	msg1 := newMessage(func(m *repository.Message) {
		m.SourceAccount = accountID.String()
	})
	action1, matched1 := engine.Evaluate(msg1)
	s.Equal("notified", action1)
	s.NotNil(matched1)

	// Non-matching account
	msg2 := newMessage(func(m *repository.Message) {
		m.SourceAccount = uuid.New().String()
	})
	action2, matched2 := engine.Evaluate(msg2)
	s.Equal("queue", action2)
	s.Nil(matched2)
}

func (s *RulesEngineSuite) TestEvaluateNilSourceAccountMatchesAll() {
	rules := []*repository.RoutingRule{
		newRule(func(r *repository.RoutingRule) {
			r.SourceAccount = nil // nil matches any account
			r.ChannelPattern = ""
		}),
	}

	engine := decisionengine.NewRulesEngine(rules)

	msg := newMessage(func(m *repository.Message) {
		m.SourceAccount = uuid.New().String()
	})
	action, matched := engine.Evaluate(msg)
	s.Equal("notified", action, "nil SourceAccount should match any account")
	s.NotNil(matched)
}
