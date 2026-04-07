package wasmhost_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"fyne.io/fyne/v2"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/character/wasmhost"
	"github.com/stretchr/testify/suite"
)

type CanvasHostSuite struct {
	suite.Suite
	host *wasmhost.FyneCanvasHost
}

func TestCanvasHost(t *testing.T) {
	suite.Run(t, new(CanvasHostSuite))
}

func (s *CanvasHostSuite) SetupTest() {
	s.host = wasmhost.NewFyneCanvasHost()
}

func (s *CanvasHostSuite) containerObjects() []fyne.CanvasObject {
	w := s.host.Widget()
	c, ok := w.(*fyne.Container)
	s.Require().True(ok, "Widget() must return a *fyne.Container")
	return c.Objects
}

func minimalPNG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// TestSetCircleAddsChild verifies that SetCircle adds a visible child to the container.
func (s *CanvasHostSuite) TestSetCircleAddsChild() {
	s.host.SetCircle(0, 0.5, 0.5, 0.1, 0, 255, 0, 255)

	objects := s.containerObjects()
	s.GreaterOrEqual(len(objects), 1, "SetCircle should add at least one child to the container")
}

// TestSetCircleSameIDDoesNotDuplicate verifies that calling SetCircle twice with the same ID
// results in exactly one child, not two.
func (s *CanvasHostSuite) TestSetCircleSameIDDoesNotDuplicate() {
	s.host.SetCircle(0, 0.5, 0.5, 0.1, 0, 255, 0, 255)
	s.host.SetCircle(0, 0.6, 0.6, 0.2, 255, 0, 0, 255)

	objects := s.containerObjects()
	s.Equal(1, len(objects), "SetCircle with same ID twice should result in exactly 1 child")
}

// TestRemoveCircleRemovesChild verifies that RemoveCircle removes the previously added circle.
func (s *CanvasHostSuite) TestRemoveCircleRemovesChild() {
	s.host.SetCircle(0, 0.5, 0.5, 0.1, 0, 255, 0, 255)

	// Precondition: circle was added
	s.GreaterOrEqual(len(s.containerObjects()), 1, "precondition: SetCircle should add a child")

	s.host.RemoveCircle(0)

	objects := s.containerObjects()
	s.Equal(0, len(objects), "RemoveCircle should remove the circle from the container")
}

// TestSetImageAddsChild verifies that SetImage adds a visible child to the container.
func (s *CanvasHostSuite) TestSetImageAddsChild() {
	s.host.SetImage(100, 0.5, 0.5, 1.0, 1.0, minimalPNG())

	objects := s.containerObjects()
	s.GreaterOrEqual(len(objects), 1, "SetImage should add at least one child to the container")
}

// TestRemoveImageRemovesChild verifies that RemoveImage removes the previously added image.
func (s *CanvasHostSuite) TestRemoveImageRemovesChild() {
	s.host.SetImage(100, 0.5, 0.5, 1.0, 1.0, minimalPNG())

	// Precondition: image was added
	s.GreaterOrEqual(len(s.containerObjects()), 1, "precondition: SetImage should add a child")

	s.host.RemoveImage(100)

	objects := s.containerObjects()
	s.Equal(0, len(objects), "RemoveImage should remove the image from the container")
}

// TestWidgetReturnsNonNil verifies the Widget method returns a non-nil canvas object.
func (s *CanvasHostSuite) TestWidgetReturnsNonNil() {
	w := s.host.Widget()
	s.NotNil(w)
}

// TestFyneCanvasHostImplementsCanvasHost verifies the interface is satisfied at compile time.
func (s *CanvasHostSuite) TestFyneCanvasHostImplementsCanvasHost() {
	var _ wasmhost.CanvasHost = s.host
}
