package ui_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/ui"
)

// FocusRailSuite tests the FocusRail component which provides the persistent
// left column with timer, task info, and navigation buttons.
type FocusRailSuite struct {
	suite.Suite
	router *ui.CenterViewRouter
}

func TestFocusRail(t *testing.T) {
	suite.Run(t, new(FocusRailSuite))
}

func (s *FocusRailSuite) SetupTest() {
	s.router = ui.NewCenterViewRouter()
}

func (s *FocusRailSuite) TestNewFocusRailReturnsNonNil() {
	rail := ui.NewFocusRail(s.router)

	s.NotNil(rail, "NewFocusRail should return a non-nil component")
}

func (s *FocusRailSuite) TestFocusRailPlanButtonVisibleByDefault() {
	rail := ui.NewFocusRail(s.router)

	s.True(rail.PlanButton().Visible(),
		"Plan button should be visible when view is Character (default)")
}

func (s *FocusRailSuite) TestFocusRailBackButtonHiddenByDefault() {
	rail := ui.NewFocusRail(s.router)

	s.False(rail.BackButton().Visible(),
		"Back button should be hidden when view is Character (default)")
}

func (s *FocusRailSuite) TestFocusRailBackButtonVisibleInPlanView() {
	rail := ui.NewFocusRail(s.router)

	s.router.NavigateTo(ui.ViewPlan)

	s.True(rail.BackButton().Visible(),
		"Back button should be visible when in Plan view")
}

func (s *FocusRailSuite) TestFocusRailBackButtonVisibleInWizardView() {
	rail := ui.NewFocusRail(s.router)

	s.router.NavigateTo(ui.ViewWizard)

	s.True(rail.BackButton().Visible(),
		"Back button should be visible when in Wizard view")
}

func (s *FocusRailSuite) TestFocusRailPlanButtonHiddenInPlanView() {
	rail := ui.NewFocusRail(s.router)

	s.router.NavigateTo(ui.ViewPlan)

	s.False(rail.PlanButton().Visible(),
		"Plan button should be hidden when in Plan view")
}

func (s *FocusRailSuite) TestFocusRailBackAndPlanMutuallyExclusive() {
	rail := ui.NewFocusRail(s.router)

	// In Character view: Plan visible, Back hidden.
	s.True(rail.PlanButton().Visible(), "Plan should be visible in Character view")
	s.False(rail.BackButton().Visible(), "Back should be hidden in Character view")

	// Navigate to Plan: Back visible, Plan hidden.
	s.router.NavigateTo(ui.ViewPlan)
	s.True(rail.BackButton().Visible(), "Back should be visible in Plan view")
	s.False(rail.PlanButton().Visible(), "Plan should be hidden in Plan view")

	// Navigate back to Character: Plan visible, Back hidden.
	s.router.NavigateTo(ui.ViewCharacter)
	s.True(rail.PlanButton().Visible(), "Plan should be visible after returning to Character")
	s.False(rail.BackButton().Visible(), "Back should be hidden after returning to Character")
}

func (s *FocusRailSuite) TestFocusRailPlanButtonNavigatesToPlan() {
	rail := ui.NewFocusRail(s.router)

	// Simulate tapping the Plan button.
	rail.PlanButton().OnTapped()

	s.Equal(ui.ViewPlan, s.router.CurrentView(),
		"tapping Plan button should navigate to ViewPlan")
}

func (s *FocusRailSuite) TestFocusRailBackButtonNavigatesToCharacter() {
	s.router.NavigateTo(ui.ViewPlan)
	rail := ui.NewFocusRail(s.router)

	// Simulate tapping the Back button.
	rail.BackButton().OnTapped()

	s.Equal(ui.ViewCharacter, s.router.CurrentView(),
		"tapping Back button should navigate to ViewCharacter")
}

func (s *FocusRailSuite) TestFocusRailTimerHiddenWithoutActivePlan() {
	rail := ui.NewFocusRail(s.router)

	s.False(rail.Timer().Visible(),
		"timer should be hidden when no active plan exists")
}

func (s *FocusRailSuite) TestFocusRailTimerVisibleWithActivePlan() {
	rail := ui.NewFocusRail(s.router)

	rail.SetActivePlan(true)

	s.True(rail.Timer().Visible(),
		"timer should be visible when active plan is set")
}

func (s *FocusRailSuite) TestFocusRailTaskNameHiddenWithoutActivePlan() {
	rail := ui.NewFocusRail(s.router)

	s.False(rail.TaskLabel().Visible(),
		"task label should be hidden when no active plan exists")
}

func (s *FocusRailSuite) TestFocusRailTaskNameVisibleWithActivePlan() {
	rail := ui.NewFocusRail(s.router)

	rail.SetActivePlan(true)

	s.True(rail.TaskLabel().Visible(),
		"task label should be visible when active plan is set")
}

func (s *FocusRailSuite) TestFocusRailDoneButtonHiddenWithoutActivePlan() {
	rail := ui.NewFocusRail(s.router)

	s.False(rail.DoneButton().Visible(),
		"Done button should be hidden when no active plan exists")
}

func (s *FocusRailSuite) TestFocusRailDoneButtonVisibleWithActivePlan() {
	rail := ui.NewFocusRail(s.router)

	rail.SetActivePlan(true)

	s.True(rail.DoneButton().Visible(),
		"Done button should be visible when active plan is set")
}

func (s *FocusRailSuite) TestFocusRailReviewButtonHiddenByDefault() {
	rail := ui.NewFocusRail(s.router)

	s.False(rail.ReviewButton().Visible(),
		"Review button should be hidden when notifications are not expanded")
}

func (s *FocusRailSuite) TestFocusRailReviewButtonVisibleWhenNotificationsExpanded() {
	rail := ui.NewFocusRail(s.router)

	rail.SetNotificationsExpanded(true)

	s.True(rail.ReviewButton().Visible(),
		"Review button should be visible when notifications are expanded")
}

func (s *FocusRailSuite) TestFocusRailSetCurrentTask() {
	rail := ui.NewFocusRail(s.router)

	rail.SetCurrentTask("Write TDD tests")

	s.Equal("Write TDD tests", rail.TaskLabel().Text,
		"SetCurrentTask should update the task label text")
}

func (s *FocusRailSuite) TestFocusRailDoneButtonCallback() {
	rail := ui.NewFocusRail(s.router)

	called := false
	rail.SetOnDone(func() {
		called = true
	})

	rail.DoneButton().OnTapped()

	s.True(called, "tapping Done button should invoke the onDone callback")
}
