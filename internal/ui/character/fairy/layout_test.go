package fairy

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// LayoutSuite verifies the jar rendered rect calculation for ImageFillContain
// letterboxing.
type LayoutSuite struct {
	suite.Suite
}

func TestLayout(t *testing.T) {
	suite.Run(t, new(LayoutSuite))
}

func (s *LayoutSuite) TestPillarboxedWhenContainerWiderThanJar() {
	// Container 400x300, jar aspect 0.5 (tall jar, width < height).
	// Container aspect = 400/300 = 1.333, which is > 0.5, so pillarboxed.
	// Expected: h = 300, w = 300 * 0.5 = 150, x = (400 - 150) / 2 = 125, y = 0.
	x, y, w, h := jarRenderedRect(400, 300, 0.5)

	s.InDelta(125.0, x, 0.01, "pillarbox x offset")
	s.InDelta(0.0, y, 0.01, "pillarbox y offset should be 0")
	s.InDelta(150.0, w, 0.01, "pillarbox rendered width")
	s.InDelta(300.0, h, 0.01, "pillarbox rendered height should equal container height")
}

func (s *LayoutSuite) TestLetterboxedWhenContainerTallerThanJar() {
	// Container 300x400, jar aspect 1.5 (wide jar, width > height).
	// Container aspect = 300/400 = 0.75, which is < 1.5, so letterboxed.
	// Expected: w = 300, h = 300 / 1.5 = 200, x = 0, y = (400 - 200) / 2 = 100.
	x, y, w, h := jarRenderedRect(300, 400, 1.5)

	s.InDelta(0.0, x, 0.01, "letterbox x offset should be 0")
	s.InDelta(100.0, y, 0.01, "letterbox y offset")
	s.InDelta(300.0, w, 0.01, "letterbox rendered width should equal container width")
	s.InDelta(200.0, h, 0.01, "letterbox rendered height")
}

func (s *LayoutSuite) TestNoGapsWhenAspectRatiosMatch() {
	// Container 300x400, jar aspect 0.75 (= 300/400).
	// Container aspect matches jar aspect exactly, so no gaps.
	// Expected: x = 0, y = 0, w = 300, h = 400.
	x, y, w, h := jarRenderedRect(300, 400, 0.75)

	s.InDelta(0.0, x, 0.01, "matching aspect x offset should be 0")
	s.InDelta(0.0, y, 0.01, "matching aspect y offset should be 0")
	s.InDelta(300.0, w, 0.01, "matching aspect width should equal container width")
	s.InDelta(400.0, h, 0.01, "matching aspect height should equal container height")
}
