package character

import (
	"context"
	"math"
	"sync"
	"time"
)

const (
	// Idle breathing animation parameters.
	IdleBreathCycleSec = 3.0 // Period of the breathing glow cycle in seconds
	IdleGlowMin        = 0.3 // Minimum glow intensity during breathing
	IdleGlowMax        = 0.8 // Maximum glow intensity during breathing

	// Animation timing parameters.
	AnimationFPS    = 30                  // Target frames per second
	AnimationTickMs = 1000 / AnimationFPS // Milliseconds between animation frames
)

// IdleGlowIntensity computes the glow intensity at time t using a sinusoidal
// breathing pattern. The result oscillates between IdleGlowMin and IdleGlowMax
// with a period of IdleBreathCycleSec.
func IdleGlowIntensity(t float64) float64 {
	// sin(-1 to +1) -> normalized(0 to 1) -> intensity(min to max)
	phase := 2 * math.Pi * t / IdleBreathCycleSec
	sinWave := math.Sin(phase)
	normalizedSin := (sinWave + 1.0) / 2.0
	return IdleGlowMin + (IdleGlowMax-IdleGlowMin)*normalizedSin
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

	a.initializeFairyState(fairy)

	startTime := a.clock.Now()
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	a.done = make(chan struct{})

	ticker := a.clock.NewTicker(time.Duration(AnimationTickMs) * time.Millisecond)
	done := a.done

	go a.runAnimationLoop(ctx, ticker, fairy, startTime, done)
}

// initializeFairyState sets the fairy to its idle appearance and position.
func (a *IdleAnimator) initializeFairyState(fairy *FairyCharacter) {
	fairy.SetPosition(IdleOriginX, IdleOriginY)    // Bottom-center
	fairy.SetBodyColor(IdleBodyColor)              // Dark green
	fairy.SetGlowIntensity(IdleGlowIntensity(0.0)) // Initial glow
}

// runAnimationLoop drives the breathing animation in a separate goroutine.
func (a *IdleAnimator) runAnimationLoop(ctx context.Context, ticker Ticker, fairy *FairyCharacter, startTime time.Time, done chan struct{}) {
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
