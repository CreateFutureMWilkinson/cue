package character

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

// NoOpCharacter is a character implementation that does nothing visually.
type NoOpCharacter struct {
	state CharacterState
}

// NewNoOpCharacter creates a new NoOpCharacter in the Starting state.
func NewNoOpCharacter() *NoOpCharacter {
	return &NoOpCharacter{state: StateStarting}
}

func (c *NoOpCharacter) Name() string {
	return NoneCharacterName
}

func (c *NoOpCharacter) TransitionTo(state CharacterState) {
	c.state = state
}

func (c *NoOpCharacter) CurrentState() CharacterState {
	return c.state
}

func (c *NoOpCharacter) Widget() fyne.CanvasObject {
	return container.NewWithoutLayout()
}

// Close is a no-op for the NoOpCharacter.
func (c *NoOpCharacter) Close() {}

// Shutdown transitions to StateShuttingDown and returns a pre-closed
// channel — the noop character has no animation to drain.
func (c *NoOpCharacter) Shutdown() <-chan struct{} {
	c.TransitionTo(StateShuttingDown)
	done := make(chan struct{})
	close(done)
	return done
}
