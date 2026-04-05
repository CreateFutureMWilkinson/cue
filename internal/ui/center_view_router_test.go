package ui_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/ui"
)

// CenterViewRouterSuite tests the CenterViewRouter state machine that controls
// which view is displayed in the center 60% column of the three-column layout.
type CenterViewRouterSuite struct {
	suite.Suite
}

func TestCenterViewRouter(t *testing.T) {
	suite.Run(t, new(CenterViewRouterSuite))
}

func (s *CenterViewRouterSuite) TestNewCenterViewRouterDefaultsToCharacterView() {
	router := ui.NewCenterViewRouter()

	s.Equal(ui.ViewCharacter, router.CurrentView(),
		"new router should default to ViewCharacter")
}

func (s *CenterViewRouterSuite) TestNavigateToChangesCurrentView() {
	router := ui.NewCenterViewRouter()

	router.NavigateTo(ui.ViewPlan)

	s.Equal(ui.ViewPlan, router.CurrentView(),
		"CurrentView should return ViewPlan after NavigateTo(ViewPlan)")
}

func (s *CenterViewRouterSuite) TestNavigateToTriggersCallback() {
	router := ui.NewCenterViewRouter()

	var received ui.CenterView
	called := false
	router.SetOnViewChange(func(v ui.CenterView) {
		called = true
		received = v
	})

	router.NavigateTo(ui.ViewWizard)

	s.True(called, "callback should have been called")
	s.Equal(ui.ViewWizard, received,
		"callback should receive the new view")
}

func (s *CenterViewRouterSuite) TestNavigateToSameViewStillTriggersCallback() {
	router := ui.NewCenterViewRouter()

	callCount := 0
	router.SetOnViewChange(func(_ ui.CenterView) {
		callCount++
	})

	// Default is ViewCharacter, navigating to it again should still fire.
	router.NavigateTo(ui.ViewCharacter)

	s.Equal(1, callCount,
		"callback should fire even when navigating to the current view")
}

func (s *CenterViewRouterSuite) TestCallbackNotCalledWithoutSet() {
	router := ui.NewCenterViewRouter()

	// No callback set — NavigateTo must not panic.
	s.NotPanics(func() {
		router.NavigateTo(ui.ViewPlan)
	}, "NavigateTo should not panic when no callback is set")

	s.Equal(ui.ViewPlan, router.CurrentView(),
		"view should still change even without a callback")
}

func (s *CenterViewRouterSuite) TestNavigateToSettingsFiresCallback() {
	router := ui.NewCenterViewRouter()

	var received ui.CenterView
	called := false
	router.SetOnViewChange(func(v ui.CenterView) {
		called = true
		received = v
	})

	router.NavigateTo(ui.ViewSettings)

	s.True(called, "callback should have been called")
	s.Equal(ui.ViewSettings, received,
		"callback should receive ViewSettings")
	s.Equal(ui.ViewSettings, router.CurrentView(),
		"CurrentView should return ViewSettings after NavigateTo(ViewSettings)")
}

func (s *CenterViewRouterSuite) TestMultipleNavigations() {
	router := ui.NewCenterViewRouter()

	views := []ui.CenterView{
		ui.ViewCharacter,
		ui.ViewPlan,
		ui.ViewWizard,
		ui.ViewCharacter,
	}

	var history []ui.CenterView
	router.SetOnViewChange(func(v ui.CenterView) {
		history = append(history, v)
	})

	for _, v := range views {
		router.NavigateTo(v)
	}

	s.Equal(views, history,
		"callback should have recorded all navigations in order")
	s.Equal(ui.ViewCharacter, router.CurrentView(),
		"final view should be ViewCharacter after full cycle")
}
