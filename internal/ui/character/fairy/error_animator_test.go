package fairy_test

import (
	"image/color"
	"math"
	"testing"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/character/fairy"
	"github.com/stretchr/testify/suite"
)

// ErrorAnimatorSuite tests the ErrorAnimator and the ErrorGlowIntensity / ErrorPosition functions.
type ErrorAnimatorSuite struct {
	suite.Suite
	clock *mockClock
}

func TestErrorAnimator(t *testing.T) {
	suite.Run(t, new(ErrorAnimatorSuite))
}

func (s *ErrorAnimatorSuite) SetupTest() {
	s.clock = newMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
}

// --- Glow intensity function tests (table-driven) ---

func (s *ErrorAnimatorSuite) TestErrorGlowIntensityAtKeyPoints() {
	const (
		glowMin = 0.4
		glowMax = 0.9
		mid     = (glowMin + glowMax) / 2.0 // 0.65
	)

	tests := []struct {
		name     string
		t        float64
		expected float64
	}{
		{"t=0.0 midpoint (sin=0)", 0.0, mid},
		{"t=0.125 peak (sin=1)", 0.125, glowMax},
		{"t=0.25 midpoint descending (sin=0)", 0.25, mid},
		{"t=0.375 trough (sin=-1)", 0.375, glowMin},
		{"t=0.5 back to midpoint (sin=0)", 0.5, mid},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			got := fairy.ErrorGlowIntensity(tc.t)
			s.InDelta(tc.expected, got, 1e-9,
				"ErrorGlowIntensity(%v) should be %v", tc.t, tc.expected)
		})
	}
}

// --- Glow min/max bounds ---

func (s *ErrorAnimatorSuite) TestErrorGlowIntensityBounds() {
	tests := []struct {
		name     string
		t        float64
		expected float64
		desc     string
	}{
		{"minimum at trough", 0.375, fairy.ErrorGlowMin, "glow should reach minimum"},
		{"maximum at peak", 0.125, fairy.ErrorGlowMax, "glow should reach maximum"},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			got := fairy.ErrorGlowIntensity(tc.t)
			s.InDelta(tc.expected, got, 1e-9, tc.desc)
		})
	}
}

func (s *ErrorAnimatorSuite) TestErrorGlowIntensityNeverExceedsBounds() {
	for i := range 1000 {
		t := float64(i) * 0.01
		got := fairy.ErrorGlowIntensity(t)
		s.GreaterOrEqual(got, 0.4, "glow at t=%v must be >= 0.4", t)
		s.LessOrEqual(got, 0.9, "glow at t=%v must be <= 0.9", t)
	}
}

// --- 0.5-second cycle period ---

func (s *ErrorAnimatorSuite) TestErrorGlowIntensityPeriodIs0Point5Seconds() {
	v0 := fairy.ErrorGlowIntensity(0.0)
	v05 := fairy.ErrorGlowIntensity(0.5)
	s.InDelta(v0, v05, 1e-9,
		"glow intensity must be periodic with period 0.5s")

	// Also check an arbitrary offset.
	v1 := fairy.ErrorGlowIntensity(0.321)
	v2 := fairy.ErrorGlowIntensity(0.321 + 0.5)
	s.InDelta(v1, v2, 1e-9,
		"glow intensity must be periodic with period 0.5s at arbitrary offset")
}

// --- Sinusoidal shape ---

func (s *ErrorAnimatorSuite) TestErrorGlowIntensityIsSinusoidal() {
	// Manually compute expected value for an arbitrary t.
	t := 0.17
	normalized := math.Sin(2 * math.Pi * t / 0.5)
	expected := 0.4 + (0.9-0.4)*(normalized+1.0)/2.0
	got := fairy.ErrorGlowIntensity(t)
	s.InDelta(expected, got, 1e-9,
		"glow intensity should follow the sinusoidal formula")
}

// --- ErrorPosition oscillates around center ---

func (s *ErrorAnimatorSuite) TestErrorPositionOscillatesAroundCenter() {
	// At t=0, sin(0)=0 so x=0.5, y=0.5.
	x0, y0 := fairy.ErrorPosition(0.0)
	s.InDelta(0.5, x0, 1e-9, "x at t=0 should be 0.5")
	s.InDelta(0.5, y0, 1e-9, "y at t=0 should be 0.5")

	// At t=1/(4*15) = 1/60, sin(pi/2) = 1 so x = 0.5 + 0.04.
	tQuarter := 1.0 / (4.0 * fairy.ErrorVibrateFreqHz)
	xQ, yQ := fairy.ErrorPosition(tQuarter)
	s.InDelta(0.54, xQ, 1e-9, "x at quarter period should be 0.54")
	s.InDelta(0.5, yQ, 1e-9, "y should always be 0.5")
}

// --- ErrorPosition horizontal amplitude <= 0.04 ---

func (s *ErrorAnimatorSuite) TestErrorPositionHorizontalAmplitude() {
	for i := range 1000 {
		t := float64(i) * 0.001
		x, _ := fairy.ErrorPosition(t)
		deviation := math.Abs(x - 0.5)
		s.LessOrEqual(deviation, fairy.ErrorVibrateAmplitude+1e-9,
			"at t=%v, |x - 0.5| = %v must be <= %v", t, deviation, fairy.ErrorVibrateAmplitude)
	}
}

// --- ErrorPosition vertical stays at 0.5 ---

func (s *ErrorAnimatorSuite) TestErrorPositionVerticalStaysAtHalf() {
	for i := range 1000 {
		t := float64(i) * 0.001
		_, y := fairy.ErrorPosition(t)
		s.InDelta(0.5, y, 1e-9, "y at t=%v must be 0.5", t)
	}
}

// --- Start sets body color to #88FF00 ---

func (s *ErrorAnimatorSuite) TestStartSetsBodyColorToErrorGreen() {
	f := fairy.NewFairyCharacter()
	// Change body color away from default.
	f.SetBodyColor(color.RGBA{R: 0xFF, G: 0x00, B: 0x00, A: 0xFF})

	animator := fairy.NewErrorAnimator(s.clock)
	animator.Start(f)
	defer animator.Stop()

	bodyCircle := f.BodyCircle()
	s.Require().NotNil(bodyCircle)

	expected := color.RGBA{R: 0x88, G: 0xFF, B: 0x00, A: 0xFF}
	r1, g1, b1, a1 := bodyCircle.FillColor.RGBA()
	r2, g2, b2, a2 := expected.RGBA()
	s.Equal(r2, r1, "body red channel should be 0x88")
	s.Equal(g2, g1, "body green channel should be 0xFF")
	s.Equal(b2, b1, "body blue channel should be 0x00")
	s.Equal(a2, a1, "body alpha channel should be 0xFF")
}

// --- Immediate glow on start ---

func (s *ErrorAnimatorSuite) TestStartSetsImmediateGlow() {
	f := fairy.NewFairyCharacter()
	animator := fairy.NewErrorAnimator(s.clock)

	animator.Start(f)
	defer animator.Stop()

	// Glow should snap to ErrorGlowIntensity(0) = midpoint (0.65) immediately.
	expected := fairy.ErrorGlowIntensity(0.0)
	s.InDelta(expected, f.GlowIntensity(), 1e-9,
		"glow should be ErrorGlowIntensity(0) immediately after Start")
}

// --- Start snaps to center position ---

func (s *ErrorAnimatorSuite) TestStartSnapsToCenter() {
	f := fairy.NewFairyCharacter()
	// Fairy starts at idle (0.5, 1.0).
	x, y := f.Position()
	s.Equal(0.5, x, "fairy should start at x=0.5")
	s.Equal(1.0, y, "fairy should start at y=1.0 (idle)")

	animator := fairy.NewErrorAnimator(s.clock)
	animator.Start(f)
	defer animator.Stop()

	x, y = f.Position()
	s.InDelta(0.5, x, 1e-9, "after Start, x should be 0.5 (center)")
	s.InDelta(0.5, y, 1e-9, "after Start, y should be 0.5 (center)")
}

// --- Start/Stop lifecycle ---

func (s *ErrorAnimatorSuite) TestStartStopLifecycle() {
	f := fairy.NewFairyCharacter()
	animator := fairy.NewErrorAnimator(s.clock)

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

// --- State() returns StateError ---

func (s *ErrorAnimatorSuite) TestStateReturnsStateError() {
	animator := fairy.NewErrorAnimator(s.clock)
	s.Equal(character.StateError, animator.State(),
		"ErrorAnimator.State() must return StateError")
}

// --- Animation constants are correct ---

func (s *ErrorAnimatorSuite) TestErrorAnimationConstants() {
	s.Equal(0.04, fairy.ErrorVibrateAmplitude,
		"error vibrate amplitude must be 0.04")
	s.Equal(15.0, fairy.ErrorVibrateFreqHz,
		"error vibrate frequency must be 15 Hz")
	s.Equal(0.5, fairy.ErrorPulseCycleSec,
		"error pulse cycle must be 0.5 seconds")
	s.Equal(0.4, fairy.ErrorGlowMin,
		"error glow minimum must be 0.4")
	s.Equal(0.9, fairy.ErrorGlowMax,
		"error glow maximum must be 0.9")

	expectedColor := color.RGBA{R: 0x88, G: 0xFF, B: 0x00, A: 0xFF}
	s.Equal(expectedColor, fairy.ErrorBodyColor,
		"error body color must be #88FF00")
}

// --- Vibration updates position over time ---

func (s *ErrorAnimatorSuite) TestVibrationUpdatesPositionOverTime() {
	f := fairy.NewFairyCharacter()
	animator := fairy.NewErrorAnimator(s.clock)

	animator.Start(f)
	defer animator.Stop()

	// At t=0, position should be center (0.5, 0.5).
	x0, y0 := f.Position()
	s.InDelta(0.5, x0, 1e-9, "initial x should be 0.5")
	s.InDelta(0.5, y0, 1e-9, "initial y should be 0.5")

	// Advance clock by a fraction of the vibration period so position changes.
	// 1/(4*15) = ~16.67ms is a quarter vibration cycle; x should be 0.54.
	freqHz := fairy.ErrorVibrateFreqHz
	quarterVibration := time.Duration(float64(time.Second) / (4.0 * freqHz))
	s.clock.Advance(quarterVibration)
	time.Sleep(5 * time.Millisecond) // Let animation goroutine tick.

	x1, _ := f.Position()
	// Position should have moved away from 0.5 due to vibration.
	s.NotEqual(x0, x1,
		"x position should change after advancing clock by quarter vibration period")
}
