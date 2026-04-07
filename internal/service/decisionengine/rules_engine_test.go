package decisionengine_test

import (
	"testing"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/service/decisionengine"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// --- Suite ---

type RulesEngineSuite struct {
	suite.Suite
}

func TestRulesEngine(t *testing.T) {
	suite.Run(t, new(RulesEngineSuite))
}

func (s *RulesEngineSuite) TestNewRulesEngineFiltersDisabledRules() {
	now := time.Now()
	rules := []*repository.RoutingRule{
		{
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
		},
		{
			ID:        uuid.New(),
			Priority:  1,
			Source:    "slack",
			Field:     "channel",
			Negate:    false,
			Pattern:   "^random$",
			Action:    "notified",
			Enabled:   true,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        uuid.New(),
			Priority:  2,
			Source:    "slack",
			Field:     "sender",
			Negate:    false,
			Pattern:   "^boss$",
			Action:    "notified",
			Enabled:   false, // disabled — should be excluded
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	engine := decisionengine.NewRulesEngine(rules)
	s.NotNil(engine)

	// A message matching the disabled rule's pattern should NOT match.
	msg := &repository.Message{
		Source: "slack",
		Sender: "boss",
	}
	action, matched := engine.Evaluate(msg)
	s.Equal("queue", action, "disabled rule should not match; expected 'queue'")
	s.Nil(matched, "disabled rule should not produce a matched rule")
}

func (s *RulesEngineSuite) TestNewRulesEngineSkipsInvalidPatterns() {
	now := time.Now()
	rules := []*repository.RoutingRule{
		{
			ID:        uuid.New(),
			Priority:  0,
			Source:    "slack",
			Field:     "content",
			Negate:    false,
			Pattern:   "^hello$",
			Action:    "notified",
			Enabled:   true,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        uuid.New(),
			Priority:  1,
			Source:    "slack",
			Field:     "content",
			Negate:    false,
			Pattern:   "[invalid", // invalid regex — should be silently excluded
			Action:    "ignored",
			Enabled:   true,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	engine := decisionengine.NewRulesEngine(rules)
	s.NotNil(engine)

	// The valid rule should still work.
	msg := &repository.Message{
		Source:     "slack",
		RawContent: "hello",
	}
	action, matched := engine.Evaluate(msg)
	s.Equal("notified", action)
	if s.NotNil(matched) {
		s.Equal("^hello$", matched.Pattern)
	}
}

func (s *RulesEngineSuite) TestNewRulesEngineSortsByPriority() {
	now := time.Now()
	// Pass rules in REVERSE priority order to verify sorting.
	rules := []*repository.RoutingRule{
		{
			ID:        uuid.New(),
			Priority:  1, // higher number = lower priority
			Source:    "slack",
			Field:     "channel",
			Negate:    false,
			Pattern:   "^alerts$",
			Action:    "ignored",
			Enabled:   true,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        uuid.New(),
			Priority:  0, // lower number = higher priority — should win
			Source:    "slack",
			Field:     "channel",
			Negate:    false,
			Pattern:   "^alerts$",
			Action:    "notified",
			Enabled:   true,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	engine := decisionengine.NewRulesEngine(rules)
	s.NotNil(engine)

	msg := &repository.Message{
		Source:  "slack",
		Channel: "alerts",
	}
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

	msg := &repository.Message{
		Source:     "slack",
		Channel:    "general",
		RawContent: "hello world",
	}
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
	now := time.Now()
	rules := []*repository.RoutingRule{
		{
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
		},
	}

	engine := decisionengine.NewRulesEngine(rules)

	msg := &repository.Message{
		Source:  "email",
		Channel: "general",
	}
	action, matched := engine.Evaluate(msg)
	s.Equal("queue", action, "source mismatch should fall through to queue")
	s.Nil(matched, "source mismatch should not produce a matched rule")
}

func (s *RulesEngineSuite) TestEvaluateFieldExtraction() {
	now := time.Now()

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
			msg: &repository.Message{
				Source: "slack",
				Sender: "alice",
			},
			wantAction: "notified",
		},
		{
			name:    "channel field",
			field:   "channel",
			pattern: "^alerts$",
			msg: &repository.Message{
				Source:  "slack",
				Channel: "alerts",
			},
			wantAction: "notified",
		},
		{
			name:    "content field",
			field:   "content",
			pattern: "urgent",
			msg: &repository.Message{
				Source:     "slack",
				RawContent: "this is urgent please respond",
			},
			wantAction: "notified",
		},
		{
			name:    "message_type field",
			field:   "message_type",
			pattern: "^channel_join$",
			msg: &repository.Message{
				Source:      "slack",
				MessageType: "channel_join",
			},
			wantAction: "notified",
		},
		{
			name:    "subject field extracts first line",
			field:   "subject",
			pattern: "^Important Subject$",
			msg: &repository.Message{
				Source:     "email",
				RawContent: "Important Subject\nBody text goes here",
			},
			wantAction: "notified",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			rules := []*repository.RoutingRule{
				{
					ID:        uuid.New(),
					Priority:  0,
					Source:    tc.msg.Source,
					Field:     tc.field,
					Negate:    false,
					Pattern:   tc.pattern,
					Action:    "notified",
					Enabled:   true,
					CreatedAt: now,
					UpdatedAt: now,
				},
			}
			engine := decisionengine.NewRulesEngine(rules)
			action, matched := engine.Evaluate(tc.msg)
			s.Equal(tc.wantAction, action)
			s.NotNil(matched)
		})
	}
}

func (s *RulesEngineSuite) TestEvaluateSubjectExtractionNoNewline() {
	now := time.Now()
	rules := []*repository.RoutingRule{
		{
			ID:        uuid.New(),
			Priority:  0,
			Source:    "email",
			Field:     "subject",
			Negate:    false,
			Pattern:   "^Single line content$",
			Action:    "notified",
			Enabled:   true,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	engine := decisionengine.NewRulesEngine(rules)

	msg := &repository.Message{
		Source:     "email",
		RawContent: "Single line content",
	}
	action, matched := engine.Evaluate(msg)
	s.Equal("notified", action, "subject should be entire RawContent when no newline present")
	s.NotNil(matched)
}

func (s *RulesEngineSuite) TestEvaluateNegation() {
	now := time.Now()
	rules := []*repository.RoutingRule{
		{
			ID:        uuid.New(),
			Priority:  0,
			Source:    "slack",
			Field:     "sender",
			Negate:    true,
			Pattern:   "^spam$",
			Action:    "notified",
			Enabled:   true,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	engine := decisionengine.NewRulesEngine(rules)

	// "boss" does NOT match "^spam$", negation inverts → true → should match rule
	msgBoss := &repository.Message{
		Source: "slack",
		Sender: "boss",
	}
	action, matched := engine.Evaluate(msgBoss)
	s.Equal("notified", action, "negated rule should match when pattern does NOT match field")
	s.NotNil(matched)

	// "spam" matches "^spam$", negation inverts → false → should NOT match rule
	msgSpam := &repository.Message{
		Source: "slack",
		Sender: "spam",
	}
	action2, matched2 := engine.Evaluate(msgSpam)
	s.Equal("queue", action2, "negated rule should not match when pattern matches field")
	s.Nil(matched2)
}

func (s *RulesEngineSuite) TestEvaluateFirstMatchWins() {
	now := time.Now()
	rules := []*repository.RoutingRule{
		{
			ID:        uuid.New(),
			Priority:  2,
			Source:    "slack",
			Field:     "channel",
			Negate:    false,
			Pattern:   "^alerts$",
			Action:    "ignored",
			Enabled:   true,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        uuid.New(),
			Priority:  0,
			Source:    "slack",
			Field:     "channel",
			Negate:    false,
			Pattern:   "^alerts$",
			Action:    "notified",
			Enabled:   true,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        uuid.New(),
			Priority:  1,
			Source:    "slack",
			Field:     "channel",
			Negate:    false,
			Pattern:   "^alerts$",
			Action:    "ignored",
			Enabled:   true,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	engine := decisionengine.NewRulesEngine(rules)

	msg := &repository.Message{
		Source:  "slack",
		Channel: "alerts",
	}
	action, matched := engine.Evaluate(msg)
	s.Equal("notified", action, "priority 0 rule should win (first match after sorting)")
	if s.NotNil(matched) {
		s.Equal(0, matched.Priority, "matched rule should be priority 0")
	}
}
