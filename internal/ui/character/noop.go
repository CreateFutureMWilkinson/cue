package character

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

// NoOpCharacter is a character implementation that does nothing visually.
type NoOpCharacter struct {
	state CharacterState
}

// NewNoOpCharacter creates a new NoOpCharacter in the Idle state.
func NewNoOpCharacter() *NoOpCharacter {
	return &NoOpCharacter{state: StateIdle}
}

func (c *NoOpCharacter) Name() string {
	return "none"
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
