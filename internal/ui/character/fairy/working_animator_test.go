package fairy_test

import (
	"image/color"
	"testing"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/character/fairy"
	"github.com/stretchr/testify/suite"
)

// WorkingAnimatorSuite tests the WorkingAnimator and the WorkingPosition function.
type WorkingAnimatorSuite struct {
	suite.Suite
	clock *mockClock
}

func TestWorkingAnimator(t *testing.T) {
	suite.Run(t, new(WorkingAnimatorSuite))
}

func (s *WorkingAnimatorSuite) SetupTest() {
	s.clock = newMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
}

// --- Position bounds ---

func (s *WorkingAnimatorSuite) TestPositionStaysWithinBounds() {
	// Sample many time points across several circuits.
	for i := range 2000 {
		t := float64(i) * 0.01 // 0.0s to 20.0s, covering ~5 circuits
		x, y := fairy.WorkingPosition(t)
		s.GreaterOrEqual(x, 0.0, "x at t=%v must be >= 0.0", t)
		s.LessOrEqual(x, 1.0, "x at t=%v must be <= 1.0", t)
		s.GreaterOrEqual(y, 0.0, "y at t=%v must be >= 0.0", t)
		s.LessOrEqual(y, 1.0, "y at t=%v must be <= 1.0", t)
	}
}

// --- Approximate 4-second periodicity ---

func (s *WorkingAnimatorSuite) TestFourSecondApproximatePeriodicity() {
	// The primary circuit is 4 seconds. Noise layers make it approximate,
	// but positions at t and t+4 should be close.
	x0, y0 := fairy.WorkingPosition(0.0)
	x4, y4 := fairy.WorkingPosition(4.0)

	s.InDelta(x0, x4, 0.2,
		"x position at t=0 and t=4 should be close (within noise tolerance)")
	s.InDelta(y0, y4, 0.2,
		"y position at t=0 and t=4 should be close (within noise tolerance)")
}

// --- Body color after entry transition ---

func (s *WorkingAnimatorSuite) TestBodyColorIsWorkingGreen() {
	f := fairy.NewFairyCharacter()
	animator := fairy.NewWorkingAnimator(s.clock)

	animator.Start(f)
	defer animator.Stop()

	// Advance past the 0.5s entry transition so color should be final.
	s.clock.Advance(600 * time.Millisecond)

	// Send a tick to let the animator process.
	// After entry completes, body color must be #00DD00.
	expected := color.RGBA{R: 0x00, G: 0xDD, B: 0x00, A: 0xFF}
	time.Sleep(50 * time.Millisecond) // Allow goroutine to process

	bodyCircle := f.BodyCircle()
	s.Require().NotNil(bodyCircle)

	r1, g1, b1, a1 := bodyCircle.FillColor.RGBA()
	r2, g2, b2, a2 := expected.RGBA()
	s.Equal(r2, r1, "body red channel should be 0x00")
	s.Equal(g2, g1, "body green channel should be 0xDD")
	s.Equal(b2, b1, "body blue channel should be 0x00")
	s.Equal(a2, a1, "body alpha channel should be 0xFF")
}

// --- Breathing glow maintained ---

func (s *WorkingAnimatorSuite) TestBreathingGlowMaintained() {
	f := fairy.NewFairyCharacter()
	animator := fairy.NewWorkingAnimator(s.clock)

	animator.Start(f)
	defer animator.Stop()

	// At t=0, glow should match IdleGlowIntensity(0.0) -- same breathing cycle.
	expected := fairy.IdleGlowIntensity(0.0)
	s.InDelta(expected, f.GlowIntensity(), 1e-9,
		"working animator should use the same breathing glow cycle as idle")
}

// --- Entry transition position interpolation ---

func (s *WorkingAnimatorSuite) TestEntryTransitionInterpolatesPosition() {
	f := fairy.NewFairyCharacter()
	animator := fairy.NewWorkingAnimator(s.clock)

	animator.Start(f)
	defer animator.Stop()

	// At t=0, position should be at idle origin (0.5, 1.0).
	x0, y0 := f.Position()
	s.InDelta(0.5, x0, 1e-9, "initial x should be idle position 0.5")
	s.InDelta(1.0, y0, 1e-9, "initial y should be idle position 1.0")

	// At t=0.25s (midway through entry), position should be interpolated
	// between (0.5, 1.0) and the first drift position.
	s.clock.Advance(250 * time.Millisecond)
	time.Sleep(50 * time.Millisecond)

	xMid, yMid := f.Position()
	driftX, driftY := fairy.WorkingPosition(0.25)

	// Midway position should be between idle origin and drift target.
	// At 50% interpolation: pos = idle + 0.5 * (drift - idle)
	expectedMidX := 0.5 + 0.5*(driftX-0.5)
	expectedMidY := 1.0 + 0.5*(driftY-1.0)
	s.InDelta(expectedMidX, xMid, 0.1,
		"midway x should be interpolated between idle and drift")
	s.InDelta(expectedMidY, yMid, 0.1,
		"midway y should be interpolated between idle and drift")

	// At t > 0.5s, position should be at drift position.
	s.clock.Advance(300 * time.Millisecond)
	time.Sleep(50 * time.Millisecond)

	xPost, yPost := f.Position()
	driftXPost, driftYPost := fairy.WorkingPosition(0.55)
	s.InDelta(driftXPost, xPost, 0.05,
		"post-entry x should match drift position")
	s.InDelta(driftYPost, yPost, 0.05,
		"post-entry y should match drift position")
}

// --- Entry transition color interpolation ---

func (s *WorkingAnimatorSuite) TestEntryTransitionInterpolatesColor() {
	f := fairy.NewFairyCharacter()
	animator := fairy.NewWorkingAnimator(s.clock)

	animator.Start(f)
	defer animator.Stop()

	// At t=0, body color should be idle green (#00FF00).
	bodyCircle := f.BodyCircle()
	s.Require().NotNil(bodyCircle)

	idleGreen := fairy.IdleBodyColor
	r1, g1, b1, a1 := bodyCircle.FillColor.RGBA()
	r2, g2, b2, a2 := idleGreen.RGBA()
	s.Equal(r2, r1, "at t=0 red should be 0x00")
	s.Equal(g2, g1, "at t=0 green should be 0xFF")
	s.Equal(b2, b1, "at t=0 blue should be 0x00")
	s.Equal(a2, a1, "at t=0 alpha should be 0xFF")

	// After entry (>0.5s), body color should be working green (#00DD00).
	s.clock.Advance(600 * time.Millisecond)
	time.Sleep(50 * time.Millisecond)

	workingGreen := color.RGBA{R: 0x00, G: 0xDD, B: 0x00, A: 0xFF}
	r3, g3, b3, a3 := bodyCircle.FillColor.RGBA()
	r4, g4, b4, a4 := workingGreen.RGBA()
	s.Equal(r4, r3, "after entry red should be 0x00")
	s.Equal(g4, g3, "after entry green should be 0xDD")
	s.Equal(b4, b3, "after entry blue should be 0x00")
	s.Equal(a4, a3, "after entry alpha should be 0xFF")
}

// --- Start/Stop lifecycle ---

func (s *WorkingAnimatorSuite) TestStartStopLifecycle() {
	f := fairy.NewFairyCharacter()
	animator := fairy.NewWorkingAnimator(s.clock)

	testCases := []struct {
		name string
		fn   func()
	}{
		{"start and stop", func() {
			animator.Start(f)
			animator.Stop()
		}},
		{"stop without start", func() {
			animator.Stop()
		}},
		{"double stop", func() {
			animator.Start(f)
			animator.Stop()
			animator.Stop()
		}},
		{"double start", func() {
			animator.Start(f)
			animator.Start(f) // Should stop first then restart
			animator.Stop()
		}},
		{"multiple cycles", func() {
			animator.Start(f)
			animator.Stop()
			animator.Start(f)
			animator.Stop()
		}},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.NotPanics(tc.fn, "lifecycle operation should not panic")
		})
	}
}

// --- State() returns StateWorking ---

func (s *WorkingAnimatorSuite) TestStateReturnsStateWorking() {
	animator := fairy.NewWorkingAnimator(s.clock)
	s.Equal(character.StateWorking, animator.State(),
		"WorkingAnimator.State() must return StateWorking")
}

// --- Position varies over time ---

func (s *WorkingAnimatorSuite) TestPositionVariesOverTime() {
	x0, y0 := fairy.WorkingPosition(0.0)
	x1, y1 := fairy.WorkingPosition(1.0)

	// Position must not be static -- at least one axis should differ.
	positionChanged := (x0 != x1) || (y0 != y1)
	s.True(positionChanged,
		"WorkingPosition should vary over time: t=0 (%v,%v) vs t=1 (%v,%v)", x0, y0, x1, y1)
}

// --- Animation constants ---

func (s *WorkingAnimatorSuite) TestWorkingAnimationConstants() {
	s.Equal(4.0, fairy.WorkingCircuitSec,
		"working circuit must be 4.0 seconds")
	s.Equal(0.35, fairy.WorkingDriftRadius,
		"working drift radius must be 0.35")
	s.Equal(0.5, fairy.WorkingEntryDurationSec,
		"working entry duration must be 0.5 seconds")

	expectedColor := color.RGBA{R: 0x00, G: 0xDD, B: 0x00, A: 0xFF}
	s.Equal(expectedColor, fairy.WorkingBodyColor,
		"working body color must be #00DD00")
}
