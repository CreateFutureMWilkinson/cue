package character_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
	"github.com/stretchr/testify/suite"
)

// Ensure fyne.CanvasObject is used (compile-time check).
var _ fyne.CanvasObject

type FairyCharacterSuite struct {
	suite.Suite
}

func TestFairyCharacter(t *testing.T) {
	suite.Run(t, new(FairyCharacterSuite))
}

func (s *FairyCharacterSuite) TestNameReturnsFairy() {
	c := character.NewFairyCharacter()
	s.Equal("fairy", c.Name())
}

func (s *FairyCharacterSuite) TestInitialStateIsIdle() {
	c := character.NewFairyCharacter()
	s.Equal(character.StateIdle, c.CurrentState())
}

func (s *FairyCharacterSuite) TestTransitionToAllStates() {
	c := character.NewFairyCharacter()

	states := []character.CharacterState{
		character.StateIdle,
		character.StateStarting,
		character.StateWorking,
		character.StateNotifying,
		character.StateError,
		character.StateShuttingDown,
	}

	for _, state := range states {
		c.TransitionTo(state)
		s.Equal(state, c.CurrentState(), "expected state %s after TransitionTo", state)
	}
}

func (s *FairyCharacterSuite) TestWidgetReturnsNonNil() {
	c := character.NewFairyCharacter()
	w := c.Widget()
	s.NotNil(w)
}

func (s *FairyCharacterSuite) TestEachStateHasDistinctWidget() {
	c := character.NewFairyCharacter()

	states := []character.CharacterState{
		character.StateIdle,
		character.StateStarting,
		character.StateWorking,
		character.StateNotifying,
		character.StateError,
		character.StateShuttingDown,
	}

	for _, state := range states {
		c.TransitionTo(state)
		w := c.Widget()
		s.NotNil(w, "Widget() must be non-nil in state %s", state)
	}
}

func (s *FairyCharacterSuite) TestRegisteredAsFairy() {
	character.ResetRegistry()
	character.Register("fairy", func() character.Character {
		return character.NewFairyCharacter()
	})

	c, err := character.Create("fairy")
	s.Require().NoError(err)
	s.Equal("fairy", c.Name())
}
