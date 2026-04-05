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

// StateAnimator defines the interface for character state animators.
type StateAnimator interface {
	Start(fairy *FairyCharacter)
	Stop()
	State() CharacterState
}

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
