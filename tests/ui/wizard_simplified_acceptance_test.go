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

// SimplifiedWizardAcceptanceSuite captures the Feature 107 WP5 wizard
// simplification. The wizard collapses to a schedule-generation flow:
//
//	StepIdle (prompt)
//	   ↓ "Plan My Day" → StartPlanning(ctx) calls the generator
//	StepSchedule (focus + recovery cards with Select buttons)
//	   ↓ pick a strategy
//	StepActive (active schedule + current-focus-task hint)
//
// The legacy task-edit / estimates / priority steps are gone — the
// wizard never edits todos. Todo editing lives in the Plan view.
type SimplifiedWizardAcceptanceSuite struct {
	suite.Suite
}

func TestSimplifiedWizardAcceptance(t *testing.T) {
	suite.Run(t, new(SimplifiedWizardAcceptanceSuite))
}

// AC: The wizard never renders task checkboxes — it does not edit todos.
func (s *SimplifiedWizardAcceptanceSuite) TestWizardNeverRendersTaskCheckboxes() {
	for _, step := range []presenter.WizardStep{
		presenter.StepIdle,
		presenter.StepSchedule,
		presenter.StepActive,
	} {
		router := ui.NewCenterViewRouter()
		wvm := &stubWizardVM{step: step}
		wv := ui.NewWizardView(wvm, router)
		root := wv.Container()

		checks := uitest.FindAll[*widget.Check](root, func(_ *widget.Check) bool { return true })
		s.Empty(checks,
			"WP5: wizard at step %d must not render task checkboxes — todos are managed in the Plan view", step)
	}
}

// AC: The wizard never renders an "Add Task" button — task creation lives
// in the Plan view.
func (s *SimplifiedWizardAcceptanceSuite) TestWizardNeverRendersAddTaskButton() {
	for _, step := range []presenter.WizardStep{
		presenter.StepIdle,
		presenter.StepSchedule,
		presenter.StepActive,
	} {
		router := ui.NewCenterViewRouter()
		wvm := &stubWizardVM{step: step}
		wv := ui.NewWizardView(wvm, router)
		root := wv.Container()

		_, found := uitest.FindWidget[*widget.Button](root, func(b *widget.Button) bool {
			return b.Text == "Add Task"
		})
		s.False(found,
			"WP5: wizard at step %d must not expose an Add Task button", step)
	}
}

// AC: The wizard never renders estimate entry fields — estimates are no
// longer edited in the wizard.
func (s *SimplifiedWizardAcceptanceSuite) TestWizardNeverRendersEstimateEntries() {
	for _, step := range []presenter.WizardStep{
		presenter.StepIdle,
		presenter.StepSchedule,
		presenter.StepActive,
	} {
		router := ui.NewCenterViewRouter()
		wvm := &stubWizardVM{step: step}
		wv := ui.NewWizardView(wvm, router)
		root := wv.Container()

		entries := uitest.FindAll[*widget.Entry](root, func(_ *widget.Entry) bool { return true })
		s.Empty(entries,
			"WP5: wizard at step %d must not render any entry fields", step)
	}
}

// AC: The wizard never renders Up/Down reorder buttons — task reordering
// lives in the Plan view's todo list.
func (s *SimplifiedWizardAcceptanceSuite) TestWizardNeverRendersReorderButtons() {
	for _, step := range []presenter.WizardStep{
		presenter.StepIdle,
		presenter.StepSchedule,
		presenter.StepActive,
	} {
		router := ui.NewCenterViewRouter()
		wvm := &stubWizardVM{step: step}
		wv := ui.NewWizardView(wvm, router)
		root := wv.Container()

		_, foundUp := uitest.FindWidget[*widget.Button](root, func(b *widget.Button) bool {
			return b.Text == "Up"
		})
		_, foundDown := uitest.FindWidget[*widget.Button](root, func(b *widget.Button) bool {
			return b.Text == "Down"
		})
		s.False(foundUp, "WP5: wizard at step %d must not render an Up reorder button", step)
		s.False(foundDown, "WP5: wizard at step %d must not render a Down reorder button", step)
	}
}

// AC: StepSchedule remains the user's first interactive step. The two
// schedule preview cards must render with Select buttons. (Regression
// guard.)
func (s *SimplifiedWizardAcceptanceSuite) TestStepScheduleRendersPreviewCards() {
	router := ui.NewCenterViewRouter()
	wvm := &stubWizardVM{
		step:         presenter.StepSchedule,
		focusPrev:    &presenter.SchedulePreview{Strategy: "focus-maximized"},
		recoveryPrev: &presenter.SchedulePreview{Strategy: "recovery-balanced"},
	}
	wv := ui.NewWizardView(wvm, router)
	root := wv.Container()

	_, foundFocus := uitest.FindWidget[*widget.Button](root, func(b *widget.Button) bool {
		return b.Text == "Select focus-maximized"
	})
	_, foundRecovery := uitest.FindWidget[*widget.Button](root, func(b *widget.Button) bool {
		return b.Text == "Select recovery-balanced"
	})
	s.True(foundFocus, "StepSchedule must show a Select button for the focus-maximized card")
	s.True(foundRecovery, "StepSchedule must show a Select button for the recovery-balanced card")
}
