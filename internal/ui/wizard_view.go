package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/google/uuid"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
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
	hasAddTaskButton     bool
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
	v.renderContainer()
	return v
}

// Container returns the top-level Fyne container for the wizard view.
func (v *WizardView) Container() *fyne.Container {
	return v.container
}

// buildState reads the view model and populates all cached fields.
func (v *WizardView) buildState() {
	step := v.vm.CurrentStep()

	v.buildStepIndicator(step)
	v.buildNavigationButtons(step)
	v.buildTaskSelection()
	v.buildEstimates()
	v.buildPriorityList()
	v.buildScheduleCards()
}

// buildStepIndicator sets the step indicator text based on the current step.
func (v *WizardView) buildStepIndicator(step presenter.WizardStep) {
	stepNum := v.getStepNumber(step)
	v.stepIndicator = fmt.Sprintf("Step %d of 4", stepNum)
}

// getStepNumber returns the numeric step number for a given wizard step.
func (v *WizardView) getStepNumber(step presenter.WizardStep) int {
	switch step {
	case presenter.StepTaskSelect:
		return 1
	case presenter.StepEstimates:
		return 2
	case presenter.StepPriority:
		return 3
	case presenter.StepSchedule:
		return 4
	default:
		return 0
	}
}

// buildNavigationButtons sets navigation button visibility based on the current step.
func (v *WizardView) buildNavigationButtons(step presenter.WizardStep) {
	v.hasCancelButton = step == presenter.StepTaskSelect
	v.hasNextButton = v.isNextButtonStep(step)
	v.hasBackButton = v.isBackButtonStep(step)
	v.hasUpDownButtons = step == presenter.StepPriority
	v.hasAddTaskButton = step == presenter.StepTaskSelect
}

// isNextButtonStep returns true for steps that show the Next button.
func (v *WizardView) isNextButtonStep(step presenter.WizardStep) bool {
	return step == presenter.StepTaskSelect ||
		step == presenter.StepEstimates ||
		step == presenter.StepPriority
}

// isBackButtonStep returns true for steps that show the Back button.
func (v *WizardView) isBackButtonStep(step presenter.WizardStep) bool {
	return step == presenter.StepEstimates ||
		step == presenter.StepPriority ||
		step == presenter.StepSchedule
}

// buildTaskSelection populates task selection data for step 1.
func (v *WizardView) buildTaskSelection() {
	tasks := v.vm.AvailableTasks()
	v.taskCheckboxes = make([]TaskCheckboxItem, len(tasks))

	for i, t := range tasks {
		v.taskCheckboxes[i] = TaskCheckboxItem{
			ID:         t.ID,
			Title:      t.Title,
			Selected:   t.Selected,
			Categories: v.extractCategoryNames(t.Categories),
		}
	}

	v.nextButtonEnabled = v.vm.SelectedCount() > 0
}

// extractCategoryNames converts category structs to display strings.
//
// TODO(feat-109 Loop 7): switch to a single-category embed once the
// task DTO carries `category: {key, name}` instead of a slice.
func (v *WizardView) extractCategoryNames(categories []repository.Category) []string {
	names := make([]string, len(categories))
	for i, c := range categories {
		names[i] = repository.PresentCategoryName(c.NameKey)
	}
	return names
}

// buildEstimates populates estimate data for step 2.
func (v *WizardView) buildEstimates() {
	v.estimateRows = v.vm.Estimates()

	summary := v.vm.EstimateSummary()
	v.summaryText = fmt.Sprintf("%d of %d Pomodoros", summary.TotalPomos, summary.AvailableBlocks)
	v.overloadWarning = summary.Overloaded
}

// buildPriorityList populates priority list data for step 3.
func (v *WizardView) buildPriorityList() {
	estimates := v.vm.Estimates()
	v.priorityList = make([]string, len(estimates))

	for i, e := range estimates {
		v.priorityList[i] = e.Title
	}
}

// buildScheduleCards populates schedule card data for step 4.
func (v *WizardView) buildScheduleCards() {
	focus := v.vm.FocusSchedule()
	recovery := v.vm.RecoverySchedule()

	v.scheduleCards = 0
	v.buildScheduleCard(focus, &v.focusCardStrategy, &v.focusCardStats)
	v.buildScheduleCard(recovery, &v.recoveryCardStrategy, &v.recoveryCardStats)
}

// buildScheduleCard populates a single schedule card's data if the preview is not nil.
func (v *WizardView) buildScheduleCard(preview *presenter.SchedulePreview, strategy *string, stats *ScheduleCardStats) {
	if preview != nil {
		v.scheduleCards++
		*strategy = preview.Strategy
		*stats = buildCardStats(preview)
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

// HasAddTaskButton returns whether the "Add Task" button is shown on step 1.
func (v *WizardView) HasAddTaskButton() bool {
	return v.hasAddTaskButton
}

// renderContainer clears the container and dispatches to step-specific render methods.
func (v *WizardView) renderContainer() {
	v.container.Objects = nil

	switch v.vm.CurrentStep() {
	case presenter.StepIdle:
		v.renderIdle()
	case presenter.StepTaskSelect:
		v.renderStep1()
	case presenter.StepEstimates:
		v.renderStep2()
	case presenter.StepPriority:
		v.renderStep3()
	case presenter.StepSchedule:
		v.renderStep4()
	}

	v.container.Refresh()
}

// renderIdle renders the idle state prompt when no wizard step is active.
func (v *WizardView) renderIdle() {
	v.container.Objects = append(v.container.Objects,
		widget.NewLabel("Use \"Plan My Day\" to start planning your day."))
}

// renderStep1 renders the task selection step (step 1) widgets into the container.
func (v *WizardView) renderStep1() {
	v.container.Objects = append(v.container.Objects, widget.NewLabel(v.stepIndicator))

	for _, item := range v.taskCheckboxes {
		check := widget.NewCheck(item.Title, func(checked bool) {
			v.vm.SelectTask(item.ID, checked)
		})
		check.Checked = item.Selected
		v.container.Objects = append(v.container.Objects, check)
	}

	entry := widget.NewEntry()
	entry.SetPlaceHolder("New task")
	v.container.Objects = append(v.container.Objects, entry)

	v.container.Objects = append(v.container.Objects,
		widget.NewButton("Add Task", func() {
			text := strings.TrimSpace(entry.Text)
			if text == "" {
				return
			}
			v.vm.AddTask(context.Background(), text, 0) // #nosec G104 -- GUI callback; error logged by presenter
			entry.SetText("")
			v.Refresh()
		}),
		widget.NewButton("Next", func() { v.vm.NextStep(context.Background()) }), // #nosec G104 -- GUI callback; error logged by presenter
		widget.NewButton("Cancel", func() {
			v.vm.PreviousStep()
			v.router.NavigateTo(ViewPlan)
		}),
	)
}

// renderStep2 renders the estimate editing step (step 2) widgets into the container.
func (v *WizardView) renderStep2() {
	v.container.Objects = append(v.container.Objects, widget.NewLabel(v.stepIndicator))

	for _, row := range v.estimateRows {
		v.container.Objects = append(v.container.Objects, widget.NewLabel(row.Title))
		entry := widget.NewEntry()
		entry.SetText(fmt.Sprintf("%d", row.EffectivePomos))
		v.container.Objects = append(v.container.Objects, entry)
	}

	v.container.Objects = append(v.container.Objects, widget.NewLabel(v.summaryText))

	v.container.Objects = append(v.container.Objects,
		widget.NewButton("Back", func() { v.vm.PreviousStep() }),
		widget.NewButton("Next", func() { v.vm.NextStep(context.Background()) }), // #nosec G104 -- GUI callback; error logged by presenter
	)
}

// renderStep3 renders the priority reordering step (step 3) widgets into the container.
func (v *WizardView) renderStep3() {
	v.container.Objects = append(v.container.Objects, widget.NewLabel(v.stepIndicator))

	for i, title := range v.priorityList {
		idx := i // capture for closure
		v.container.Objects = append(v.container.Objects,
			widget.NewLabel(fmt.Sprintf("%d. %s", i+1, title)))

		if v.hasUpDownButtons {
			upBtn := widget.NewButton("Up", func() {
				v.vm.ReorderTask(idx, idx-1)
				v.Refresh()
			})
			if idx == 0 {
				upBtn.Disable()
			}

			downBtn := widget.NewButton("Down", func() {
				v.vm.ReorderTask(idx, idx+1)
				v.Refresh()
			})
			if idx == len(v.priorityList)-1 {
				downBtn.Disable()
			}

			v.container.Objects = append(v.container.Objects, upBtn, downBtn)
		}
	}

	v.container.Objects = append(v.container.Objects,
		widget.NewButton("Back", func() { v.vm.PreviousStep() }),
		widget.NewButton("Next", func() { v.vm.NextStep(context.Background()) }), // #nosec G104 -- GUI callback; error logged by presenter
	)
}

// renderStep4 renders the schedule selection step (step 4) widgets into the container.
func (v *WizardView) renderStep4() {
	v.container.Objects = append(v.container.Objects, widget.NewLabel(v.stepIndicator))

	if v.focusCardStrategy != "" {
		strategy := v.focusCardStrategy
		v.container.Objects = append(v.container.Objects,
			widget.NewButton("Select "+strategy, func() {
				v.vm.SelectSchedule(context.Background(), strategy) // #nosec G104 -- GUI callback; error logged by presenter
			}))
	}
	if v.recoveryCardStrategy != "" {
		strategy := v.recoveryCardStrategy
		v.container.Objects = append(v.container.Objects,
			widget.NewButton("Select "+strategy, func() {
				v.vm.SelectSchedule(context.Background(), strategy) // #nosec G104 -- GUI callback; error logged by presenter
			}))
	}

	v.container.Objects = append(v.container.Objects,
		widget.NewButton("Back", func() { v.vm.PreviousStep() }),
	)
}

// Refresh updates the wizard view from the current model state.
func (v *WizardView) Refresh() {
	v.buildState()
	v.renderContainer()
}
