package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/CreateFutureMWilkinson/cue/internal/service/planner"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
)

// PlannerViewModel abstracts the planner presenter for the view.
type PlannerViewModel interface {
	CurrentStep() presenter.WizardStep
	HasActivePlan() bool
	AvailableTasks() []presenter.TodoRow
	Estimates() []presenter.TaskEstimateRow
	EstimateSummary() presenter.EstimateSummary
	FocusSchedule() *presenter.SchedulePreview
	RecoverySchedule() *presenter.SchedulePreview
	ActiveSchedule() *presenter.ActiveScheduleState
}

// TimerViewModel abstracts the timer presenter for the view.
type TimerViewModel interface {
	IsRunning() bool
	ActiveSegment() int
	ElapsedFraction() float64
	IsFlashVisible() bool
	CurrentTaskName() string
	BlockType() planner.BlockType
}

// PlannerView is the Fyne component for the day planner wizard and active schedule.
type PlannerView struct {
	plannerModel PlannerViewModel
	timerModel   TimerViewModel

	planBtn         *widget.Button
	nextBtn         *widget.Button
	backBtn         *widget.Button
	completeTaskBtn *widget.Button
	abandonBtn      *widget.Button

	container *fyne.Container
}

// NewPlannerView creates a new PlannerView bound to the given view models.
func NewPlannerView(plannerModel PlannerViewModel, timerModel TimerViewModel) *PlannerView {
	v := &PlannerView{
		plannerModel: plannerModel,
		timerModel:   timerModel,
	}

	v.planBtn = widget.NewButton("Plan My Day", func() {})
	v.nextBtn = widget.NewButton("Next", func() {})
	v.backBtn = widget.NewButton("Back", func() {})
	v.completeTaskBtn = widget.NewButton("Complete Task", func() {})
	v.abandonBtn = widget.NewButton("Abandon Plan", func() {})

	v.applyVisibility()

	v.container = container.NewVBox(
		v.planBtn,
		v.nextBtn,
		v.backBtn,
		v.completeTaskBtn,
		v.abandonBtn,
	)

	return v
}

// Container returns the top-level Fyne container for the planner view.
func (v *PlannerView) Container() *fyne.Container {
	return v.container
}

// PlanButton returns the "Plan My Day" button.
func (v *PlannerView) PlanButton() *widget.Button { return v.planBtn }

// NextButton returns the "Next" wizard step button.
func (v *PlannerView) NextButton() *widget.Button { return v.nextBtn }

// BackButton returns the "Back" wizard step button.
func (v *PlannerView) BackButton() *widget.Button { return v.backBtn }

// CompleteTaskButton returns the "Complete Task" button.
func (v *PlannerView) CompleteTaskButton() *widget.Button { return v.completeTaskBtn }

// AbandonButton returns the "Abandon Plan" button.
func (v *PlannerView) AbandonButton() *widget.Button { return v.abandonBtn }

// SetPlannerModel replaces the planner view model.
func (v *PlannerView) SetPlannerModel(model PlannerViewModel) {
	v.plannerModel = model
}

// SetTimerModel replaces the timer view model.
func (v *PlannerView) SetTimerModel(model TimerViewModel) {
	v.timerModel = model
}

// Refresh updates button visibility based on the current wizard step.
func (v *PlannerView) Refresh() {
	v.applyVisibility()
}

// applyVisibility sets button show/hide state based on the current wizard step.
func (v *PlannerView) applyVisibility() {
	step := v.plannerModel.CurrentStep()

	// PlanButton: visible only at StepIdle
	if step == presenter.StepIdle {
		v.planBtn.Show()
	} else {
		v.planBtn.Hide()
	}

	// NextButton: visible at StepTaskSelect, StepEstimates, StepPriority
	switch step {
	case presenter.StepTaskSelect, presenter.StepEstimates, presenter.StepPriority:
		v.nextBtn.Show()
	default:
		v.nextBtn.Hide()
	}

	// BackButton: visible at StepTaskSelect, StepEstimates, StepPriority, StepSchedule
	switch step {
	case presenter.StepTaskSelect, presenter.StepEstimates, presenter.StepPriority, presenter.StepSchedule:
		v.backBtn.Show()
	default:
		v.backBtn.Hide()
	}

	// CompleteTaskButton: visible only at StepActive
	if step == presenter.StepActive {
		v.completeTaskBtn.Show()
	} else {
		v.completeTaskBtn.Hide()
	}

	// AbandonButton: visible only at StepActive
	if step == presenter.StepActive {
		v.abandonBtn.Show()
	} else {
		v.abandonBtn.Hide()
	}
}
