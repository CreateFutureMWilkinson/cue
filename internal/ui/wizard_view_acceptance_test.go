//go:build uitest

package ui_test

import (
	"testing"

	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/ui"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/uitest"
)

// WizardViewAcceptanceSuite verifies that the WizardView container tree
// contains the expected Fyne widgets for each wizard step.
type WizardViewAcceptanceSuite struct {
	suite.Suite
	vm     *mockWizardViewModel
	router *ui.CenterViewRouter
}

func TestWizardViewAcceptance(t *testing.T) {
	suite.Run(t, new(WizardViewAcceptanceSuite))
}

func (s *WizardViewAcceptanceSuite) SetupTest() {
	s.vm = new(mockWizardViewModel)
	s.router = ui.NewCenterViewRouter()
}

// --- Step 1 widget rendering ---

func (s *WizardViewAcceptanceSuite) TestWizardStep1ContainsCheckboxes() {
	s.setupStep1Defaults()

	view := ui.NewWizardView(s.vm, s.router)
	root := view.Container()

	checks := uitest.FindAll[*widget.Check](root, func(_ *widget.Check) bool {
		return true
	})

	s.GreaterOrEqual(len(checks), 3,
		"Step 1 container should contain at least 3 Check widgets (one per task from sampleAvailableTasks)")
}

func (s *WizardViewAcceptanceSuite) TestWizardStep1HasNavigationButtons() {
	s.setupStep1Defaults()

	view := ui.NewWizardView(s.vm, s.router)
	root := view.Container()

	_, foundNext := uitest.FindWidget[*widget.Button](root, func(b *widget.Button) bool {
		return b.Text == "Next"
	})
	s.True(foundNext, "Step 1 container should have a 'Next' button widget")

	_, foundCancel := uitest.FindWidget[*widget.Button](root, func(b *widget.Button) bool {
		return b.Text == "Cancel"
	})
	s.True(foundCancel, "Step 1 container should have a 'Cancel' button widget")
}

func (s *WizardViewAcceptanceSuite) TestWizardStep1HasInlineCreation() {
	s.setupStep1Defaults()

	view := ui.NewWizardView(s.vm, s.router)
	root := view.Container()

	entries := uitest.FindAll[*widget.Entry](root, func(_ *widget.Entry) bool {
		return true
	})

	s.GreaterOrEqual(len(entries), 1,
		"Step 1 container should have at least 1 Entry widget for inline task creation")
}

// --- Helper: reuse step 1 defaults from WizardViewSuite ---

func (s *WizardViewAcceptanceSuite) setupStep1Defaults() {
	s.vm.On("CurrentStep").Return(presenter.StepTaskSelect).Maybe()
	s.vm.On("AvailableTasks").Return(sampleAvailableTasks()).Maybe()
	s.vm.On("SelectedCount").Return(2).Maybe()
	s.vm.On("Estimates").Return([]presenter.TaskEstimateRow{}).Maybe()
	s.vm.On("EstimateSummary").Return(presenter.EstimateSummary{}).Maybe()
	s.vm.On("FocusSchedule").Return((*presenter.SchedulePreview)(nil)).Maybe()
	s.vm.On("RecoverySchedule").Return((*presenter.SchedulePreview)(nil)).Maybe()
}
