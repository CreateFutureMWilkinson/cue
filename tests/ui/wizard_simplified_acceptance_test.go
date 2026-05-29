//go:build ui_acceptance

package ui_acceptance_test

import (
	"context"
	"testing"

	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/ui"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/uitest"
)

// SimplifiedWizardAcceptanceSuite captures the Feature 107 WP5 wizard
// simplification:
//
//   - StepTaskSelect + StepPriority merged into a single StepTodoEdit.
//   - StepEstimates is deleted (the LLM estimator is gone).
//   - The active-schedule view shows a current-focus-task hint pulled
//     from PlannerPresenter.CurrentFocusTask.
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

// AC: StepTodoEdit renders the editable todo list with reorder controls.
// This subsumes the old StepTaskSelect (checkboxes for plan inclusion)
// and StepPriority (up/down reorder); the new step shows the same list
// with both behaviors fused.
func (s *SimplifiedWizardAcceptanceSuite) TestStepTodoEditRendersTodoList() {
	router := ui.NewCenterViewRouter()
	wvm := &stubWizardVM{
		step: presenter.StepTodoEdit,
		tasks: []presenter.TodoRow{
			{Title: "Task A", Priority: 1},
			{Title: "Task B", Priority: 2},
		},
	}
	wv := ui.NewWizardView(wvm, router)
	root := wv.Container()

	// The merged step must render the task titles somewhere in the tree.
	labels := uitest.FindAll[*widget.Label](root, func(l *widget.Label) bool {
		return l.Text == "Task A" || l.Text == "Task B"
	})
	s.GreaterOrEqual(len(labels), 2,
		"StepTodoEdit must render every incomplete todo as a row")
}

// AC: StepTodoEdit shows up/down reorder controls (carrying over the
// behavior previously provided only by StepPriority).
func (s *SimplifiedWizardAcceptanceSuite) TestStepTodoEditHasReorderControls() {
	router := ui.NewCenterViewRouter()
	wvm := &stubWizardVM{
		step: presenter.StepTodoEdit,
		tasks: []presenter.TodoRow{
			{Title: "Task A", Priority: 1},
			{Title: "Task B", Priority: 2},
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
	s.True(foundUp, "StepTodoEdit must expose an Up reorder button")
	s.True(foundDown, "StepTodoEdit must expose a Down reorder button")
}

// AC: StepTodoEdit advances directly to the schedule preview — there
// is no intermediate StepEstimates view.
func (s *SimplifiedWizardAcceptanceSuite) TestStepTodoEditNextGoesStraightToSchedule() {
	router := ui.NewCenterViewRouter()
	wvm := &recordingWizardVM{stubWizardVM: stubWizardVM{
		step: presenter.StepTodoEdit,
		tasks: []presenter.TodoRow{
			{Title: "Task A", Priority: 1},
		},
		selectedCount: 1,
	}}
	wv := ui.NewWizardView(wvm, router)
	root := wv.Container()

	nextBtn, found := uitest.FindWidget[*widget.Button](root, func(b *widget.Button) bool {
		return b.Text == "Next"
	})
	s.Require().True(found, "StepTodoEdit must have a Next button")
	nextBtn.OnTapped()

	// The presenter contract guarantees the next step from
	// StepTodoEdit is StepSchedule (no intermediate Estimates).
	s.Equal(1, wvm.nextCalls,
		"clicking Next on StepTodoEdit should advance the wizard exactly once")
}

// AC: StepEstimates is no longer rendered as a wizard step. If a
// view-model erroneously reports it, the wizard renders nothing
// step-specific (i.e. behaves like an unknown/idle state) rather than
// the legacy estimates table.
func (s *SimplifiedWizardAcceptanceSuite) TestStepEstimatesIsNoLongerRendered() {
	router := ui.NewCenterViewRouter()
	wvm := &stubWizardVM{
		step: presenter.StepEstimates,
		estimates: []presenter.TaskEstimateRow{
			{Title: "Should not render", EstimatedPomos: 1, EffectivePomos: 1},
		},
	}
	wv := ui.NewWizardView(wvm, router)
	root := wv.Container()

	_, found := uitest.FindWidget[*widget.Label](root, func(l *widget.Label) bool {
		return l.Text == "Should not render"
	})
	s.False(found, "StepEstimates must not render any task estimate rows after WP5")
}

// recordingWizardVM is a stubWizardVM that counts NextStep calls so
// tests can assert the wizard advanced.
type recordingWizardVM struct {
	stubWizardVM
	nextCalls int
}

func (r *recordingWizardVM) NextStep(_ context.Context) error {
	r.nextCalls++
	return nil
}
