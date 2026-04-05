package character_test

import (
	"image/color"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
	"github.com/stretchr/testify/suite"
)

// mockTicker is a controllable ticker for testing animation timing.
type mockTicker struct {
	C    chan time.Time
	stop func()
}

func (t *mockTicker) Chan() <-chan time.Time { return t.C }
func (t *mockTicker) Stop()                  { t.stop() }

// mockClock implements character.Clock for deterministic testing.
type mockClock struct {
	mu  sync.Mutex
	now time.Time
}

func newMockClock(start time.Time) *mockClock {
	return &mockClock{now: start}
}

func (c *mockClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *mockClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func (c *mockClock) NewTicker(d time.Duration) character.Ticker {
	ch := make(chan time.Time, 1)
	stopped := make(chan struct{})
	return &mockTicker{
		C: ch,
		stop: func() {
			select {
			case <-stopped:
			default:
				close(stopped)
			}
		},
	}
}

// IdleAnimatorSuite tests the IdleAnimator and the idleGlowIntensity function.
type IdleAnimatorSuite struct {
	suite.Suite
	clock *mockClock
}

func TestIdleAnimator(t *testing.T) {
	suite.Run(t, new(IdleAnimatorSuite))
}

func (s *IdleAnimatorSuite) SetupTest() {
	s.clock = newMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
}

// --- Glow intensity function tests (table-driven) ---

func (s *IdleAnimatorSuite) TestGlowIntensityAtKeyPoints() {
	const (
		glowMin = 0.3
		glowMax = 0.8
		mid     = (glowMin + glowMax) / 2.0
	)

	tests := []struct {
		name     string
		t        float64
		expected float64
	}{
		{"t=0.0 midpoint (sin=0)", 0.0, mid},
		{"t=0.75 peak (sin=1)", 0.75, glowMax},
		{"t=1.5 midpoint descending (sin=0)", 1.5, mid},
		{"t=2.25 trough (sin=-1)", 2.25, glowMin},
		{"t=3.0 full cycle back to midpoint (sin=0)", 3.0, mid},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			got := character.IdleGlowIntensity(tc.t)
			s.InDelta(tc.expected, got, 1e-9,
				"IdleGlowIntensity(%v) should be %v", tc.t, tc.expected)
		})
	}
}

// --- Breathing glow min/max tests ---

func (s *IdleAnimatorSuite) TestGlowIntensityBounds() {
	tests := []struct {
		name     string
		t        float64
		expected float64
		desc     string
	}{
		{"minimum at trough", 2.25, character.IdleGlowMin, "glow should reach minimum"},
		{"maximum at peak", 0.75, character.IdleGlowMax, "glow should reach maximum"},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			got := character.IdleGlowIntensity(tc.t)
			s.InDelta(tc.expected, got, 1e-9, tc.desc)
		})
	}
}

func (s *IdleAnimatorSuite) TestGlowIntensityNeverExceedsBounds() {
	// Sample many points across several cycles.
	for i := range 1000 {
		t := float64(i) * 0.01
		got := character.IdleGlowIntensity(t)
		s.GreaterOrEqual(got, 0.3, "glow at t=%v must be >= 0.3", t)
		s.LessOrEqual(got, 0.8, "glow at t=%v must be <= 0.8", t)
	}
}

// --- 3-second cycle period ---

func (s *IdleAnimatorSuite) TestGlowIntensityPeriodIs3Seconds() {
	// Glow at t=0.0 should equal glow at t=3.0 (full cycle).
	v0 := character.IdleGlowIntensity(0.0)
	v3 := character.IdleGlowIntensity(3.0)
	s.InDelta(v0, v3, 1e-9,
		"glow intensity must be periodic with period 3.0s")

	// Also check an arbitrary offset.
	v1 := character.IdleGlowIntensity(1.234)
	v4 := character.IdleGlowIntensity(1.234 + 3.0)
	s.InDelta(v1, v4, 1e-9,
		"glow intensity must be periodic with period 3.0s at arbitrary offset")
}

// --- Sinusoidal shape ---

func (s *IdleAnimatorSuite) TestGlowIntensityIsSinusoidal() {
	// Manually compute expected value for an arbitrary t.
	t := 1.0
	normalized := math.Sin(2 * math.Pi * t / 3.0)
	expected := 0.3 + (0.8-0.3)*(normalized+1.0)/2.0
	got := character.IdleGlowIntensity(t)
	s.InDelta(expected, got, 1e-9,
		"glow intensity should follow the sinusoidal formula")
}

// --- Position stays at (0.5, 1.0) ---

func (s *IdleAnimatorSuite) TestStartSetsPositionToBottomCenter() {
	fairy := character.NewFairyCharacter()
	// Move fairy away from the idle position first.
	fairy.SetPosition(0.0, 0.0)

	animator := character.NewIdleAnimator(s.clock)
	animator.Start(fairy)
	defer animator.Stop()

	x, y := fairy.Position()
	s.Equal(0.5, x, "idle position x must be 0.5 (center)")
	s.Equal(1.0, y, "idle position y must be 1.0 (bottom)")
}

// --- Body color is #006100 ---

func (s *IdleAnimatorSuite) TestStartSetsBodyColorToDarkGreen() {
	fairy := character.NewFairyCharacter()
	// Change body color away from default.
	fairy.SetBodyColor(color.RGBA{R: 0xFF, G: 0x00, B: 0x00, A: 0xFF})

	animator := character.NewIdleAnimator(s.clock)
	animator.Start(fairy)
	defer animator.Stop()

	bodyCircle := fairy.BodyCircle()
	s.Require().NotNil(bodyCircle)

	expected := color.RGBA{R: 0x00, G: 0x61, B: 0x00, A: 0xFF}
	r1, g1, b1, a1 := bodyCircle.FillColor.RGBA()
	r2, g2, b2, a2 := expected.RGBA()
	s.Equal(r2, r1, "body red channel should be 0x00")
	s.Equal(g2, g1, "body green channel should be 0x61")
	s.Equal(b2, b1, "body blue channel should be 0x00")
	s.Equal(a2, a1, "body alpha channel should be 0xFF")
}

// --- Start/Stop lifecycle ---

func (s *IdleAnimatorSuite) TestStartStopLifecycle() {
	fairy := character.NewFairyCharacter()
	animator := character.NewIdleAnimator(s.clock)

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
			animator.Start(fairy) // Should stop first then restart
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

// --- State() returns StateIdle ---

func (s *IdleAnimatorSuite) TestStateReturnsStateIdle() {
	animator := character.NewIdleAnimator(s.clock)
	s.Equal(character.StateIdle, animator.State(),
		"IdleAnimator.State() must return StateIdle")
}

// --- StateAnimator interface compliance ---

func (s *IdleAnimatorSuite) TestIdleAnimatorImplementsStateAnimator() {
	animator := character.NewIdleAnimator(s.clock)
	// Compile-time check that *IdleAnimator satisfies StateAnimator.
	var _ character.StateAnimator = animator
}

// --- Synchronous cleanup verification ---

func (s *IdleAnimatorSuite) TestStopIsSynchronous() {
	fairy := character.NewFairyCharacter()
	animator := character.NewIdleAnimator(s.clock)

	animator.Start(fairy)
	// Stop should block until the animation goroutine exits.
	animator.Stop()

	// After Stop returns, calling Start again must work immediately,
	// proving the previous goroutine has fully exited.
	animator.Start(fairy)
	animator.Stop()
}

// --- Animation constants ---

func (s *IdleAnimatorSuite) TestAnimationConstants() {
	s.Equal(3.0, character.IdleBreathCycleSec,
		"idle breath cycle must be 3.0 seconds")
	s.Equal(0.3, character.IdleGlowMin,
		"idle glow minimum must be 0.3")
	s.Equal(0.8, character.IdleGlowMax,
		"idle glow maximum must be 0.8")
	s.Equal(30, character.AnimationFPS,
		"animation FPS must be 30")
	s.Equal(33, character.AnimationTickMs,
		"animation tick must be ~33ms (1000/30)")
}

// --- Glow intensity is driven by animator ---

func (s *IdleAnimatorSuite) TestAnimatorDrivesGlowIntensity() {
	fairy := character.NewFairyCharacter()
	animator := character.NewIdleAnimator(s.clock)

	animator.Start(fairy)
	defer animator.Stop()

	// At t=0, the glow intensity should be set to the midpoint (0.55).
	// The animator should have set it on Start.
	expected := character.IdleGlowIntensity(0.0)
	s.InDelta(expected, fairy.GlowIntensity(), 1e-9,
		"glow intensity at start should match IdleGlowIntensity(0.0)")
}
