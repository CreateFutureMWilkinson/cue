//go:build uitest

package ui_test

import (
	"strings"
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

// --- Step 2 widget rendering ---

func (s *WizardViewAcceptanceSuite) TestWizardStep2ContainsEstimateEntries() {
	s.setupStep2Defaults()

	view := ui.NewWizardView(s.vm, s.router)
	root := view.Container()

	entries := uitest.FindAll[*widget.Entry](root, func(_ *widget.Entry) bool {
		return true
	})

	s.GreaterOrEqual(len(entries), 2,
		"Step 2 container should contain at least 2 Entry widgets (one per estimate row from sampleEstimates)")
}

func (s *WizardViewAcceptanceSuite) TestWizardStep2HasSummaryLabel() {
	s.setupStep2Defaults()

	view := ui.NewWizardView(s.vm, s.router)
	root := view.Container()

	_, found := uitest.FindWidget[*widget.Label](root, func(l *widget.Label) bool {
		return strings.Contains(l.Text, "Pomodoros")
	})
	s.True(found, "Step 2 container should have a Label containing 'Pomodoros'")
}

func (s *WizardViewAcceptanceSuite) TestWizardStep2HasBackAndNextButtons() {
	s.setupStep2Defaults()

	view := ui.NewWizardView(s.vm, s.router)
	root := view.Container()

	_, foundBack := uitest.FindWidget[*widget.Button](root, func(b *widget.Button) bool {
		return b.Text == "Back"
	})
	s.True(foundBack, "Step 2 container should have a 'Back' button widget")

	_, foundNext := uitest.FindWidget[*widget.Button](root, func(b *widget.Button) bool {
		return b.Text == "Next"
	})
	s.True(foundNext, "Step 2 container should have a 'Next' button widget")
}

// --- Helpers ---

func (s *WizardViewAcceptanceSuite) setupStep2Defaults() {
	s.vm.On("CurrentStep").Return(presenter.StepEstimates).Maybe()
	s.vm.On("AvailableTasks").Return([]presenter.TodoRow{}).Maybe()
	s.vm.On("SelectedCount").Return(0).Maybe()
	s.vm.On("Estimates").Return(sampleEstimates()).Maybe()
	s.vm.On("EstimateSummary").Return(presenter.EstimateSummary{
		TotalPomos:      5,
		AvailableBlocks: 19,
		Overloaded:      false,
	}).Maybe()
	s.vm.On("FocusSchedule").Return((*presenter.SchedulePreview)(nil)).Maybe()
	s.vm.On("RecoverySchedule").Return((*presenter.SchedulePreview)(nil)).Maybe()
}

func (s *WizardViewAcceptanceSuite) setupStep1Defaults() {
	s.vm.On("CurrentStep").Return(presenter.StepTaskSelect).Maybe()
	s.vm.On("AvailableTasks").Return(sampleAvailableTasks()).Maybe()
	s.vm.On("SelectedCount").Return(2).Maybe()
	s.vm.On("Estimates").Return([]presenter.TaskEstimateRow{}).Maybe()
	s.vm.On("EstimateSummary").Return(presenter.EstimateSummary{}).Maybe()
	s.vm.On("FocusSchedule").Return((*presenter.SchedulePreview)(nil)).Maybe()
	s.vm.On("RecoverySchedule").Return((*presenter.SchedulePreview)(nil)).Maybe()
}
