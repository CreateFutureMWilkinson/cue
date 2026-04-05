package fairy

import (
	"context"
	"image/color"
	"sync"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
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

// StartupAnimator drives the fairy's startup animation.
type StartupAnimator struct {
	clock      character.Clock
	onComplete func()
	mu         sync.Mutex
	cancel     context.CancelFunc
	done       chan struct{}
}

// NewStartupAnimator creates a new StartupAnimator using the provided clock.
func NewStartupAnimator(clock character.Clock, onComplete func()) *StartupAnimator {
	return &StartupAnimator{
		clock:      clock,
		onComplete: onComplete,
	}
}

// Start begins the startup animation on the given fairy.
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

	ticker := time.NewTicker(character.AnimationFrameInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			elapsed := a.clock.Now().Sub(startTime).Seconds()
			progress := elapsed / StartupDurationSec

			if progress >= 1.0 {
				fairy.SetBodyColor(IdleBodyColor)
				fairy.SetGlowIntensity(StartupIdleGlowIntensity)
				a.onComplete()
				return
			}

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

// Stop cancels the animation goroutine and waits for it to exit.
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
func (a *StartupAnimator) State() character.CharacterState {
	return character.StateStarting
}
