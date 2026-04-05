package character_test

import (
	"sync"
	"testing"
	"time"

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

// --- Helper: create fairy with mock clock and refresh disabled ---

func (s *FairyCharacterSuite) newTestFairy() (*character.FairyCharacter, *mockClock) {
	fairy := character.NewFairyCharacter()
	clock := newMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	fairy.SetClock(clock)
	fairy.DisableRefresh()
	return fairy, clock
}

// advanceAndTick advances the mock clock and sends a tick to trigger animation frames.
// It sends multiple ticks to ensure the animator goroutine processes them.
func (s *FairyCharacterSuite) advanceAndTick(clock *mockClock, d time.Duration) {
	// Advance in small increments to give the animator goroutine time to process.
	steps := 10
	stepDuration := d / time.Duration(steps)
	for range steps {
		clock.Advance(stepDuration)
		time.Sleep(5 * time.Millisecond) // Let the goroutine process the tick.
	}
}

// --- TransitionTo animator wiring tests ---

func (s *FairyCharacterSuite) TestTransitionToIdleStartsAnimator() {
	fairy, clock := s.newTestFairy()
	defer fairy.Close()

	// Move fairy away from idle position to verify animator resets it.
	fairy.SetPosition(0.0, 0.0)

	fairy.TransitionTo(character.StateIdle)

	// Advance clock ~3 seconds (one full breathing cycle).
	s.advanceAndTick(clock, 3*time.Second)

	// Verify position is at idle origin (0.5, 1.0).
	x, y := fairy.Position()
	s.InDelta(character.IdleOriginX, x, 0.01, "idle position x should be 0.5")
	s.InDelta(character.IdleOriginY, y, 0.01, "idle position y should be 1.0")

	// Verify body color is IdleBodyColor.
	bc := fairy.BodyCircle().FillColor
	r, g, b, a := bc.RGBA()
	er, eg, eb, ea := character.IdleBodyColor.RGBA()
	s.Equal(er, r, "body red should match IdleBodyColor")
	s.Equal(eg, g, "body green should match IdleBodyColor")
	s.Equal(eb, b, "body blue should match IdleBodyColor")
	s.Equal(ea, a, "body alpha should match IdleBodyColor")

	// Verify glow intensity oscillates (is not zero — proving idle animator is running).
	glow := fairy.GlowIntensity()
	s.Greater(glow, 0.0, "glow intensity should be non-zero after idle animator runs")
}

func (s *FairyCharacterSuite) TestTransitionToWorkingStartsAnimator() {
	fairy, clock := s.newTestFairy()
	defer fairy.Close()

	fairy.TransitionTo(character.StateWorking)

	// Advance past the entry duration (0.5s) well into drift mode.
	s.advanceAndTick(clock, 1*time.Second)

	// Verify body color has changed toward WorkingBodyColor.
	bc := fairy.BodyCircle().FillColor
	r, g, b, a := bc.RGBA()
	wr, wg, wb, wa := character.WorkingBodyColor.RGBA()
	s.Equal(wr, r, "body red should match WorkingBodyColor after entry")
	s.Equal(wg, g, "body green should match WorkingBodyColor after entry")
	s.Equal(wb, b, "body blue should match WorkingBodyColor after entry")
	s.Equal(wa, a, "body alpha should match WorkingBodyColor after entry")

	// Verify position has moved from idle origin.
	x, y := fairy.Position()
	movedFromOrigin := (x != character.IdleOriginX) || (y != character.IdleOriginY)
	s.True(movedFromOrigin, "position should drift away from idle origin in working state")
}

func (s *FairyCharacterSuite) TestTransitionToErrorStartsAnimator() {
	fairy, clock := s.newTestFairy()
	defer fairy.Close()

	fairy.TransitionTo(character.StateError)

	// Give animator time to set initial state.
	s.advanceAndTick(clock, 100*time.Millisecond)

	// Verify body color is ErrorBodyColor.
	bc := fairy.BodyCircle().FillColor
	r, g, b, a := bc.RGBA()
	er, eg, eb, ea := character.ErrorBodyColor.RGBA()
	s.Equal(er, r, "body red should match ErrorBodyColor")
	s.Equal(eg, g, "body green should match ErrorBodyColor")
	s.Equal(eb, b, "body blue should match ErrorBodyColor")
	s.Equal(ea, a, "body alpha should match ErrorBodyColor")

	// Verify position y is 0.5 (center).
	_, y := fairy.Position()
	s.InDelta(0.5, y, 0.01, "error state position y should be 0.5 (center)")
}

func (s *FairyCharacterSuite) TestTransitionToNotifyStartsAnimator() {
	fairy, clock := s.newTestFairy()
	defer fairy.Close()

	fairy.TransitionTo(character.StateNotifying)

	// Give animator time to set initial state.
	s.advanceAndTick(clock, 100*time.Millisecond)

	// Verify body color is NotifyBodyColor.
	bc := fairy.BodyCircle().FillColor
	r, g, b, a := bc.RGBA()
	nr, ng, nb, na := character.NotifyBodyColor.RGBA()
	s.Equal(nr, r, "body red should match NotifyBodyColor")
	s.Equal(ng, g, "body green should match NotifyBodyColor")
	s.Equal(nb, b, "body blue should match NotifyBodyColor")
	s.Equal(na, a, "body alpha should match NotifyBodyColor")
}

func (s *FairyCharacterSuite) TestTransitionStopsPreviousAnimator() {
	fairy, clock := s.newTestFairy()
	defer fairy.Close()

	// Start working — fairy drifts away from idle origin.
	fairy.TransitionTo(character.StateWorking)
	s.advanceAndTick(clock, 1*time.Second)

	// Verify fairy has moved from idle origin in working state.
	x, y := fairy.Position()
	movedFromOrigin := (x != character.IdleOriginX) || (y != character.IdleOriginY)
	s.True(movedFromOrigin, "working state should drift from idle origin")

	// Transition to idle — should stop working animator and start idle animator.
	fairy.TransitionTo(character.StateIdle)
	s.advanceAndTick(clock, 100*time.Millisecond)

	// Verify position returns to (0.5, 1.0) — proving the idle animator reset it.
	x, y = fairy.Position()
	s.InDelta(character.IdleOriginX, x, 0.01, "after idle transition, x should return to 0.5")
	s.InDelta(character.IdleOriginY, y, 0.01, "after idle transition, y should return to 1.0")
}

func (s *FairyCharacterSuite) TestStartupAutoTransitionsToIdle() {
	fairy, clock := s.newTestFairy()
	defer fairy.Close()

	fairy.TransitionTo(character.StateStarting)

	// Advance past startup duration (1.5s).
	s.advanceAndTick(clock, 2*time.Second)

	// Wait for the async auto-transition goroutine.
	time.Sleep(100 * time.Millisecond)

	// Verify state has auto-transitioned to Idle.
	s.Equal(character.StateIdle, fairy.CurrentState(),
		"after startup duration, state should auto-transition to Idle")
}

// --- Visual refresh / glow alpha tests ---

func (s *FairyCharacterSuite) TestSetGlowIntensityUpdatesGlowAlpha() {
	fairy := character.NewFairyCharacter()
	fairy.DisableRefresh()

	// Set glow intensity to 1.0 — glow layers should have non-zero alpha.
	fairy.SetGlowIntensity(1.0)
	layers := fairy.GlowLayers()
	s.Require().NotEmpty(layers, "fairy should have glow layers")

	for i, gl := range layers {
		_, _, _, a := gl.FillColor.RGBA()
		alpha := uint8((a >> 8) & 0xFF)
		s.Equal(uint8(30), alpha,
			"glow layer %d alpha should be glowAlpha (30) when intensity=1.0", i)
	}

	// Set glow intensity to 0.0 — glow layers should have zero alpha.
	fairy.SetGlowIntensity(0.0)
	for i, gl := range layers {
		_, _, _, a := gl.FillColor.RGBA()
		alpha := uint8((a >> 8) & 0xFF)
		s.Equal(uint8(0), alpha,
			"glow layer %d alpha should be 0 when intensity=0.0", i)
	}
}

// --- Close tests ---

func (s *FairyCharacterSuite) TestCloseStopsAnimator() {
	fairy, clock := s.newTestFairy()

	fairy.TransitionTo(character.StateWorking)
	s.advanceAndTick(clock, 100*time.Millisecond)

	// Close should not panic.
	s.NotPanics(func() { fairy.Close() })

	// State should still be Working (Close doesn't change state, just stops animator).
	s.Equal(character.StateWorking, fairy.CurrentState(),
		"Close() should not change the current state")
}

func (s *FairyCharacterSuite) TestCloseIdempotent() {
	fairy, _ := s.newTestFairy()

	fairy.TransitionTo(character.StateIdle)
	time.Sleep(10 * time.Millisecond)

	// Close twice without panic.
	s.NotPanics(func() {
		fairy.Close()
		fairy.Close()
	}, "calling Close() twice should not panic")
}

func (s *FairyCharacterSuite) TestCloseWithoutAnimator() {
	fairy := character.NewFairyCharacter()
	fairy.DisableRefresh()

	// Close without any TransitionTo that starts an animator. No panic.
	s.NotPanics(func() { fairy.Close() },
		"Close() on fresh fairy without animator should not panic")
}

// --- Thread safety test ---

func (s *FairyCharacterSuite) TestConcurrentTransitions() {
	fairy, _ := s.newTestFairy()
	defer fairy.Close()

	states := []character.CharacterState{
		character.StateIdle,
		character.StateStarting,
		character.StateWorking,
		character.StateNotifying,
		character.StateError,
		character.StateShuttingDown,
	}

	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, st := range states {
				fairy.TransitionTo(st)
			}
		}()
	}

	// Should complete without panic or race (run with -race to verify).
	s.NotPanics(func() { wg.Wait() },
		"concurrent TransitionTo calls should not panic or race")
}
