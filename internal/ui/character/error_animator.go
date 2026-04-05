package character

import (
	"context"
	"image/color"
	"math"
	"sync"
	"time"
)

const (
	// ErrorVibrateAmplitude is the horizontal vibration amplitude.
	ErrorVibrateAmplitude = 0.04

	// ErrorPulseCycleSec is the period of the error glow pulse cycle.
	ErrorPulseCycleSec = 0.5

	// ErrorGlowMin is the minimum glow intensity during the error state.
	ErrorGlowMin = 0.4

	// ErrorGlowMax is the maximum glow intensity during the error state.
	ErrorGlowMax = 0.9
)

// ErrorVibrateFreqHz is the vibration frequency in Hz.
var ErrorVibrateFreqHz = 15.0

// ErrorBodyColor is the body color used in the error state (#00B800).
var ErrorBodyColor = color.RGBA{R: 0x00, G: 0xB8, B: 0x00, A: 0xFF}

// ErrorGlowIntensity computes the glow intensity at time t using a sinusoidal
// pulse. The result oscillates between ErrorGlowMin and ErrorGlowMax with a
// period of ErrorPulseCycleSec.
func ErrorGlowIntensity(t float64) float64 {
	return glowIntensity(t, ErrorPulseCycleSec, ErrorGlowMin, ErrorGlowMax)
}

// ErrorPosition computes the fairy position at time t. The x coordinate
// vibrates horizontally around 0.5 with amplitude ErrorVibrateAmplitude at
// ErrorVibrateFreqHz. The y coordinate is always 0.5.
func ErrorPosition(t float64) (float64, float64) {
	x := 0.5 + ErrorVibrateAmplitude*math.Sin(2*math.Pi*ErrorVibrateFreqHz*t)
	return x, 0.5
}

// ErrorAnimator drives the fairy's error state animation with rapid horizontal
// vibration and a pulsing glow.
type ErrorAnimator struct {
	clock  Clock
	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

// NewErrorAnimator creates a new ErrorAnimator using the provided clock.
func NewErrorAnimator(clock Clock) *ErrorAnimator {
	return &ErrorAnimator{clock: clock}
}

// Start begins the error animation on the given fairy. It immediately sets
// the body color to ErrorBodyColor, snaps position to center (0.5, 0.5),
// and sets glow to ErrorGlowIntensity(0). A background goroutine then
// vibrates and updates glow continuously. If already running, the previous
// animation is stopped first.
func (a *ErrorAnimator) Start(fairy *FairyCharacter) {
	a.Stop()

	a.mu.Lock()
	defer a.mu.Unlock()

	fairy.SetBodyColor(ErrorBodyColor)
	fairy.SetPosition(0.5, 0.5)
	fairy.SetGlowIntensity(ErrorGlowIntensity(0))

	startTime := a.clock.Now()
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	a.done = make(chan struct{})

	done := a.done

	go a.runAnimationLoop(ctx, fairy, startTime, done)
}

// Stop cancels the animation goroutine and waits for it to exit. It is safe
// to call without a prior Start, or to call multiple times.
func (a *ErrorAnimator) Stop() {
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

// State returns StateError.
func (a *ErrorAnimator) State() CharacterState {
	return StateError
}

// runAnimationLoop drives the error animation in a separate goroutine.
func (a *ErrorAnimator) runAnimationLoop(ctx context.Context, fairy *FairyCharacter, startTime time.Time, done chan struct{}) {
	defer close(done)

	ticker := time.NewTicker(AnimationFrameInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			elapsed := a.clock.Now().Sub(startTime).Seconds()

			fairy.SetGlowIntensity(ErrorGlowIntensity(elapsed))

			x, y := ErrorPosition(elapsed)
			fairy.SetPosition(x, y)
		}
	}
}
