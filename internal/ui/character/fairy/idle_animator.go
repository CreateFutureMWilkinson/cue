package fairy

import (
	"context"
	"sync"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
)

const (
	// Idle breathing animation parameters.
	IdleBreathCycleSec = 3.0 // Period of the breathing glow cycle in seconds
	IdleGlowMin        = 0.3 // Minimum glow intensity during breathing
	IdleGlowMax        = 0.8 // Maximum glow intensity during breathing
)

// IdleGlowIntensity computes the glow intensity at time t using a sinusoidal
// breathing pattern. The result oscillates between IdleGlowMin and IdleGlowMax
// with a period of IdleBreathCycleSec.
func IdleGlowIntensity(t float64) float64 {
	return glowIntensity(t, IdleBreathCycleSec, IdleGlowMin, IdleGlowMax)
}

// IdleAnimator drives the fairy's idle breathing animation.
type IdleAnimator struct {
	clock  character.Clock
	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

// NewIdleAnimator creates a new IdleAnimator using the provided clock.
func NewIdleAnimator(clock character.Clock) *IdleAnimator {
	return &IdleAnimator{clock: clock}
}

// Start begins the idle animation on the given fairy.
func (a *IdleAnimator) Start(fairy *FairyCharacter) {
	a.Stop()

	a.mu.Lock()
	defer a.mu.Unlock()

	a.initializeFairyState(fairy)

	startTime := a.clock.Now()
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	a.done = make(chan struct{})

	ticker := a.clock.NewTicker(time.Duration(character.AnimationTickMs) * time.Millisecond)
	done := a.done

	go a.runAnimationLoop(ctx, ticker, fairy, startTime, done)
}

// initializeFairyState sets the fairy to its idle appearance and position.
func (a *IdleAnimator) initializeFairyState(fairy *FairyCharacter) {
	fairy.SetPosition(IdleOriginX, IdleOriginY)
	fairy.SetBodyColor(IdleBodyColor)
	fairy.SetGlowIntensity(IdleGlowIntensity(0.0))
}

// runAnimationLoop drives the breathing animation in a separate goroutine.
func (a *IdleAnimator) runAnimationLoop(ctx context.Context, ticker character.Ticker, fairy *FairyCharacter, startTime time.Time, done chan struct{}) {
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
}

// Stop cancels the animation goroutine and waits for it to exit.
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
func (a *IdleAnimator) State() character.CharacterState {
	return character.StateIdle
}
