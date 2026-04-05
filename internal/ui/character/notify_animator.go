package character

import (
	"context"
	"image/color"
	"math/rand"
	"sync"
	"time"
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

// NotifyBodyColor is the body color used in the notify state (#00C300).
var NotifyBodyColor = color.RGBA{R: 0x00, G: 0xC3, B: 0x00, A: 0xFF}

// NotifyGlowIntensity computes the glow intensity at time t using a sinusoidal
// breathing pattern. The result oscillates between NotifyGlowMin and NotifyGlowMax
// with a period of NotifyBreathCycleSec.
func NotifyGlowIntensity(t float64) float64 {
	return glowIntensity(t, NotifyBreathCycleSec, NotifyGlowMin, NotifyGlowMax)
}

// NotifyAnimator drives the fairy's notify state animation with rapid darting
// and a fast breathing glow.
type NotifyAnimator struct {
	clock  Clock
	rng    *rand.Rand
	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

// NewNotifyAnimator creates a new NotifyAnimator using the provided clock and RNG.
func NewNotifyAnimator(clock Clock, rng *rand.Rand) *NotifyAnimator {
	return &NotifyAnimator{clock: clock, rng: rng}
}

// Start begins the notify animation on the given fairy. It immediately sets
// the body color to NotifyBodyColor, glow to NotifyGlowMax, and darts to a
// random position. A background goroutine then darts every 0.5s and updates
// glow continuously. If already running, the previous animation is stopped first.
func (a *NotifyAnimator) Start(fairy *FairyCharacter) {
	a.Stop()

	a.mu.Lock()
	defer a.mu.Unlock()

	// Immediate state changes.
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

// Stop cancels the animation goroutine and waits for it to exit. It is safe
// to call without a prior Start, or to call multiple times.
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
func (a *NotifyAnimator) State() CharacterState {
	return StateNotifying
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

	ticker := time.NewTicker(AnimationFrameInterval)
	defer ticker.Stop()

	lastDartCount := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			elapsed := a.clock.Now().Sub(startTime).Seconds()

			// Update glow continuously.
			fairy.SetGlowIntensity(NotifyGlowIntensity(elapsed))

			// Dart when a new 0.5s interval is reached.
			dartCount := int(elapsed / NotifyDartIntervalSec)
			if dartCount > lastDartCount {
				a.dart(fairy)
				lastDartCount = dartCount
			}
		}
	}
}
