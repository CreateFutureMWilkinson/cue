package fairy_test

import (
	"testing"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/character/fairy"
	"github.com/stretchr/testify/suite"
)

// FairyEmbedSuite verifies that the fairy constructor uses embedded PNG assets
// rather than file-based SVGs, so it works without a build_assets/ directory.
type FairyEmbedSuite struct {
	suite.Suite
}

func TestFairyEmbed(t *testing.T) {
	suite.Run(t, new(FairyEmbedSuite))
}

func (s *FairyEmbedSuite) TestConstructorSucceedsWithoutBuildAssetsDir() {
	// NewFairyCharacter should work from any working directory because
	// it uses go:embed PNGs, not file-system SVG paths.
	// If it relied on build_assets/ files, this would fail when run from
	// a directory that lacks that folder.
	fc := fairy.NewFairyCharacter()
	s.NotNil(fc, "NewFairyCharacter should succeed without build_assets/ directory")
}

func (s *FairyEmbedSuite) TestWidgetReturnsNonNilCanvasObject() {
	fc := fairy.NewFairyCharacter()
	w := fc.Widget()
	s.NotNil(w, "Widget() should return a non-nil canvas object from embedded assets")
}

func (s *FairyEmbedSuite) TestConstructorProducesValidBodyCircle() {
	fc := fairy.NewFairyCharacter()
	body := fc.BodyCircle()
	s.NotNil(body, "BodyCircle() should be non-nil after construction with embedded assets")
}

func (s *FairyEmbedSuite) TestConstructorProducesValidGlowLayers() {
	fc := fairy.NewFairyCharacter()
	layers := fc.GlowLayers()
	s.NotEmpty(layers, "GlowLayers() should contain glow circles from embedded-asset constructor")
}
