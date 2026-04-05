package character

import (
	"context"
	"image/color"
	"math"
	"sync"
	"time"
)

const (
	// WorkingCircuitSec is the period of the primary drift circuit in seconds.
	WorkingCircuitSec = 4.0

	// WorkingDriftRadius is the radius of the primary drift circle in normalized units.
	WorkingDriftRadius = 0.35

	// WorkingEntryDurationSec is the duration of the entry transition in seconds.
	WorkingEntryDurationSec = 0.5

	// workingFrameInterval is how often the working animator goroutine checks for updates.
	workingFrameInterval = time.Millisecond
)

// WorkingBodyColor is the body color used after the entry transition completes.
var WorkingBodyColor = color.RGBA{R: 0x00, G: 0x92, B: 0x00, A: 0xFF}

// idleOriginX and idleOriginY are the idle position coordinates.
const (
	idleOriginX = 0.5
	idleOriginY = 1.0
)

// idleBodyColor is the idle state body color (#006100).
var idleBodyColor = color.RGBA{R: 0x00, G: 0x61, B: 0x00, A: 0xFF}

// WorkingPosition computes the drift position at elapsed time t using layered
// sinusoidal motion. The result is clamped to [0.0, 1.0].
func WorkingPosition(t float64) (x, y float64) {
	// Primary circuit: 4.0s period, radius 0.35, centered at 0.5.
	primaryPhase := 2 * math.Pi * t / WorkingCircuitSec
	px := 0.5 + WorkingDriftRadius*math.Cos(primaryPhase)
	py := 0.5 + WorkingDriftRadius*math.Sin(primaryPhase)

	// Secondary noise: 7.3s and 5.7s periods, amplitude 0.08.
	px += 0.08 * math.Sin(2*math.Pi*t/7.3)
	py += 0.08 * math.Cos(2*math.Pi*t/5.7)

	// Tertiary wobble: 1.9s and 2.3s periods, amplitude 0.03.
	px += 0.03 * math.Sin(2*math.Pi*t/1.9)
	py += 0.03 * math.Cos(2*math.Pi*t/2.3)

	return clamp01(px), clamp01(py)
}

// WorkingAnimator drives the fairy's working state animation with an entry
// transition from idle position/color to working drift.
type WorkingAnimator struct {
	clock  Clock
	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

// NewWorkingAnimator creates a new WorkingAnimator using the provided clock.
func NewWorkingAnimator(clock Clock) *WorkingAnimator {
	return &WorkingAnimator{clock: clock}
}

// Start begins the working animation on the given fairy. It sets the initial
// idle position and color, then starts a goroutine that interpolates through
// the entry transition and into drift mode. If already running, the previous
// animation is stopped first.
func (a *WorkingAnimator) Start(fairy *FairyCharacter) {
	a.Stop()

	a.mu.Lock()
	defer a.mu.Unlock()

	// Set initial state: idle position and color.
	fairy.SetPosition(idleOriginX, idleOriginY)
	fairy.SetBodyColor(idleBodyColor)
	fairy.SetGlowIntensity(IdleGlowIntensity(0.0))

	startTime := a.clock.Now()
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	a.done = make(chan struct{})

	done := a.done

	go a.runAnimationLoop(ctx, fairy, startTime, done)
}

// Stop cancels the animation goroutine and waits for it to exit. It is safe
// to call without a prior Start, or to call multiple times.
func (a *WorkingAnimator) Stop() {
	a.mu.Lock()
	cancel := a.cancel
	done := a.done
	a.cancel = nil
	a.done = nil
	a.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

// State returns StateWorking.
func (a *WorkingAnimator) State() CharacterState {
	return StateWorking
}

// runAnimationLoop drives the working animation in a separate goroutine.
// It uses a real-time ticker for frame pacing and reads elapsed time from
// the clock interface, allowing mock clocks to control time progression.
func (a *WorkingAnimator) runAnimationLoop(ctx context.Context, fairy *FairyCharacter, startTime time.Time, done chan struct{}) {
	defer close(done)

	ticker := time.NewTicker(workingFrameInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			elapsed := a.clock.Now().Sub(startTime).Seconds()
			a.updateFrame(fairy, elapsed)
		}
	}
}

// updateFrame updates the fairy's position, color, and glow for the current
// elapsed time.
func (a *WorkingAnimator) updateFrame(fairy *FairyCharacter, elapsed float64) {
	// Glow always uses the idle breathing cycle.
	fairy.SetGlowIntensity(IdleGlowIntensity(elapsed))

	if elapsed < WorkingEntryDurationSec {
		// Entry transition: interpolate from idle to working.
		t := elapsed / WorkingEntryDurationSec
		a.interpolateEntry(fairy, elapsed, t)
	} else {
		// Pure drift mode.
		driftX, driftY := WorkingPosition(elapsed)
		fairy.SetPosition(driftX, driftY)
		fairy.SetBodyColor(WorkingBodyColor)
	}
}

// interpolateEntry handles the entry transition, interpolating position and
// color between idle and working states.
func (a *WorkingAnimator) interpolateEntry(fairy *FairyCharacter, elapsed, t float64) {
	driftX, driftY := WorkingPosition(elapsed)

	// Linearly interpolate position.
	x := idleOriginX + t*(driftX-idleOriginX)
	y := idleOriginY + t*(driftY-idleOriginY)
	fairy.SetPosition(x, y)

	// Linearly interpolate color channels.
	fairy.SetBodyColor(lerpColor(idleBodyColor, WorkingBodyColor, t))
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
