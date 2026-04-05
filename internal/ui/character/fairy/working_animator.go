package fairy

import (
	"context"
	"image/color"
	"math"
	"sync"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
)

const (
	// WorkingCircuitSec is the period of the primary drift circuit in seconds.
	WorkingCircuitSec = 4.0

	// WorkingDriftRadius is the radius of the primary drift circle in normalized units.
	WorkingDriftRadius = 0.35

	// WorkingEntryDurationSec is the duration of the entry transition in seconds.
	WorkingEntryDurationSec = 0.5
)

// WorkingBodyColor is the body color used after the entry transition completes.
var WorkingBodyColor = color.RGBA{R: 0x00, G: 0x92, B: 0x00, A: 0xFF}

// WorkingPosition computes the drift position at elapsed time t.
func WorkingPosition(t float64) (x, y float64) {
	primaryPhase := 2 * math.Pi * t / WorkingCircuitSec
	px := 0.5 + WorkingDriftRadius*math.Sin(primaryPhase)
	py := 0.5 + WorkingDriftRadius*math.Cos(primaryPhase)

	px += 0.08 * math.Sin(2*math.Pi*t/7.3)
	py += 0.08 * math.Cos(2*math.Pi*t/5.7)

	px += 0.03 * math.Sin(2*math.Pi*t/1.9)
	py += 0.03 * math.Cos(2*math.Pi*t/2.3)

	return clamp01(px), clamp01(py)
}

// WorkingAnimator drives the fairy's working state animation.
type WorkingAnimator struct {
	clock  character.Clock
	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

// NewWorkingAnimator creates a new WorkingAnimator using the provided clock.
func NewWorkingAnimator(clock character.Clock) *WorkingAnimator {
	return &WorkingAnimator{clock: clock}
}

// Start begins the working animation on the given fairy.
func (a *WorkingAnimator) Start(fairy *FairyCharacter) {
	a.Stop()

	a.mu.Lock()
	defer a.mu.Unlock()

	fairy.SetPosition(IdleOriginX, IdleOriginY)
	fairy.SetBodyColor(IdleBodyColor)
	fairy.SetGlowIntensity(IdleGlowIntensity(0.0))

	startTime := a.clock.Now()
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	a.done = make(chan struct{})

	done := a.done

	go a.runAnimationLoop(ctx, fairy, startTime, done)
}

// Stop cancels the animation goroutine and waits for it to exit.
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
func (a *WorkingAnimator) State() character.CharacterState {
	return character.StateWorking
}

// runAnimationLoop drives the working animation in a separate goroutine.
func (a *WorkingAnimator) runAnimationLoop(ctx context.Context, fairy *FairyCharacter, startTime time.Time, done chan struct{}) {
	defer close(done)

	ticker := time.NewTicker(character.AnimationFrameInterval)
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

// updateFrame updates the fairy's position, color, and glow for the current elapsed time.
func (a *WorkingAnimator) updateFrame(fairy *FairyCharacter, elapsed float64) {
	fairy.SetGlowIntensity(IdleGlowIntensity(elapsed))

	if elapsed < WorkingEntryDurationSec {
		t := elapsed / WorkingEntryDurationSec
		a.interpolateEntry(fairy, elapsed, t)
	} else {
		driftX, driftY := WorkingPosition(elapsed)
		fairy.SetPosition(driftX, driftY)
		fairy.SetBodyColor(WorkingBodyColor)
	}
}

// interpolateEntry handles the entry transition.
func (a *WorkingAnimator) interpolateEntry(fairy *FairyCharacter, elapsed, t float64) {
	driftX, driftY := WorkingPosition(elapsed)

	x := IdleOriginX + t*(driftX-IdleOriginX)
	y := IdleOriginY + t*(driftY-IdleOriginY)
	fairy.SetPosition(x, y)

	fairy.SetBodyColor(lerpColor(IdleBodyColor, WorkingBodyColor, t))
}
