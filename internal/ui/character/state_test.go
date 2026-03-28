package character_test

import (
	"testing"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
	"github.com/stretchr/testify/suite"
)

type CharacterStateSuite struct {
	suite.Suite
}

func TestCharacterState(t *testing.T) {
	suite.Run(t, new(CharacterStateSuite))
}

func (s *CharacterStateSuite) TestAllStatesHaveCorrectStringRepresentations() {
	tests := []struct {
		state    character.CharacterState
		expected string
	}{
		{character.StateIdle, "Idle"},
		{character.StateStarting, "Starting"},
		{character.StateWorking, "Working"},
		{character.StateNotifying, "Notifying"},
		{character.StateError, "Error"},
		{character.StateShuttingDown, "ShuttingDown"},
	}

	for _, tc := range tests {
		s.Run(tc.expected, func() {
			s.Equal(tc.expected, tc.state.String())
		})
	}
}

func (s *CharacterStateSuite) TestAllStateValuesAreDistinct() {
	states := []character.CharacterState{
		character.StateIdle,
		character.StateStarting,
		character.StateWorking,
		character.StateNotifying,
		character.StateError,
		character.StateShuttingDown,
	}

	seen := make(map[character.CharacterState]bool)
	for _, state := range states {
		s.False(seen[state], "duplicate state value found: %v", state)
		seen[state] = true
	}
}
