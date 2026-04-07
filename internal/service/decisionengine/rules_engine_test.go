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
