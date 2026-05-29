package ui_test

import (
	"context"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
	"github.com/CreateFutureMWilkinson/cue/internal/service/planner"
	"github.com/CreateFutureMWilkinson/cue/internal/ui"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
)

// stubPlannerTimerVM satisfies both PlannerViewModel and TimerViewModel
// with zero-value returns for all methods. Used in layout wiring tests.
type stubPlannerTimerVM struct{}

func (s *stubPlannerTimerVM) CurrentStep() presenter.WizardStep              { return presenter.StepIdle }
func (s *stubPlannerTimerVM) HasActivePlan() bool                            { return false }
func (s *stubPlannerTimerVM) ActiveSchedule() *presenter.ActiveScheduleState { return nil }
func (s *stubPlannerTimerVM) CurrentFocusTask(_ context.Context) (*presenter.TodoRow, error) {
	return nil, nil
}
func (s *stubPlannerTimerVM) IsRunning() bool              { return false }
func (s *stubPlannerTimerVM) ActiveSegment() int           { return 0 }
func (s *stubPlannerTimerVM) ElapsedFraction() float64     { return 0 }
func (s *stubPlannerTimerVM) IsFlashVisible() bool         { return false }
func (s *stubPlannerTimerVM) CurrentTaskName() string      { return "" }
func (s *stubPlannerTimerVM) BlockType() planner.BlockType { return planner.BlockFocus }

// ThreeColumnLayoutSuite tests that the MainWindow API accepts the
// CenterViewRouter parameter required by the three-column layout.
type ThreeColumnLayoutSuite struct {
	suite.Suite
}

func TestThreeColumnLayout(t *testing.T) {
	suite.Run(t, new(ThreeColumnLayoutSuite))
}

// newTestMainWindow is a helper that creates a MainWindow with a given
// CenterViewRouter and nil presenters (sufficient for layout tests).
// The fyne.App is passed as the first argument to NewMainWindow, replacing
// the old package-level factory pattern.
func newTestMainWindow(fyneApp fyne.App, router *ui.CenterViewRouter) *ui.MainWindow {
	cfg := config.GUIConfig{
		WindowWidth:  1200,
		WindowHeight: 800,
	}
	return ui.NewMainWindow(
		fyneApp,
		cfg,
		(*presenter.NotificationPresenter)(nil),
		(*presenter.ActivityPresenter)(nil),
		(*presenter.FeedbackPresenter)(nil),
		(*presenter.AppPresenter)(nil),
		(*presenter.SettingsPresenter)(nil),
		(*presenter.ServiceSettingsPresenter)(nil),
		nil, // rp
		config.OllamaConfig{},
		nil, // characterWidget
		router,
		nil, // plannerVM
		nil, // timerVM
		nil, // wizardVM
		nil, // rightPanelOverride
	)
}

// TestNewMainWindowAcceptsFyneApp verifies that NewMainWindow accepts a
// fyne.App as its first parameter, allowing callers to inject the application
// instance instead of relying on a package-level factory variable.
func (s *ThreeColumnLayoutSuite) TestNewMainWindowAcceptsFyneApp() {
	fyneApp := test.NewApp()
	router := ui.NewCenterViewRouter()
	mw := newTestMainWindow(fyneApp, router)
	s.NotNil(mw, "NewMainWindow should return a non-nil *MainWindow when given an injected fyne.App")
}

func (s *ThreeColumnLayoutSuite) TestNewMainWindowAcceptsCenterViewRouter() {
	// This test verifies the NewMainWindow signature accepts a *CenterViewRouter.
	// It is a compile-time contract test -- if the parameter is missing, this
	// file will not compile.
	//
	// We pass nil for all presenter dependencies because we are only testing
	// that the function signature is correct, not that it produces a working
	// window. The function should still return a non-nil *MainWindow.
	fyneApp := test.NewApp()
	router := ui.NewCenterViewRouter()
	mw := newTestMainWindow(fyneApp, router)
	s.NotNil(mw, "NewMainWindow should return a non-nil *MainWindow")
}

func (s *ThreeColumnLayoutSuite) TestCenterViewDefaultsToCharacterContent() {
	fyneApp := test.NewApp()
	router := ui.NewCenterViewRouter()
	mw := newTestMainWindow(fyneApp, router)

	content := mw.CenterContent()
	s.NotNil(content, "CenterContent should return a non-nil canvas object on startup")

	// The router should default to ViewCharacter.
	s.Equal(ui.ViewCharacter, router.CurrentView(),
		"CenterViewRouter should default to ViewCharacter")
}

func (s *ThreeColumnLayoutSuite) TestNavigateToPlanSwapsCenterContent() {
	fyneApp := test.NewApp()
	router := ui.NewCenterViewRouter()
	mw := newTestMainWindow(fyneApp, router)

	originalContent := mw.CenterContent()
	s.NotNil(originalContent)

	router.NavigateTo(ui.ViewPlan)

	newContent := mw.CenterContent()
	s.NotNil(newContent, "CenterContent should not be nil after navigating to ViewPlan")
	s.NotEqual(originalContent, newContent,
		"Navigating to ViewPlan should swap the center pane to different content")
}

func (s *ThreeColumnLayoutSuite) TestNavigateToWizardSwapsCenterContent() {
	fyneApp := test.NewApp()
	router := ui.NewCenterViewRouter()
	mw := newTestMainWindow(fyneApp, router)

	originalContent := mw.CenterContent()
	s.NotNil(originalContent)

	router.NavigateTo(ui.ViewWizard)

	newContent := mw.CenterContent()
	s.NotNil(newContent, "CenterContent should not be nil after navigating to ViewWizard")
	s.NotEqual(originalContent, newContent,
		"Navigating to ViewWizard should swap the center pane to different content")
}

func (s *ThreeColumnLayoutSuite) TestNavigateToSettingsSwapsCenterContent() {
	fyneApp := test.NewApp()
	router := ui.NewCenterViewRouter()
	mw := newTestMainWindow(fyneApp, router)

	originalContent := mw.CenterContent()
	s.NotNil(originalContent)

	router.NavigateTo(ui.ViewSettings)

	newContent := mw.CenterContent()
	s.NotNil(newContent, "CenterContent should not be nil after navigating to ViewSettings")
	s.NotEqual(originalContent, newContent,
		"Navigating to ViewSettings should swap the center pane to different content")
}

func (s *ThreeColumnLayoutSuite) TestViewPlanShowsPlannerViewWhenVMsProvided() {
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
		nil, // rp
		config.OllamaConfig{},
		nil, // characterWidget
		router,
		vm,  // plannerVM
		vm,  // timerVM
		nil, // wizardVM
		nil, // rightPanelOverride
	)

	router.NavigateTo(ui.ViewPlan)

	content := mw.CenterContent()
	s.Require().NotNil(content, "CenterContent should not be nil after navigating to ViewPlan")

	// When PlannerViewModel and TimerViewModel are provided, ViewPlan should
	// show a real PlannerView container, NOT a placeholder label.
	_, isLabel := content.(*widget.Label)
	s.False(isLabel, "ViewPlan content should be a *fyne.Container from PlannerView, not a *widget.Label placeholder")
}

// stubWizardVM satisfies WizardViewModel with zero-value returns for all methods.
// Separate from stubPlannerTimerVM for clarity.
type stubWizardVM struct{}

func (s *stubWizardVM) CurrentStep() presenter.WizardStep                { return presenter.StepIdle }
func (s *stubWizardVM) FocusSchedule() *presenter.SchedulePreview        { return nil }
func (s *stubWizardVM) RecoverySchedule() *presenter.SchedulePreview     { return nil }
func (s *stubWizardVM) StartPlanning(_ context.Context) error            { return nil }
func (s *stubWizardVM) PreviousStep()                                    {}
func (s *stubWizardVM) SelectSchedule(_ context.Context, _ string) error { return nil }

func (s *ThreeColumnLayoutSuite) TestViewWizardShowsWizardViewWhenVMProvided() {
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
		nil, // rp
		config.OllamaConfig{},
		nil, // characterWidget
		router,
		nil, // plannerVM
		nil, // timerVM
		wvm, // wizardVM
		nil, // rightPanelOverride
	)

	router.NavigateTo(ui.ViewWizard)

	content := mw.CenterContent()
	s.Require().NotNil(content, "CenterContent should not be nil after navigating to ViewWizard")

	// When WizardViewModel is provided, ViewWizard should show a real
	// WizardView container, NOT a placeholder label.
	_, isLabel := content.(*widget.Label)
	s.False(isLabel, "ViewWizard content should be a *fyne.Container from WizardView, not a *widget.Label placeholder")
}

func (s *ThreeColumnLayoutSuite) TestNavigateBackToCharacterRestoresContent() {
	fyneApp := test.NewApp()
	router := ui.NewCenterViewRouter()
	mw := newTestMainWindow(fyneApp, router)

	characterContent := mw.CenterContent()
	s.NotNil(characterContent)

	// Navigate away to Plan view.
	router.NavigateTo(ui.ViewPlan)
	planContent := mw.CenterContent()
	s.NotEqual(characterContent, planContent,
		"Plan content should differ from character content")

	// Navigate back to Character view.
	router.NavigateTo(ui.ViewCharacter)
	restoredContent := mw.CenterContent()
	s.NotNil(restoredContent, "CenterContent should not be nil after returning to ViewCharacter")
	s.NotEqual(planContent, restoredContent,
		"Returning to ViewCharacter should swap away from plan content")
}

func (s *ThreeColumnLayoutSuite) TestRightPanelOverrideReplacesNotificationPanel() {
	fyneApp := test.NewApp()
	router := ui.NewCenterViewRouter()
	cfg := config.GUIConfig{
		WindowWidth:  1200,
		WindowHeight: 800,
	}

	customPanel := widget.NewLabel("UAT Panel")

	mw := ui.NewMainWindow(
		fyneApp, cfg,
		(*presenter.NotificationPresenter)(nil),
		(*presenter.ActivityPresenter)(nil),
		(*presenter.FeedbackPresenter)(nil),
		(*presenter.AppPresenter)(nil),
		(*presenter.SettingsPresenter)(nil),
		(*presenter.ServiceSettingsPresenter)(nil),
		nil, // rp
		config.OllamaConfig{},
		nil, router,
		nil, nil, nil,
		customPanel, // right panel override
	)

	s.NotNil(mw)
	// The custom panel should appear in the widget tree.
	found := findWidget(mw.Content(), customPanel)
	s.True(found, "right panel override should appear in the main window content tree")
}

func (s *ThreeColumnLayoutSuite) TestSetCharacterWidgetSwapsCenterContent() {
	fyneApp := test.NewApp()
	router := ui.NewCenterViewRouter()
	mw := newTestMainWindow(fyneApp, router)

	originalContent := mw.CenterContent()
	s.NotNil(originalContent)

	newWidget := widget.NewLabel("new character widget")
	mw.SetCharacterWidget(newWidget)

	updatedContent := mw.CenterContent()
	s.Equal(newWidget, updatedContent,
		"SetCharacterWidget should replace the character view content")
}

func (s *ThreeColumnLayoutSuite) TestSetCharacterWidgetWhileOnPlanDoesNotAffectPlan() {
	fyneApp := test.NewApp()
	router := ui.NewCenterViewRouter()
	mw := newTestMainWindow(fyneApp, router)

	router.NavigateTo(ui.ViewPlan)
	planContent := mw.CenterContent()

	newWidget := widget.NewLabel("new character widget")
	mw.SetCharacterWidget(newWidget)

	// Plan view should still be active and unchanged.
	s.Equal(planContent, mw.CenterContent(),
		"SetCharacterWidget while on Plan view should not change current center content")

	// Navigate back to character — should show the new widget.
	router.NavigateTo(ui.ViewCharacter)
	s.Equal(newWidget, mw.CenterContent(),
		"After navigating back to character, the new widget should be shown")
}

// findWidget recursively checks whether target exists in the widget tree rooted at root.
func findWidget(root fyne.CanvasObject, target fyne.CanvasObject) bool {
	if root == target {
		return true
	}
	if c, ok := root.(*fyne.Container); ok {
		for _, child := range c.Objects {
			if findWidget(child, target) {
				return true
			}
		}
	}
	if sp, ok := root.(*container.Split); ok {
		return findWidget(sp.Leading, target) || findWidget(sp.Trailing, target)
	}
	return false
}
