package fairy

import (
	"crypto/rand"
	"encoding/binary"
	"image/color"
	mathrand "math/rand"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
)

const (
	// fairyIndicatorSize is the diameter of the state indicator circle.
	fairyIndicatorSize = 40

	// fairyGlowLayerCount is the number of concentric glow layers.
	fairyGlowLayerCount = 8

	// bodyRatio is the body circle diameter as a fraction of jar width.
	bodyRatio = 0.05

	// glowRatio is the outermost glow circle diameter as a fraction of jar width.
	glowRatio = 0.25
)

// stateAnimator defines the interface for character state animators.
// This is fairy-local and unexported — each character owns its animators.
type stateAnimator interface {
	Start(fairy *FairyCharacter)
	Stop()
	State() character.CharacterState
}

// FairyCharacter is a character implementation that renders a fairy inside a
// glass jar. The jar is composed of PNG back/front layers with the fairy's
// body and glow circles sandwiched between them.
type FairyCharacter struct {
	state     character.CharacterState
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
	clock character.Clock

	// Current running animator (nil if none).
	currentAnimator stateAnimator

	// layoutRefreshCount tracks how many times the layout has called Refresh on jar images.
	layoutRefreshCount int

	// Mutex for thread-safe transitions.
	mu sync.Mutex

	// refreshFunc is called after visual updates; wired to fyne.Do for thread-safety.
	// Can be replaced via DisableRefresh or SetRefreshHook in tests.
	refreshFunc func()
}

// NewFairyCharacter creates a new FairyCharacter in the Starting state with jar
// rendering. The fairy starts at the bottom-center position (0.5, 1.0) with
// bright green body color (#00FF00). Jar images are loaded from embedded PNGs.
func NewFairyCharacter() *FairyCharacter {
	// State indicator (hidden but maintained for TransitionTo method compatibility).
	indicator := canvas.NewCircle(stateColor(character.StateStarting))
	indicator.Resize(fyne.NewSize(fairyIndicatorSize, fairyIndicatorSize))
	indicator.Hide()

	// Jar PNG layers from embedded assets.
	jarBack := canvas.NewImageFromResource(fyne.NewStaticResource("jar_back.png", jarBackPNG))
	jarBack.FillMode = canvas.ImageFillContain
	jarFront := canvas.NewImageFromResource(fyne.NewStaticResource("jar_front.png", jarFrontPNG))
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
		state:         character.StateStarting,
		indicator:     indicator,
		jarBack:       jarBack,
		jarFront:      jarFront,
		bodyCircle:    bodyCircle,
		glowLayers:    glowLayers,
		posX:          0.5,
		posY:          1.0,
		glowIntensity: 0.0,
		clock:         character.WallClock{},
	}

	// Build the container with custom layout for proportional sizing.
	objects := make([]fyne.CanvasObject, 0, 3+fairyGlowLayerCount)
	objects = append(objects, jarBack)
	for _, gl := range glowLayers {
		objects = append(objects, gl)
	}
	objects = append(objects, bodyCircle)
	objects = append(objects, jarFront)
	objects = append(objects, indicator)

	f.container = container.New(&fairyJarLayout{fairy: f}, objects...)
	f.refreshFunc = func() {
		if fyne.CurrentApp() != nil {
			fyne.Do(func() { f.container.Refresh() })
		}
	}

	return f
}

// Name returns the character name.
func (f *FairyCharacter) Name() string { return "fairy" }

// TransitionTo changes the character's state, stops the current animator,
// creates a new animator for the target state, and starts it.
func (f *FairyCharacter) TransitionTo(state character.CharacterState) {
	animator := f.stopAndUpdateState(state)
	if animator != nil {
		animator.Start(f)
	}
}

// stopAndUpdateState is a helper that stops the current animator (if any),
// updates the character state and indicator, and creates a new animator for the target state.
func (f *FairyCharacter) stopAndUpdateState(state character.CharacterState) stateAnimator {
	f.mu.Lock()
	prev := f.currentAnimator
	f.currentAnimator = nil
	f.mu.Unlock()

	if prev != nil {
		prev.Stop()
	}

	f.mu.Lock()
	f.state = state
	f.indicator.FillColor = stateColor(state)
	f.refreshFunc()

	animator := f.createAnimatorForState(state)
	f.currentAnimator = animator
	f.mu.Unlock()

	return animator
}

// createAnimatorForState creates the appropriate animator for the given state.
func (f *FairyCharacter) createAnimatorForState(state character.CharacterState) stateAnimator {
	switch state {
	case character.StateIdle:
		return NewIdleAnimator(f.clock)
	case character.StateStarting:
		return NewStartupAnimator(f.clock, func() {
			go f.TransitionTo(character.StateIdle)
		})
	case character.StateWorking:
		return NewWorkingAnimator(f.clock)
	case character.StateNotifying:
		var seed int64
		_ = binary.Read(rand.Reader, binary.LittleEndian, &seed)
		return NewNotifyAnimator(f.clock, mathrand.New(mathrand.NewSource(seed))) // #nosec G404 -- animation dart positions are visual-only
	case character.StateError:
		return NewErrorAnimator(f.clock)
	case character.StateShuttingDown:
		return NewShutdownAnimator(f.clock)
	default:
		return nil
	}
}

// CurrentState returns the current character state.
func (f *FairyCharacter) CurrentState() character.CharacterState {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.state
}

// SetClock injects a clock implementation (used for testing).
func (f *FairyCharacter) SetClock(c character.Clock) { f.clock = c }

// DisableRefresh replaces the refresh function with a no-op (used for testing).
func (f *FairyCharacter) DisableRefresh() { f.refreshFunc = func() {} }

// SetRefreshHook replaces the refresh function with a caller-provided function for test observability.
func (f *FairyCharacter) SetRefreshHook(fn func()) { f.refreshFunc = fn }

// IsNoopRefresh reports whether the current refreshFunc is the default no-op.
func (f *FairyCharacter) IsNoopRefresh() bool {
	return false // stub: not implemented
}

// Close stops the current animator without changing the character state.
func (f *FairyCharacter) Close() {
	f.mu.Lock()
	prev := f.currentAnimator
	f.currentAnimator = nil
	f.mu.Unlock()

	if prev != nil {
		prev.Stop()
	}
}

// Shutdown stops the current animator, transitions to StateShuttingDown, starts
// a ShutdownAnimator, and returns a channel that closes when the animation completes.
func (f *FairyCharacter) Shutdown() <-chan struct{} {
	animator := f.stopAndUpdateState(character.StateShuttingDown)
	animator.Start(f)

	if shutdownAnimator, ok := animator.(interface{ Done() <-chan struct{} }); ok {
		return shutdownAnimator.Done()
	}

	done := make(chan struct{})
	close(done)
	return done
}

// Widget returns the jar container as a canvas object.
func (f *FairyCharacter) Widget() fyne.CanvasObject { return f.container }

// SetPosition sets the fairy's position in normalized coordinates (0.0-1.0).
func (f *FairyCharacter) SetPosition(x, y float64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.posX = clamp01(x)
	f.posY = clamp01(y)
	f.refreshFunc()
}

// Position returns the fairy's current normalized position.
func (f *FairyCharacter) Position() (x, y float64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.posX, f.posY
}

// SetBodyColor changes the fill color of the body circle.
func (f *FairyCharacter) SetBodyColor(c color.Color) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.bodyCircle.FillColor = c
	f.refreshFunc()
}

// SetGlowIntensity sets the glow intensity (0.0-1.0).
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

// GlowIntensity returns the current glow intensity.
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

// JarBack returns the jar back image layer.
func (f *FairyCharacter) JarBack() *canvas.Image { return f.jarBack }

// JarFront returns the jar front image layer.
func (f *FairyCharacter) JarFront() *canvas.Image { return f.jarFront }

// LayoutRefreshCount returns how many times the layout has called Refresh on jar images.
func (f *FairyCharacter) LayoutRefreshCount() int { return f.layoutRefreshCount }

// newGlowCircle creates a new glow circle with the default glow color and given alpha.
func newGlowCircle(alpha uint8) *canvas.Circle {
	return canvas.NewCircle(color.RGBA{R: 0x00, G: 0x61, B: 0x00, A: alpha})
}
