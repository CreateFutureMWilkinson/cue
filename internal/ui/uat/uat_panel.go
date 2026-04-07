package uat

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
)

// UATPanel is the character UAT control panel providing a character dropdown,
// state trigger buttons, and diagnostic labels.
type UATPanel struct {
	root            fyne.CanvasObject
	characterSelect *widget.Select
	stateButtons    []*widget.Button
	stateLabel      *widget.Label
	charLabel       *widget.Label
	currentChar     character.Character
	onCharChanged   func(character.Character)
}

// NewUATPanel creates a UAT control panel with a character dropdown populated
// from the character registry, 6 state trigger buttons in a 2x3 grid, a state
// label, and a character label. onCharChanged is called when the user selects
// a different character from the dropdown.
func NewUATPanel(onCharChanged func(character.Character)) *UATPanel {
	p := &UATPanel{
		onCharChanged: onCharChanged,
	}

	names := character.Available()
	p.characterSelect = widget.NewSelect(names, func(_ string) {})

	stateNames := []string{"Idle", "Starting", "Working", "Notifying", "Error", "Shutdown"}
	p.stateButtons = make([]*widget.Button, len(stateNames))
	buttonObjects := make([]fyne.CanvasObject, len(stateNames))
	for i, name := range stateNames {
		p.stateButtons[i] = widget.NewButton(name, func() {})
		buttonObjects[i] = p.stateButtons[i]
	}

	p.stateLabel = widget.NewLabel("Current State: \u2014")
	p.charLabel = widget.NewLabel("Character: \u2014")

	p.root = container.NewVBox(
		widget.NewLabel("Character UAT Controls"),
		widget.NewLabel("Character:"),
		p.characterSelect,
		widget.NewLabel("State Triggers:"),
		container.NewGridWithColumns(2, buttonObjects...),
		widget.NewSeparator(),
		p.stateLabel,
		p.charLabel,
	)

	return p
}

// Container returns the root canvas object of the UAT panel.
func (p *UATPanel) Container() fyne.CanvasObject {
	return p.root
}
