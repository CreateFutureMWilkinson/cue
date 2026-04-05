package character

import (
	"context"
	"image/color"
	"sync"
	"time"
)

const (
	// StartupDurationSec is the total duration of the startup animation in seconds.
	StartupDurationSec = 1.5

	// DormantGlowIntensity is the glow intensity at the start of the startup animation.
	DormantGlowIntensity = 0.1

	// StartupIdleGlowIntensity is the target glow intensity at the end of startup.
	StartupIdleGlowIntensity = 0.5
)

var (
	// DormantColor is the fairy body color at the start of the startup animation (#004900).
	DormantColor = color.RGBA{R: 0x00, G: 0x49, B: 0x00, A: 0xFF}
)

// StartupAnimator drives the fairy's startup animation, transitioning from
// dormant state to idle state over StartupDurationSec seconds.
type StartupAnimator struct {
	clock      Clock
	onComplete func()
	mu         sync.Mutex
	cancel     context.CancelFunc
	done       chan struct{}
}

// NewStartupAnimator creates a new StartupAnimator using the provided clock.
// The onComplete callback is invoked once when the animation finishes.
func NewStartupAnimator(clock Clock, onComplete func()) *StartupAnimator {
	return &StartupAnimator{
		clock:      clock,
		onComplete: onComplete,
	}
}

// Start begins the startup animation on the given fairy. It sets the fairy to
// dormant state (position 0.5, 1.0; color #004900; glow 0.1) and starts a
// background goroutine that interpolates to idle state over 1.5 seconds.
// If already running, the previous animation is stopped first.
func (a *StartupAnimator) Start(fairy *FairyCharacter) {
	a.Stop()

	a.mu.Lock()
	defer a.mu.Unlock()

	a.initializeFairyState(fairy)

	startTime := a.clock.Now()
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	a.done = make(chan struct{})

	done := a.done

	go a.runAnimationLoop(ctx, fairy, startTime, done)
}

// runAnimationLoop drives the startup animation in a separate goroutine.
func (a *StartupAnimator) runAnimationLoop(ctx context.Context, fairy *FairyCharacter, startTime time.Time, done chan struct{}) {
	defer close(done)

	ticker := time.NewTicker(AnimationFrameInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			elapsed := a.clock.Now().Sub(startTime).Seconds()
			progress := elapsed / StartupDurationSec

			if progress >= 1.0 {
				// Set final state and fire completion callback.
				fairy.SetBodyColor(IdleBodyColor)
				fairy.SetGlowIntensity(StartupIdleGlowIntensity)
				a.onComplete()
				return
			}

			// Apply eased interpolation.
			eased := EaseInOut(clamp01(progress))
			fairy.SetBodyColor(lerpColor(DormantColor, IdleBodyColor, eased))
			fairy.SetGlowIntensity(DormantGlowIntensity + (StartupIdleGlowIntensity-DormantGlowIntensity)*eased)
		}
	}
}

// initializeFairyState sets the fairy to its dormant appearance and position.
func (a *StartupAnimator) initializeFairyState(fairy *FairyCharacter) {
	fairy.SetPosition(IdleOriginX, IdleOriginY)
	fairy.SetBodyColor(DormantColor)
	fairy.SetGlowIntensity(DormantGlowIntensity)
}

// Stop cancels the animation goroutine and waits for it to exit. It is safe
// to call without a prior Start, or to call multiple times.
func (a *StartupAnimator) Stop() {
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

// State returns StateStarting.
func (a *StartupAnimator) State() CharacterState {
	return StateStarting
}
