package character_test

import (
	"image/color"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
	"github.com/stretchr/testify/suite"
)

// StartupAnimatorSuite tests the StartupAnimator lifecycle and animation behavior.
type StartupAnimatorSuite struct {
	suite.Suite
	clock *mockClock
}

func TestStartupAnimator(t *testing.T) {
	suite.Run(t, new(StartupAnimatorSuite))
}

func (s *StartupAnimatorSuite) SetupTest() {
	s.clock = newMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
}

// --- State() returns StateStarting ---

func (s *StartupAnimatorSuite) TestStateReturnsStarting() {
	animator := character.NewStartupAnimator(s.clock, func() {})
	s.Equal(character.StateStarting, animator.State(),
		"StartupAnimator.State() must return StateStarting")
}

// --- Initial state is dormant ---

func (s *StartupAnimatorSuite) TestInitialStateIsDormant() {
	fairy := character.NewFairyCharacter()
	animator := character.NewStartupAnimator(s.clock, func() {})

	animator.Start(fairy)
	defer animator.Stop()

	// Position should be (0.5, 1.0).
	x, y := fairy.Position()
	s.Equal(0.5, x, "startup position x must be 0.5")
	s.Equal(1.0, y, "startup position y must be 1.0")

	// Body color should be dormant (#004900).
	bodyCircle := fairy.BodyCircle()
	s.Require().NotNil(bodyCircle)

	expected := color.RGBA{R: 0x00, G: 0x49, B: 0x00, A: 0xFF}
	r1, g1, b1, a1 := bodyCircle.FillColor.RGBA()
	r2, g2, b2, a2 := expected.RGBA()
	s.Equal(r2, r1, "body red channel should be 0x00")
	s.Equal(g2, g1, "body green channel should be 0x49")
	s.Equal(b2, b1, "body blue channel should be 0x00")
	s.Equal(a2, a1, "body alpha channel should be 0xFF")

	// Glow intensity should be dormant (0.1).
	s.InDelta(0.1, fairy.GlowIntensity(), 1e-9,
		"glow intensity at start should be dormant (0.1)")
}

// --- Final state is idle after 1.5s ---

func (s *StartupAnimatorSuite) TestFinalStateIsIdle() {
	fairy := character.NewFairyCharacter()
	completed := make(chan struct{})
	animator := character.NewStartupAnimator(s.clock, func() {
		close(completed)
	})

	animator.Start(fairy)
	defer animator.Stop()

	// Advance the mock clock past the full 1.5s startup duration.
	s.clock.Advance(1600 * time.Millisecond)
	time.Sleep(10 * time.Millisecond) // Let animation goroutine process.

	// Wait for completion callback.
	select {
	case <-completed:
	case <-time.After(time.Second):
		s.Fail("onComplete callback was not called within timeout")
	}

	// Body color should be idle (#006100).
	bodyCircle := fairy.BodyCircle()
	s.Require().NotNil(bodyCircle)

	expected := color.RGBA{R: 0x00, G: 0x61, B: 0x00, A: 0xFF}
	r1, g1, b1, a1 := bodyCircle.FillColor.RGBA()
	r2, g2, b2, a2 := expected.RGBA()
	s.Equal(r2, r1, "final body red channel should be 0x00")
	s.Equal(g2, g1, "final body green channel should be 0x61")
	s.Equal(b2, b1, "final body blue channel should be 0x00")
	s.Equal(a2, a1, "final body alpha channel should be 0xFF")

	// Glow intensity should be idle target (0.5).
	s.InDelta(0.5, fairy.GlowIntensity(), 0.05,
		"glow intensity at end should be approximately idle target (0.5)")
}

// --- onComplete callback fires ---

func (s *StartupAnimatorSuite) TestOnCompleteCallbackFires() {
	fairy := character.NewFairyCharacter()
	var callCount atomic.Int32
	animator := character.NewStartupAnimator(s.clock, func() {
		callCount.Add(1)
	})

	animator.Start(fairy)
	defer animator.Stop()

	// Advance past the full 1.5s duration.
	s.clock.Advance(1600 * time.Millisecond)
	time.Sleep(10 * time.Millisecond) // Let animation goroutine process.

	s.Equal(int32(1), callCount.Load(),
		"onComplete callback should be called exactly once")
}

// --- Color interpolation at midpoint ---

func (s *StartupAnimatorSuite) TestColorInterpolationAtMidpoint() {
	fairy := character.NewFairyCharacter()
	animator := character.NewStartupAnimator(s.clock, func() {})

	animator.Start(fairy)
	defer animator.Stop()

	// Advance to ~0.75s (midpoint of 1.5s animation).
	s.clock.Advance(750 * time.Millisecond)
	time.Sleep(10 * time.Millisecond) // Let animation goroutine process.

	// At midpoint, color should be between dormant (#004900) and idle (#006100).
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

func (s *StartupAnimatorSuite) TestGlowInterpolationAtMidpoint() {
	fairy := character.NewFairyCharacter()
	animator := character.NewStartupAnimator(s.clock, func() {})

	animator.Start(fairy)
	defer animator.Stop()

	// Advance to ~0.75s (midpoint of 1.5s animation).
	s.clock.Advance(750 * time.Millisecond)
	time.Sleep(10 * time.Millisecond) // Let animation goroutine process.

	// At midpoint with easeInOut(0.5)=0.5, glow should be ~0.3 (lerp 0.1 to 0.5).
	glow := fairy.GlowIntensity()
	s.Greater(glow, 0.1,
		"glow intensity at midpoint should be above dormant (0.1)")
	s.Less(glow, 0.5,
		"glow intensity at midpoint should be below idle target (0.5)")
}

// --- Stop cancels cleanly ---

func (s *StartupAnimatorSuite) TestStopCancelsCleanly() {
	fairy := character.NewFairyCharacter()
	animator := character.NewStartupAnimator(s.clock, func() {})

	testCases := []struct {
		name string
		fn   func()
	}{
		{"start and stop", func() {
			animator.Start(fairy)
			animator.Stop()
		}},
		{"stop without start", func() {
			animator.Stop()
		}},
		{"double stop", func() {
			animator.Start(fairy)
			animator.Stop()
			animator.Stop()
		}},
		{"double start", func() {
			animator.Start(fairy)
			animator.Start(fairy)
			animator.Stop()
		}},
		{"multiple cycles", func() {
			animator.Start(fairy)
			animator.Stop()
			animator.Start(fairy)
			animator.Stop()
		}},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.NotPanics(tc.fn, "lifecycle operation should not panic")
		})
	}
}

// --- Duration is correct ---

func (s *StartupAnimatorSuite) TestDurationIsCorrect() {
	s.Equal(1.5, character.StartupDurationSec,
		"startup duration must be 1.5 seconds")
}

// --- Startup animation constants ---

func (s *StartupAnimatorSuite) TestStartupAnimationConstants() {
	s.Equal(1.5, character.StartupDurationSec,
		"startup duration must be 1.5 seconds")

	expectedDormant := color.RGBA{R: 0x00, G: 0x49, B: 0x00, A: 0xFF}
	s.Equal(expectedDormant, character.DormantColor,
		"dormant color must be #004900")

	s.Equal(0.1, character.DormantGlowIntensity,
		"dormant glow intensity must be 0.1")

	s.Equal(0.5, character.StartupIdleGlowIntensity,
		"idle glow intensity target for startup must be 0.5")
}

// --- StateAnimator interface compliance ---

func (s *StartupAnimatorSuite) TestImplementsStateAnimator() {
	animator := character.NewStartupAnimator(s.clock, func() {})
	var _ character.StateAnimator = animator
}
