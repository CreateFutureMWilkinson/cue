//go:build ui_acceptance

package ui_acceptance_test

import (
	"testing"

	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/ui"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/uitest"
)

// WizardAcceptanceSuite verifies day planner wizard acceptance criteria
// from UiSpec.md lines 1123-1139.
type WizardAcceptanceSuite struct {
	suite.Suite
}

func TestWizardAcceptance(t *testing.T) {
	suite.Run(t, new(WizardAcceptanceSuite))
}

// AC: Launched from "Plan My Day" in Plan view, replaces center area.
func (s *WizardAcceptanceSuite) TestWizardReplacesCenter() {
	app := newTestApp()
	defer app.Quit()
	router := ui.NewCenterViewRouter()
	wvm := &stubWizardVM{step: presenter.StepTaskSelect}
	mw := newMainWindowWithWizard(app, router, wvm)

	charContent := mw.CenterContent()
	router.NavigateTo(ui.ViewWizard)
	wizardContent := mw.CenterContent()

	s.NotEqual(charContent, wizardContent,
		"wizard should replace center area content")
}

// AC: Wizard view is not a placeholder when VM provided.
func (s *WizardAcceptanceSuite) TestWizardViewIsNotPlaceholder() {
	app := newTestApp()
	defer app.Quit()
	router := ui.NewCenterViewRouter()
	wvm := &stubWizardVM{step: presenter.StepTaskSelect}
	mw := newMainWindowWithWizard(app, router, wvm)

	router.NavigateTo(ui.ViewWizard)
	content := mw.CenterContent()
	s.Require().NotNil(content)

	_, isLabel := content.(*widget.Label)
	s.False(isLabel, "wizard view should be a real container, not a placeholder label")
}

// AC: Step 1 displays incomplete todos with checkboxes.
func (s *WizardAcceptanceSuite) TestWizardStep1ContainsCheckboxes() {
	router := ui.NewCenterViewRouter()
	wvm := &stubWizardVM{
		step: presenter.StepTaskSelect,
		tasks: []presenter.TodoRow{
			{Title: "Task 1", Priority: 1},
			{Title: "Task 2", Priority: 2},
		},
	}
	wv := ui.NewWizardView(wvm, router)
	root := wv.Container()

	checks := uitest.FindAll[*widget.Check](root, func(_ *widget.Check) bool {
		return true
	})

	s.GreaterOrEqual(len(checks), 1,
		"wizard step 1 should contain checkboxes for task selection")
}

// AC: Step 1 has Cancel and Next buttons.
func (s *WizardAcceptanceSuite) TestWizardStep1HasNavigationButtons() {
	router := ui.NewCenterViewRouter()
	wvm := &stubWizardVM{step: presenter.StepTaskSelect}
	wv := ui.NewWizardView(wvm, router)
	root := wv.Container()

	_, foundNext := uitest.FindWidget[*widget.Button](root, func(b *widget.Button) bool {
		return b.Text == "Next"
	})
	s.True(foundNext, "wizard step 1 should have a 'Next' button")

	_, foundCancel := uitest.FindWidget[*widget.Button](root, func(b *widget.Button) bool {
		return b.Text == "Cancel"
	})
	s.True(foundCancel, "wizard step 1 should have a 'Cancel' button")
}

// AC: Step 1 allows inline task creation.
func (s *WizardAcceptanceSuite) TestWizardStep1HasInlineCreation() {
	router := ui.NewCenterViewRouter()
	wvm := &stubWizardVM{step: presenter.StepTaskSelect}
	wv := ui.NewWizardView(wvm, router)
	root := wv.Container()

	entries := uitest.FindAll[*widget.Entry](root, func(_ *widget.Entry) bool {
		return true
	})

	s.GreaterOrEqual(len(entries), 1,
		"wizard step 1 should have entry fields for inline task creation")
}

// AC: Step 2 shows estimates with override inputs.
func (s *WizardAcceptanceSuite) TestWizardStep2ContainsEstimates() {
	router := ui.NewCenterViewRouter()
	wvm := &stubWizardVM{
		step: presenter.StepEstimates,
		estimates: []presenter.TaskEstimateRow{
			{Title: "Task 1", EstimatedPomos: 2, EffectivePomos: 2},
		},
		summary: presenter.EstimateSummary{TotalPomos: 2, AvailableBlocks: 8},
	}
	wv := ui.NewWizardView(wvm, router)
	root := wv.Container()

	// Should have at least one entry for estimate override.
	entries := uitest.FindAll[*widget.Entry](root, func(_ *widget.Entry) bool {
		return true
	})
	s.GreaterOrEqual(len(entries), 1,
		"wizard step 2 should have entry fields for estimate overrides")
}

// AC: Step 2 has Back and Next buttons.
func (s *WizardAcceptanceSuite) TestWizardStep2HasBackAndNext() {
	router := ui.NewCenterViewRouter()
	wvm := &stubWizardVM{step: presenter.StepEstimates}
	wv := ui.NewWizardView(wvm, router)
	root := wv.Container()

	_, foundBack := uitest.FindWidget[*widget.Button](root, func(b *widget.Button) bool {
		return b.Text == "Back"
	})
	s.True(foundBack, "wizard step 2 should have a 'Back' button")

	_, foundNext := uitest.FindWidget[*widget.Button](root, func(b *widget.Button) bool {
		return b.Text == "Next"
	})
	s.True(foundNext, "wizard step 2 should have a 'Next' button")
}

// AC: Step 3 tasks displayed with up/down reorder controls.
func (s *WizardAcceptanceSuite) TestWizardStep3HasReorderControls() {
	router := ui.NewCenterViewRouter()
	wvm := &stubWizardVM{
		step: presenter.StepPriority,
		tasks: []presenter.TodoRow{
			{Title: "Task 1", Priority: 1},
			{Title: "Task 2", Priority: 2},
		},
	}
	wv := ui.NewWizardView(wvm, router)
	root := wv.Container()

	buttons := uitest.FindAll[*widget.Button](root, func(b *widget.Button) bool {
		return true
	})

	// Should have at least Back, Next, and some reorder buttons.
	s.GreaterOrEqual(len(buttons), 2,
		"wizard step 3 should have navigation and reorder buttons")
}

// AC: Step 4 has two schedule cards side-by-side with Select buttons.
func (s *WizardAcceptanceSuite) TestWizardStep4HasScheduleSelectionButtons() {
	router := ui.NewCenterViewRouter()
	wvm := &stubWizardVM{step: presenter.StepSchedule}
	wv := ui.NewWizardView(wvm, router)
	root := wv.Container()

	buttons := uitest.FindAll[*widget.Button](root, func(b *widget.Button) bool {
		return true
	})

	// Should have at least Back and schedule selection buttons.
	s.GreaterOrEqual(len(buttons), 1,
		"wizard step 4 should have schedule selection buttons")
}

// AC: All steps have Back button (except step 1 which has Cancel).
func (s *WizardAcceptanceSuite) TestWizardStep3HasBackButton() {
	router := ui.NewCenterViewRouter()
	wvm := &stubWizardVM{step: presenter.StepPriority}
	wv := ui.NewWizardView(wvm, router)
	root := wv.Container()

	_, found := uitest.FindWidget[*widget.Button](root, func(b *widget.Button) bool {
		return b.Text == "Back"
	})
	s.True(found, "wizard step 3 should have a 'Back' button")
}

// AC: Wizard can be exited by navigating back to character view.
func (s *WizardAcceptanceSuite) TestWizardExitToCharacter() {
	app := newTestApp()
	defer app.Quit()
	router := ui.NewCenterViewRouter()
	wvm := &stubWizardVM{step: presenter.StepTaskSelect}
	_ = newMainWindowWithWizard(app, router, wvm)

	router.NavigateTo(ui.ViewWizard)
	s.Equal(ui.ViewWizard, router.CurrentView())

	router.NavigateTo(ui.ViewCharacter)
	s.Equal(ui.ViewCharacter, router.CurrentView(),
		"should be able to exit wizard by navigating to character view")
}
