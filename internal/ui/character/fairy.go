package character

import (
	"crypto/rand"
	"encoding/binary"
	"image/color"
	"math"
	mathrand "math/rand"
	"sync"

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

	// Idle position coordinates.
	IdleOriginX = 0.5
	IdleOriginY = 1.0
)

var (
	// IdleBodyColor is the idle state body color (#006100).
	IdleBodyColor = color.RGBA{R: 0x00, G: 0x61, B: 0x00, A: 0xFF}

	// initialFairyColor is the default body color (#006100).
	initialFairyColor = IdleBodyColor

	// State colors for the fairy character indicator.
	colorIdle         = color.RGBA{R: 200, G: 200, B: 255, A: 255} // Light blue
	colorStarting     = color.RGBA{R: 255, G: 255, B: 200, A: 255} // Light yellow
	colorWorking      = color.RGBA{R: 200, G: 255, B: 200, A: 255} // Light green
	colorNotifying    = color.RGBA{R: 255, G: 200, B: 100, A: 255} // Orange
	colorError        = color.RGBA{R: 255, G: 100, B: 100, A: 255} // Light red
	colorShuttingDown = color.RGBA{R: 150, G: 150, B: 150, A: 255} // Gray

	// glowBaseAlphas stores the graduated base alpha values for each glow layer.
	// Index 0 = innermost (brightest at 128), index 7 = outermost (dimmest at 16).
	// Final alpha = base_alpha * glow_intensity for smooth breathing effects.
	glowBaseAlphas = [fairyGlowLayerCount]uint8{128, 112, 96, 80, 64, 48, 32, 16}
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

	// Clock for creating animators.
	clock Clock

	// Current running animator (nil if none).
	currentAnimator StateAnimator

	// Mutex for thread-safe transitions.
	mu sync.Mutex

	// refreshFunc is called after visual updates; replaced by DisableRefresh in tests.
	refreshFunc func()
}

// NewFairyCharacter creates a new FairyCharacter in the Idle state with jar
// rendering. The fairy starts at the bottom-center position (0.5, 1.0) with
// dark green body color (#006100).
func NewFairyCharacter() *FairyCharacter {
	// State indicator (hidden but maintained for TransitionTo method compatibility).
	// This circle tracks state colors but is not visually displayed.
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
	// Initial glow intensity is 0.0, so all alphas start at 0.
	glowLayers := make([]*canvas.Circle, fairyGlowLayerCount)
	for i := range glowLayers {
		glowLayers[i] = newGlowCircle(0)
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
		clock:         WallClock{},
		refreshFunc:   func() {}, // Default no-op, replaced by container refresh in production
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

// TransitionTo changes the character's state, stops the current animator,
// creates a new animator for the target state, and starts it.
func (f *FairyCharacter) TransitionTo(state CharacterState) {
	// Stop any existing animator and update to new state.
	animator := f.stopAndUpdateState(state)

	// Start the new animator if one was created.
	if animator != nil {
		animator.Start(f)
	}
}

// stopAndUpdateState is a helper that stops the current animator (if any),
// updates the character state and indicator, and creates a new animator for the target state.
// This encapsulates the common pattern shared by TransitionTo and Shutdown.
func (f *FairyCharacter) stopAndUpdateState(state CharacterState) StateAnimator {
	// Phase 1: Safely extract and clear the current animator while holding the lock.
	f.mu.Lock()
	prev := f.currentAnimator
	f.currentAnimator = nil
	f.mu.Unlock()

	// Phase 2: Stop the previous animator outside the lock to avoid deadlock.
	// Animator goroutines may call Set* methods that acquire the same mutex.
	if prev != nil {
		prev.Stop()
	}

	// Phase 3: Update state, indicator, and create the new animator while holding the lock.
	f.mu.Lock()
	f.state = state
	f.indicator.FillColor = stateColor(state)
	f.indicator.Refresh()

	animator := f.createAnimatorForState(state)
	f.currentAnimator = animator
	f.mu.Unlock()

	return animator
}

// createAnimatorForState creates the appropriate animator for the given state.
func (f *FairyCharacter) createAnimatorForState(state CharacterState) StateAnimator {
	switch state {
	case StateIdle:
		return NewIdleAnimator(f.clock)
	case StateStarting:
		return NewStartupAnimator(f.clock, func() {
			go f.TransitionTo(StateIdle)
		})
	case StateWorking:
		return NewWorkingAnimator(f.clock)
	case StateNotifying:
		var seed int64
		_ = binary.Read(rand.Reader, binary.LittleEndian, &seed)
		return NewNotifyAnimator(f.clock, mathrand.New(mathrand.NewSource(seed))) // #nosec G404 -- animation dart positions are visual-only, not security-sensitive; seed is cryptographic
	case StateError:
		return NewErrorAnimator(f.clock)
	case StateShuttingDown:
		return NewShutdownAnimator(f.clock)
	default:
		return nil
	}
}

// CurrentState returns the current character state.
func (f *FairyCharacter) CurrentState() CharacterState {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.state
}

// SetClock injects a clock implementation (used for testing).
func (f *FairyCharacter) SetClock(c Clock) { f.clock = c }

// DisableRefresh replaces the refresh function with a no-op (used for testing).
func (f *FairyCharacter) DisableRefresh() { f.refreshFunc = func() {} }

// Close stops the current animator without changing the character state.
func (f *FairyCharacter) Close() {
	// Extract and clear the current animator while holding the lock.
	f.mu.Lock()
	prev := f.currentAnimator
	f.currentAnimator = nil
	f.mu.Unlock()

	// Stop the animator outside the lock to avoid deadlock.
	if prev != nil {
		prev.Stop()
	}
}

// Shutdown stops the current animator, transitions to StateShuttingDown, starts
// a ShutdownAnimator, and returns a channel that closes when the animation completes.
func (f *FairyCharacter) Shutdown() <-chan struct{} {
	// Stop any existing animator and transition to ShuttingDown.
	animator := f.stopAndUpdateState(StateShuttingDown)

	// Start the shutdown animator (guaranteed to be non-nil).
	animator.Start(f)

	// Return the completion channel from the shutdown animator.
	if shutdownAnimator, ok := animator.(interface{ Done() <-chan struct{} }); ok {
		return shutdownAnimator.Done()
	}

	// Fallback: create a closed channel if animator doesn't support Done().
	done := make(chan struct{})
	close(done)
	return done
}

// Widget returns the jar container as a canvas object.
func (f *FairyCharacter) Widget() fyne.CanvasObject { return f.container }

// SetPosition sets the fairy's position in normalized coordinates (0.0-1.0).
// Values are clamped to the valid range. Thread-safe.
func (f *FairyCharacter) SetPosition(x, y float64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.posX = clamp01(x)
	f.posY = clamp01(y)
	f.refreshFunc()
}

// Position returns the fairy's current normalized position. Thread-safe.
func (f *FairyCharacter) Position() (x, y float64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.posX, f.posY
}

// SetBodyColor changes the fill color of the body circle. Thread-safe.
func (f *FairyCharacter) SetBodyColor(c color.Color) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.bodyCircle.FillColor = c
	f.bodyCircle.Refresh()
	f.refreshFunc()
}

// SetGlowIntensity sets the glow intensity (0.0-1.0). Values are clamped.
// It also updates the alpha channel of all glow layers based on the intensity,
// using per-layer graduated base alphas (inner = brightest, outer = dimmest).
func (f *FairyCharacter) SetGlowIntensity(intensity float64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.glowIntensity = clamp01(intensity)
	for i, gl := range f.glowLayers {
		r, g, b, _ := gl.FillColor.RGBA()
		gl.FillColor = color.RGBA{
			R: uint8((r >> 8) & 0xFF),
			G: uint8((g >> 8) & 0xFF),
			B: uint8((b >> 8) & 0xFF),
			A: uint8(float64(glowBaseAlphas[i]) * f.glowIntensity),
		}
	}
	f.refreshFunc()
}

// GlowIntensity returns the current glow intensity. Thread-safe.
func (f *FairyCharacter) GlowIntensity() float64 {
	f.mu.Lock()
	defer f.mu.Unlock()
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

// newGlowCircle creates a new glow circle with the default idle color and given alpha.
func newGlowCircle(alpha uint8) *canvas.Circle {
	return canvas.NewCircle(color.RGBA{R: 0x00, G: 0x61, B: 0x00, A: alpha})
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

// lerpColor linearly interpolates between two RGBA colors.
func lerpColor(a, b color.RGBA, t float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(a.R) + t*(float64(b.R)-float64(a.R))),
		G: uint8(float64(a.G) + t*(float64(b.G)-float64(a.G))),
		B: uint8(float64(a.B) + t*(float64(b.B)-float64(a.B))),
		A: uint8(float64(a.A) + t*(float64(b.A)-float64(a.A))),
	}
}

// glowIntensity computes the glow intensity at time t using a sinusoidal
// breathing pattern. The result oscillates between min and max intensities
// with the specified period.
func glowIntensity(t, period, min, max float64) float64 {
	phase := 2 * math.Pi * t / period
	sinWave := math.Sin(phase)
	normalizedSin := (sinWave + 1.0) / 2.0
	return min + (max-min)*normalizedSin
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

	// Position jar SVGs to fill the entire container.
	l.positionJarLayers(size)

	// Position fairy circles (body + glow layers).
	l.positionFairyCircles(w, h)

	// Position hidden indicator (maintained for compatibility).
	l.fairy.indicator.Resize(fyne.NewSize(fairyIndicatorSize, fairyIndicatorSize))
}

// positionJarLayers positions the jar back and front SVG layers.
func (l *fairyJarLayout) positionJarLayers(size fyne.Size) {
	origin := fyne.NewPos(0, 0)

	l.fairy.jarBack.Resize(size)
	l.fairy.jarBack.Move(origin)

	l.fairy.jarFront.Resize(size)
	l.fairy.jarFront.Move(origin)
}

// positionFairyCircles positions the body circle and glow layers.
func (l *fairyJarLayout) positionFairyCircles(containerWidth, containerHeight float32) {
	// Body circle: 10% of jar width.
	bodyDiam := containerWidth * bodyRatio
	l.positionCircle(l.fairy.bodyCircle, bodyDiam, containerWidth, containerHeight)

	// Glow layers: linearly interpolated from body size to 25% of jar width.
	glowDiam := containerWidth * glowRatio
	for i, gl := range l.fairy.glowLayers {
		// Layer 0 is innermost (smallest), layer N-1 is outermost (largest).
		t := float32(i+1) / float32(fairyGlowLayerCount)
		d := bodyDiam + (glowDiam-bodyDiam)*t
		l.positionCircle(gl, d, containerWidth, containerHeight)
	}
}

// positionCircle positions and resizes a circle at the fairy's current position.
func (l *fairyJarLayout) positionCircle(circle *canvas.Circle, diameter, containerWidth, containerHeight float32) {
	circle.Resize(fyne.NewSize(diameter, diameter))
	circle.Move(fyne.NewPos(
		float32(l.fairy.posX)*containerWidth-diameter/2,
		float32(l.fairy.posY)*containerHeight-diameter/2,
	))
}
