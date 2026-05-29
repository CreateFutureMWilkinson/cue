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
// StepTaskSelect, StepEstimates, and StepPriority are gone — the
// wizard never edits todos. Todo editing lives in the Plan view.
//
// These tests are written before the implementation lands; they are
// expected to fail until WP5's view rewrite is in place. They are the
// outer verification gate per CLAUDE.md's UI Feature Workflow.
type SimplifiedWizardAcceptanceSuite struct {
	suite.Suite
}

func TestSimplifiedWizardAcceptance(t *testing.T) {
	suite.Run(t, new(SimplifiedWizardAcceptanceSuite))
}

// AC: StepTaskSelect must no longer render task checkboxes — the
// wizard never edits todos. Today the view renders a *widget.Check per
// task; this test fails until the legacy step is removed from the view.
func (s *SimplifiedWizardAcceptanceSuite) TestStepTaskSelectNoLongerRendersCheckboxes() {
	router := ui.NewCenterViewRouter()
	wvm := &stubWizardVM{
		step: presenter.StepTaskSelect,
		tasks: []presenter.TodoRow{
			{Title: "Task 1", Priority: 1},
			{Title: "Task 2", Priority: 2},
		},
		selectedCount: 1,
	}
	wv := ui.NewWizardView(wvm, router)
	root := wv.Container()

	checks := uitest.FindAll[*widget.Check](root, func(_ *widget.Check) bool { return true })
	s.Empty(checks,
		"StepTaskSelect must not render task checkboxes after WP5 — the wizard does not edit todos")
}

// AC: StepTaskSelect must no longer render an "Add Task" button — task
// creation lives in the Plan view, not the wizard.
func (s *SimplifiedWizardAcceptanceSuite) TestStepTaskSelectNoLongerHasAddTaskButton() {
	router := ui.NewCenterViewRouter()
	wvm := &stubWizardVM{step: presenter.StepTaskSelect}
	wv := ui.NewWizardView(wvm, router)
	root := wv.Container()

	_, found := uitest.FindWidget[*widget.Button](root, func(b *widget.Button) bool {
		return b.Text == "Add Task"
	})
	s.False(found,
		"the wizard must not expose an Add Task button after WP5")
}

// AC: StepEstimates must not render the legacy estimates table even
// when the VM reports estimate rows. Today the view renders a label +
// entry per row; this test fails until the legacy step is removed from
// the view.
func (s *SimplifiedWizardAcceptanceSuite) TestStepEstimatesNoLongerRendersEntries() {
	router := ui.NewCenterViewRouter()
	wvm := &stubWizardVM{
		step: presenter.StepEstimates,
		estimates: []presenter.TaskEstimateRow{
			{Title: "Task 1", EstimatedPomos: 2, EffectivePomos: 2},
		},
	}
	wv := ui.NewWizardView(wvm, router)
	root := wv.Container()

	entries := uitest.FindAll[*widget.Entry](root, func(_ *widget.Entry) bool { return true })
	s.Empty(entries,
		"StepEstimates must not render any estimate entry fields after WP5")
}

// AC: StepPriority must not render Up/Down reorder buttons — task
// reordering lives in the Plan view's todo list, not the wizard.
// Today the view renders Up + Down per row; this test fails until the
// legacy step is removed.
func (s *SimplifiedWizardAcceptanceSuite) TestStepPriorityNoLongerRendersReorderButtons() {
	router := ui.NewCenterViewRouter()
	wvm := &stubWizardVM{
		step: presenter.StepPriority,
		// Today the wizard reads priority rows from Estimates(), so
		// providing estimate rows is what triggers the legacy Up/Down
		// render path. WP5 must remove that path entirely.
		estimates: []presenter.TaskEstimateRow{
			{Title: "Task 1", EstimatedPomos: 1, EffectivePomos: 1},
			{Title: "Task 2", EstimatedPomos: 1, EffectivePomos: 1},
		},
	}
	wv := ui.NewWizardView(wvm, router)
	root := wv.Container()

	_, foundUp := uitest.FindWidget[*widget.Button](root, func(b *widget.Button) bool {
		return b.Text == "Up"
	})
	_, foundDown := uitest.FindWidget[*widget.Button](root, func(b *widget.Button) bool {
		return b.Text == "Down"
	})
	s.False(foundUp, "StepPriority must not render an Up reorder button after WP5")
	s.False(foundDown, "StepPriority must not render a Down reorder button after WP5")
}

// AC: StepSchedule remains the user's first interactive step. The two
// schedule preview cards must render with Select buttons. (This test
// already passes against the current wizard; it stays here as a
// regression guard so WP5's simplification can't accidentally break
// the schedule-pick flow.)
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
