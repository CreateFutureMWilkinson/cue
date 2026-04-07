package uat_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/uat"
)

type UATPanelSuite struct {
	suite.Suite
}

func (s *UATPanelSuite) SetupSuite() {
	test.NewApp()
}

func TestUATPanel(t *testing.T) {
	suite.Run(t, new(UATPanelSuite))
}

// collectWidgets recursively walks a fyne.CanvasObject tree and returns all
// objects found, including containers and their children.
func collectWidgets(obj fyne.CanvasObject) []fyne.CanvasObject {
	var result []fyne.CanvasObject
	result = append(result, obj)
	if c, ok := obj.(*fyne.Container); ok {
		for _, child := range c.Objects {
			result = append(result, collectWidgets(child)...)
		}
	}
	return result
}

func (s *UATPanelSuite) TestPanelRendersAllComponents() {
	character.Register("test-char", func() character.Character {
		return character.NewNoOpCharacter()
	})
	defer character.ResetRegistry()

	panel := uat.NewUATPanel(func(_ character.Character) {})

	container := panel.Container()
	s.Require().NotNil(container, "Container() must not be nil")

	widgets := collectWidgets(container)

	var selectCount int
	var buttonCount int
	var labelCount int

	for _, w := range widgets {
		switch w.(type) {
		case *widget.Select:
			selectCount++
		case *widget.Button:
			buttonCount++
		case *widget.Label:
			labelCount++
		}
	}

	s.GreaterOrEqual(selectCount, 1, "expected at least 1 *widget.Select (character dropdown)")
	s.GreaterOrEqual(buttonCount, 6, "expected at least 6 *widget.Button (state trigger buttons)")
	s.GreaterOrEqual(labelCount, 2, "expected at least 2 *widget.Label (state + character labels)")
}

func (s *UATPanelSuite) TestCharacterSelectionFiresCallback() {
	character.Register("test-char", func() character.Character {
		return character.NewNoOpCharacter()
	})
	defer character.ResetRegistry()

	var received character.Character
	panel := uat.NewUATPanel(func(c character.Character) {
		received = c
	})

	container := panel.Container()
	s.Require().NotNil(container, "Container() must not be nil")

	widgets := collectWidgets(container)

	var sel *widget.Select
	for _, w := range widgets {
		if s, ok := w.(*widget.Select); ok {
			sel = s
			break
		}
	}
	s.Require().NotNil(sel, "expected a *widget.Select in the widget tree")

	// Simulate user picking "test-char" from the dropdown.
	sel.OnChanged("test-char")

	s.Require().NotNil(received, "onCharChanged callback should have been called with a character")
	s.Equal("none", received.Name(), "NoOpCharacter.Name() should return \"none\"")
	s.Equal("Character: test-char", panel.CharacterLabel(), "charLabel should reflect selected character name")
}

func (s *UATPanelSuite) TestStateTriggerUpdatesLabel() {
	character.Register("test-char", func() character.Character {
		return character.NewNoOpCharacter()
	})
	defer character.ResetRegistry()

	panel := uat.NewUATPanel(func(_ character.Character) {})

	container := panel.Container()
	s.Require().NotNil(container, "Container() must not be nil")

	// Select "test-char" so buttons are active and currentChar is set.
	widgets := collectWidgets(container)
	var sel *widget.Select
	for _, w := range widgets {
		if ws, ok := w.(*widget.Select); ok {
			sel = ws
			break
		}
	}
	s.Require().NotNil(sel, "expected a *widget.Select in the widget tree")
	sel.OnChanged("test-char")

	// Collect all buttons from the widget tree.
	var buttons []*widget.Button
	for _, w := range widgets {
		if b, ok := w.(*widget.Button); ok {
			buttons = append(buttons, b)
		}
	}
	s.Require().GreaterOrEqual(len(buttons), 6, "expected at least 6 state trigger buttons")

	// Tap the "Working" button (index 2).
	s.Require().NotNil(buttons[2].OnTapped, "Working button must have an OnTapped handler")
	buttons[2].OnTapped()

	s.Equal("Current State: Working", panel.StateLabel(),
		"StateLabel should reflect the tapped state")
}

func (s *UATPanelSuite) TestButtonsDisabledForNone() {
	character.Register("test-char", func() character.Character {
		return character.NewNoOpCharacter()
	})
	defer character.ResetRegistry()

	panel := uat.NewUATPanel(func(_ character.Character) {})

	container := panel.Container()
	s.Require().NotNil(container, "Container() must not be nil")

	widgets := collectWidgets(container)
	var sel *widget.Select
	for _, w := range widgets {
		if ws, ok := w.(*widget.Select); ok {
			sel = ws
			break
		}
	}
	s.Require().NotNil(sel, "expected a *widget.Select in the widget tree")

	// Collect all state trigger buttons.
	var buttons []*widget.Button
	for _, w := range widgets {
		if b, ok := w.(*widget.Button); ok {
			buttons = append(buttons, b)
		}
	}
	s.Require().GreaterOrEqual(len(buttons), 6, "expected at least 6 state trigger buttons")

	// First select "test-char" — buttons should be enabled.
	sel.OnChanged("test-char")
	for i := 0; i < 6; i++ {
		s.False(buttons[i].Disabled(),
			"button %d (%s) should be enabled after selecting test-char", i, buttons[i].Text)
	}

	// Now select "none" — buttons should be disabled.
	sel.OnChanged("none")
	for i := 0; i < 6; i++ {
		s.True(buttons[i].Disabled(),
			"button %d (%s) should be disabled after selecting none", i, buttons[i].Text)
	}
}
