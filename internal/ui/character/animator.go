package character

import "time"

// Ticker abstracts time.Ticker for testability.
type Ticker interface {
	Chan() <-chan time.Time
	Stop()
}

// Clock abstracts time operations for testability.
type Clock interface {
	Now() time.Time
	NewTicker(d time.Duration) Ticker
}

// Animation timing constants shared across all character implementations.
const (
	AnimationFPS           = 30                    // Target frames per second
	AnimationTickMs        = 1000 / AnimationFPS   // Milliseconds between animation frames
	AnimationFrameInterval = 16 * time.Millisecond // ~60fps frame interval for animators
)

// wallTicker wraps a standard time.Ticker to implement the Ticker interface.
type wallTicker struct {
	t *time.Ticker
}

func (wt *wallTicker) Chan() <-chan time.Time { return wt.t.C }
func (wt *wallTicker) Stop()                  { wt.t.Stop() }

// WallClock implements Clock using the real system clock.
type WallClock struct{}

// Now returns the current wall clock time.
func (WallClock) Now() time.Time { return time.Now() }

// NewTicker returns a new Ticker that wraps time.NewTicker.
func (WallClock) NewTicker(d time.Duration) Ticker {
	return &wallTicker{t: time.NewTicker(d)}
}
