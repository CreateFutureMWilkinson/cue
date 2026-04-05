package ui

import (
	"context"

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
}

// NewWizardView creates a new WizardView bound to the given view model and router.
func NewWizardView(vm WizardViewModel, router *CenterViewRouter) *WizardView {
	v := &WizardView{
		vm:        vm,
		router:    router,
		container: container.NewVBox(),
	}
	return v
}

// Container returns the top-level Fyne container for the wizard view.
func (v *WizardView) Container() *fyne.Container {
	return v.container
}

// StepIndicator returns the step indicator text, e.g. "Step 1 of 4".
func (v *WizardView) StepIndicator() string {
	return ""
}

// TaskCheckboxes returns the task selection checkboxes for step 1.
func (v *WizardView) TaskCheckboxes() []TaskCheckboxItem {
	return nil
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
	return false
}

// CancelButton returns true if the cancel button is present.
func (v *WizardView) HasCancelButton() bool {
	return false
}

// EstimateRows returns the estimate table rows for step 2.
func (v *WizardView) EstimateRows() []presenter.TaskEstimateRow {
	return nil
}

// SummaryText returns the estimate summary text for step 2, e.g. "3 of 19 Pomodoros".
func (v *WizardView) SummaryText() string {
	return ""
}

// OverloadWarningVisible returns whether the overload warning is shown.
func (v *WizardView) OverloadWarningVisible() bool {
	return false
}

// PriorityList returns the ordered task list for step 3.
func (v *WizardView) PriorityList() []string {
	return nil
}

// HasUpDownButtons returns whether up/down reorder buttons are present.
func (v *WizardView) HasUpDownButtons() bool {
	return false
}

// ScheduleCards returns the number of schedule choice cards for step 4.
func (v *WizardView) ScheduleCards() int {
	return 0
}

// FocusCardStrategy returns the strategy name on the focus card.
func (v *WizardView) FocusCardStrategy() string {
	return ""
}

// RecoveryCardStrategy returns the strategy name on the recovery card.
func (v *WizardView) RecoveryCardStrategy() string {
	return ""
}

// FocusCardStats returns the stats for the focus card.
func (v *WizardView) FocusCardStats() ScheduleCardStats {
	return ScheduleCardStats{}
}

// RecoveryCardStats returns the stats for the recovery card.
func (v *WizardView) RecoveryCardStats() ScheduleCardStats {
	return ScheduleCardStats{}
}

// ScheduleCardStats holds the display stats for a schedule card.
type ScheduleCardStats struct {
	FocusBlocks int
	Breaks      int
	TotalTime   string
}

// HasBackButton returns whether the back button is shown.
func (v *WizardView) HasBackButton() bool {
	return false
}

// HasNextButton returns whether the next button is shown.
func (v *WizardView) HasNextButton() bool {
	return false
}

// Refresh updates the wizard view from the current model state.
func (v *WizardView) Refresh() {}
