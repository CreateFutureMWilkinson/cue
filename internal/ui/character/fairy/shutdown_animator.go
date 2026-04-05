package fairy

import (
	"context"
	"image/color"
	"sync"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
)

const (
	// ShutdownDurationSec is the total duration of the shutdown animation in seconds.
	ShutdownDurationSec = 1.5

	// ShutdownDormantGlowIntensity is the target glow intensity at the end of shutdown.
	ShutdownDormantGlowIntensity = 0.15
)

// ShutdownAnimator drives the fairy's shutdown animation.
type ShutdownAnimator struct {
	clock  character.Clock
	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

// NewShutdownAnimator creates a new ShutdownAnimator using the provided clock.
func NewShutdownAnimator(clock character.Clock) *ShutdownAnimator {
	return &ShutdownAnimator{
		clock: clock,
	}
}

// Start begins the shutdown animation on the given fairy.
func (a *ShutdownAnimator) Start(fairy *FairyCharacter) {
	a.mu.Lock()
	defer a.mu.Unlock()

	startX, startY, startColor, startGlow := a.captureFairyState(fairy)

	startTime := a.clock.Now()
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	a.done = make(chan struct{})

	done := a.done

	go a.runAnimationLoop(ctx, fairy, startTime, startX, startY, startColor, startGlow, done)
}

// runAnimationLoop drives the shutdown animation in a separate goroutine.
func (a *ShutdownAnimator) runAnimationLoop(
	ctx context.Context,
	fairy *FairyCharacter,
	startTime time.Time,
	startX, startY float64,
	startColor color.RGBA,
	startGlow float64,
	done chan struct{},
) {
	defer close(done)

	const (
		targetX    = 0.5
		targetY    = 1.0
		targetGlow = ShutdownDormantGlowIntensity
	)
	targetColor := DormantColor

	ticker := time.NewTicker(character.AnimationFrameInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			elapsed := a.clock.Now().Sub(startTime).Seconds()
			progress := elapsed / ShutdownDurationSec

			if progress >= 1.0 {
				fairy.SetPosition(targetX, targetY)
				fairy.SetBodyColor(targetColor)
				fairy.SetGlowIntensity(targetGlow)
				return
			}

			eased := EaseInOut(clamp01(progress))

			x := startX + eased*(targetX-startX)
			y := startY + eased*(targetY-startY)
			fairy.SetPosition(x, y)

			fairy.SetBodyColor(lerpColor(startColor, targetColor, eased))
			fairy.SetGlowIntensity(startGlow + (targetGlow-startGlow)*eased)
		}
	}
}

// captureFairyState extracts the current position, body color, and glow intensity from the fairy.
func (a *ShutdownAnimator) captureFairyState(fairy *FairyCharacter) (float64, float64, color.RGBA, float64) {
	x, y := fairy.Position()
	glow := fairy.GlowIntensity()

	fc := fairy.BodyCircle().FillColor
	r, g, b, al := fc.RGBA()
	bodyColor := color.RGBA{
		R: uint8((r >> 8) & 0xFF),
		G: uint8((g >> 8) & 0xFF),
		B: uint8((b >> 8) & 0xFF),
		A: uint8((al >> 8) & 0xFF),
	}

	return x, y, bodyColor, glow
}

// Stop cancels the animation goroutine and waits for it to exit.
func (a *ShutdownAnimator) Stop() {
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

// State returns StateShuttingDown.
func (a *ShutdownAnimator) State() character.CharacterState {
	return character.StateShuttingDown
}

// Done returns a channel that is closed when the animation completes.
func (a *ShutdownAnimator) Done() <-chan struct{} {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.done
}
