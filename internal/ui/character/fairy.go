package character

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

// FairyCharacter is a character implementation that uses colored circles to
// represent each state visually.
type FairyCharacter struct {
	state     CharacterState
	container *fyne.Container
	indicator *canvas.Circle
}

// NewFairyCharacter creates a new FairyCharacter in the Idle state.
func NewFairyCharacter() *FairyCharacter {
	indicator := canvas.NewCircle(stateColor(StateIdle))
	indicator.Resize(fyne.NewSize(40, 40))
	c := container.NewWithoutLayout(indicator)
	return &FairyCharacter{
		state:     StateIdle,
		container: c,
		indicator: indicator,
	}
}

func (f *FairyCharacter) Name() string { return "fairy" }

func (f *FairyCharacter) TransitionTo(state CharacterState) {
	f.state = state
	f.indicator.FillColor = stateColor(state)
	f.indicator.Refresh()
}

func (f *FairyCharacter) CurrentState() CharacterState { return f.state }

func (f *FairyCharacter) Widget() fyne.CanvasObject { return f.container }

func stateColor(s CharacterState) color.Color {
	switch s {
	case StateIdle:
		return color.RGBA{R: 200, G: 200, B: 255, A: 255}
	case StateStarting:
		return color.RGBA{R: 255, G: 255, B: 200, A: 255}
	case StateWorking:
		return color.RGBA{R: 200, G: 255, B: 200, A: 255}
	case StateNotifying:
		return color.RGBA{R: 255, G: 200, B: 100, A: 255}
	case StateError:
		return color.RGBA{R: 255, G: 100, B: 100, A: 255}
	case StateShuttingDown:
		return color.RGBA{R: 150, G: 150, B: 150, A: 255}
	default:
		return color.RGBA{R: 200, G: 200, B: 255, A: 255}
	}
}
