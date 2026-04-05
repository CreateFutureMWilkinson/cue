package character

import (
	"context"
	"image/color"
	"math"
	"sync"
	"time"
)

const (
	// IdleBreathCycleSec is the period of the breathing glow cycle in seconds.
	IdleBreathCycleSec = 3.0

	// IdleGlowMin is the minimum glow intensity during the breathing cycle.
	IdleGlowMin = 0.3

	// IdleGlowMax is the maximum glow intensity during the breathing cycle.
	IdleGlowMax = 0.8

	// AnimationFPS is the target frames per second for animations.
	AnimationFPS = 30

	// AnimationTickMs is the milliseconds between animation frames.
	AnimationTickMs = 1000 / AnimationFPS
)

// IdleGlowIntensity computes the glow intensity at time t using a sinusoidal
// breathing pattern. The result oscillates between IdleGlowMin and IdleGlowMax
// with a period of IdleBreathCycleSec.
func IdleGlowIntensity(t float64) float64 {
	normalized := math.Sin(2 * math.Pi * t / IdleBreathCycleSec)
	return IdleGlowMin + (IdleGlowMax-IdleGlowMin)*(normalized+1.0)/2.0
}

// IdleAnimator drives the fairy's idle breathing animation.
type IdleAnimator struct {
	clock  Clock
	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

// NewIdleAnimator creates a new IdleAnimator using the provided clock.
func NewIdleAnimator(clock Clock) *IdleAnimator {
	return &IdleAnimator{clock: clock}
}

// Start begins the idle animation on the given fairy. It sets the fairy's
// position to bottom-center (0.5, 1.0), body color to dark green (#006100),
// and glow intensity to the initial value. A background goroutine updates
// glow intensity each tick. If already running, the previous animation is
// stopped first.
func (a *IdleAnimator) Start(fairy *FairyCharacter) {
	a.Stop()

	a.mu.Lock()
	defer a.mu.Unlock()

	// Set initial fairy state.
	fairy.SetPosition(0.5, 1.0)
	fairy.SetBodyColor(color.RGBA{R: 0x00, G: 0x61, B: 0x00, A: 0xFF})

	startTime := a.clock.Now()
	fairy.SetGlowIntensity(IdleGlowIntensity(0.0))

	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	a.done = make(chan struct{})

	ticker := a.clock.NewTicker(time.Duration(AnimationTickMs) * time.Millisecond)
	done := a.done

	go func() {
		defer close(done)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.Chan():
				elapsed := a.clock.Now().Sub(startTime).Seconds()
				fairy.SetGlowIntensity(IdleGlowIntensity(elapsed))
			}
		}
	}()
}

// Stop cancels the animation goroutine and waits for it to exit. It is safe
// to call without a prior Start, or to call multiple times.
func (a *IdleAnimator) Stop() {
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

// State returns StateIdle.
func (a *IdleAnimator) State() CharacterState {
	return StateIdle
}
