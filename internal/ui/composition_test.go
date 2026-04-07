package ui_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
	"github.com/CreateFutureMWilkinson/cue/internal/ui"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
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

// TestMainWindowLeftColumnIsFocusRailContainer verifies that the left column
// of the three-column layout contains a *fyne.Container (from FocusRail.Container())
// rather than a placeholder *widget.Label.
func (s *CompositionSuite) TestMainWindowLeftColumnIsFocusRailContainer() {
	fyneApp := test.NewApp()
	router := ui.NewCenterViewRouter()
	mw := newTestMainWindow(fyneApp, router)

	content := mw.Content()
	s.Require().NotNil(content)

	outerSplit, ok := content.(*container.Split)
	s.Require().True(ok, "root content should be a *container.Split, got %T", content)

	// The left column (Leading) should be a *fyne.Container (VBox from FocusRail),
	// not a *widget.Label placeholder.
	_, isLabel := outerSplit.Leading.(*widget.Label)
	s.False(isLabel, "left column should not be a *widget.Label placeholder")

	_, isContainer := outerSplit.Leading.(*fyne.Container)
	s.True(isContainer, "left column should be a *fyne.Container (from FocusRail), got %T", outerSplit.Leading)
}

// TestPlannerViewRefReturnsNonNilWhenVMProvided verifies that PlannerViewRef()
// returns a non-nil RefreshableView when PlannerViewModel and TimerViewModel
// are provided to NewMainWindow.
func (s *CompositionSuite) TestPlannerViewRefReturnsNonNilWhenVMProvided() {
	fyneApp := test.NewApp()
	router := ui.NewCenterViewRouter()
	vm := &stubPlannerTimerVM{}

	cfg := config.GUIConfig{
		WindowWidth:  1200,
		WindowHeight: 800,
	}
	mw := ui.NewMainWindow(
		fyneApp,
		cfg,
		(*presenter.NotificationPresenter)(nil),
		(*presenter.ActivityPresenter)(nil),
		(*presenter.FeedbackPresenter)(nil),
		(*presenter.AppPresenter)(nil),
		(*presenter.SettingsPresenter)(nil),
		(*presenter.ServiceSettingsPresenter)(nil),
		config.OllamaConfig{},
		nil, // characterWidget
		router,
		vm,  // plannerVM
		vm,  // timerVM
		nil, // wizardVM
	)

	ref := mw.PlannerViewRef()
	s.NotNil(ref, "PlannerViewRef() should return non-nil when plannerVM and timerVM are provided")
}

// TestWizardViewRefReturnsNonNilWhenVMProvided verifies that WizardViewRef()
// returns a non-nil RefreshableView when WizardViewModel is provided to
// NewMainWindow.
func (s *CompositionSuite) TestWizardViewRefReturnsNonNilWhenVMProvided() {
	fyneApp := test.NewApp()
	router := ui.NewCenterViewRouter()
	wvm := &stubWizardVM{}

	cfg := config.GUIConfig{
		WindowWidth:  1200,
		WindowHeight: 800,
	}
	mw := ui.NewMainWindow(
		fyneApp,
		cfg,
		(*presenter.NotificationPresenter)(nil),
		(*presenter.ActivityPresenter)(nil),
		(*presenter.FeedbackPresenter)(nil),
		(*presenter.AppPresenter)(nil),
		(*presenter.SettingsPresenter)(nil),
		(*presenter.ServiceSettingsPresenter)(nil),
		config.OllamaConfig{},
		nil, // characterWidget
		router,
		nil, // plannerVM
		nil, // timerVM
		wvm, // wizardVM
	)

	ref := mw.WizardViewRef()
	s.NotNil(ref, "WizardViewRef() should return non-nil when wizardVM is provided")
}
