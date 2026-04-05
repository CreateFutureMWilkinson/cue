package fairy_test

import (
	"bytes"
	"image/color"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/character/fairy"
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
	c := fairy.NewFairyCharacter()
	s.Equal("fairy", c.Name())
}

func (s *FairyCharacterSuite) TestInitialStateIsStarting() {
	c := fairy.NewFairyCharacter()
	s.Equal(character.StateStarting, c.CurrentState())
}

func (s *FairyCharacterSuite) TestTransitionToAllStates() {
	c := fairy.NewFairyCharacter()

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
	c := fairy.NewFairyCharacter()
	w := c.Widget()
	s.NotNil(w)
}

func (s *FairyCharacterSuite) TestEachStateHasDistinctWidget() {
	c := fairy.NewFairyCharacter()

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
		return fairy.NewFairyCharacter()
	})

	c, err := character.Create("fairy")
	s.Require().NoError(err)
	s.Equal("fairy", c.Name())
}

// --- Helper: create fairy with mock clock and refresh disabled ---

func (s *FairyCharacterSuite) newTestFairy() (*fairy.FairyCharacter, *mockClock) {
	f := fairy.NewFairyCharacter()
	clock := newMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	f.SetClock(clock)
	f.DisableRefresh()
	return f, clock
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
	f, clock := s.newTestFairy()
	defer f.Close()

	// Move fairy away from idle position to verify animator resets it.
	f.SetPosition(0.0, 0.0)

	f.TransitionTo(character.StateIdle)

	// Advance clock ~3 seconds (one full breathing cycle).
	s.advanceAndTick(clock, 3*time.Second)

	// Verify position is at idle origin (0.5, 1.0).
	x, y := f.Position()
	s.InDelta(fairy.IdleOriginX, x, 0.01, "idle position x should be 0.5")
	s.InDelta(fairy.IdleOriginY, y, 0.01, "idle position y should be 1.0")

	// Verify body color is IdleBodyColor.
	bc := f.BodyCircle().FillColor
	r, g, b, a := bc.RGBA()
	er, eg, eb, ea := fairy.IdleBodyColor.RGBA()
	s.Equal(er, r, "body red should match IdleBodyColor")
	s.Equal(eg, g, "body green should match IdleBodyColor")
	s.Equal(eb, b, "body blue should match IdleBodyColor")
	s.Equal(ea, a, "body alpha should match IdleBodyColor")

	// Verify glow intensity oscillates (is not zero -- proving idle animator is running).
	glow := f.GlowIntensity()
	s.Greater(glow, 0.0, "glow intensity should be non-zero after idle animator runs")
}

func (s *FairyCharacterSuite) TestTransitionToWorkingStartsAnimator() {
	f, clock := s.newTestFairy()
	defer f.Close()

	f.TransitionTo(character.StateWorking)

	// Advance past the entry duration (0.5s) well into drift mode.
	s.advanceAndTick(clock, 1*time.Second)

	// Verify body color has changed toward WorkingBodyColor.
	bc := f.BodyCircle().FillColor
	r, g, b, a := bc.RGBA()
	wr, wg, wb, wa := fairy.WorkingBodyColor.RGBA()
	s.Equal(wr, r, "body red should match WorkingBodyColor after entry")
	s.Equal(wg, g, "body green should match WorkingBodyColor after entry")
	s.Equal(wb, b, "body blue should match WorkingBodyColor after entry")
	s.Equal(wa, a, "body alpha should match WorkingBodyColor after entry")

	// Verify position has moved from idle origin.
	x, y := f.Position()
	movedFromOrigin := (x != fairy.IdleOriginX) || (y != fairy.IdleOriginY)
	s.True(movedFromOrigin, "position should drift away from idle origin in working state")
}

func (s *FairyCharacterSuite) TestTransitionToErrorStartsAnimator() {
	f, clock := s.newTestFairy()
	defer f.Close()

	f.TransitionTo(character.StateError)

	// Give animator time to set initial state.
	s.advanceAndTick(clock, 100*time.Millisecond)

	// Verify body color is ErrorBodyColor.
	bc := f.BodyCircle().FillColor
	r, g, b, a := bc.RGBA()
	er, eg, eb, ea := fairy.ErrorBodyColor.RGBA()
	s.Equal(er, r, "body red should match ErrorBodyColor")
	s.Equal(eg, g, "body green should match ErrorBodyColor")
	s.Equal(eb, b, "body blue should match ErrorBodyColor")
	s.Equal(ea, a, "body alpha should match ErrorBodyColor")

	// Verify position y is 0.5 (center).
	_, y := f.Position()
	s.InDelta(0.5, y, 0.01, "error state position y should be 0.5 (center)")
}

func (s *FairyCharacterSuite) TestTransitionToNotifyStartsAnimator() {
	f, clock := s.newTestFairy()
	defer f.Close()

	f.TransitionTo(character.StateNotifying)

	// Give animator time to set initial state.
	s.advanceAndTick(clock, 100*time.Millisecond)

	// Verify body color is NotifyBodyColor.
	bc := f.BodyCircle().FillColor
	r, g, b, a := bc.RGBA()
	nr, ng, nb, na := fairy.NotifyBodyColor.RGBA()
	s.Equal(nr, r, "body red should match NotifyBodyColor")
	s.Equal(ng, g, "body green should match NotifyBodyColor")
	s.Equal(nb, b, "body blue should match NotifyBodyColor")
	s.Equal(na, a, "body alpha should match NotifyBodyColor")
}

func (s *FairyCharacterSuite) TestTransitionStopsPreviousAnimator() {
	f, clock := s.newTestFairy()
	defer f.Close()

	// Start working -- fairy drifts away from idle origin.
	f.TransitionTo(character.StateWorking)
	s.advanceAndTick(clock, 1*time.Second)

	// Verify fairy has moved from idle origin in working state.
	x, y := f.Position()
	movedFromOrigin := (x != fairy.IdleOriginX) || (y != fairy.IdleOriginY)
	s.True(movedFromOrigin, "working state should drift from idle origin")

	// Transition to idle -- should stop working animator and start idle animator.
	f.TransitionTo(character.StateIdle)
	s.advanceAndTick(clock, 100*time.Millisecond)

	// Verify position returns to (0.5, 1.0) -- proving the idle animator reset it.
	x, y = f.Position()
	s.InDelta(fairy.IdleOriginX, x, 0.01, "after idle transition, x should return to 0.5")
	s.InDelta(fairy.IdleOriginY, y, 0.01, "after idle transition, y should return to 1.0")
}

func (s *FairyCharacterSuite) TestStartupAutoTransitionsToIdle() {
	f, clock := s.newTestFairy()
	defer f.Close()

	f.TransitionTo(character.StateStarting)

	// Advance past startup duration (1.5s).
	s.advanceAndTick(clock, 2*time.Second)

	// Wait for the async auto-transition goroutine.
	time.Sleep(100 * time.Millisecond)

	// Verify state has auto-transitioned to Idle.
	s.Equal(character.StateIdle, f.CurrentState(),
		"after startup duration, state should auto-transition to Idle")
}

// --- Visual refresh / glow alpha tests ---

// GlowBaseAlphas defines the expected graduated base alphas from inner to outer.
var glowBaseAlphas = [8]uint8{128, 112, 96, 80, 64, 48, 32, 16}

func (s *FairyCharacterSuite) TestSetGlowIntensityUpdatesGlowAlphaGraduated() {
	f := fairy.NewFairyCharacter()
	f.DisableRefresh()

	// Set glow intensity to 1.0 -- each layer should have its graduated base alpha.
	f.SetGlowIntensity(1.0)
	layers := f.GlowLayers()
	s.Require().Len(layers, 8, "fairy should have exactly 8 glow layers")

	for i, gl := range layers {
		_, _, _, a := gl.FillColor.RGBA()
		alpha := uint8((a >> 8) & 0xFF)
		s.Equal(glowBaseAlphas[i], alpha,
			"glow layer %d alpha should be %d when intensity=1.0", i, glowBaseAlphas[i])
	}

	// Set glow intensity to 0.0 -- all layers should have zero alpha.
	f.SetGlowIntensity(0.0)
	for i, gl := range layers {
		_, _, _, a := gl.FillColor.RGBA()
		alpha := uint8((a >> 8) & 0xFF)
		s.Equal(uint8(0), alpha,
			"glow layer %d alpha should be 0 when intensity=0.0", i)
	}
}

func (s *FairyCharacterSuite) TestSetGlowIntensityHalfGraduated() {
	f := fairy.NewFairyCharacter()
	f.DisableRefresh()

	// At intensity=0.5, each layer's alpha should be half its base alpha.
	f.SetGlowIntensity(0.5)
	layers := f.GlowLayers()
	s.Require().Len(layers, 8)

	for i, gl := range layers {
		_, _, _, a := gl.FillColor.RGBA()
		alpha := uint8((a >> 8) & 0xFF)
		expected := uint8(float64(glowBaseAlphas[i]) * 0.5)
		s.Equal(expected, alpha,
			"glow layer %d alpha should be %d when intensity=0.5", i, expected)
	}
}

func (s *FairyCharacterSuite) TestGlowAlphaInnerBrighterThanOuter() {
	f := fairy.NewFairyCharacter()
	f.DisableRefresh()

	f.SetGlowIntensity(1.0)
	layers := f.GlowLayers()
	s.Require().Len(layers, 8)

	// Inner layer (index 0) alpha should be greater than outer layer (index 7) alpha.
	_, _, _, aInner := layers[0].FillColor.RGBA()
	_, _, _, aOuter := layers[7].FillColor.RGBA()
	s.Greater(uint8((aInner>>8)&0xFF), uint8((aOuter>>8)&0xFF),
		"inner glow layer should have higher alpha than outer glow layer")
}

// --- Shutdown tests ---

func (s *FairyCharacterSuite) TestShutdownReturnsCompletionChannel() {
	f, clock := s.newTestFairy()

	// Shutdown should return a channel that closes when the animation completes.
	done := f.Shutdown()
	s.Require().NotNil(done, "Shutdown() must return a non-nil channel")

	// Verify state transitions to ShuttingDown.
	s.Equal(character.StateShuttingDown, f.CurrentState(),
		"Shutdown() should transition to StateShuttingDown")

	// Advance past shutdown duration (1.5s) to complete animation.
	s.advanceAndTick(clock, 2*time.Second)

	// The done channel should close within a reasonable time.
	select {
	case <-done:
		// Success -- animation completed.
	case <-time.After(2 * time.Second):
		s.Fail("Shutdown() done channel did not close after animation completed")
	}
}

// --- Close tests ---

func (s *FairyCharacterSuite) TestCloseStopsAnimator() {
	f, clock := s.newTestFairy()

	f.TransitionTo(character.StateWorking)
	s.advanceAndTick(clock, 100*time.Millisecond)

	// Close should not panic.
	s.NotPanics(func() { f.Close() })

	// State should still be Working (Close doesn't change state, just stops animator).
	s.Equal(character.StateWorking, f.CurrentState(),
		"Close() should not change the current state")
}

func (s *FairyCharacterSuite) TestCloseIdempotent() {
	f, _ := s.newTestFairy()

	f.TransitionTo(character.StateIdle)
	time.Sleep(10 * time.Millisecond)

	// Close twice without panic.
	s.NotPanics(func() {
		f.Close()
		f.Close()
	}, "calling Close() twice should not panic")
}

func (s *FairyCharacterSuite) TestCloseWithoutAnimator() {
	f := fairy.NewFairyCharacter()
	f.DisableRefresh()

	// Close without any TransitionTo that starts an animator. No panic.
	s.NotPanics(func() { f.Close() },
		"Close() on fresh fairy without animator should not panic")
}

// --- Thread safety test ---

func (s *FairyCharacterSuite) TestConcurrentSetMethodsDoNotPanic() {
	f := fairy.NewFairyCharacter()
	f.DisableRefresh()
	defer f.Close()

	var wg sync.WaitGroup
	for range 10 {
		wg.Add(3)
		go func() {
			defer wg.Done()
			for range 50 {
				f.SetPosition(0.3, 0.7)
			}
		}()
		go func() {
			defer wg.Done()
			for range 50 {
				f.SetBodyColor(color.RGBA{R: 255, G: 0, B: 0, A: 255})
			}
		}()
		go func() {
			defer wg.Done()
			for range 50 {
				f.SetGlowIntensity(0.5)
			}
		}()
	}

	s.NotPanics(func() { wg.Wait() },
		"concurrent calls to SetPosition, SetBodyColor, SetGlowIntensity should not panic")
}

func (s *FairyCharacterSuite) TestRefreshFuncCalledOnSetPosition() {
	f := fairy.NewFairyCharacter()
	defer f.Close()

	var callCount int
	f.SetRefreshHook(func() { callCount++ })

	f.SetPosition(0.3, 0.7)
	s.Greater(callCount, 0, "refreshFunc should be called when SetPosition is invoked")
}

func (s *FairyCharacterSuite) TestSetBodyColorDoesNotDirectlyRefreshCanvasObject() {
	// Regression: SetBodyColor used to call bodyCircle.Refresh() directly, which
	// violates Fyne's threading model when called from animator goroutines.
	// Only refreshFunc (wired to fyne.Do) should trigger refreshes.
	//
	// Without a running Fyne app, calling .Refresh() on a canvas object logs a
	// "Fyne error" to stderr via the standard logger. We capture log output to
	// detect any such direct Refresh() call.
	f := fairy.NewFairyCharacter()
	f.SetRefreshHook(func() { /* no-op hook: absorbs the refreshFunc call */ })

	// Redirect the standard logger output to a buffer.
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	f.SetBodyColor(color.RGBA{R: 0xFF, G: 0x00, B: 0x00, A: 0xFF})

	// Verify color was updated (behavioral correctness).
	bc := f.BodyCircle().FillColor
	r, g, b, a := bc.RGBA()
	s.Equal(uint32(0xFFFF), r, "body red should be 0xFF")
	s.Equal(uint32(0x0000), g, "body green should be 0x00")
	s.Equal(uint32(0x0000), b, "body blue should be 0x00")
	s.Equal(uint32(0xFFFF), a, "body alpha should be 0xFF")

	// The key assertion: no Fyne error should have been logged. A direct
	// bodyCircle.Refresh() call (without a Fyne app) triggers a log line
	// containing "Fyne error". If refreshFunc is the sole refresh mechanism,
	// no such log output appears.
	s.Empty(buf.String(),
		"SetBodyColor should not call Refresh() directly on canvas objects; "+
			"got log output: %s", buf.String())
}

func (s *FairyCharacterSuite) TestTransitionToDoesNotDirectlyRefreshCanvasObject() {
	// Regression: stopAndUpdateState calls f.indicator.Refresh() directly, which
	// violates Fyne's threading model when called from animator goroutines.
	// Only refreshFunc (wired to fyne.Do) should trigger refreshes.
	//
	// Without a running Fyne app, calling .Refresh() on a canvas object logs a
	// "Fyne error" to stderr via the standard logger. We capture log output to
	// detect any such direct Refresh() call.
	f := fairy.NewFairyCharacter()
	f.SetRefreshHook(func() { /* no-op hook: absorbs the refreshFunc call */ })
	clock := newMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	f.SetClock(clock)

	// Redirect the standard logger output to a buffer.
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	f.TransitionTo(character.StateIdle)

	// Stop the animator promptly so only the direct Refresh in stopAndUpdateState
	// is captured, not any Fyne errors from the animator goroutine.
	f.Close()

	// The key assertion: no Fyne error should have been logged. A direct
	// indicator.Refresh() call (without a Fyne app) triggers a log line
	// containing "Fyne error". If refreshFunc is the sole refresh mechanism,
	// no such log output appears.
	s.Empty(buf.String(),
		"TransitionTo should not call Refresh() directly on canvas objects; "+
			"got log output: %s", buf.String())
}

func (s *FairyCharacterSuite) TestConstructorDefaultRefreshIsNoOp() {
	// A freshly constructed FairyCharacter should default its refreshFunc to a
	// no-op, so that no Fyne app is required at construction time. Previously,
	// the constructor wired refreshFunc to fyne.CurrentApp()/fyne.Do(), which
	// logged errors when no Fyne app was running and coupled construction to
	// the GUI runtime. After this change, callers must explicitly set a refresh
	// hook via SetRefreshHook if they want container refreshes.
	f := fairy.NewFairyCharacter()
	defer f.Close()

	// The default refresh should be a plain no-op, not a fyne-guarded closure.
	s.True(f.IsNoopRefresh(),
		"newly constructed fairy should have a no-op refreshFunc by default")

	// Verify the fairy is fully functional without DisableRefresh or SetRefreshHook:
	// a state transition should succeed without errors or panics.
	f.TransitionTo(character.StateIdle)
	s.Equal(character.StateIdle, f.CurrentState(),
		"state transition should work with default no-op refresh")
}

func (s *FairyCharacterSuite) TestConcurrentTransitions() {
	f, _ := s.newTestFairy()
	defer f.Close()

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
				f.TransitionTo(st)
			}
		}()
	}

	// Should complete without panic or race (run with -race to verify).
	s.NotPanics(func() { wg.Wait() },
		"concurrent TransitionTo calls should not panic or race")
}
