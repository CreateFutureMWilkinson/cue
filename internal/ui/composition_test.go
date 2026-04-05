package ui_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/ui"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/uitest"
)

// CompositionSuite verifies that MainWindow composes the expected
// top-level widget tree: a three-column HSplit layout with real
// widgets in each column position.
type CompositionSuite struct {
	suite.Suite
}

func TestComposition(t *testing.T) {
	suite.Run(t, new(CompositionSuite))
}

// TestMainWindowContentExposesWidgetTree verifies that MainWindow has a
// Content() method that returns a non-nil canvas object representing the
// window's widget tree.
func (s *CompositionSuite) TestMainWindowContentExposesWidgetTree() {
	fyneApp := test.NewApp()
	router := ui.NewCenterViewRouter()
	mw := newTestMainWindow(fyneApp, router)

	content := mw.Content()
	s.NotNil(content, "Content() should return a non-nil canvas object")
}

// TestMainWindowContainsOuterHSplit verifies that the root content returned
// by Content() is a *container.Split configured as a horizontal split
// (the outer HSplit of the three-column layout).
func (s *CompositionSuite) TestMainWindowContainsOuterHSplit() {
	fyneApp := test.NewApp()
	router := ui.NewCenterViewRouter()
	mw := newTestMainWindow(fyneApp, router)

	content := mw.Content()
	s.Require().NotNil(content)

	outerSplit, ok := content.(*container.Split)
	s.Require().True(ok, "root content should be a *container.Split, got %T", content)
	s.True(outerSplit.Horizontal, "outer split should be horizontal")
}

// TestMainWindowContainsCenterStack verifies that the widget tree contains
// a *fyne.Container (the center stack) nested within the inner split's
// leading position.
func (s *CompositionSuite) TestMainWindowContainsCenterStack() {
	fyneApp := test.NewApp()
	router := ui.NewCenterViewRouter()
	mw := newTestMainWindow(fyneApp, router)

	content := mw.Content()
	s.Require().NotNil(content)

	// The outer split's Trailing is the inner split.
	outerSplit, ok := content.(*container.Split)
	s.Require().True(ok, "root content should be a *container.Split")

	innerSplit, ok := outerSplit.Trailing.(*container.Split)
	s.Require().True(ok, "outer split's trailing should be inner *container.Split, got %T", outerSplit.Trailing)

	// The inner split's Leading is the center stack (*fyne.Container).
	centerStack, found := uitest.FindWidget[*fyne.Container](innerSplit.Leading, func(c *fyne.Container) bool {
		return true
	})
	s.True(found, "inner split's leading should contain a *fyne.Container (center stack)")
	s.NotNil(centerStack)
}

// TestMainWindowContainsNotificationArea verifies that the right column
// of the three-column layout contains a notification-related widget.
// When no NotificationPresenter is provided (nil), the notification area
// is a placeholder label.
func (s *CompositionSuite) TestMainWindowContainsNotificationArea() {
	fyneApp := test.NewApp()
	router := ui.NewCenterViewRouter()
	mw := newTestMainWindow(fyneApp, router)

	content := mw.Content()
	s.Require().NotNil(content)

	// Navigate to the inner split's trailing (notification area).
	outerSplit, ok := content.(*container.Split)
	s.Require().True(ok, "root content should be a *container.Split")

	innerSplit, ok := outerSplit.Trailing.(*container.Split)
	s.Require().True(ok, "outer split's trailing should be inner *container.Split")

	// The notification area is the inner split's trailing element.
	s.NotNil(innerSplit.Trailing,
		"inner split's trailing (notification area) should not be nil")
}
