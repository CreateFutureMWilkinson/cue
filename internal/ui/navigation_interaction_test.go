package ui_test

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/ui"
)

// NavigationInteractionSuite contains Tier 2 interaction tests that verify
// tapping FocusRail buttons via fyne/test.Tap changes the CenterViewRouter
// to the expected view.
type NavigationInteractionSuite struct {
	suite.Suite
	router *ui.CenterViewRouter
	rail   *ui.FocusRail
}

func TestNavigationInteraction(t *testing.T) {
	suite.Run(t, new(NavigationInteractionSuite))
}

func (s *NavigationInteractionSuite) SetupTest() {
	s.router = ui.NewCenterViewRouter()
	s.rail = ui.NewFocusRail(s.router)
}

func (s *NavigationInteractionSuite) TestTapPlanButtonNavigatesToPlanView() {
	test.Tap(s.rail.PlanButton())

	s.Equal(ui.ViewPlan, s.router.CurrentView(),
		"tapping Plan button should navigate router to ViewPlan")
}

func (s *NavigationInteractionSuite) TestTapBackButtonNavigatesToCharacterView() {
	// Start in Plan view so Back button is visible and active.
	s.router.NavigateTo(ui.ViewPlan)

	test.Tap(s.rail.BackButton())

	s.Equal(ui.ViewCharacter, s.router.CurrentView(),
		"tapping Back button from Plan view should navigate router to ViewCharacter")
}

func (s *NavigationInteractionSuite) TestTapSettingsButtonNavigatesToSettingsView() {
	test.Tap(s.rail.SettingsButton())

	s.Equal(ui.ViewSettings, s.router.CurrentView(),
		"tapping Settings button should navigate router to ViewSettings")
}

func (s *NavigationInteractionSuite) TestTapBackFromSettingsNavigatesToCharacterView() {
	// Navigate to Settings first so Back button is visible.
	s.router.NavigateTo(ui.ViewSettings)

	test.Tap(s.rail.BackButton())

	s.Equal(ui.ViewCharacter, s.router.CurrentView(),
		"tapping Back button from Settings view should navigate router to ViewCharacter")
}

func (s *NavigationInteractionSuite) TestTapPlanThenBackRoundTrip() {
	// Tap Plan — should go to ViewPlan.
	test.Tap(s.rail.PlanButton())
	s.Equal(ui.ViewPlan, s.router.CurrentView(),
		"after tapping Plan, router should be at ViewPlan")

	// Tap Back — should return to ViewCharacter.
	test.Tap(s.rail.BackButton())
	s.Equal(ui.ViewCharacter, s.router.CurrentView(),
		"after tapping Back, router should return to ViewCharacter")
}
