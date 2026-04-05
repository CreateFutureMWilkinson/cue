package characteruat_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	characteruat "github.com/CreateFutureMWilkinson/cue/cmd/character-uat"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/character/fairy"
	"github.com/stretchr/testify/suite"
)

type UATWindowSuite struct {
	suite.Suite
}

func TestUATWindow(t *testing.T) {
	suite.Run(t, new(UATWindowSuite))
}

func (s *UATWindowSuite) SetupTest() {
	character.ResetRegistry()
	character.Register("fairy", func() character.Character {
		return fairy.NewFairyCharacter()
	})
}

func (s *UATWindowSuite) TestAvailableCharactersIncludesFairy() {
	names := character.Available()
	s.Contains(names, "fairy", "registry should include fairy after registration")
}

func (s *UATWindowSuite) TestCreateCharacterReturnsValidCharacter() {
	ch, err := character.Create("fairy")
	s.Require().NoError(err, "creating a registered character should not error")
	s.Require().NotNil(ch, "created character should not be nil")
	s.Equal("fairy", ch.Name(), "character Name() should match registered name")
}

func (s *UATWindowSuite) TestCreatedCharacterStartsInStartingState() {
	ch, err := character.Create("fairy")
	s.Require().NoError(err)
	s.Equal(character.StateStarting, ch.CurrentState(),
		"newly created character should start in StateStarting")
}

func (s *UATWindowSuite) TestTriggerStateUpdatesCharacter() {
	ch, err := character.Create("fairy")
	s.Require().NoError(err)

	ch.TransitionTo(character.StateWorking)
	s.Equal(character.StateWorking, ch.CurrentState(),
		"CurrentState should reflect the state set via TransitionTo")
}

func (s *UATWindowSuite) TestAllStatesAccessible() {
	ch, err := character.Create("fairy")
	s.Require().NoError(err)

	allStates := []character.CharacterState{
		character.StateIdle,
		character.StateStarting,
		character.StateWorking,
		character.StateNotifying,
		character.StateError,
		character.StateShuttingDown,
	}

	for _, state := range allStates {
		ch.TransitionTo(state)
		s.Equal(state, ch.CurrentState(),
			"CurrentState should be %s after TransitionTo(%s)", state, state)
	}
}

func (s *UATWindowSuite) TestCharacterSwapResetsToInitialState() {
	// Create first character and change its state.
	ch1, err := character.Create("fairy")
	s.Require().NoError(err)
	ch1.TransitionTo(character.StateError)
	s.Equal(character.StateError, ch1.CurrentState(),
		"first character should be in Error state")

	// Create a second character (simulating a swap in the UAT window).
	ch2, err := character.Create("fairy")
	s.Require().NoError(err)
	s.Equal(character.StateStarting, ch2.CurrentState(),
		"newly created character after swap should start in StateStarting")
}

func (s *UATWindowSuite) TestProductionFairyWiresRefreshHook() {
	// The SetupTest registers fairy the same way production callers do.
	// Production callers MUST wire SetRefreshHook in the factory so the
	// fairy actually refreshes visually. This test verifies that the
	// standard registration (shared with cmd/cue and cmd/cue-uat) produces
	// a fairy whose refreshFunc is NOT the default no-op.
	ch, err := character.Create("fairy")
	s.Require().NoError(err)

	fc, ok := ch.(*fairy.FairyCharacter)
	s.Require().True(ok, "created character should be *fairy.FairyCharacter")

	s.False(fc.IsNoopRefresh(),
		"production fairy registration must wire SetRefreshHook so IsNoopRefresh returns false")
}

func (s *UATWindowSuite) TestCharacterPanelFillsSpace() {
	ch, err := character.Create("fairy")
	s.Require().NoError(err)

	charContainer := container.NewStack()
	charContainer.Add(ch.Widget())

	panel := characteruat.NewCharacterPanel(charContainer)
	s.Require().NotNil(panel, "NewCharacterPanel must return a non-nil container")

	// Resize the panel and verify the charContainer fills the space.
	panel.Resize(fyne.NewSize(400, 600))
	s.InDelta(400, charContainer.Size().Width, 1.0,
		"character container should fill panel width")
	s.InDelta(600, charContainer.Size().Height, 1.0,
		"character container should fill panel height")
}
