package ui

import (
	"context"
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"github.com/google/uuid"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
)

// WizardViewModel abstracts the wizard presenter for the WizardView.
// It combines read methods (from the planner presenter) with write/mutation
// methods needed to drive the wizard steps.
type WizardViewModel interface {
	// Read methods
	CurrentStep() presenter.WizardStep
	AvailableTasks() []presenter.TodoRow
	Estimates() []presenter.TaskEstimateRow
	EstimateSummary() presenter.EstimateSummary
	FocusSchedule() *presenter.SchedulePreview
	RecoverySchedule() *presenter.SchedulePreview

	// Write/mutation methods
	SelectTask(id uuid.UUID, selected bool)
	AddTask(ctx context.Context, title string, priority int) error
	NextStep(ctx context.Context) error
	PreviousStep()
	OverrideEstimate(todoID uuid.UUID, pomos int)
	ReorderTask(from, to int)
	SelectSchedule(ctx context.Context, strategy string) error

	// Derived state
	SelectedCount() int
}

// WizardView renders the day planner wizard step content.
type WizardView struct {
	vm     WizardViewModel
	router *CenterViewRouter

	container *fyne.Container

	// Cached state from buildState
	stepIndicator        string
	taskCheckboxes       []TaskCheckboxItem
	nextButtonEnabled    bool
	hasCancelButton      bool
	hasNextButton        bool
	hasBackButton        bool
	estimateRows         []presenter.TaskEstimateRow
	summaryText          string
	overloadWarning      bool
	priorityList         []string
	hasUpDownButtons     bool
	scheduleCards        int
	focusCardStrategy    string
	recoveryCardStrategy string
	focusCardStats       ScheduleCardStats
	recoveryCardStats    ScheduleCardStats
}

// NewWizardView creates a new WizardView bound to the given view model and router.
func NewWizardView(vm WizardViewModel, router *CenterViewRouter) *WizardView {
	v := &WizardView{
		vm:        vm,
		router:    router,
		container: container.NewVBox(),
	}
	v.buildState()
	return v
}

// Container returns the top-level Fyne container for the wizard view.
func (v *WizardView) Container() *fyne.Container {
	return v.container
}

// buildState reads the view model and populates all cached fields.
func (v *WizardView) buildState() {
	step := v.vm.CurrentStep()

	// Step indicator
	stepNum := 0
	switch step {
	case presenter.StepTaskSelect:
		stepNum = 1
	case presenter.StepEstimates:
		stepNum = 2
	case presenter.StepPriority:
		stepNum = 3
	case presenter.StepSchedule:
		stepNum = 4
	}
	v.stepIndicator = fmt.Sprintf("Step %d of 4", stepNum)

	// Navigation buttons
	v.hasCancelButton = step == presenter.StepTaskSelect
	v.hasNextButton = step == presenter.StepTaskSelect || step == presenter.StepEstimates || step == presenter.StepPriority
	v.hasBackButton = step == presenter.StepEstimates || step == presenter.StepPriority || step == presenter.StepSchedule
	v.hasUpDownButtons = step == presenter.StepPriority

	// Step 1: Task selection
	tasks := v.vm.AvailableTasks()
	v.taskCheckboxes = make([]TaskCheckboxItem, len(tasks))
	for i, t := range tasks {
		cats := make([]string, len(t.Categories))
		for j, c := range t.Categories {
			cats[j] = c.Name
		}
		v.taskCheckboxes[i] = TaskCheckboxItem{
			ID:         t.ID,
			Title:      t.Title,
			Selected:   t.Selected,
			Categories: cats,
		}
	}
	v.nextButtonEnabled = v.vm.SelectedCount() > 0

	// Step 2: Estimates
	v.estimateRows = v.vm.Estimates()
	summary := v.vm.EstimateSummary()
	v.summaryText = fmt.Sprintf("%d of %d Pomodoros", summary.TotalPomos, summary.AvailableBlocks)
	v.overloadWarning = summary.Overloaded

	// Step 3: Priority
	estimates := v.vm.Estimates()
	v.priorityList = make([]string, len(estimates))
	for i, e := range estimates {
		v.priorityList[i] = e.Title
	}

	// Step 4: Schedule
	focus := v.vm.FocusSchedule()
	recovery := v.vm.RecoverySchedule()

	v.scheduleCards = 0
	if focus != nil {
		v.scheduleCards++
		v.focusCardStrategy = focus.Strategy
		v.focusCardStats = buildCardStats(focus)
	}
	if recovery != nil {
		v.scheduleCards++
		v.recoveryCardStrategy = recovery.Strategy
		v.recoveryCardStats = buildCardStats(recovery)
	}
}

func buildCardStats(preview *presenter.SchedulePreview) ScheduleCardStats {
	focusBlocks := 0
	for _, b := range preview.Blocks {
		if b.Type == "focus" {
			focusBlocks++
		}
	}
	return ScheduleCardStats{
		FocusBlocks: focusBlocks,
		Breaks:      preview.BreakCount,
		TotalTime:   formatDuration(preview.TotalFocusTime),
	}
}

func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%dm", h, m)
}

// StepIndicator returns the step indicator text, e.g. "Step 1 of 4".
func (v *WizardView) StepIndicator() string {
	return v.stepIndicator
}

// TaskCheckboxes returns the task selection checkboxes for step 1.
func (v *WizardView) TaskCheckboxes() []TaskCheckboxItem {
	return v.taskCheckboxes
}

// TaskCheckboxItem represents a single task checkbox in step 1.
type TaskCheckboxItem struct {
	ID         uuid.UUID
	Title      string
	Selected   bool
	Categories []string
}

// NextButtonEnabled returns whether the Next button should be enabled.
func (v *WizardView) NextButtonEnabled() bool {
	return v.nextButtonEnabled
}

// HasCancelButton returns true if the cancel button is present.
func (v *WizardView) HasCancelButton() bool {
	return v.hasCancelButton
}

// EstimateRows returns the estimate table rows for step 2.
func (v *WizardView) EstimateRows() []presenter.TaskEstimateRow {
	return v.estimateRows
}

// SummaryText returns the estimate summary text for step 2, e.g. "3 of 19 Pomodoros".
func (v *WizardView) SummaryText() string {
	return v.summaryText
}

// OverloadWarningVisible returns whether the overload warning is shown.
func (v *WizardView) OverloadWarningVisible() bool {
	return v.overloadWarning
}

// PriorityList returns the ordered task list for step 3.
func (v *WizardView) PriorityList() []string {
	return v.priorityList
}

// HasUpDownButtons returns whether up/down reorder buttons are present.
func (v *WizardView) HasUpDownButtons() bool {
	return v.hasUpDownButtons
}

// ScheduleCards returns the number of schedule choice cards for step 4.
func (v *WizardView) ScheduleCards() int {
	return v.scheduleCards
}

// FocusCardStrategy returns the strategy name on the focus card.
func (v *WizardView) FocusCardStrategy() string {
	return v.focusCardStrategy
}

// RecoveryCardStrategy returns the strategy name on the recovery card.
func (v *WizardView) RecoveryCardStrategy() string {
	return v.recoveryCardStrategy
}

// FocusCardStats returns the stats for the focus card.
func (v *WizardView) FocusCardStats() ScheduleCardStats {
	return v.focusCardStats
}

// RecoveryCardStats returns the stats for the recovery card.
func (v *WizardView) RecoveryCardStats() ScheduleCardStats {
	return v.recoveryCardStats
}

// ScheduleCardStats holds the display stats for a schedule card.
type ScheduleCardStats struct {
	FocusBlocks int
	Breaks      int
	TotalTime   string
}

// HasBackButton returns whether the back button is shown.
func (v *WizardView) HasBackButton() bool {
	return v.hasBackButton
}

// HasNextButton returns whether the next button is shown.
func (v *WizardView) HasNextButton() bool {
	return v.hasNextButton
}

// Refresh updates the wizard view from the current model state.
func (v *WizardView) Refresh() {
	v.buildState()
}
