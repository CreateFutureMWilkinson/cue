package fairy_test

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/character/fairy"
	"github.com/stretchr/testify/suite"
)

// Compile-time usage check.
var _ fyne.CanvasObject

type FairyJarSuite struct {
	suite.Suite
}

func TestFairyJar(t *testing.T) {
	suite.Run(t, new(FairyJarSuite))
}

func (s *FairyJarSuite) TestWidgetReturnsNonNilContainer() {
	c := fairy.NewFairyCharacter()
	w := c.Widget()
	s.NotNil(w, "Widget() must return a non-nil container")
}

func (s *FairyJarSuite) TestSVGLayersPresent() {
	c := fairy.NewFairyCharacter()
	w := c.Widget()

	// The widget should be a container with at least 3 children:
	// jar_back SVG, fairy layer (body+glow), jar_front SVG.
	cont, ok := w.(*fyne.Container)
	s.Require().True(ok, "Widget() must return a *fyne.Container")
	s.GreaterOrEqual(len(cont.Objects), 3,
		"jar container must have at least 3 layers (back SVG, fairy, front SVG)")
}

func (s *FairyJarSuite) TestBodyCircleSizedAt10Percent() {
	c := fairy.NewFairyCharacter()
	w := c.Widget()

	// Resize the container to a known size so proportions are testable.
	jarWidth := float32(200)
	w.Resize(fyne.NewSize(jarWidth, 400))

	bodyCircle := c.BodyCircle()
	s.Require().NotNil(bodyCircle, "BodyCircle() must return the body circle")

	expectedDiameter := jarWidth * 0.10
	s.InDelta(expectedDiameter, bodyCircle.Size().Width, 1.0,
		"body circle width should be 10%% of jar width")
	s.InDelta(expectedDiameter, bodyCircle.Size().Height, 1.0,
		"body circle height should be 10%% of jar width")
}

func (s *FairyJarSuite) TestGlowCircleSizedAt25Percent() {
	c := fairy.NewFairyCharacter()
	w := c.Widget()

	jarWidth := float32(200)
	w.Resize(fyne.NewSize(jarWidth, 400))

	glowCircle := c.GlowCircle()
	s.Require().NotNil(glowCircle, "GlowCircle() must return the outermost glow circle")

	expectedDiameter := jarWidth * 0.25
	s.InDelta(expectedDiameter, glowCircle.Size().Width, 1.0,
		"glow circle width should be 25%% of jar width")
	s.InDelta(expectedDiameter, glowCircle.Size().Height, 1.0,
		"glow circle height should be 25%% of jar width")
}

func (s *FairyJarSuite) TestSetPositionClampingAboveOne() {
	c := fairy.NewFairyCharacter()
	c.SetPosition(1.5, 2.0)
	x, y := c.Position()
	s.Equal(1.0, x, "x > 1.0 must be clamped to 1.0")
	s.Equal(1.0, y, "y > 1.0 must be clamped to 1.0")
}

func (s *FairyJarSuite) TestSetPositionClampingBelowZero() {
	c := fairy.NewFairyCharacter()
	c.SetPosition(-0.5, -1.0)
	x, y := c.Position()
	s.Equal(0.0, x, "x < 0.0 must be clamped to 0.0")
	s.Equal(0.0, y, "y < 0.0 must be clamped to 0.0")
}

func (s *FairyJarSuite) TestPositionRoundTrip() {
	c := fairy.NewFairyCharacter()

	cases := []struct {
		x, y float64
	}{
		{0.0, 0.0},
		{0.5, 0.5},
		{1.0, 1.0},
		{0.3, 0.7},
	}

	for _, tc := range cases {
		c.SetPosition(tc.x, tc.y)
		gotX, gotY := c.Position()
		s.Equal(tc.x, gotX, "Position().x round-trip for input %v", tc.x)
		s.Equal(tc.y, gotY, "Position().y round-trip for input %v", tc.y)
	}
}

func (s *FairyJarSuite) TestSetBodyColorApplied() {
	c := fairy.NewFairyCharacter()

	newColor := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	c.SetBodyColor(newColor)

	bodyCircle := c.BodyCircle()
	s.Require().NotNil(bodyCircle, "BodyCircle() must return the body circle")

	r1, g1, b1, a1 := bodyCircle.FillColor.RGBA()
	r2, g2, b2, a2 := newColor.RGBA()
	s.Equal(r2, r1, "red channel should match after SetBodyColor")
	s.Equal(g2, g1, "green channel should match after SetBodyColor")
	s.Equal(b2, b1, "blue channel should match after SetBodyColor")
	s.Equal(a2, a1, "alpha channel should match after SetBodyColor")
}

func (s *FairyJarSuite) TestSetGlowIntensityClampingAboveOne() {
	c := fairy.NewFairyCharacter()
	c.SetGlowIntensity(1.5)
	s.Equal(1.0, c.GlowIntensity(),
		"glow intensity > 1.0 must be clamped to 1.0")
}

func (s *FairyJarSuite) TestSetGlowIntensityClampingBelowZero() {
	c := fairy.NewFairyCharacter()
	c.SetGlowIntensity(-0.5)
	s.Equal(0.0, c.GlowIntensity(),
		"glow intensity < 0.0 must be clamped to 0.0")
}

func (s *FairyJarSuite) TestGlowLayerCountIsEight() {
	c := fairy.NewFairyCharacter()
	layers := c.GlowLayers()
	s.Len(layers, 8, "there must be exactly 8 concentric glow layers")
}

func (s *FairyJarSuite) TestInitialColorIsDarkGreen() {
	c := fairy.NewFairyCharacter()

	bodyCircle := c.BodyCircle()
	s.Require().NotNil(bodyCircle, "BodyCircle() must return the body circle")

	// #006100 = R:0, G:97, B:0, A:255
	expected := color.RGBA{R: 0x00, G: 0x61, B: 0x00, A: 0xFF}
	r1, g1, b1, a1 := bodyCircle.FillColor.RGBA()
	r2, g2, b2, a2 := expected.RGBA()
	s.Equal(r2, r1, "initial red channel should be 0x00")
	s.Equal(g2, g1, "initial green channel should be 0x61")
	s.Equal(b2, b1, "initial blue channel should be 0x00")
	s.Equal(a2, a1, "initial alpha channel should be 0xFF")
}

func (s *FairyJarSuite) TestInitialPositionIsBottomCenter() {
	c := fairy.NewFairyCharacter()
	x, y := c.Position()
	s.Equal(0.5, x, "initial x position must be 0.5 (center)")
	s.Equal(1.0, y, "initial y position must be 1.0 (bottom)")
}

func (s *FairyJarSuite) TestCharacterInterfaceSatisfied() {
	c := fairy.NewFairyCharacter()

	// Widget returns non-nil.
	s.NotNil(c.Widget(), "Widget() must be non-nil")

	// Name returns "fairy".
	s.Equal("fairy", c.Name(), "Name() must return fairy")

	// State transitions work.
	c.TransitionTo(character.StateWorking)
	s.Equal(character.StateWorking, c.CurrentState(),
		"CurrentState() must reflect TransitionTo")
}
