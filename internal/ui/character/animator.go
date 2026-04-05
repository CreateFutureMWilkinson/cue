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
