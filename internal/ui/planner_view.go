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
// It displays different sets of buttons based on the current wizard step or active plan state.
type PlannerView struct {
	plannerModel PlannerViewModel
	timerModel   TimerViewModel

	// Navigation and control buttons
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

	v.initializeButtons()
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

// initializeButtons creates all the buttons with their default text and empty callbacks.
func (v *PlannerView) initializeButtons() {
	v.planBtn = widget.NewButton("Plan My Day", func() {})
	v.nextBtn = widget.NewButton("Next", func() {})
	v.backBtn = widget.NewButton("Back", func() {})
	v.completeTaskBtn = widget.NewButton("Complete Task", func() {})
	v.abandonBtn = widget.NewButton("Abandon Plan", func() {})
}

// Container returns the top-level Fyne container for the planner view.
func (v *PlannerView) Container() *fyne.Container {
	return v.container
}

// PlanButton returns the "Plan My Day" button for initiating the day planning wizard.
func (v *PlannerView) PlanButton() *widget.Button { return v.planBtn }

// NextButton returns the "Next" button for progressing through wizard steps.
func (v *PlannerView) NextButton() *widget.Button { return v.nextBtn }

// BackButton returns the "Back" button for returning to previous wizard steps.
func (v *PlannerView) BackButton() *widget.Button { return v.backBtn }

// CompleteTaskButton returns the "Complete Task" button for finishing the current active task.
func (v *PlannerView) CompleteTaskButton() *widget.Button { return v.completeTaskBtn }

// AbandonButton returns the "Abandon Plan" button for cancelling the active schedule.
func (v *PlannerView) AbandonButton() *widget.Button { return v.abandonBtn }

// SetPlannerModel replaces the planner view model and updates button visibility.
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

	// Configure visibility for each button based on the current step
	v.setButtonVisibility(v.planBtn, v.isPlanButtonVisible(step))
	v.setButtonVisibility(v.nextBtn, v.isNextButtonVisible(step))
	v.setButtonVisibility(v.backBtn, v.isBackButtonVisible(step))
	v.setButtonVisibility(v.completeTaskBtn, v.isActiveStepButton(step))
	v.setButtonVisibility(v.abandonBtn, v.isActiveStepButton(step))
}

// setButtonVisibility shows or hides a button based on the visible flag.
func (v *PlannerView) setButtonVisibility(button *widget.Button, visible bool) {
	if visible {
		button.Show()
	} else {
		button.Hide()
	}
}

// isPlanButtonVisible returns true if the Plan button should be visible for the given step.
func (v *PlannerView) isPlanButtonVisible(step presenter.WizardStep) bool {
	return step == presenter.StepIdle
}

// isNextButtonVisible returns true if the Next button should be visible for the given step.
func (v *PlannerView) isNextButtonVisible(step presenter.WizardStep) bool {
	return step == presenter.StepTaskSelect ||
		step == presenter.StepEstimates ||
		step == presenter.StepPriority
}

// isBackButtonVisible returns true if the Back button should be visible for the given step.
func (v *PlannerView) isBackButtonVisible(step presenter.WizardStep) bool {
	return step == presenter.StepTaskSelect ||
		step == presenter.StepEstimates ||
		step == presenter.StepPriority ||
		step == presenter.StepSchedule
}

// isActiveStepButton returns true if active step buttons (Complete/Abandon) should be visible.
func (v *PlannerView) isActiveStepButton(step presenter.WizardStep) bool {
	return step == presenter.StepActive
}
