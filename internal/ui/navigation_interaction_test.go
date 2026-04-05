package ui_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/ui"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/uitest"
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

// TestWiredFocusRailPlanButtonNavigates verifies that the Plan button embedded
// in the MainWindow's left column (via FocusRail wiring) navigates the shared
// CenterViewRouter to ViewPlan when tapped. This is a Tier 2 integration test
// exercising the full composition path: MainWindow → FocusRail → button tap →
// CenterViewRouter navigation.
func (s *NavigationInteractionSuite) TestWiredFocusRailPlanButtonNavigates() {
	fyneApp := test.NewApp()
	router := ui.NewCenterViewRouter()
	mw := newTestMainWindow(fyneApp, router)

	// Dig into MainWindow content: outerSplit.Leading is the FocusRail VBox.
	outerSplit, ok := mw.Content().(*container.Split)
	s.Require().True(ok, "MainWindow content should be *container.Split, got %T", mw.Content())

	leftCol, ok := outerSplit.Leading.(*fyne.Container)
	s.Require().True(ok, "outer split Leading should be *fyne.Container (FocusRail), got %T", outerSplit.Leading)

	// Find the Plan button in the FocusRail container tree.
	planBtn := uitest.RequireWidget[*widget.Button](s.T(), leftCol, func(b *widget.Button) bool {
		return b.Text == "Plan"
	})

	// Tap the Plan button and verify navigation.
	planBtn.OnTapped()

	s.Equal(ui.ViewPlan, router.CurrentView(),
		"tapping the wired Plan button in MainWindow should navigate router to ViewPlan")
}
