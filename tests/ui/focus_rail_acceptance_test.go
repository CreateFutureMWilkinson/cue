//go:build ui_acceptance

package ui_acceptance_test

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/ui"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/uitest"
)

// FocusRailAcceptanceSuite verifies focus rail acceptance criteria
// from UiSpec.md lines 1023-1031.
type FocusRailAcceptanceSuite struct {
	suite.Suite
}

func TestFocusRailAcceptance(t *testing.T) {
	suite.Run(t, new(FocusRailAcceptanceSuite))
}

// AC: Plan button visible when center area is character (switches to Plan view).
func (s *FocusRailAcceptanceSuite) TestPlanButtonVisibleInCharacterView() {
	router := ui.NewCenterViewRouter()
	rail := ui.NewFocusRail(router)

	s.Equal(ui.ViewCharacter, router.CurrentView())
	s.True(rail.PlanButton().Visible(), "Plan button should be visible in character view")
}

// AC: Back button visible when center area is Plan view or Wizard (returns to character).
func (s *FocusRailAcceptanceSuite) TestBackButtonVisibleInPlanView() {
	router := ui.NewCenterViewRouter()
	rail := ui.NewFocusRail(router)

	router.NavigateTo(ui.ViewPlan)

	s.True(rail.BackButton().Visible(), "Back button should be visible in Plan view")
}

// AC: Back button visible when center area is Plan view or Wizard (returns to character).
func (s *FocusRailAcceptanceSuite) TestBackButtonVisibleInWizardView() {
	router := ui.NewCenterViewRouter()
	rail := ui.NewFocusRail(router)

	router.NavigateTo(ui.ViewWizard)

	s.True(rail.BackButton().Visible(), "Back button should be visible in Wizard view")
}

// AC: Back and Plan are mutually exclusive.
func (s *FocusRailAcceptanceSuite) TestPlanAndBackMutuallyExclusive() {
	router := ui.NewCenterViewRouter()
	rail := ui.NewFocusRail(router)

	// In character view: Plan visible, Back hidden.
	s.True(rail.PlanButton().Visible(), "Plan visible in character view")
	s.False(rail.BackButton().Visible(), "Back hidden in character view")

	// In plan view: Back visible, Plan hidden.
	router.NavigateTo(ui.ViewPlan)
	s.True(rail.BackButton().Visible(), "Back visible in plan view")
	s.False(rail.PlanButton().Visible(), "Plan hidden in plan view")
}

// AC: Tapping Plan button navigates to Plan view.
func (s *FocusRailAcceptanceSuite) TestTapPlanNavigatesToPlanView() {
	router := ui.NewCenterViewRouter()
	rail := ui.NewFocusRail(router)

	test.Tap(rail.PlanButton())

	s.Equal(ui.ViewPlan, router.CurrentView(),
		"tapping Plan should navigate to ViewPlan")
}

// AC: Tapping Back button returns to character view.
func (s *FocusRailAcceptanceSuite) TestTapBackReturnsToCharacterView() {
	router := ui.NewCenterViewRouter()
	rail := ui.NewFocusRail(router)

	router.NavigateTo(ui.ViewPlan)
	test.Tap(rail.BackButton())

	s.Equal(ui.ViewCharacter, router.CurrentView(),
		"tapping Back should return to ViewCharacter")
}

// AC: Review button only visible when notifications are expanded.
func (s *FocusRailAcceptanceSuite) TestReviewButtonHiddenByDefault() {
	app := newTestApp()
	defer app.Quit()
	router := ui.NewCenterViewRouter()
	_, np, _ := newMainWindowWithFeedback(app, router, sampleNotifiedMessages(), sampleBufferedMessages())

	rail := ui.NewFocusRail(router)
	// Wire the review button visibility to notification expanded state.
	// In a fully-wired MainWindow, this is handled by app_binder.
	// For this test, we check the default state.
	_ = np // Notifications are not expanded by default.

	s.False(rail.ReviewButton().Visible(),
		"Review button should be hidden when notifications are collapsed")
}

// AC: Focus rail contains a Done button.
func (s *FocusRailAcceptanceSuite) TestFocusRailContainsDoneButton() {
	router := ui.NewCenterViewRouter()
	rail := ui.NewFocusRail(router)
	root := rail.Container()

	_, found := uitest.FindWidget[*widget.Button](root, func(b *widget.Button) bool {
		return b.Text == "Done"
	})

	s.True(found, "Focus rail should contain a 'Done' button")
}

// AC: Tapping Plan then Back completes a round-trip.
func (s *FocusRailAcceptanceSuite) TestPlanBackRoundTrip() {
	router := ui.NewCenterViewRouter()
	rail := ui.NewFocusRail(router)

	test.Tap(rail.PlanButton())
	s.Equal(ui.ViewPlan, router.CurrentView())

	test.Tap(rail.BackButton())
	s.Equal(ui.ViewCharacter, router.CurrentView(),
		"round trip should return to ViewCharacter")
}
