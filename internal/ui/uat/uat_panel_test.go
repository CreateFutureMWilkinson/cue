package uat_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/uat"
)

type UATPanelSuite struct {
	suite.Suite
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
