package ui

import (
	"fmt"
	"math/rand"
	"time"

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

// placeholderMessages are the possible messages shown when there is no active plan.
var placeholderMessages = []string{
	"Who even knows",
	"It's your time you're wasting",
	"A goal without a plan is just a wish",
	"Winging it, are we?",
	"The plan is there is no plan",
	"Chaos is also a strategy, I suppose",
	"Bold of you to go planless",
}

// PlannerView is the Fyne component for the day planner wizard and active schedule.
// It displays different sets of buttons based on the current wizard step or active plan state.
type PlannerView struct {
	plannerModel PlannerViewModel
	timerModel   TimerViewModel
	router       *CenterViewRouter

	// Navigation and control buttons
	planBtn         *widget.Button
	nextBtn         *widget.Button
	backBtn         *widget.Button
	completeTaskBtn *widget.Button
	abandonBtn      *widget.Button

	// Todo list (trailing pane)
	todoList *TodoListView

	// Content state
	placeholderText string
	scheduleTree    *ScheduleTree
	centerContent   *fyne.Container

	container *fyne.Container
}

// NewPlannerView creates a new PlannerView bound to the given view models.
// The todoVM parameter, when non-nil, provides the data source for the trailing
// todo list pane in the horizontal split layout.
func NewPlannerView(plannerModel PlannerViewModel, timerModel TimerViewModel, router *CenterViewRouter, todoVM TodoListViewModel) *PlannerView {
	v := &PlannerView{
		plannerModel: plannerModel,
		timerModel:   timerModel,
		router:       router,
	}

	v.initializeButtons()
	v.applyVisibility()
	v.centerContent = container.NewStack()
	v.buildContent()

	if todoVM != nil {
		v.todoList = NewTodoListView(todoVM)
	}

	buttons := container.NewVBox(
		v.planBtn,
		v.nextBtn,
		v.backBtn,
		v.completeTaskBtn,
		v.abandonBtn,
	)

	leading := container.NewBorder(buttons, nil, nil, nil, v.centerContent)

	var trailing fyne.CanvasObject
	if v.todoList != nil {
		trailing = v.todoList.Container()
	} else {
		trailing = container.NewVBox()
	}

	split := container.NewHSplit(leading, trailing)
	v.container = container.NewStack(split)

	return v
}

// initializeButtons creates all the buttons with their default text and empty callbacks.
func (v *PlannerView) initializeButtons() {
	v.planBtn = widget.NewButton("Plan My Day", func() {
		if v.router != nil {
			v.router.NavigateTo(ViewWizard)
		}
	})
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

// PlaceholderText returns the placeholder message shown when there is no active plan.
// Returns empty string when there is an active plan.
func (v *PlannerView) PlaceholderText() string {
	return v.placeholderText
}

// ScheduleTree returns the schedule tree widget, or nil when there is no active plan.
func (v *PlannerView) ScheduleTree() *ScheduleTree {
	return v.scheduleTree
}

// SetPlannerModel replaces the planner view model and updates button visibility.
func (v *PlannerView) SetPlannerModel(model PlannerViewModel) {
	v.plannerModel = model
}

// SetTimerModel replaces the timer view model.
func (v *PlannerView) SetTimerModel(model TimerViewModel) {
	v.timerModel = model
}

// Refresh updates button visibility and content based on the current wizard step.
func (v *PlannerView) Refresh() {
	v.applyVisibility()
	v.buildContent()
}

// buildContent rebuilds the placeholder text and schedule tree from the current model state.
func (v *PlannerView) buildContent() {
	step := v.plannerModel.CurrentStep()
	hasActivePlan := v.plannerModel.HasActivePlan()

	if step == presenter.StepIdle && !hasActivePlan {
		v.buildNoActivePlanContent()
		return
	}

	v.placeholderText = ""
	v.scheduleTree = v.buildScheduleTree(hasActivePlan)
	v.updateCenterContent()
}

// updateCenterContent replaces the centerContent container's children based on current state.
func (v *PlannerView) updateCenterContent() {
	v.centerContent.RemoveAll()

	if v.placeholderText != "" {
		label := widget.NewLabel(v.placeholderText)
		label.Alignment = fyne.TextAlignCenter
		v.centerContent.Add(container.NewCenter(label))
		return
	}

	if v.scheduleTree != nil {
		cycles := v.scheduleTree.Cycles()
		if len(cycles) > 0 {
			label := widget.NewLabel(fmt.Sprintf("Schedule: %d cycles", len(cycles)))
			v.centerContent.Add(container.NewCenter(label))
		}
	}
}

// buildNoActivePlanContent sets up content when there's no active plan.
func (v *PlannerView) buildNoActivePlanContent() {
	v.placeholderText = placeholderMessages[rand.Intn(len(placeholderMessages))] // #nosec G404 -- math/rand is fine for placeholder text selection
	v.scheduleTree = nil
	v.updateCenterContent()
}

// buildScheduleTree creates a schedule tree if there's an active plan with blocks.
func (v *PlannerView) buildScheduleTree(hasActivePlan bool) *ScheduleTree {
	if !hasActivePlan {
		return nil
	}

	sched := v.plannerModel.ActiveSchedule()
	if sched == nil || len(sched.Blocks) == 0 {
		return nil
	}

	return NewScheduleTree(sched.Blocks, time.Now())
}

// applyVisibility sets button show/hide state based on the current wizard step.
func (v *PlannerView) applyVisibility() {
	step := v.plannerModel.CurrentStep()

	// Define visibility for each button type based on current step
	buttonVisibility := map[*widget.Button]bool{
		v.planBtn:         step == presenter.StepIdle,
		v.nextBtn:         v.isWizardProgressStep(step),
		v.backBtn:         v.isWizardNavigableStep(step),
		v.completeTaskBtn: step == presenter.StepActive,
		v.abandonBtn:      step == presenter.StepActive,
	}

	// Apply visibility to all buttons
	for button, visible := range buttonVisibility {
		v.setButtonVisibility(button, visible)
	}
}

// setButtonVisibility shows or hides a button based on the visible flag.
func (v *PlannerView) setButtonVisibility(button *widget.Button, visible bool) {
	if visible {
		button.Show()
	} else {
		button.Hide()
	}
}

// isWizardProgressStep returns true for wizard steps that allow progressing forward.
func (v *PlannerView) isWizardProgressStep(step presenter.WizardStep) bool {
	return step == presenter.StepTaskSelect ||
		step == presenter.StepEstimates ||
		step == presenter.StepPriority
}

// isWizardNavigableStep returns true for wizard steps that allow navigation (forward/back).
func (v *PlannerView) isWizardNavigableStep(step presenter.WizardStep) bool {
	return step == presenter.StepTaskSelect ||
		step == presenter.StepEstimates ||
		step == presenter.StepPriority ||
		step == presenter.StepSchedule
}
