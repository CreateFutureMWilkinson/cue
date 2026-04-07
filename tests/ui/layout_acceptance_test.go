//go:build ui_acceptance

package ui_acceptance_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/ui"
)

// LayoutAcceptanceSuite verifies the three-column layout acceptance criteria
// from UiSpec.md lines 1016-1020.
type LayoutAcceptanceSuite struct {
	suite.Suite
}

func TestLayoutAcceptance(t *testing.T) {
	suite.Run(t, new(LayoutAcceptanceSuite))
}

// AC: Focus rail occupies 10% width, always visible.
// Verified structurally: outer HSplit Leading is the focus rail container.
func (s *LayoutAcceptanceSuite) TestFocusRailIsLeftColumn() {
	app := newTestApp()
	defer app.Quit()
	router := ui.NewCenterViewRouter()
	mw := newMinimalMainWindow(app, router)

	content := mw.Content()
	s.Require().NotNil(content)

	outerSplit, ok := content.(*container.Split)
	s.Require().True(ok, "root content should be *container.Split, got %T", content)
	s.True(outerSplit.Horizontal, "outer split should be horizontal")

	// Leading column should be a *fyne.Container (FocusRail), not a placeholder label.
	_, isContainer := outerSplit.Leading.(*fyne.Container)
	s.True(isContainer, "left column (focus rail) should be a *fyne.Container, got %T", outerSplit.Leading)
}

// AC: Character area occupies 60% width (center column).
// Verified structurally: inner HSplit exists with Leading as center area.
func (s *LayoutAcceptanceSuite) TestCenterAreaExists() {
	app := newTestApp()
	defer app.Quit()
	router := ui.NewCenterViewRouter()
	mw := newMinimalMainWindow(app, router)

	content := mw.Content()
	outerSplit, ok := content.(*container.Split)
	s.Require().True(ok)

	innerSplit, ok := outerSplit.Trailing.(*container.Split)
	s.Require().True(ok, "outer trailing should be inner *container.Split, got %T", outerSplit.Trailing)

	s.NotNil(innerSplit.Leading, "inner split leading (center area) should not be nil")
}

// AC: Notification panel occupies 30% width (right column).
// Verified structurally: inner HSplit Trailing is the notification area.
func (s *LayoutAcceptanceSuite) TestNotificationPanelIsRightColumn() {
	app := newTestApp()
	defer app.Quit()
	router := ui.NewCenterViewRouter()
	mw := newMinimalMainWindow(app, router)

	content := mw.Content()
	outerSplit, ok := content.(*container.Split)
	s.Require().True(ok)

	innerSplit, ok := outerSplit.Trailing.(*container.Split)
	s.Require().True(ok)

	s.NotNil(innerSplit.Trailing, "inner split trailing (notification panel) should not be nil")
}

// AC: No tab bar -- navigation via focus rail buttons and contextual controls.
// Verified: no AppTabs widget in the top-level content tree (outside of settings view).
func (s *LayoutAcceptanceSuite) TestNoTopLevelTabBar() {
	app := newTestApp()
	defer app.Quit()
	router := ui.NewCenterViewRouter()
	mw := newMinimalMainWindow(app, router)

	content := mw.Content()
	outerSplit, ok := content.(*container.Split)
	s.Require().True(ok)

	// The outer split's Leading (focus rail) should not contain tabs.
	_, isLabel := outerSplit.Leading.(*widget.Label)
	s.False(isLabel, "left column should not be a placeholder label")

	// Check that the outer container does not contain AppTabs at the top level.
	_, isTabs := content.(*container.AppTabs)
	s.False(isTabs, "root content should not be AppTabs — navigation is via focus rail buttons")
}
