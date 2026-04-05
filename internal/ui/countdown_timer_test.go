package ui_test

import (
	"image/color"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/ui"
)

// CountdownTimerSuite tests the CountdownTimer custom Fyne widget that displays
// a circular burndown timer with 45 line segments at 8-degree intervals.
type CountdownTimerSuite struct {
	suite.Suite
}

func TestCountdownTimer(t *testing.T) {
	suite.Run(t, new(CountdownTimerSuite))
}

func (s *CountdownTimerSuite) TestNewCountdownTimerReturnsNonNil() {
	timer := ui.NewCountdownTimer()

	s.NotNil(timer, "NewCountdownTimer should return a non-nil widget")
}

func (s *CountdownTimerSuite) TestCountdownTimerImplementsWidget() {
	timer := ui.NewCountdownTimer()

	// Verify it satisfies fyne.Widget by calling Widget methods.
	// If CountdownTimer does not implement fyne.Widget, this will not compile.
	s.NotNil(timer.MinSize(), "widget should report a MinSize")
	renderer := timer.CreateRenderer()
	s.NotNil(renderer, "widget should create a renderer")
}

func (s *CountdownTimerSuite) TestCountdownTimerHas45Segments() {
	timer := ui.NewCountdownTimer()
	segments := timer.Segments()

	s.Len(segments, 45, "countdown timer must have exactly 45 segments")
}

func (s *CountdownTimerSuite) TestCountdownTimerSegmentAngles() {
	timer := ui.NewCountdownTimer()
	segments := timer.Segments()

	// Segments should be at 8-degree intervals starting at 8 degrees,
	// ending at 360 (which is 0, i.e. 12 o'clock).
	for i, seg := range segments {
		expectedAngle := float64((i + 1) * 8)
		s.InDelta(expectedAngle, seg.AngleDeg, 0.001,
			"segment %d should be at %.0f degrees", i, expectedAngle)
	}
}

func (s *CountdownTimerSuite) TestCountdownTimerCardinalLinesAreLong() {
	timer := ui.NewCountdownTimer()
	segments := timer.Segments()

	// Cardinal angles: 0° (=360°), 90°, 180°, 270° should be 3x short length (36px).
	cardinalAngles := map[float64]bool{360: true, 90: true, 180: true, 270: true}

	for _, seg := range segments {
		if cardinalAngles[seg.AngleDeg] {
			s.InDelta(36.0, seg.Length, 0.001,
				"cardinal line at %.0f degrees should be 36px (3x short)", seg.AngleDeg)
		}
	}
}

func (s *CountdownTimerSuite) TestCountdownTimerDiagonalLinesAreMedium() {
	timer := ui.NewCountdownTimer()
	segments := timer.Segments()

	// Diagonal angles: 45°, 135°, 225°, 315° should be 2x short length (24px).
	diagonalAngles := map[float64]bool{45: true, 135: true, 225: true, 315: true}

	for _, seg := range segments {
		if diagonalAngles[seg.AngleDeg] {
			s.InDelta(24.0, seg.Length, 0.001,
				"diagonal line at %.0f degrees should be 24px (2x short)", seg.AngleDeg)
		}
	}
}

func (s *CountdownTimerSuite) TestCountdownTimerRegularLinesAreShort() {
	timer := ui.NewCountdownTimer()
	segments := timer.Segments()

	// All lines that are not cardinal or diagonal should be 1x short length (12px).
	specialAngles := map[float64]bool{
		360: true, 90: true, 180: true, 270: true, // cardinal
		45: true, 135: true, 225: true, 315: true, // diagonal
	}

	for _, seg := range segments {
		if !specialAngles[seg.AngleDeg] {
			s.InDelta(12.0, seg.Length, 0.001,
				"regular line at %.0f degrees should be 12px (1x short)", seg.AngleDeg)
		}
	}
}

func (s *CountdownTimerSuite) TestCountdownTimerDefaultColors() {
	timer := ui.NewCountdownTimer()
	segments := timer.Segments()

	// All segments should be future (yellow #FFCE1B) by default.
	expectedColor := color.NRGBA{R: 0xFF, G: 0xCE, B: 0x1B, A: 0xFF}
	for i, seg := range segments {
		s.Equal(expectedColor, seg.Color,
			"segment %d should have future color #FFCE1B", i)
	}
}

func (s *CountdownTimerSuite) TestCountdownTimerElapsedSegmentsDimmed() {
	timer := ui.NewCountdownTimer()

	// Set half progress so some segments are elapsed.
	timer.SetProgress(0.5)
	segments := timer.Segments()

	elapsedColor := color.NRGBA{R: 0xFF, G: 0xCE, B: 0x1B, A: 64}

	// With 50% progress, roughly 22-23 segments should be elapsed.
	// Check that at least the first segment is elapsed/dimmed.
	s.Equal(elapsedColor, segments[0].Color,
		"first segment should be dimmed after 50%% progress")
}

func (s *CountdownTimerSuite) TestCountdownTimerSetProgress() {
	timer := ui.NewCountdownTimer()

	timer.SetProgress(0.5)
	segments := timer.Segments()

	// Count elapsed vs future segments.
	elapsedCount := 0
	for _, seg := range segments {
		if seg.State == ui.SegmentElapsed {
			elapsedCount++
		}
	}

	// 50% of 45 = 22.5, so expect 22 or 23 elapsed segments.
	s.True(elapsedCount >= 22 && elapsedCount <= 23,
		"50%% progress should yield ~22-23 elapsed segments, got %d", elapsedCount)
}

func (s *CountdownTimerSuite) TestCountdownTimerProgressZeroAllFuture() {
	timer := ui.NewCountdownTimer()

	timer.SetProgress(0.0)
	segments := timer.Segments()

	for i, seg := range segments {
		s.Equal(ui.SegmentFuture, seg.State,
			"segment %d should be Future at 0%% progress", i)
	}
}

func (s *CountdownTimerSuite) TestCountdownTimerProgressOneAllElapsed() {
	timer := ui.NewCountdownTimer()

	timer.SetProgress(1.0)
	segments := timer.Segments()

	for i, seg := range segments {
		s.Equal(ui.SegmentElapsed, seg.State,
			"segment %d should be Elapsed at 100%% progress", i)
	}
}

func (s *CountdownTimerSuite) TestCountdownTimerMinSize() {
	timer := ui.NewCountdownTimer()
	minSize := timer.MinSize()

	s.Greater(minSize.Width, float32(0), "MinSize width should be positive")
	s.Greater(minSize.Height, float32(0), "MinSize height should be positive")
}

func (s *CountdownTimerSuite) TestCountdownTimerResetClearsProgress() {
	timer := ui.NewCountdownTimer()

	timer.SetProgress(0.75)
	timer.Reset()
	segments := timer.Segments()

	for i, seg := range segments {
		s.Equal(ui.SegmentFuture, seg.State,
			"segment %d should be Future after Reset()", i)
	}
}
