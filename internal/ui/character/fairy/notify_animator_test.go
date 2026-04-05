package fairy_test

import (
	"image/color"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/character/fairy"
	"github.com/stretchr/testify/suite"
)

// NotifyAnimatorSuite tests the NotifyAnimator and the NotifyGlowIntensity function.
type NotifyAnimatorSuite struct {
	suite.Suite
	clock *mockClock
}

func TestNotifyAnimator(t *testing.T) {
	suite.Run(t, new(NotifyAnimatorSuite))
}

func (s *NotifyAnimatorSuite) SetupTest() {
	s.clock = newMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
}

// --- Glow intensity function tests (table-driven) ---

func (s *NotifyAnimatorSuite) TestNotifyGlowIntensityAtKeyPoints() {
	const (
		glowMin = 0.5
		glowMax = 0.9
		mid     = (glowMin + glowMax) / 2.0 // 0.7
	)

	tests := []struct {
		name     string
		t        float64
		expected float64
	}{
		{"t=0.0 midpoint (sin=0)", 0.0, mid},
		{"t=0.375 peak (sin=1)", 0.375, glowMax},
		{"t=0.75 midpoint descending (sin=0)", 0.75, mid},
		{"t=1.125 trough (sin=-1)", 1.125, glowMin},
		{"t=1.5 back to midpoint (sin=0)", 1.5, mid},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			got := fairy.NotifyGlowIntensity(tc.t)
			s.InDelta(tc.expected, got, 1e-9,
				"NotifyGlowIntensity(%v) should be %v", tc.t, tc.expected)
		})
	}
}

// --- Breathing glow min/max tests ---

func (s *NotifyAnimatorSuite) TestNotifyGlowIntensityBounds() {
	tests := []struct {
		name     string
		t        float64
		expected float64
		desc     string
	}{
		{"minimum at trough", 1.125, fairy.NotifyGlowMin, "glow should reach minimum"},
		{"maximum at peak", 0.375, fairy.NotifyGlowMax, "glow should reach maximum"},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			got := fairy.NotifyGlowIntensity(tc.t)
			s.InDelta(tc.expected, got, 1e-9, tc.desc)
		})
	}
}

func (s *NotifyAnimatorSuite) TestNotifyGlowIntensityNeverExceedsBounds() {
	for i := range 1000 {
		t := float64(i) * 0.01
		got := fairy.NotifyGlowIntensity(t)
		s.GreaterOrEqual(got, 0.5, "glow at t=%v must be >= 0.5", t)
		s.LessOrEqual(got, 0.9, "glow at t=%v must be <= 0.9", t)
	}
}

// --- 1.5-second cycle period ---

func (s *NotifyAnimatorSuite) TestNotifyGlowIntensityPeriodIs1Point5Seconds() {
	v0 := fairy.NotifyGlowIntensity(0.0)
	v15 := fairy.NotifyGlowIntensity(1.5)
	s.InDelta(v0, v15, 1e-9,
		"glow intensity must be periodic with period 1.5s")

	// Also check an arbitrary offset.
	v1 := fairy.NotifyGlowIntensity(0.789)
	v2 := fairy.NotifyGlowIntensity(0.789 + 1.5)
	s.InDelta(v1, v2, 1e-9,
		"glow intensity must be periodic with period 1.5s at arbitrary offset")
}

// --- Sinusoidal shape ---

func (s *NotifyAnimatorSuite) TestNotifyGlowIntensityIsSinusoidal() {
	// Manually compute expected value for an arbitrary t.
	t := 0.6
	normalized := math.Sin(2 * math.Pi * t / 1.5)
	expected := 0.5 + (0.9-0.5)*(normalized+1.0)/2.0
	got := fairy.NotifyGlowIntensity(t)
	s.InDelta(expected, got, 1e-9,
		"glow intensity should follow the sinusoidal formula")
}

// --- Body color is #00C300 ---

func (s *NotifyAnimatorSuite) TestStartSetsBodyColorToBrightGreen() {
	f := fairy.NewFairyCharacter()
	// Change body color away from default.
	f.SetBodyColor(color.RGBA{R: 0xFF, G: 0x00, B: 0x00, A: 0xFF})

	rng := rand.New(rand.NewSource(42))
	animator := fairy.NewNotifyAnimator(s.clock, rng)
	animator.Start(f)
	defer animator.Stop()

	bodyCircle := f.BodyCircle()
	s.Require().NotNil(bodyCircle)

	expected := color.RGBA{R: 0x00, G: 0xC3, B: 0x00, A: 0xFF}
	r1, g1, b1, a1 := bodyCircle.FillColor.RGBA()
	r2, g2, b2, a2 := expected.RGBA()
	s.Equal(r2, r1, "body red channel should be 0x00")
	s.Equal(g2, g1, "body green channel should be 0xC3")
	s.Equal(b2, b1, "body blue channel should be 0x00")
	s.Equal(a2, a1, "body alpha channel should be 0xFF")
}

// --- Immediate glow on start ---

func (s *NotifyAnimatorSuite) TestStartSetsImmediateGlow() {
	f := fairy.NewFairyCharacter()
	rng := rand.New(rand.NewSource(42))
	animator := fairy.NewNotifyAnimator(s.clock, rng)

	animator.Start(f)
	defer animator.Stop()

	// Glow should snap to NotifyGlowMax (0.9) immediately, no transition.
	s.InDelta(fairy.NotifyGlowMax, f.GlowIntensity(), 1e-9,
		"glow should be NotifyGlowMax (0.9) immediately after Start")
}

// --- Immediate dart on start ---

func (s *NotifyAnimatorSuite) TestStartTriggersImmediateDart() {
	f := fairy.NewFairyCharacter()
	rng := rand.New(rand.NewSource(42))
	animator := fairy.NewNotifyAnimator(s.clock, rng)

	animator.Start(f)
	defer animator.Stop()

	x, y := f.Position()
	// Position should NOT be at idle origin (0.5, 1.0) -- it should have
	// darted to a random position.
	atIdleOrigin := (x == 0.5) && (y == 1.0)
	s.False(atIdleOrigin,
		"position after Start should not be idle origin (0.5, 1.0), got (%v, %v)", x, y)
}

// --- Dart every 0.5 seconds ---

func (s *NotifyAnimatorSuite) TestDartPositionsChangeEveryHalfSecond() {
	f := fairy.NewFairyCharacter()
	rng := rand.New(rand.NewSource(42))
	animator := fairy.NewNotifyAnimator(s.clock, rng)

	animator.Start(f)
	defer animator.Stop()

	x0, y0 := f.Position()

	// Advance 500ms -- dart should happen.
	s.clock.Advance(500 * time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	x1, y1 := f.Position()
	posChanged := (x0 != x1) || (y0 != y1)
	s.True(posChanged,
		"position should change after 500ms dart: was (%v,%v), still (%v,%v)", x0, y0, x1, y1)

	// Advance another 500ms -- another dart.
	s.clock.Advance(500 * time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	x2, y2 := f.Position()
	posChanged2 := (x1 != x2) || (y1 != y2)
	s.True(posChanged2,
		"position should change again after another 500ms: was (%v,%v), now (%v,%v)", x1, y1, x2, y2)
}

// --- Dart positions within bounds ---

func (s *NotifyAnimatorSuite) TestDartPositionsWithinBounds() {
	f := fairy.NewFairyCharacter()
	rng := rand.New(rand.NewSource(42))
	animator := fairy.NewNotifyAnimator(s.clock, rng)

	animator.Start(f)
	defer animator.Stop()

	// Check initial position.
	x, y := f.Position()
	s.GreaterOrEqual(x, 0.0, "x must be >= 0.0")
	s.LessOrEqual(x, 1.0, "x must be <= 1.0")
	s.GreaterOrEqual(y, 0.0, "y must be >= 0.0")
	s.LessOrEqual(y, 1.0, "y must be <= 1.0")

	// Advance through several darts and check bounds.
	for i := range 10 {
		s.clock.Advance(500 * time.Millisecond)
		time.Sleep(5 * time.Millisecond)

		x, y = f.Position()
		s.GreaterOrEqual(x, 0.0, "dart %d: x must be >= 0.0", i)
		s.LessOrEqual(x, 1.0, "dart %d: x must be <= 1.0", i)
		s.GreaterOrEqual(y, 0.0, "dart %d: y must be >= 0.0", i)
		s.LessOrEqual(y, 1.0, "dart %d: y must be <= 1.0", i)
	}
}

// --- Deterministic RNG ---

func (s *NotifyAnimatorSuite) TestDeterministicRNG() {
	fairy1 := fairy.NewFairyCharacter()
	rng1 := rand.New(rand.NewSource(99))
	animator1 := fairy.NewNotifyAnimator(s.clock, rng1)
	animator1.Start(fairy1)

	x1, y1 := fairy1.Position()
	animator1.Stop()

	fairy2 := fairy.NewFairyCharacter()
	rng2 := rand.New(rand.NewSource(99))
	clock2 := newMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	animator2 := fairy.NewNotifyAnimator(clock2, rng2)
	animator2.Start(fairy2)

	x2, y2 := fairy2.Position()
	animator2.Stop()

	s.Equal(x1, x2, "same seed should produce same initial x position")
	s.Equal(y1, y2, "same seed should produce same initial y position")
}

// --- Start/Stop lifecycle ---

func (s *NotifyAnimatorSuite) TestStartStopLifecycle() {
	f := fairy.NewFairyCharacter()
	rng := rand.New(rand.NewSource(42))
	animator := fairy.NewNotifyAnimator(s.clock, rng)

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

// --- State() returns StateNotifying ---

func (s *NotifyAnimatorSuite) TestStateReturnsStateNotifying() {
	rng := rand.New(rand.NewSource(42))
	animator := fairy.NewNotifyAnimator(s.clock, rng)
	s.Equal(character.StateNotifying, animator.State(),
		"NotifyAnimator.State() must return StateNotifying")
}

// --- Animation constants ---

func (s *NotifyAnimatorSuite) TestNotifyAnimationConstants() {
	s.Equal(0.5, fairy.NotifyDartIntervalSec,
		"notify dart interval must be 0.5 seconds")
	s.Equal(1.5, fairy.NotifyBreathCycleSec,
		"notify breath cycle must be 1.5 seconds")
	s.Equal(0.5, fairy.NotifyGlowMin,
		"notify glow minimum must be 0.5")
	s.Equal(0.9, fairy.NotifyGlowMax,
		"notify glow maximum must be 0.9")

	expectedColor := color.RGBA{R: 0x00, G: 0xC3, B: 0x00, A: 0xFF}
	s.Equal(expectedColor, fairy.NotifyBodyColor,
		"notify body color must be #00C300")
}
