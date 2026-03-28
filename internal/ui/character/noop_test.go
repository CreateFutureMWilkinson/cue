package character_test

import (
	"testing"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
	"github.com/stretchr/testify/suite"
)

type NoOpCharacterSuite struct {
	suite.Suite
}

func TestNoOpCharacter(t *testing.T) {
	suite.Run(t, new(NoOpCharacterSuite))
}

func (s *NoOpCharacterSuite) TestNameReturnsNone() {
	c := character.NewNoOpCharacter()
	s.Equal("none", c.Name())
}

func (s *NoOpCharacterSuite) TestTransitionToAcceptsAllStatesWithoutPanicking() {
	c := character.NewNoOpCharacter()

	states := []character.CharacterState{
		character.StateIdle,
		character.StateStarting,
		character.StateWorking,
		character.StateNotifying,
		character.StateError,
		character.StateShuttingDown,
	}

	for _, state := range states {
		s.NotPanics(func() {
			c.TransitionTo(state)
		})
	}
}

func (s *NoOpCharacterSuite) TestCurrentStateReturnsIdleInitiallyThenLastTransitioned() {
	c := character.NewNoOpCharacter()

	// Initially idle
	s.Equal(character.StateIdle, c.CurrentState())

	// After transition, returns last state
	c.TransitionTo(character.StateWorking)
	s.Equal(character.StateWorking, c.CurrentState())

	c.TransitionTo(character.StateNotifying)
	s.Equal(character.StateNotifying, c.CurrentState())
}

func (s *NoOpCharacterSuite) TestWidgetReturnsNonNil() {
	c := character.NewNoOpCharacter()
	w := c.Widget()
	s.NotNil(w)
}
