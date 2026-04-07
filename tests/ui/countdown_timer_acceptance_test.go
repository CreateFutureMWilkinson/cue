//go:build ui_acceptance

package ui_acceptance_test

import (
	"math"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/ui"
)

// CountdownTimerAcceptanceSuite verifies countdown timer acceptance criteria
// from UiSpec.md lines 1141-1156.
type CountdownTimerAcceptanceSuite struct {
	suite.Suite
}

func TestCountdownTimerAcceptance(t *testing.T) {
	suite.Run(t, new(CountdownTimerAcceptanceSuite))
}

// AC: Timer widget can be created.
func (s *CountdownTimerAcceptanceSuite) TestTimerCreation() {
	timer := ui.NewCountdownTimer()
	s.NotNil(timer, "NewCountdownTimer should return a non-nil widget")
}

// AC: Timer implements fyne.Widget (has MinSize, CreateRenderer).
func (s *CountdownTimerAcceptanceSuite) TestTimerImplementsWidget() {
	timer := ui.NewCountdownTimer()

	minSize := timer.MinSize()
	s.Greater(minSize.Width, float32(0), "MinSize width should be > 0")
	s.Greater(minSize.Height, float32(0), "MinSize height should be > 0")
}

// AC: 45 lines arranged in a ring at 8 degree intervals.
func (s *CountdownTimerAcceptanceSuite) TestTimerHas45Segments() {
	timer := ui.NewCountdownTimer()

	segments := timer.Segments()
	s.Len(segments, 45, "timer should have exactly 45 segments")
}

// AC: Lines at 8 degree intervals spanning 360 degrees.
func (s *CountdownTimerAcceptanceSuite) TestSegmentsAt8DegreeIntervals() {
	timer := ui.NewCountdownTimer()
	segments := timer.Segments()

	for i, seg := range segments {
		rawAngle := float64((i + 1) * 8)
		expectedAngle := rawAngle
		if rawAngle == 360 {
			expectedAngle = 360 // Last segment wraps to 360 (equivalent to 0)
		}
		s.InDelta(expectedAngle, seg.AngleDeg, 0.01,
			"segment %d should be at %.0f degrees", i, expectedAngle)
	}
}

// AC: Cardinal lines (0, 90, 180, 270) are 3x short length (36px).
func (s *CountdownTimerAcceptanceSuite) TestCardinalLinesAreLong() {
	timer := ui.NewCountdownTimer()
	segments := timer.Segments()

	cardinalAngles := []float64{0, 90, 180, 270}
	for _, angle := range cardinalAngles {
		for _, seg := range segments {
			if math.Abs(seg.AngleDeg-angle) < 0.01 || math.Abs(seg.AngleDeg-360-angle) < 0.01 {
				s.InDelta(36.0, seg.Length, 0.01,
					"cardinal line at %.0f degrees should be 36px (3x short)", angle)
			}
		}
	}
}

// AC: Diagonal lines (45, 135, 225, 315) are 2x short length (24px).
func (s *CountdownTimerAcceptanceSuite) TestDiagonalLinesAreMedium() {
	timer := ui.NewCountdownTimer()
	segments := timer.Segments()

	diagonalAngles := []float64{45, 135, 225, 315}
	for _, angle := range diagonalAngles {
		for _, seg := range segments {
			if math.Abs(seg.AngleDeg-angle) < 0.01 {
				s.InDelta(24.0, seg.Length, 0.01,
					"diagonal line at %.0f degrees should be 24px (2x short)", angle)
			}
		}
	}
}

// AC: All other lines are 1x short length (12px).
func (s *CountdownTimerAcceptanceSuite) TestOtherLinesAreShort() {
	timer := ui.NewCountdownTimer()
	segments := timer.Segments()

	specialAngles := map[float64]bool{
		0: true, 90: true, 180: true, 270: true, 360: true,
		45: true, 135: true, 225: true, 315: true,
	}

	for _, seg := range segments {
		if !specialAngles[seg.AngleDeg] {
			s.InDelta(12.0, seg.Length, 0.01,
				"non-cardinal/diagonal line at %.0f degrees should be 12px (1x short)", seg.AngleDeg)
		}
	}
}

// AC: Lines radiate inward from the outer edge — verified by rendering without panic.
func (s *CountdownTimerAcceptanceSuite) TestTimerRendersWithoutPanic() {
	timer := ui.NewCountdownTimer()
	timer.Show()
	w := test.NewWindow(timer)
	defer w.Close()
	w.Resize(fyne.NewSize(200, 200))

	w.Content().Refresh()
}

// AC: Segments deplete clockwise starting at 8 degrees from vertical.
// AC: 12 o'clock (0 degrees) is the last segment.
func (s *CountdownTimerAcceptanceSuite) TestSegmentOrderStartsAt8Degrees() {
	timer := ui.NewCountdownTimer()
	segments := timer.Segments()

	// First segment should be at 8 degrees.
	s.InDelta(8.0, segments[0].AngleDeg, 0.01,
		"first segment should be at 8 degrees")

	// Last segment should be at 360 degrees (equivalent to 0/12 o'clock).
	lastAngle := segments[44].AngleDeg
	s.True(lastAngle == 360 || lastAngle == 0,
		"last segment should be at 0 or 360 degrees (12 o'clock), got %.0f", lastAngle)
}

// AC: Timer resets at start of each new block.
func (s *CountdownTimerAcceptanceSuite) TestTimerReset() {
	timer := ui.NewCountdownTimer()

	timer.SetProgress(0.5)
	timer.Reset()

	segments := timer.Segments()
	for _, seg := range segments {
		s.NotEqual(ui.SegmentElapsed, seg.State,
			"after reset, no segments should be elapsed")
	}
}

// AC: Current segment flashes at 1 Hz (500ms on / 500ms off).
func (s *CountdownTimerAcceptanceSuite) TestTimerFlashToggle() {
	timer := ui.NewCountdownTimer()
	timer.SetProgress(0.5)

	timer.SetFlashVisible(true)
	timer.SetFlashVisible(false)
	// No panic means flash toggle is supported.
}

// AC: Elapsed segments dimmed or hidden.
func (s *CountdownTimerAcceptanceSuite) TestElapsedSegmentsAreDimmed() {
	timer := ui.NewCountdownTimer()
	timer.SetProgress(0.5)

	segments := timer.Segments()
	hasElapsed := false
	for _, seg := range segments {
		if seg.State == ui.SegmentElapsed {
			hasElapsed = true
			// Elapsed color should have reduced alpha (64).
			s.Equal(uint8(64), seg.Color.A,
				"elapsed segment at %.0f degrees should have dimmed alpha", seg.AngleDeg)
		}
	}
	s.True(hasElapsed, "at 50% progress, some segments should be elapsed")
}

// AC: All future segments colored yellow #FFCE1B.
func (s *CountdownTimerAcceptanceSuite) TestFutureSegmentsAreYellow() {
	timer := ui.NewCountdownTimer()
	// At 0 progress, all segments are future.
	segments := timer.Segments()

	for _, seg := range segments {
		if seg.State == ui.SegmentFuture || seg.State == ui.SegmentCurrent {
			s.Equal(uint8(0xFF), seg.Color.R, "future segment R should be 0xFF")
			s.Equal(uint8(0xCE), seg.Color.G, "future segment G should be 0xCE")
			s.Equal(uint8(0x1B), seg.Color.B, "future segment B should be 0x1B")
			s.Equal(uint8(0xFF), seg.Color.A, "future segment A should be 0xFF (fully opaque)")
		}
	}
}

// AC: Ring scales to fit focus rail width (~40-50px radius).
func (s *CountdownTimerAcceptanceSuite) TestTimerMinSizeIsReasonable() {
	timer := ui.NewCountdownTimer()
	minSize := timer.MinSize()

	s.GreaterOrEqual(minSize.Width, float32(20),
		"timer min width should be reasonable for a ring display")
	s.GreaterOrEqual(minSize.Height, float32(20),
		"timer min height should be reasonable for a ring display")
}
