package character

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

const (
	// fairyIndicatorSize is the diameter of the state indicator circle.
	fairyIndicatorSize = 40

	// fairyGlowLayerCount is the number of concentric glow layers.
	fairyGlowLayerCount = 8

	// bodyRatio is the body circle diameter as a fraction of jar width.
	bodyRatio = 0.10

	// glowRatio is the outermost glow circle diameter as a fraction of jar width.
	glowRatio = 0.25
)

var (
	// initialFairyColor is the default body color (#006100).
	initialFairyColor = color.RGBA{R: 0x00, G: 0x61, B: 0x00, A: 0xFF}

	// State colors for the fairy character indicator.
	colorIdle         = color.RGBA{R: 200, G: 200, B: 255, A: 255} // Light blue
	colorStarting     = color.RGBA{R: 255, G: 255, B: 200, A: 255} // Light yellow
	colorWorking      = color.RGBA{R: 200, G: 255, B: 200, A: 255} // Light green
	colorNotifying    = color.RGBA{R: 255, G: 200, B: 100, A: 255} // Orange
	colorError        = color.RGBA{R: 255, G: 100, B: 100, A: 255} // Light red
	colorShuttingDown = color.RGBA{R: 150, G: 150, B: 150, A: 255} // Gray
)

// FairyCharacter is a character implementation that renders a fairy inside a
// glass jar. The jar is composed of SVG back/front layers with the fairy's
// body and glow circles sandwiched between them.
type FairyCharacter struct {
	state     CharacterState
	container *fyne.Container
	indicator *canvas.Circle

	// Jar layers.
	jarBack  *canvas.Image
	jarFront *canvas.Image

	// Fairy circles.
	bodyCircle *canvas.Circle
	glowLayers []*canvas.Circle

	// Position in normalized 0..1 coordinates.
	posX, posY float64

	// Glow intensity 0..1.
	glowIntensity float64
}

// NewFairyCharacter creates a new FairyCharacter in the Idle state with jar
// rendering. The fairy starts at the bottom-center position (0.5, 1.0) with
// dark green body color (#006100).
func NewFairyCharacter() *FairyCharacter {
	// State indicator (kept for backward compatibility with TransitionTo).
	indicator := canvas.NewCircle(stateColor(StateIdle))
	indicator.Resize(fyne.NewSize(fairyIndicatorSize, fairyIndicatorSize))
	indicator.Hide()

	// Jar SVG layers.
	jarBack := canvas.NewImageFromFile("build_assets/images/jar_back.svg")
	jarBack.FillMode = canvas.ImageFillContain
	jarFront := canvas.NewImageFromFile("build_assets/images/jar_front.svg")
	jarFront.FillMode = canvas.ImageFillContain

	// Body circle.
	bodyCircle := canvas.NewCircle(initialFairyColor)

	// Glow layers — 8 concentric circles from innermost to outermost.
	glowLayers := make([]*canvas.Circle, fairyGlowLayerCount)
	for i := range glowLayers {
		c := canvas.NewCircle(color.RGBA{R: 0x00, G: 0x61, B: 0x00, A: 30})
		glowLayers[i] = c
	}

	f := &FairyCharacter{
		state:         StateIdle,
		indicator:     indicator,
		jarBack:       jarBack,
		jarFront:      jarFront,
		bodyCircle:    bodyCircle,
		glowLayers:    glowLayers,
		posX:          0.5,
		posY:          1.0,
		glowIntensity: 0.0,
	}

	// Build the container with custom layout for proportional sizing.
	// Layer order (back to front): jarBack, glow layers, body, jarFront, indicator.
	objects := make([]fyne.CanvasObject, 0, 3+fairyGlowLayerCount)
	objects = append(objects, jarBack)
	for _, gl := range glowLayers {
		objects = append(objects, gl)
	}
	objects = append(objects, bodyCircle)
	objects = append(objects, jarFront)
	objects = append(objects, indicator)

	f.container = container.New(&fairyJarLayout{fairy: f}, objects...)

	return f
}

// Name returns the character name.
func (f *FairyCharacter) Name() string { return "fairy" }

// TransitionTo changes the character's state and updates the indicator color.
func (f *FairyCharacter) TransitionTo(state CharacterState) {
	f.state = state
	f.indicator.FillColor = stateColor(state)
	f.indicator.Refresh()
}

// CurrentState returns the current character state.
func (f *FairyCharacter) CurrentState() CharacterState { return f.state }

// Widget returns the jar container as a canvas object.
func (f *FairyCharacter) Widget() fyne.CanvasObject { return f.container }

// SetPosition sets the fairy's position in normalized coordinates (0.0-1.0).
// Values are clamped to the valid range.
func (f *FairyCharacter) SetPosition(x, y float64) {
	f.posX = clamp01(x)
	f.posY = clamp01(y)
}

// Position returns the fairy's current normalized position.
func (f *FairyCharacter) Position() (x, y float64) {
	return f.posX, f.posY
}

// SetBodyColor changes the fill color of the body circle.
func (f *FairyCharacter) SetBodyColor(c color.Color) {
	f.bodyCircle.FillColor = c
	f.bodyCircle.Refresh()
}

// SetGlowIntensity sets the glow intensity (0.0-1.0). Values are clamped.
func (f *FairyCharacter) SetGlowIntensity(intensity float64) {
	f.glowIntensity = clamp01(intensity)
}

// GlowIntensity returns the current glow intensity.
func (f *FairyCharacter) GlowIntensity() float64 {
	return f.glowIntensity
}

// BodyCircle returns the fairy's body circle.
func (f *FairyCharacter) BodyCircle() *canvas.Circle {
	return f.bodyCircle
}

// GlowCircle returns the outermost glow layer circle.
func (f *FairyCharacter) GlowCircle() *canvas.Circle {
	if len(f.glowLayers) == 0 {
		return nil
	}
	return f.glowLayers[len(f.glowLayers)-1]
}

// GlowLayers returns all glow layer circles.
func (f *FairyCharacter) GlowLayers() []*canvas.Circle {
	return f.glowLayers
}

// clamp01 clamps a value to the range [0.0, 1.0].
func clamp01(v float64) float64 {
	if v < 0.0 {
		return 0.0
	}
	if v > 1.0 {
		return 1.0
	}
	return v
}

func stateColor(s CharacterState) color.Color {
	switch s {
	case StateIdle:
		return colorIdle
	case StateStarting:
		return colorStarting
	case StateWorking:
		return colorWorking
	case StateNotifying:
		return colorNotifying
	case StateError:
		return colorError
	case StateShuttingDown:
		return colorShuttingDown
	default:
		return colorIdle
	}
}

// fairyJarLayout is a custom Fyne layout that sizes jar layers to fill the
// container and sizes fairy circles proportionally to the container width.
type fairyJarLayout struct {
	fairy *FairyCharacter
}

func (l *fairyJarLayout) MinSize(_ []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(100, 200)
}

func (l *fairyJarLayout) Layout(_ []fyne.CanvasObject, size fyne.Size) {
	w := size.Width
	h := size.Height

	// Jar SVGs fill the entire container.
	l.fairy.jarBack.Resize(size)
	l.fairy.jarBack.Move(fyne.NewPos(0, 0))

	l.fairy.jarFront.Resize(size)
	l.fairy.jarFront.Move(fyne.NewPos(0, 0))

	// Body circle: 10% of jar width.
	bodyDiam := w * bodyRatio
	bodySize := fyne.NewSize(bodyDiam, bodyDiam)
	l.fairy.bodyCircle.Resize(bodySize)
	l.fairy.bodyCircle.Move(fyne.NewPos(
		float32(l.fairy.posX)*w-bodyDiam/2,
		float32(l.fairy.posY)*h-bodyDiam/2,
	))

	// Glow layers: linearly interpolated from body size to 25% of jar width.
	glowDiam := w * glowRatio
	for i, gl := range l.fairy.glowLayers {
		// Layer 0 is innermost (smallest), layer N-1 is outermost (largest).
		t := float32(i+1) / float32(fairyGlowLayerCount)
		d := bodyDiam + (glowDiam-bodyDiam)*t
		s := fyne.NewSize(d, d)
		gl.Resize(s)
		gl.Move(fyne.NewPos(
			float32(l.fairy.posX)*w-d/2,
			float32(l.fairy.posY)*h-d/2,
		))
	}

	// Hidden indicator.
	l.fairy.indicator.Resize(fyne.NewSize(fairyIndicatorSize, fairyIndicatorSize))
}
