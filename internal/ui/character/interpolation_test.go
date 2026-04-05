package character_test

import (
	"testing"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
	"github.com/stretchr/testify/suite"
)

// InterpolationSuite tests the easeInOut interpolation function.
type InterpolationSuite struct {
	suite.Suite
}

func TestInterpolation(t *testing.T) {
	suite.Run(t, new(InterpolationSuite))
}

// --- easeInOut at boundary and midpoint values ---

func (s *InterpolationSuite) TestEaseInOutAtZero() {
	s.Equal(0.0, character.EaseInOut(0.0),
		"easeInOut(0.0) must be 0.0")
}

func (s *InterpolationSuite) TestEaseInOutAtHalf() {
	s.Equal(0.5, character.EaseInOut(0.5),
		"easeInOut(0.5) must be 0.5")
}

func (s *InterpolationSuite) TestEaseInOutAtOne() {
	s.Equal(1.0, character.EaseInOut(1.0),
		"easeInOut(1.0) must be 1.0")
}

// --- Symmetry property ---

func (s *InterpolationSuite) TestEaseInOutSymmetry() {
	// Hermite smoothstep is symmetric: f(t) + f(1-t) = 1.0
	v025 := character.EaseInOut(0.25)
	v075 := character.EaseInOut(0.75)
	s.InDelta(1.0, v025+v075, 1e-9,
		"easeInOut(0.25) + easeInOut(0.75) must equal 1.0 (symmetry)")
}

// --- Monotonicity ---

func (s *InterpolationSuite) TestEaseInOutMonotonic() {
	// Values must increase monotonically from 0 to 1.
	const steps = 100
	prev := character.EaseInOut(0.0)
	for i := 1; i <= steps; i++ {
		t := float64(i) / float64(steps)
		curr := character.EaseInOut(t)
		s.GreaterOrEqual(curr, prev,
			"easeInOut must be monotonically increasing: easeInOut(%v)=%v < easeInOut(%v)=%v",
			float64(i-1)/float64(steps), prev, t, curr)
		prev = curr
	}
}
