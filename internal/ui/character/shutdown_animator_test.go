package character_test

import (
	"image/color"
	"testing"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
	"github.com/stretchr/testify/suite"
)

// ShutdownAnimatorSuite tests the ShutdownAnimator lifecycle and animation behavior.
type ShutdownAnimatorSuite struct {
	suite.Suite
	clock *mockClock
}

func TestShutdownAnimator(t *testing.T) {
	suite.Run(t, new(ShutdownAnimatorSuite))
}

func (s *ShutdownAnimatorSuite) SetupTest() {
	s.clock = newMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
}

// --- State() returns StateShuttingDown ---

func (s *ShutdownAnimatorSuite) TestStateReturnsShuttingDown() {
	animator := character.NewShutdownAnimator(s.clock)
	s.Equal(character.StateShuttingDown, animator.State(),
		"ShutdownAnimator.State() must return StateShuttingDown")
}

// --- Captures current state on Start ---

func (s *ShutdownAnimatorSuite) TestCapturesCurrentStateOnStart() {
	fairy := character.NewFairyCharacter()
	// Set fairy to a non-default state.
	fairy.SetPosition(0.2, 0.3)
	fairy.SetBodyColor(color.RGBA{R: 0x00, G: 0xB8, B: 0x00, A: 0xFF})
	fairy.SetGlowIntensity(0.8)

	animator := character.NewShutdownAnimator(s.clock)
	animator.Start(fairy)
	defer animator.Stop()

	// Advance to midpoint (0.75s).
	s.clock.Advance(750 * time.Millisecond)
	time.Sleep(10 * time.Millisecond) // Let animation goroutine process.

	// Position should be between start (0.2, 0.3) and dormant (0.5, 1.0).
	x, y := fairy.Position()
	s.Greater(x, 0.2, "x at midpoint should be above start (0.2)")
	s.Less(x, 0.5, "x at midpoint should be below dormant (0.5)")
	s.Greater(y, 0.3, "y at midpoint should be above start (0.3)")
	s.Less(y, 1.0, "y at midpoint should be below dormant (1.0)")

	// Glow should be between start (0.8) and dormant (0.15).
	glow := fairy.GlowIntensity()
	s.Less(glow, 0.8, "glow at midpoint should be below start (0.8)")
	s.Greater(glow, 0.15, "glow at midpoint should be above dormant (0.15)")
}

// --- Final state is dormant after 1.5s ---

func (s *ShutdownAnimatorSuite) TestFinalStateIsDormant() {
	fairy := character.NewFairyCharacter()
	// Start from idle defaults.
	fairy.SetGlowIntensity(0.5)

	animator := character.NewShutdownAnimator(s.clock)
	animator.Start(fairy)
	defer animator.Stop()

	// Advance past the full 1.5s shutdown duration.
	s.clock.Advance(1600 * time.Millisecond)
	time.Sleep(10 * time.Millisecond) // Let animation goroutine process.

	// Position should be dormant (0.5, 1.0).
	x, y := fairy.Position()
	s.InDelta(0.5, x, 1e-9, "final position x must be 0.5")
	s.InDelta(1.0, y, 1e-9, "final position y must be 1.0")

	// Body color should be dormant (#004900).
	bodyCircle := fairy.BodyCircle()
	s.Require().NotNil(bodyCircle)

	expected := color.RGBA{R: 0x00, G: 0x49, B: 0x00, A: 0xFF}
	r1, g1, b1, a1 := bodyCircle.FillColor.RGBA()
	r2, g2, b2, a2 := expected.RGBA()
	s.Equal(r2, r1, "final body red channel should be 0x00")
	s.Equal(g2, g1, "final body green channel should be 0x49")
	s.Equal(b2, b1, "final body blue channel should be 0x00")
	s.Equal(a2, a1, "final body alpha channel should be 0xFF")

	// Glow intensity should be shutdown dormant (0.15).
	s.InDelta(character.ShutdownDormantGlowIntensity, fairy.GlowIntensity(), 0.01,
		"glow intensity at end should be shutdown dormant (0.15)")
}

// --- Done channel closes on completion ---

func (s *ShutdownAnimatorSuite) TestDoneChannelClosesOnCompletion() {
	fairy := character.NewFairyCharacter()
	animator := character.NewShutdownAnimator(s.clock)

	animator.Start(fairy)
	defer animator.Stop()

	// Advance past the full 1.5s shutdown duration.
	s.clock.Advance(1600 * time.Millisecond)
	time.Sleep(10 * time.Millisecond) // Let animation goroutine process.

	// Done channel should be closed.
	select {
	case <-animator.Done():
		// Success — channel is closed.
	case <-time.After(time.Second):
		s.Fail("Done() channel was not closed within timeout")
	}
}

// --- Done channel before Start ---

func (s *ShutdownAnimatorSuite) TestDoneChannelBeforeStart() {
	animator := character.NewShutdownAnimator(s.clock)

	// Done() should be safe to call before Start — returns a non-nil channel
	// that is not yet closed.
	done := animator.Done()
	s.NotNil(done, "Done() should return a non-nil channel even before Start")

	// The channel should NOT be closed before Start.
	select {
	case <-done:
		s.Fail("Done() channel should not be closed before Start")
	default:
		// Expected — channel is open.
	}
}

// --- Position interpolation from custom start ---

func (s *ShutdownAnimatorSuite) TestPositionInterpolationFromCustomStart() {
	fairy := character.NewFairyCharacter()
	fairy.SetPosition(0.2, 0.3)

	animator := character.NewShutdownAnimator(s.clock)
	animator.Start(fairy)
	defer animator.Stop()

	// Advance past the full 1.5s shutdown duration.
	s.clock.Advance(1600 * time.Millisecond)
	time.Sleep(10 * time.Millisecond) // Let animation goroutine process.

	// Final position should be dormant (0.5, 1.0).
	x, y := fairy.Position()
	s.InDelta(0.5, x, 1e-9, "final position x must be 0.5")
	s.InDelta(1.0, y, 1e-9, "final position y must be 1.0")
}

// --- Color interpolation at midpoint ---

func (s *ShutdownAnimatorSuite) TestColorInterpolationAtMidpoint() {
	fairy := character.NewFairyCharacter()
	// Set body color to idle (#006100) — a non-dormant color.
	fairy.SetBodyColor(character.IdleBodyColor)

	animator := character.NewShutdownAnimator(s.clock)
	animator.Start(fairy)
	defer animator.Stop()

	// Advance to ~0.75s (midpoint of 1.5s animation).
	s.clock.Advance(750 * time.Millisecond)
	time.Sleep(10 * time.Millisecond) // Let animation goroutine process.

	// At midpoint, color should be between idle (#006100) and dormant (#004900).
	bodyCircle := fairy.BodyCircle()
	s.Require().NotNil(bodyCircle)
	_, g, _, _ := bodyCircle.FillColor.RGBA()

	// Green channel in pre-multiplied 16-bit: dormant=0x4949, idle=0x6161.
	dormantG := uint32(0x49) * 0x101
	idleG := uint32(0x61) * 0x101
	s.Greater(g, dormantG, "green channel at midpoint should be above dormant")
	s.Less(g, idleG, "green channel at midpoint should be below idle")
}

// --- Glow interpolation at midpoint ---

func (s *ShutdownAnimatorSuite) TestGlowInterpolationAtMidpoint() {
	fairy := character.NewFairyCharacter()
	fairy.SetGlowIntensity(0.8)

	animator := character.NewShutdownAnimator(s.clock)
	animator.Start(fairy)
	defer animator.Stop()

	// Advance to ~0.75s (midpoint of 1.5s animation).
	s.clock.Advance(750 * time.Millisecond)
	time.Sleep(10 * time.Millisecond) // Let animation goroutine process.

	// At midpoint, glow should be between start (0.8) and dormant (0.15).
	glow := fairy.GlowIntensity()
	s.Less(glow, 0.8,
		"glow intensity at midpoint should be below start (0.8)")
	s.Greater(glow, character.ShutdownDormantGlowIntensity,
		"glow intensity at midpoint should be above shutdown dormant (0.15)")
}

// --- Animation from idle state ---

func (s *ShutdownAnimatorSuite) TestAnimationFromIdleState() {
	fairy := character.NewFairyCharacter()
	// Fairy starts at idle defaults: position (0.5, 1.0), color #006100.
	fairy.SetGlowIntensity(0.5)

	animator := character.NewShutdownAnimator(s.clock)
	animator.Start(fairy)
	defer animator.Stop()

	// Advance past the full 1.5s shutdown duration.
	s.clock.Advance(1600 * time.Millisecond)
	time.Sleep(10 * time.Millisecond) // Let animation goroutine process.

	// Final position should still be (0.5, 1.0) — same as idle origin.
	x, y := fairy.Position()
	s.InDelta(0.5, x, 1e-9, "final position x must be 0.5")
	s.InDelta(1.0, y, 1e-9, "final position y must be 1.0")

	// Body color should be dormant (#004900).
	bodyCircle := fairy.BodyCircle()
	s.Require().NotNil(bodyCircle)

	expected := color.RGBA{R: 0x00, G: 0x49, B: 0x00, A: 0xFF}
	r1, g1, b1, a1 := bodyCircle.FillColor.RGBA()
	r2, g2, b2, a2 := expected.RGBA()
	s.Equal(r2, r1, "body red channel should be dormant")
	s.Equal(g2, g1, "body green channel should be dormant")
	s.Equal(b2, b1, "body blue channel should be dormant")
	s.Equal(a2, a1, "body alpha channel should be dormant")

	// Glow intensity should be shutdown dormant (0.15).
	s.InDelta(character.ShutdownDormantGlowIntensity, fairy.GlowIntensity(), 0.01,
		"glow intensity at end should be shutdown dormant (0.15)")
}

// --- Animation from error state ---

func (s *ShutdownAnimatorSuite) TestAnimationFromErrorState() {
	fairy := character.NewFairyCharacter()
	// Simulate error state: position (0.5, 0.5), color #00B800, glow 0.65.
	fairy.SetPosition(0.5, 0.5)
	fairy.SetBodyColor(color.RGBA{R: 0x00, G: 0xB8, B: 0x00, A: 0xFF})
	fairy.SetGlowIntensity(0.65)

	animator := character.NewShutdownAnimator(s.clock)
	animator.Start(fairy)
	defer animator.Stop()

	// Advance past the full 1.5s shutdown duration.
	s.clock.Advance(1600 * time.Millisecond)
	time.Sleep(10 * time.Millisecond) // Let animation goroutine process.

	// Final position should be dormant (0.5, 1.0).
	x, y := fairy.Position()
	s.InDelta(0.5, x, 1e-9, "final position x must be 0.5")
	s.InDelta(1.0, y, 1e-9, "final position y must be 1.0")

	// Body color should be dormant (#004900).
	bodyCircle := fairy.BodyCircle()
	s.Require().NotNil(bodyCircle)

	expected := color.RGBA{R: 0x00, G: 0x49, B: 0x00, A: 0xFF}
	r1, g1, b1, a1 := bodyCircle.FillColor.RGBA()
	r2, g2, b2, a2 := expected.RGBA()
	s.Equal(r2, r1, "body red channel should be dormant")
	s.Equal(g2, g1, "body green channel should be dormant")
	s.Equal(b2, b1, "body blue channel should be dormant")
	s.Equal(a2, a1, "body alpha channel should be dormant")

	// Glow intensity should be shutdown dormant (0.15).
	s.InDelta(character.ShutdownDormantGlowIntensity, fairy.GlowIntensity(), 0.01,
		"glow intensity at end should be shutdown dormant (0.15)")
}

// --- Shutdown duration constant ---

func (s *ShutdownAnimatorSuite) TestShutdownDurationConstant() {
	s.Equal(1.5, character.ShutdownDurationSec,
		"shutdown duration must be 1.5 seconds")
}

// --- Shutdown dormant glow constant ---

func (s *ShutdownAnimatorSuite) TestShutdownDormantGlowConstant() {
	s.Equal(0.15, character.ShutdownDormantGlowIntensity,
		"shutdown dormant glow intensity must be 0.15")
}

// --- Stop waits for completion ---

func (s *ShutdownAnimatorSuite) TestStopWaitsForCompletion() {
	fairy := character.NewFairyCharacter()
	animator := character.NewShutdownAnimator(s.clock)

	animator.Start(fairy)

	// Advance clock past full 1.5s duration so the animation can complete.
	s.clock.Advance(1600 * time.Millisecond)
	time.Sleep(10 * time.Millisecond) // Let animation goroutine process.

	// Stop should return without hanging because animation has completed.
	done := make(chan struct{})
	go func() {
		animator.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Success — Stop() returned.
	case <-time.After(time.Second):
		s.Fail("Stop() did not return within timeout — may be blocking indefinitely")
	}
}

// --- StateAnimator interface compliance ---

func (s *ShutdownAnimatorSuite) TestImplementsStateAnimator() {
	animator := character.NewShutdownAnimator(s.clock)
	// Compile-time check that *ShutdownAnimator satisfies StateAnimator.
	var _ character.StateAnimator = animator
}

// --- Color capture preserves arbitrary channel values ---

func (s *ShutdownAnimatorSuite) TestColorCapturePreservesChannelValues() {
	fairy := character.NewFairyCharacter()
	// Set fairy to a color with distinct channel values.
	fairy.SetBodyColor(color.RGBA{R: 0xAA, G: 0xBB, B: 0xCC, A: 0xFF})

	animator := character.NewShutdownAnimator(s.clock)
	animator.Start(fairy)
	defer animator.Stop()

	// Advance past the full 1.5s shutdown duration.
	s.clock.Advance(1600 * time.Millisecond)
	time.Sleep(10 * time.Millisecond) // Let animation goroutine process.

	// Final body color should be exactly DormantColor (#004900).
	bodyCircle := fairy.BodyCircle()
	s.Require().NotNil(bodyCircle)

	expected := color.RGBA{R: 0x00, G: 0x49, B: 0x00, A: 0xFF}
	r1, g1, b1, a1 := bodyCircle.FillColor.RGBA()
	r2, g2, b2, a2 := expected.RGBA()
	s.Equal(r2, r1, "final body red channel should be 0x00")
	s.Equal(g2, g1, "final body green channel should be 0x49")
	s.Equal(b2, b1, "final body blue channel should be 0x00")
	s.Equal(a2, a1, "final body alpha channel should be 0xFF")
}

// --- Color capture from bright color at midpoint ---

func (s *ShutdownAnimatorSuite) TestColorCaptureFromBrightColor() {
	fairy := character.NewFairyCharacter()
	// Set fairy to brightest green (#00FF00).
	fairy.SetBodyColor(color.RGBA{R: 0x00, G: 0xFF, B: 0x00, A: 0xFF})

	animator := character.NewShutdownAnimator(s.clock)
	animator.Start(fairy)
	defer animator.Stop()

	// Advance to midpoint (750ms of 1.5s animation).
	s.clock.Advance(750 * time.Millisecond)
	time.Sleep(10 * time.Millisecond) // Let animation goroutine process.

	// At midpoint, green channel should be between 0xFF and 0x49 (dormant).
	bodyCircle := fairy.BodyCircle()
	s.Require().NotNil(bodyCircle)
	_, g, _, _ := bodyCircle.FillColor.RGBA()

	// Green channel in pre-multiplied 16-bit: dormant=0x4949, bright=0xFFFF.
	dormantG := uint32(0x49) * 0x101
	brightG := uint32(0xFF) * 0x101
	s.Greater(g, dormantG, "green channel at midpoint should be above dormant (0x49)")
	s.Less(g, brightG, "green channel at midpoint should be below bright (0xFF)")
}
