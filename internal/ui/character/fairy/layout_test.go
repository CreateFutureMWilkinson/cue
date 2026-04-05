package fairy

import (
	"testing"

	"fyne.io/fyne/v2"
	"github.com/stretchr/testify/suite"
)

// LayoutSuite verifies jar rendered rect and interior bounds mapping.
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

// --- Behavior 2: Body Circle Edge Inset ---
//
// The ENTIRE circle must stay within the jar interior. At pos=0 the circle's
// leading edge touches the interior wall; at pos=1 the trailing edge touches
// the opposite wall. The circle diameter is subtracted from the available
// interior span so half the body never extends outside the jar walls.
// Jar image is 375x795 (aspect ≈ 0.4717).

func (s *LayoutSuite) TestBodyEntirelyInsideAtOrigin() {
	f := NewFairyCharacter()
	f.DisableRefresh()
	f.SetPosition(0.0, 0.0)

	// Use container matching jar aspect ratio (375x795) → no letterboxing.
	f.Widget().Resize(fyne.NewSize(375, 795))

	body := f.BodyCircle()

	// At pos (0,0) the circle's top-left corner should align with the
	// interior top-left corner — the entire circle is inside the jar.
	expectedX := float32(jarInteriorLeft) * 375
	expectedY := float32(jarInteriorTop) * 795

	s.InDelta(expectedX, body.Position().X, 1.0,
		"position (0,0) body left edge should align with interior left wall")
	s.InDelta(expectedY, body.Position().Y, 1.0,
		"position (0,0) body top edge should align with interior top wall")
}

func (s *LayoutSuite) TestBodyEntirelyInsideAtMax() {
	f := NewFairyCharacter()
	f.DisableRefresh()
	f.SetPosition(1.0, 1.0)

	f.Widget().Resize(fyne.NewSize(375, 795))

	body := f.BodyCircle()
	bodyDiam := float32(375) * bodyRatio

	// At pos (1,1) the circle's right/bottom edges should align with the
	// interior right/bottom walls. Top-left = wall - diameter.
	expectedX := float32(jarInteriorRight)*375 - bodyDiam
	expectedY := float32(jarInteriorBottom)*795 - bodyDiam

	s.InDelta(expectedX, body.Position().X, 1.0,
		"position (1,1) body right edge should align with interior right wall")
	s.InDelta(expectedY, body.Position().Y, 1.0,
		"position (1,1) body bottom edge should align with interior bottom wall")
}

func (s *LayoutSuite) TestGlowLayersConcentricWithBody() {
	f := NewFairyCharacter()
	f.DisableRefresh()
	f.SetPosition(0.0, 0.0)

	f.Widget().Resize(fyne.NewSize(375, 795))

	body := f.BodyCircle()
	bodyDiam := float32(375) * bodyRatio
	bodyCenterX := body.Position().X + bodyDiam/2
	bodyCenterY := body.Position().Y + bodyDiam/2

	glowDiam := float32(375) * glowRatio
	for i, glowLayer := range f.GlowLayers() {
		interpolation := float32(i+1) / float32(fairyGlowLayerCount)
		diameter := bodyDiam + (glowDiam-bodyDiam)*interpolation

		glowCenterX := glowLayer.Position().X + diameter/2
		glowCenterY := glowLayer.Position().Y + diameter/2

		s.InDelta(bodyCenterX, glowCenterX, 0.01,
			"glow layer %d center X should match body center X", i)
		s.InDelta(bodyCenterY, glowCenterY, 0.01,
			"glow layer %d center Y should match body center Y", i)
	}
}

func (s *LayoutSuite) TestBodyCenteredAtHalf() {
	f := NewFairyCharacter()
	f.DisableRefresh()
	f.SetPosition(0.5, 0.5)

	f.Widget().Resize(fyne.NewSize(375, 795))

	body := f.BodyCircle()
	bodyDiam := float32(375) * bodyRatio

	// Interior pixel boundaries.
	intLeft := float32(jarInteriorLeft) * 375
	intRight := float32(jarInteriorRight) * 375
	intTop := float32(jarInteriorTop) * 795
	intBottom := float32(jarInteriorBottom) * 795

	// At pos (0.5, 0.5) the body is centered in the available interior span
	// (which is interior size minus the body diameter).
	expectedX := intLeft + 0.5*(intRight-intLeft-bodyDiam)
	expectedY := intTop + 0.5*(intBottom-intTop-bodyDiam)

	s.InDelta(expectedX, body.Position().X, 1.0,
		"position (0.5,0.5) body X should be centered in interior")
	s.InDelta(expectedY, body.Position().Y, 1.0,
		"position (0.5,0.5) body Y should be centered in interior")
}
