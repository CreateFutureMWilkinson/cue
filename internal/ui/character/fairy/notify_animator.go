package fairy

import (
	"context"
	"image/color"
	"math/rand"
	"sync"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
)

const (
	// NotifyDartIntervalSec is how often the fairy darts to a new random position.
	NotifyDartIntervalSec = 0.5

	// NotifyBreathCycleSec is the period of the notify glow breathing cycle.
	NotifyBreathCycleSec = 1.5

	// NotifyGlowMin is the minimum glow intensity during the notify state.
	NotifyGlowMin = 0.5

	// NotifyGlowMax is the maximum glow intensity during the notify state.
	NotifyGlowMax = 0.9
)

// NotifyBodyColor is the body color used in the notify state (#00FF88).
var NotifyBodyColor = color.RGBA{R: 0x00, G: 0xFF, B: 0x88, A: 0xFF}

// NotifyGlowIntensity computes the glow intensity at time t.
func NotifyGlowIntensity(t float64) float64 {
	return glowIntensity(t, NotifyBreathCycleSec, NotifyGlowMin, NotifyGlowMax)
}

// NotifyAnimator drives the fairy's notify state animation.
type NotifyAnimator struct {
	clock  character.Clock
	rng    *rand.Rand
	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

// NewNotifyAnimator creates a new NotifyAnimator using the provided clock and RNG.
func NewNotifyAnimator(clock character.Clock, rng *rand.Rand) *NotifyAnimator {
	return &NotifyAnimator{clock: clock, rng: rng}
}

// Start begins the notify animation on the given fairy.
func (a *NotifyAnimator) Start(fairy *FairyCharacter) {
	a.Stop()

	a.mu.Lock()
	defer a.mu.Unlock()

	fairy.SetBodyColor(NotifyBodyColor)
	fairy.SetGlowIntensity(NotifyGlowMax)
	a.dart(fairy)

	startTime := a.clock.Now()
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	a.done = make(chan struct{})

	done := a.done

	go a.runAnimationLoop(ctx, fairy, startTime, done)
}

// Stop cancels the animation goroutine and waits for it to exit.
func (a *NotifyAnimator) Stop() {
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

// State returns StateNotifying.
func (a *NotifyAnimator) State() character.CharacterState {
	return character.StateNotifying
}

// dart moves the fairy to a new random position using the RNG.
func (a *NotifyAnimator) dart(fairy *FairyCharacter) {
	x := a.rng.Float64()
	y := a.rng.Float64()
	fairy.SetPosition(x, y)
}

// runAnimationLoop drives the notify animation in a separate goroutine.
func (a *NotifyAnimator) runAnimationLoop(ctx context.Context, fairy *FairyCharacter, startTime time.Time, done chan struct{}) {
	defer close(done)

	ticker := time.NewTicker(character.AnimationFrameInterval)
	defer ticker.Stop()

	lastDartCount := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			elapsed := a.clock.Now().Sub(startTime).Seconds()

			fairy.SetGlowIntensity(NotifyGlowIntensity(elapsed))

			dartCount := int(elapsed / NotifyDartIntervalSec)
			if dartCount > lastDartCount {
				a.dart(fairy)
				lastDartCount = dartCount
			}
		}
	}
}
