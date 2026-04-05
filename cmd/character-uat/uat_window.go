package characteruat

import (
	"fmt"
	"sort"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
)

const (
	// windowWidth is the default UAT window width.
	windowWidth = 800
	// windowHeight is the default UAT window height.
	windowHeight = 600
	// fpsUpdateInterval is how often the FPS label refreshes.
	fpsUpdateInterval = 500 * time.Millisecond
)

// UATWindow is a standalone Fyne window for visually testing character
// animations. It provides a character dropdown, state trigger buttons,
// and diagnostic labels (character name, state, FPS).
type UATWindow struct {
	window     fyne.Window
	charWidget fyne.CanvasObject
	character  character.Character
	fps        *FPSCounter
	stateLabel *widget.Label
	charLabel  *widget.Label
	fpsLabel   *widget.Label

	charContainer *fyne.Container
	stateButtons  []*widget.Button
	dropdown      *widget.Select
	fpsLoop       *FPSLoop
}

// NewUATWindow creates a new UAT harness window attached to the given app.
// It discovers registered characters via the character registry and sets up
// the full UI layout. If no characters are available, controls are disabled.
func NewUATWindow(app fyne.App) *UATWindow {
	w := &UATWindow{
		window:     app.NewWindow("Character UAT"),
		fps:        NewFPSCounter(),
		charLabel:  widget.NewLabel("Character: (none)"),
		stateLabel: widget.NewLabel("State: (none)"),
		fpsLabel:   widget.NewLabel("FPS: 0"),
	}

	w.fpsLoop = NewFPSLoop(FPSLoopConfig{
		Counter:  w.fps,
		Interval: fpsUpdateInterval,
		OnFPSUpdate: func(text string) {
			fyne.Do(func() { w.fpsLabel.SetText(text) })
		},
	})

	w.charContainer = container.NewStack()

	// Build controls.
	w.buildControls()

	// Build layout.
	w.buildLayout()

	w.window.Resize(fyne.NewSize(windowWidth, windowHeight))

	// Select the first available character if any.
	names := availableCharacterNames()
	if len(names) > 0 {
		w.selectCharacter(names[0])
		w.dropdown.SetSelected(names[0])
	}

	return w
}

// Run starts the FPS update goroutine and shows the window. It blocks
// until the window is closed.
func (w *UATWindow) Run() {
	w.fpsLoop.Start()
	w.window.ShowAndRun()
	w.fpsLoop.Stop()
}

// selectCharacter creates a character by name from the registry and swaps
// the displayed widget. If creation fails, the previous character is kept.
func (w *UATWindow) selectCharacter(name string) {
	ch, err := character.Create(name)
	if err != nil {
		w.charLabel.SetText(fmt.Sprintf("Character: error (%s)", err))
		return
	}

	w.character = ch
	w.charWidget = ch.Widget()
	w.charContainer.RemoveAll()
	w.charContainer.Add(w.charWidget)

	w.charLabel.SetText(fmt.Sprintf("Character: %s", ch.Name()))
	w.stateLabel.SetText(fmt.Sprintf("State: %s", ch.CurrentState()))
	w.setStateButtonsEnabled(true)
}

// triggerState transitions the current character to the given state and
// updates the diagnostics label.
func (w *UATWindow) triggerState(state character.CharacterState) {
	if w.character == nil {
		return
	}
	w.character.TransitionTo(state)
	w.fps.Tick()
	w.stateLabel.SetText(fmt.Sprintf("State: %s", state))
}

// buildControls creates the character dropdown and state buttons.
func (w *UATWindow) buildControls() {
	names := availableCharacterNames()

	// Character dropdown.
	if len(names) == 0 {
		names = []string{"No characters available"}
	}
	w.dropdown = widget.NewSelect(names, func(selected string) {
		w.selectCharacter(selected)
	})

	// State buttons.
	states := []struct {
		label string
		state character.CharacterState
	}{
		{"Idle", character.StateIdle},
		{"Starting", character.StateStarting},
		{"Working", character.StateWorking},
		{"Notifying", character.StateNotifying},
		{"Error", character.StateError},
		{"Shutdown", character.StateShuttingDown},
	}

	w.stateButtons = make([]*widget.Button, 0, len(states))
	for _, st := range states {
		s := st.state
		btn := widget.NewButton(st.label, func() {
			w.triggerState(s)
		})
		w.stateButtons = append(w.stateButtons, btn)
	}

	// Disable buttons if no characters are registered.
	if len(character.Available()) == 0 {
		w.setStateButtonsEnabled(false)
	}
}

// buildLayout assembles the 60/40 split layout with footer.
func (w *UATWindow) buildLayout() {
	// Diagnostics section (top-right).
	diagnosticsHeader := widget.NewLabel("Diagnostics")
	diagnosticsHeader.TextStyle = fyne.TextStyle{Bold: true}
	diagnostics := container.NewVBox(
		diagnosticsHeader,
		w.charLabel,
		w.stateLabel,
		w.fpsLabel,
	)

	// Controls section (bottom-right).
	controlsHeader := widget.NewLabel("Controls")
	controlsHeader.TextStyle = fyne.TextStyle{Bold: true}

	// Arrange state buttons in a 3x2 grid.
	buttonGrid := container.NewGridWithColumns(2)
	for _, btn := range w.stateButtons {
		buttonGrid.Add(btn)
	}

	controls := container.NewVBox(
		controlsHeader,
		widget.NewLabel("Character:"),
		w.dropdown,
		buttonGrid,
	)

	// Right panel: diagnostics on top, controls on bottom.
	rightPanel := container.NewBorder(diagnostics, nil, nil, nil, controls)

	// Left panel: character widget centered.
	leftPanel := NewCharacterPanel(w.charContainer)

	// 60/40 horizontal split.
	split := container.NewHSplit(leftPanel, rightPanel)
	split.SetOffset(0.6)

	// Footer.
	footer := widget.NewLabel("UAT Harness \u2014 select character and trigger states")
	footer.Alignment = fyne.TextAlignCenter

	content := container.New(layout.NewBorderLayout(nil, footer, nil, nil), footer, split)
	w.window.SetContent(content)
}

// setStateButtonsEnabled enables or disables all state trigger buttons.
func (w *UATWindow) setStateButtonsEnabled(enabled bool) {
	for _, btn := range w.stateButtons {
		if enabled {
			btn.Enable()
		} else {
			btn.Disable()
		}
	}
}

// NewCharacterPanel creates the left panel container that holds the character widget.
// The character widget fills the available space to match the main application's embedding.
func NewCharacterPanel(charContainer *fyne.Container) fyne.CanvasObject {
	return container.NewStack(charContainer)
}

// availableCharacterNames returns sorted character names from the registry,
// including the "none" no-op character for baseline testing.
func availableCharacterNames() []string {
	all := character.Available()
	sort.Strings(all)
	return all
}
