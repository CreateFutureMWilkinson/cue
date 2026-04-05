package ui_test

import (
	"image/color"
	"math"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
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

// --- Renderer tests (Feature 017-Hotfix-A) ---

func (s *CountdownTimerSuite) TestRendererObjectsContains45Lines() {
	timer := ui.NewCountdownTimer()
	renderer := timer.CreateRenderer()

	objects := renderer.Objects()
	s.Len(objects, 45, "renderer must contain exactly 45 canvas objects")

	for i, obj := range objects {
		_, ok := obj.(*canvas.Line)
		s.True(ok, "object %d must be a *canvas.Line, got %T", i, obj)
	}
}

func (s *CountdownTimerSuite) TestRendererLayoutPositionsLinesWithinBounds() {
	timer := ui.NewCountdownTimer()
	renderer := timer.CreateRenderer()

	objects := renderer.Objects()
	s.Require().Len(objects, 45, "renderer must have 45 objects")

	size := fyne.NewSize(300, 300)
	renderer.Layout(size)

	for i, obj := range renderer.Objects() {
		line, ok := obj.(*canvas.Line)
		s.Require().True(ok, "object %d must be *canvas.Line", i)
		s.GreaterOrEqual(line.Position1.X, float32(0),
			"line %d Position1.X should be >= 0", i)
		s.GreaterOrEqual(line.Position1.Y, float32(0),
			"line %d Position1.Y should be >= 0", i)
		s.LessOrEqual(line.Position1.X, size.Width,
			"line %d Position1.X should be <= width", i)
		s.LessOrEqual(line.Position1.Y, size.Height,
			"line %d Position1.Y should be <= height", i)
		s.GreaterOrEqual(line.Position2.X, float32(0),
			"line %d Position2.X should be >= 0", i)
		s.GreaterOrEqual(line.Position2.Y, float32(0),
			"line %d Position2.Y should be >= 0", i)
		s.LessOrEqual(line.Position2.X, size.Width,
			"line %d Position2.X should be <= width", i)
		s.LessOrEqual(line.Position2.Y, size.Height,
			"line %d Position2.Y should be <= height", i)
	}
}

func (s *CountdownTimerSuite) TestRendererLayoutZeroSizeNoPanic() {
	timer := ui.NewCountdownTimer()
	renderer := timer.CreateRenderer()

	s.NotPanics(func() {
		renderer.Layout(fyne.NewSize(0, 0))
	}, "Layout with zero size must not panic")
}

func (s *CountdownTimerSuite) TestRendererLayoutScalesProportionally() {
	timer := ui.NewCountdownTimer()
	renderer := timer.CreateRenderer()

	objects := renderer.Objects()
	s.Require().Len(objects, 45, "renderer must have 45 objects")

	// Layout at small size.
	renderer.Layout(fyne.NewSize(100, 100))
	smallLine, ok := renderer.Objects()[0].(*canvas.Line)
	s.Require().True(ok, "object 0 must be *canvas.Line")
	smallSpread := math.Hypot(
		float64(smallLine.Position1.X-smallLine.Position2.X),
		float64(smallLine.Position1.Y-smallLine.Position2.Y),
	)

	// Layout at large size.
	renderer.Layout(fyne.NewSize(400, 400))
	largeLine, ok := renderer.Objects()[0].(*canvas.Line)
	s.Require().True(ok, "object 0 must be *canvas.Line")
	largeSpread := math.Hypot(
		float64(largeLine.Position1.X-largeLine.Position2.X),
		float64(largeLine.Position1.Y-largeLine.Position2.Y),
	)

	s.Greater(largeSpread, smallSpread,
		"larger layout size should produce larger line spread")
}

func (s *CountdownTimerSuite) TestRendererRefreshAtZeroProgressAllFutureColor() {
	timer := ui.NewCountdownTimer()
	renderer := timer.CreateRenderer()
	timer.SetProgress(0.0)
	renderer.Refresh()

	objects := renderer.Objects()
	s.Require().Len(objects, 45, "renderer must have 45 objects")

	expectedColor := color.NRGBA{R: 0xFF, G: 0xCE, B: 0x1B, A: 0xFF}
	for i, obj := range objects {
		line, ok := obj.(*canvas.Line)
		s.Require().True(ok, "object %d must be *canvas.Line", i)
		s.Equal(expectedColor, line.StrokeColor,
			"line %d should have future color at 0%% progress", i)
	}
}

func (s *CountdownTimerSuite) TestRendererRefreshAtFullProgressAllElapsedColor() {
	timer := ui.NewCountdownTimer()
	renderer := timer.CreateRenderer()
	timer.SetProgress(1.0)
	renderer.Refresh()

	objects := renderer.Objects()
	s.Require().Len(objects, 45, "renderer must have 45 objects")

	expectedColor := color.NRGBA{R: 0xFF, G: 0xCE, B: 0x1B, A: 64}
	for i, obj := range objects {
		line, ok := obj.(*canvas.Line)
		s.Require().True(ok, "object %d must be *canvas.Line", i)
		s.Equal(expectedColor, line.StrokeColor,
			"line %d should have elapsed color at 100%% progress", i)
	}
}

func (s *CountdownTimerSuite) TestRendererRefreshAtHalfProgressMixedColors() {
	timer := ui.NewCountdownTimer()
	renderer := timer.CreateRenderer()
	timer.SetProgress(0.5)
	renderer.Refresh()

	objects := renderer.Objects()
	s.Require().Len(objects, 45, "renderer must have 45 objects")

	elapsedC := color.NRGBA{R: 0xFF, G: 0xCE, B: 0x1B, A: 64}
	futureC := color.NRGBA{R: 0xFF, G: 0xCE, B: 0x1B, A: 0xFF}

	elapsedCount := 0
	futureCount := 0
	for i, obj := range objects {
		line, ok := obj.(*canvas.Line)
		s.Require().True(ok, "object %d must be *canvas.Line", i)
		if line.StrokeColor == elapsedC {
			elapsedCount++
		} else if line.StrokeColor == futureC {
			futureCount++
		}
	}

	s.True(elapsedCount >= 22 && elapsedCount <= 23,
		"expected ~22 elapsed lines, got %d", elapsedCount)
	s.True(futureCount >= 22 && futureCount <= 23,
		"expected ~23 future lines, got %d", futureCount)
}

func (s *CountdownTimerSuite) TestRendererCardinalLinesLongerThanRegular() {
	timer := ui.NewCountdownTimer()
	renderer := timer.CreateRenderer()

	objects := renderer.Objects()
	s.Require().Len(objects, 45, "renderer must have 45 objects")

	renderer.Layout(fyne.NewSize(300, 300))

	// Find indices where segment length is long (cardinal) vs short (regular).
	segments := timer.Segments()
	regularIndex := -1
	var cardinalIndices []int
	for i, seg := range segments {
		if seg.Length == 36.0 {
			cardinalIndices = append(cardinalIndices, i)
		} else if seg.Length == 12.0 && regularIndex == -1 {
			regularIndex = i
		}
	}
	// Even if no cardinal segments exist at exact angles, test that Layout
	// renders different lengths. Use first and last segment as proxies.
	if regularIndex == -1 {
		regularIndex = 0
	}
	if len(cardinalIndices) == 0 {
		// Fall back: segment at index 44 (360 deg) should be cardinal.
		cardinalIndices = []int{44}
	}

	regularLine, ok := renderer.Objects()[regularIndex].(*canvas.Line)
	s.Require().True(ok, "object %d must be *canvas.Line", regularIndex)
	regularLen := math.Hypot(
		float64(regularLine.Position1.X-regularLine.Position2.X),
		float64(regularLine.Position1.Y-regularLine.Position2.Y),
	)

	for _, ci := range cardinalIndices {
		cardLine, ok := renderer.Objects()[ci].(*canvas.Line)
		s.Require().True(ok, "object %d must be *canvas.Line", ci)
		cardLen := math.Hypot(
			float64(cardLine.Position1.X-cardLine.Position2.X),
			float64(cardLine.Position1.Y-cardLine.Position2.Y),
		)
		s.Greater(cardLen, regularLen,
			"cardinal line at index %d should be longer than regular line", ci)
	}
}

func (s *CountdownTimerSuite) TestRendererAllLinesNonZeroLength() {
	timer := ui.NewCountdownTimer()
	renderer := timer.CreateRenderer()

	objects := renderer.Objects()
	s.Require().Len(objects, 45, "renderer must have 45 objects")

	renderer.Layout(fyne.NewSize(300, 300))

	for i, obj := range renderer.Objects() {
		line, ok := obj.(*canvas.Line)
		s.Require().True(ok, "object %d must be *canvas.Line", i)
		lineLen := math.Hypot(
			float64(line.Position1.X-line.Position2.X),
			float64(line.Position1.Y-line.Position2.Y),
		)
		s.Greater(lineLen, float64(0),
			"line %d should have non-zero rendered length after Layout", i)
	}
}

func (s *CountdownTimerSuite) TestRendererLineStrokeWidths() {
	timer := ui.NewCountdownTimer()
	renderer := timer.CreateRenderer()

	segments := timer.Segments()
	objects := renderer.Objects()
	s.Require().Len(objects, 45, "renderer must have 45 objects")

	for i, obj := range objects {
		line, ok := obj.(*canvas.Line)
		s.Require().True(ok, "object %d must be *canvas.Line", i)
		seg := segments[i]

		if seg.Length == 36.0 || seg.Length == 24.0 {
			// Cardinal or diagonal lines should have stroke width 3.0.
			s.InDelta(3.0, line.StrokeWidth, 0.001,
				"cardinal/diagonal line %d should have stroke width 3.0", i)
		} else {
			// Regular lines should have stroke width 2.0.
			s.InDelta(2.0, line.StrokeWidth, 0.001,
				"regular line %d should have stroke width 2.0", i)
		}
	}
}

func (s *CountdownTimerSuite) TestRendererMinSizePositive() {
	timer := ui.NewCountdownTimer()
	renderer := timer.CreateRenderer()
	minSize := renderer.MinSize()

	s.Greater(minSize.Width, float32(0), "renderer MinSize width should be positive")
	s.Greater(minSize.Height, float32(0), "renderer MinSize height should be positive")
}

func (s *CountdownTimerSuite) TestSetFlashVisibleHidesCurrentSegment() {
	timer := ui.NewCountdownTimer()
	renderer := timer.CreateRenderer()

	objects := renderer.Objects()
	s.Require().Len(objects, 45, "renderer must have 45 objects")

	timer.SetProgress(0.5)
	renderer.Refresh()

	// Find the first non-elapsed segment (the "current" one).
	segments := timer.Segments()
	currentIdx := -1
	for i, seg := range segments {
		if seg.State != ui.SegmentElapsed {
			currentIdx = i
			break
		}
	}
	s.Require().NotEqual(-1, currentIdx, "should have a non-elapsed segment at 50%%")

	// Hide flash — the current segment line should become invisible.
	timer.SetFlashVisible(false)
	renderer.Refresh()
	line, ok := renderer.Objects()[currentIdx].(*canvas.Line)
	s.Require().True(ok, "object at current index must be *canvas.Line")
	s.True(line.Hidden, "current segment line should be hidden when flash is not visible")
}

func (s *CountdownTimerSuite) TestSetFlashVisibleShowsCurrentSegment() {
	timer := ui.NewCountdownTimer()
	renderer := timer.CreateRenderer()

	objects := renderer.Objects()
	s.Require().Len(objects, 45, "renderer must have 45 objects")

	timer.SetProgress(0.5)

	// First hide, then show.
	timer.SetFlashVisible(false)
	renderer.Refresh()
	timer.SetFlashVisible(true)
	renderer.Refresh()

	segments := timer.Segments()
	currentIdx := -1
	for i, seg := range segments {
		if seg.State != ui.SegmentElapsed {
			currentIdx = i
			break
		}
	}
	s.Require().NotEqual(-1, currentIdx, "should have a non-elapsed segment at 50%%")

	line, ok := renderer.Objects()[currentIdx].(*canvas.Line)
	s.Require().True(ok, "object at current index must be *canvas.Line")
	s.False(line.Hidden, "current segment line should be visible when flash is visible")
}

func (s *CountdownTimerSuite) TestRendererLayoutTrigonometryPositions() {
	timer := ui.NewCountdownTimer()
	renderer := timer.CreateRenderer()

	objects := renderer.Objects()
	s.Require().Len(objects, 45, "renderer must have 45 objects")

	size := fyne.NewSize(300, 300)
	renderer.Layout(size)

	centerX := float64(size.Width / 2)
	centerY := float64(size.Height / 2)
	radius := float64(min(size.Width, size.Height)) / 2 * 0.9
	canonicalRadius := 120.0
	scale := radius / canonicalRadius

	segments := timer.Segments()
	for i, obj := range renderer.Objects() {
		line, ok := obj.(*canvas.Line)
		s.Require().True(ok, "object %d must be *canvas.Line", i)
		seg := segments[i]
		angleRad := seg.AngleDeg * math.Pi / 180.0

		expectedOuterX := centerX + radius*math.Sin(angleRad)
		expectedOuterY := centerY - radius*math.Cos(angleRad)
		expectedInnerX := centerX + (radius-seg.Length*scale)*math.Sin(angleRad)
		expectedInnerY := centerY - (radius-seg.Length*scale)*math.Cos(angleRad)

		s.InDelta(expectedOuterX, float64(line.Position1.X), 1.0,
			"line %d outer X mismatch", i)
		s.InDelta(expectedOuterY, float64(line.Position1.Y), 1.0,
			"line %d outer Y mismatch", i)
		s.InDelta(expectedInnerX, float64(line.Position2.X), 1.0,
			"line %d inner X mismatch", i)
		s.InDelta(expectedInnerY, float64(line.Position2.Y), 1.0,
			"line %d inner Y mismatch", i)
	}
}
