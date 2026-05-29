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
// contains the expected Fyne widgets for the schedule-only wizard flow
// introduced by Feature 107 WP5.
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

func (s *WizardViewAcceptanceSuite) setupScheduleDefaults() {
	s.vm.On("CurrentStep").Return(presenter.StepSchedule).Maybe()
	s.vm.On("FocusSchedule").Return(sampleFocusSchedule()).Maybe()
	s.vm.On("RecoverySchedule").Return(sampleRecoverySchedule()).Maybe()
}

func (s *WizardViewAcceptanceSuite) TestScheduleHasSelectionButtons() {
	s.setupScheduleDefaults()

	view := ui.NewWizardView(s.vm, s.router)
	root := view.Container()

	buttons := uitest.FindAll[*widget.Button](root, func(b *widget.Button) bool {
		return strings.Contains(b.Text, "focus-maximized") ||
			strings.Contains(b.Text, "recovery-balanced")
	})

	s.GreaterOrEqual(len(buttons), 2,
		"schedule step should contain at least 2 Select buttons (one per strategy)")
}

func (s *WizardViewAcceptanceSuite) TestScheduleHasBackButton() {
	s.setupScheduleDefaults()

	view := ui.NewWizardView(s.vm, s.router)
	root := view.Container()

	_, found := uitest.FindWidget[*widget.Button](root, func(b *widget.Button) bool {
		return b.Text == "Back"
	})
	s.True(found, "schedule step should have a 'Back' button")
}

func (s *WizardViewAcceptanceSuite) TestScheduleHasNoNextButton() {
	s.setupScheduleDefaults()

	view := ui.NewWizardView(s.vm, s.router)
	root := view.Container()

	_, found := uitest.FindWidget[*widget.Button](root, func(b *widget.Button) bool {
		return b.Text == "Next"
	})
	s.False(found, "schedule step should NOT have a 'Next' button")
}
