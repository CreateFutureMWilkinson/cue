package ui_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/ui"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/uitest"
)

// --- Mock WizardViewModel ---

type mockWizardViewModel struct {
	mock.Mock
}

func (m *mockWizardViewModel) CurrentStep() presenter.WizardStep {
	args := m.Called()
	return args.Get(0).(presenter.WizardStep)
}

func (m *mockWizardViewModel) AvailableTasks() []presenter.TodoRow {
	args := m.Called()
	return args.Get(0).([]presenter.TodoRow)
}

func (m *mockWizardViewModel) Estimates() []presenter.TaskEstimateRow {
	args := m.Called()
	return args.Get(0).([]presenter.TaskEstimateRow)
}

func (m *mockWizardViewModel) EstimateSummary() presenter.EstimateSummary {
	args := m.Called()
	return args.Get(0).(presenter.EstimateSummary)
}

func (m *mockWizardViewModel) FocusSchedule() *presenter.SchedulePreview {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*presenter.SchedulePreview)
}

func (m *mockWizardViewModel) RecoverySchedule() *presenter.SchedulePreview {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*presenter.SchedulePreview)
}

func (m *mockWizardViewModel) SelectTask(id uuid.UUID, selected bool) {
	m.Called(id, selected)
}

func (m *mockWizardViewModel) AddTask(ctx context.Context, title string, priority int) error {
	args := m.Called(ctx, title, priority)
	return args.Error(0)
}

func (m *mockWizardViewModel) NextStep(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockWizardViewModel) PreviousStep() {
	m.Called()
}

func (m *mockWizardViewModel) OverrideEstimate(todoID uuid.UUID, pomos int) {
	m.Called(todoID, pomos)
}

func (m *mockWizardViewModel) ReorderTask(from, to int) {
	m.Called(from, to)
}

func (m *mockWizardViewModel) SelectSchedule(ctx context.Context, strategy string) error {
	args := m.Called(ctx, strategy)
	return args.Error(0)
}

func (m *mockWizardViewModel) SelectedCount() int {
	args := m.Called()
	return args.Int(0)
}

// --- Suite ---

type WizardViewSuite struct {
	suite.Suite
	vm     *mockWizardViewModel
	router *ui.CenterViewRouter
}

func TestWizardView(t *testing.T) {
	suite.Run(t, new(WizardViewSuite))
}

func (s *WizardViewSuite) SetupTest() {
	s.vm = new(mockWizardViewModel)
	s.router = ui.NewCenterViewRouter()
}

// --- Helper: sample data ---

var (
	taskID1 = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	taskID2 = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	taskID3 = uuid.MustParse("00000000-0000-0000-0000-000000000003")
)

func sampleAvailableTasks() []presenter.TodoRow {
	due := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	return []presenter.TodoRow{
		{
			ID:       taskID1,
			Title:    "Write tests",
			Priority: 1,
			DueDate:  &due,
			Categories: []repository.Category{
				{ID: uuid.New(), Name: "work", Color: "#FF0000"},
			},
			Selected: true,
		},
		{
			ID:       taskID2,
			Title:    "Review PR",
			Priority: 2,
			Selected: false,
		},
		{
			ID:       taskID3,
			Title:    "Deploy fix",
			Priority: 3,
			Selected: true,
		},
	}
}

func sampleEstimates() []presenter.TaskEstimateRow {
	override := 3
	return []presenter.TaskEstimateRow{
		{
			TodoID:         taskID1,
			Title:          "Write tests",
			EstimatedPomos: 2,
			EffectivePomos: 2,
		},
		{
			TodoID:         taskID3,
			Title:          "Deploy fix",
			EstimatedPomos: 4,
			UserOverride:   &override,
			EffectivePomos: 3,
		},
	}
}

func sampleFocusSchedule() *presenter.SchedulePreview {
	return &presenter.SchedulePreview{
		Strategy:       "focus-maximized",
		TotalFocusTime: 3 * time.Hour,
		BreakCount:     4,
		Blocks: []presenter.TimeBlockPreview{
			{Type: "focus", TaskName: "Write tests"},
			{Type: "short_break"},
			{Type: "focus", TaskName: "Deploy fix"},
		},
	}
}

func sampleRecoverySchedule() *presenter.SchedulePreview {
	return &presenter.SchedulePreview{
		Strategy:       "recovery-balanced",
		TotalFocusTime: 2 * time.Hour,
		BreakCount:     6,
		Blocks: []presenter.TimeBlockPreview{
			{Type: "focus", TaskName: "Write tests"},
			{Type: "long_break"},
			{Type: "focus", TaskName: "Deploy fix"},
		},
	}
}

// --- Step-specific default setup ---

func (s *WizardViewSuite) setupStep1Defaults() {
	s.vm.On("CurrentStep").Return(presenter.StepTaskSelect).Maybe()
	s.vm.On("AvailableTasks").Return(sampleAvailableTasks()).Maybe()
	s.vm.On("SelectedCount").Return(2).Maybe()
	s.vm.On("Estimates").Return([]presenter.TaskEstimateRow{}).Maybe()
	s.vm.On("EstimateSummary").Return(presenter.EstimateSummary{}).Maybe()
	s.vm.On("FocusSchedule").Return((*presenter.SchedulePreview)(nil)).Maybe()
	s.vm.On("RecoverySchedule").Return((*presenter.SchedulePreview)(nil)).Maybe()
}

func (s *WizardViewSuite) setupStep1NoSelection() {
	s.vm.On("CurrentStep").Return(presenter.StepTaskSelect).Maybe()
	s.vm.On("AvailableTasks").Return(sampleAvailableTasks()).Maybe()
	s.vm.On("SelectedCount").Return(0).Maybe()
	s.vm.On("Estimates").Return([]presenter.TaskEstimateRow{}).Maybe()
	s.vm.On("EstimateSummary").Return(presenter.EstimateSummary{}).Maybe()
	s.vm.On("FocusSchedule").Return((*presenter.SchedulePreview)(nil)).Maybe()
	s.vm.On("RecoverySchedule").Return((*presenter.SchedulePreview)(nil)).Maybe()
}

func (s *WizardViewSuite) setupStep2Defaults() {
	s.vm.On("CurrentStep").Return(presenter.StepEstimates).Maybe()
	s.vm.On("AvailableTasks").Return([]presenter.TodoRow{}).Maybe()
	s.vm.On("SelectedCount").Return(0).Maybe()
	s.vm.On("Estimates").Return(sampleEstimates()).Maybe()
	s.vm.On("EstimateSummary").Return(presenter.EstimateSummary{
		TotalPomos:      5,
		AvailableBlocks: 19,
		Overloaded:      false,
	}).Maybe()
	s.vm.On("FocusSchedule").Return((*presenter.SchedulePreview)(nil)).Maybe()
	s.vm.On("RecoverySchedule").Return((*presenter.SchedulePreview)(nil)).Maybe()
}

func (s *WizardViewSuite) setupStep2Overloaded() {
	s.vm.On("CurrentStep").Return(presenter.StepEstimates).Maybe()
	s.vm.On("AvailableTasks").Return([]presenter.TodoRow{}).Maybe()
	s.vm.On("SelectedCount").Return(0).Maybe()
	s.vm.On("Estimates").Return(sampleEstimates()).Maybe()
	s.vm.On("EstimateSummary").Return(presenter.EstimateSummary{
		TotalPomos:      25,
		AvailableBlocks: 19,
		Overloaded:      true,
	}).Maybe()
	s.vm.On("FocusSchedule").Return((*presenter.SchedulePreview)(nil)).Maybe()
	s.vm.On("RecoverySchedule").Return((*presenter.SchedulePreview)(nil)).Maybe()
}

func (s *WizardViewSuite) setupStep3Defaults() {
	s.vm.On("CurrentStep").Return(presenter.StepPriority).Maybe()
	s.vm.On("AvailableTasks").Return([]presenter.TodoRow{}).Maybe()
	s.vm.On("SelectedCount").Return(0).Maybe()
	s.vm.On("Estimates").Return(sampleEstimates()).Maybe()
	s.vm.On("EstimateSummary").Return(presenter.EstimateSummary{}).Maybe()
	s.vm.On("FocusSchedule").Return((*presenter.SchedulePreview)(nil)).Maybe()
	s.vm.On("RecoverySchedule").Return((*presenter.SchedulePreview)(nil)).Maybe()
}

func (s *WizardViewSuite) setupStep4Defaults() {
	s.vm.On("CurrentStep").Return(presenter.StepSchedule).Maybe()
	s.vm.On("AvailableTasks").Return([]presenter.TodoRow{}).Maybe()
	s.vm.On("SelectedCount").Return(0).Maybe()
	s.vm.On("Estimates").Return([]presenter.TaskEstimateRow{}).Maybe()
	s.vm.On("EstimateSummary").Return(presenter.EstimateSummary{}).Maybe()
	s.vm.On("FocusSchedule").Return(sampleFocusSchedule()).Maybe()
	s.vm.On("RecoverySchedule").Return(sampleRecoverySchedule()).Maybe()
}

// =====================================================================
// Constructor Tests
// =====================================================================

func (s *WizardViewSuite) TestNewWizardViewReturnsNonNil() {
	s.setupStep1Defaults()

	view := ui.NewWizardView(s.vm, s.router)

	s.NotNil(view, "NewWizardView should return a non-nil component")
}

func (s *WizardViewSuite) TestContainerReturnsNonNil() {
	s.setupStep1Defaults()

	view := ui.NewWizardView(s.vm, s.router)

	s.NotNil(view.Container(), "Container() should return a non-nil fyne container")
}

// =====================================================================
// Step 1: Task Selection
// =====================================================================

func (s *WizardViewSuite) TestStep1ShowsStepIndicator() {
	s.setupStep1Defaults()

	view := ui.NewWizardView(s.vm, s.router)

	s.Equal("Step 1 of 4", view.StepIndicator(),
		"Step indicator should show 'Step 1 of 4' during task selection")
}

func (s *WizardViewSuite) TestStep1ShowsTaskCheckboxes() {
	s.setupStep1Defaults()

	view := ui.NewWizardView(s.vm, s.router)
	checkboxes := view.TaskCheckboxes()

	s.Require().Equal(3, len(checkboxes),
		"Should show one checkbox per available task")
	s.Equal("Write tests", checkboxes[0].Title)
	s.True(checkboxes[0].Selected, "First task should be selected")
	s.Equal("Review PR", checkboxes[1].Title)
	s.False(checkboxes[1].Selected, "Second task should not be selected")
}

func (s *WizardViewSuite) TestStep1ShowsCategoryBadges() {
	s.setupStep1Defaults()

	view := ui.NewWizardView(s.vm, s.router)
	checkboxes := view.TaskCheckboxes()

	s.Require().Equal(3, len(checkboxes))
	s.Require().Equal(1, len(checkboxes[0].Categories),
		"First task should have one category badge")
	s.Equal("work", checkboxes[0].Categories[0],
		"Category badge should display the category name")
}

func (s *WizardViewSuite) TestStep1NextDisabledWhenNoTasksSelected() {
	s.setupStep1NoSelection()

	view := ui.NewWizardView(s.vm, s.router)

	s.False(view.NextButtonEnabled(),
		"Next button should be disabled when SelectedCount() == 0")
}

func (s *WizardViewSuite) TestStep1NextEnabledWhenTasksSelected() {
	s.setupStep1Defaults()

	view := ui.NewWizardView(s.vm, s.router)

	s.True(view.NextButtonEnabled(),
		"Next button should be enabled when SelectedCount() > 0")
}

func (s *WizardViewSuite) TestStep1HasCancelButton() {
	s.setupStep1Defaults()

	view := ui.NewWizardView(s.vm, s.router)

	s.True(view.HasCancelButton(),
		"Cancel button should be present on step 1")
}

func (s *WizardViewSuite) TestStep1HasNextButton() {
	s.setupStep1Defaults()

	view := ui.NewWizardView(s.vm, s.router)

	s.True(view.HasNextButton(),
		"Next button should be present on step 1")
}

func (s *WizardViewSuite) TestStep1HasAddTaskButton() {
	s.setupStep1Defaults()

	view := ui.NewWizardView(s.vm, s.router)

	s.True(view.HasAddTaskButton(),
		"Add Task button should be present on step 1 so the user can add new tasks")
}

// =====================================================================
// Step 2: Pomodoro Estimates
// =====================================================================

func (s *WizardViewSuite) TestStep2ShowsStepIndicator() {
	s.setupStep2Defaults()

	view := ui.NewWizardView(s.vm, s.router)

	s.Equal("Step 2 of 4", view.StepIndicator(),
		"Step indicator should show 'Step 2 of 4' during estimates")
}

func (s *WizardViewSuite) TestStep2ShowsEstimateRows() {
	s.setupStep2Defaults()

	view := ui.NewWizardView(s.vm, s.router)
	rows := view.EstimateRows()

	s.Require().Equal(2, len(rows),
		"Should show one row per selected task estimate")
	s.Equal("Write tests", rows[0].Title)
	s.Equal(2, rows[0].EffectivePomos)
	s.Equal("Deploy fix", rows[1].Title)
	s.Equal(3, rows[1].EffectivePomos)
}

func (s *WizardViewSuite) TestStep2ShowsSummaryText() {
	s.setupStep2Defaults()

	view := ui.NewWizardView(s.vm, s.router)

	s.Equal("5 of 19 Pomodoros", view.SummaryText(),
		"Summary should show 'TotalPomos of AvailableBlocks Pomodoros'")
}

func (s *WizardViewSuite) TestStep2OverloadWarningHiddenWhenNotOverloaded() {
	s.setupStep2Defaults()

	view := ui.NewWizardView(s.vm, s.router)

	s.False(view.OverloadWarningVisible(),
		"Overload warning should be hidden when Overloaded=false")
}

func (s *WizardViewSuite) TestStep2OverloadWarningVisibleWhenOverloaded() {
	s.setupStep2Overloaded()

	view := ui.NewWizardView(s.vm, s.router)

	s.True(view.OverloadWarningVisible(),
		"Overload warning should be visible when Overloaded=true")
}

func (s *WizardViewSuite) TestStep2HasBackButton() {
	s.setupStep2Defaults()

	view := ui.NewWizardView(s.vm, s.router)

	s.True(view.HasBackButton(),
		"Back button should be present on step 2")
}

func (s *WizardViewSuite) TestStep2HasNextButton() {
	s.setupStep2Defaults()

	view := ui.NewWizardView(s.vm, s.router)

	s.True(view.HasNextButton(),
		"Next button should be present on step 2")
}

// =====================================================================
// Step 3: Priority Ordering
// =====================================================================

func (s *WizardViewSuite) TestStep3ShowsStepIndicator() {
	s.setupStep3Defaults()

	view := ui.NewWizardView(s.vm, s.router)

	s.Equal("Step 3 of 4", view.StepIndicator(),
		"Step indicator should show 'Step 3 of 4' during priority ordering")
}

func (s *WizardViewSuite) TestStep3ShowsNumberedTaskList() {
	s.setupStep3Defaults()

	view := ui.NewWizardView(s.vm, s.router)
	list := view.PriorityList()

	s.Require().Equal(2, len(list),
		"Priority list should show one entry per estimate")
	s.Equal("Write tests", list[0])
	s.Equal("Deploy fix", list[1])
}

func (s *WizardViewSuite) TestStep3HasUpDownButtons() {
	s.setupStep3Defaults()

	view := ui.NewWizardView(s.vm, s.router)

	s.True(view.HasUpDownButtons(),
		"Up/down reorder buttons should be present on step 3")
}

func (s *WizardViewSuite) TestStep3HasBackButton() {
	s.setupStep3Defaults()

	view := ui.NewWizardView(s.vm, s.router)

	s.True(view.HasBackButton(),
		"Back button should be present on step 3")
}

func (s *WizardViewSuite) TestStep3HasNextButton() {
	s.setupStep3Defaults()

	view := ui.NewWizardView(s.vm, s.router)

	s.True(view.HasNextButton(),
		"Next button should be present on step 3")
}

// =====================================================================
// Step 4: Schedule Choice
// =====================================================================

func (s *WizardViewSuite) TestStep4ShowsStepIndicator() {
	s.setupStep4Defaults()

	view := ui.NewWizardView(s.vm, s.router)

	s.Equal("Step 4 of 4", view.StepIndicator(),
		"Step indicator should show 'Step 4 of 4' during schedule choice")
}

func (s *WizardViewSuite) TestStep4ShowsTwoScheduleCards() {
	s.setupStep4Defaults()

	view := ui.NewWizardView(s.vm, s.router)

	s.Equal(2, view.ScheduleCards(),
		"Should show exactly two schedule cards (focus and recovery)")
}

func (s *WizardViewSuite) TestStep4FocusCardShowsStrategy() {
	s.setupStep4Defaults()

	view := ui.NewWizardView(s.vm, s.router)

	s.Equal("focus-maximized", view.FocusCardStrategy(),
		"Focus card should display the focus-maximized strategy name")
}

func (s *WizardViewSuite) TestStep4RecoveryCardShowsStrategy() {
	s.setupStep4Defaults()

	view := ui.NewWizardView(s.vm, s.router)

	s.Equal("recovery-balanced", view.RecoveryCardStrategy(),
		"Recovery card should display the recovery-balanced strategy name")
}

func (s *WizardViewSuite) TestStep4FocusCardShowsStats() {
	s.setupStep4Defaults()

	view := ui.NewWizardView(s.vm, s.router)
	stats := view.FocusCardStats()

	// Focus schedule has 2 focus blocks, 4 breaks, 3h total focus time
	s.Equal(2, stats.FocusBlocks,
		"Focus card should show correct number of focus blocks")
	s.Equal(4, stats.Breaks,
		"Focus card should show correct number of breaks")
	s.Equal("3h0m", stats.TotalTime,
		"Focus card should show correct total focus time")
}

func (s *WizardViewSuite) TestStep4RecoveryCardShowsStats() {
	s.setupStep4Defaults()

	view := ui.NewWizardView(s.vm, s.router)
	stats := view.RecoveryCardStats()

	// Recovery schedule has 2 focus blocks, 6 breaks, 2h total focus time
	s.Equal(2, stats.FocusBlocks,
		"Recovery card should show correct number of focus blocks")
	s.Equal(6, stats.Breaks,
		"Recovery card should show correct number of breaks")
	s.Equal("2h0m", stats.TotalTime,
		"Recovery card should show correct total focus time")
}

func (s *WizardViewSuite) TestStep4HasBackButton() {
	s.setupStep4Defaults()

	view := ui.NewWizardView(s.vm, s.router)

	s.True(view.HasBackButton(),
		"Back button should be present on step 4")
}

func (s *WizardViewSuite) TestStep4NoNextButton() {
	s.setupStep4Defaults()

	view := ui.NewWizardView(s.vm, s.router)

	s.False(view.HasNextButton(),
		"Next button should NOT be present on step 4 (use schedule selection instead)")
}

// =====================================================================
// Refresh
// =====================================================================

func (s *WizardViewSuite) TestRefreshUpdatesStepIndicator() {
	s.setupStep1Defaults()

	view := ui.NewWizardView(s.vm, s.router)

	s.Equal("Step 1 of 4", view.StepIndicator(),
		"Initially should show step 1")

	// Transition to step 2
	s.vm = new(mockWizardViewModel)
	s.setupStep2Defaults()
	// We need a way to set the model — the view should support this
	// For now, create a new view to verify step 2 rendering
	view2 := ui.NewWizardView(s.vm, s.router)

	s.Equal("Step 2 of 4", view2.StepIndicator(),
		"After model change should show step 2")
}

// --- Helper: find Nth button by text in widget tree ---

func findNthButton(root *fyne.Container, text string, n int) *widget.Button {
	buttons := uitest.FindAll[*widget.Button](root, func(b *widget.Button) bool {
		return b.Text == text
	})
	if n < len(buttons) {
		return buttons[n]
	}
	return nil
}

// =====================================================================
// Step 3: Per-Row Up/Down Button Behavior
// =====================================================================

func (s *WizardViewSuite) TestStep3UpButtonCallsReorderTask() {
	s.setupStep3Defaults()
	s.vm.On("ReorderTask", 1, 0).Once()

	view := ui.NewWizardView(s.vm, s.router)

	// Find the Up button for the second item (index 1)
	upBtn := findNthButton(view.Container(), "Up", 1)
	s.Require().NotNil(upBtn, "Should find a second Up button (for item at index 1)")

	upBtn.OnTapped()

	s.vm.AssertCalled(s.T(), "ReorderTask", 1, 0)
}

func (s *WizardViewSuite) TestStep3DownButtonCallsReorderTask() {
	s.setupStep3Defaults()
	s.vm.On("ReorderTask", 0, 1).Once()

	view := ui.NewWizardView(s.vm, s.router)

	// Find the Down button for the first item (index 0)
	downBtn := findNthButton(view.Container(), "Down", 0)
	s.Require().NotNil(downBtn, "Should find a Down button for the first item")

	downBtn.OnTapped()

	s.vm.AssertCalled(s.T(), "ReorderTask", 0, 1)
}

func (s *WizardViewSuite) TestStep3FirstItemUpDisabled() {
	s.setupStep3Defaults()

	view := ui.NewWizardView(s.vm, s.router)

	// The first item's Up button should be disabled (can't move up from index 0)
	upBtn := findNthButton(view.Container(), "Up", 0)
	s.Require().NotNil(upBtn, "Should find an Up button for the first item")

	s.True(upBtn.Disabled(), "First item's Up button should be disabled")
}

func (s *WizardViewSuite) TestStep3LastItemDownDisabled() {
	s.setupStep3Defaults()

	view := ui.NewWizardView(s.vm, s.router)

	// The last item's Down button should be disabled (can't move down from last index)
	downBtn := findNthButton(view.Container(), "Down", 1)
	s.Require().NotNil(downBtn, "Should find a Down button for the last item")

	s.True(downBtn.Disabled(), "Last item's Down button should be disabled")
}

func (s *WizardViewSuite) TestStep3UpButtonRefreshesView() {
	// Initial order: "Write tests" (index 0), "Deploy fix" (index 1)
	s.setupStep3Defaults()

	// After ReorderTask(1, 0), the VM should return swapped estimates
	s.vm.On("ReorderTask", 1, 0).Run(func(args mock.Arguments) {
		// After reorder, clear and re-setup with swapped order
		// The view calls Refresh which re-reads Estimates()
	}).Once()

	view := ui.NewWizardView(s.vm, s.router)

	// Verify initial order
	s.Equal("Write tests", view.PriorityList()[0])
	s.Equal("Deploy fix", view.PriorityList()[1])

	// Tap Up on the second item to move it to position 0
	upBtn := findNthButton(view.Container(), "Up", 1)
	s.Require().NotNil(upBtn, "Should find a second Up button")

	// Reconfigure mock to return swapped order after reorder
	swappedEstimates := []presenter.TaskEstimateRow{
		{
			TodoID:         taskID3,
			Title:          "Deploy fix",
			EstimatedPomos: 4,
			EffectivePomos: 3,
		},
		{
			TodoID:         taskID1,
			Title:          "Write tests",
			EstimatedPomos: 2,
			EffectivePomos: 2,
		},
	}
	s.vm.ExpectedCalls = filterCallsByMethod(s.vm.ExpectedCalls, "Estimates")
	s.vm.On("Estimates").Return(swappedEstimates).Maybe()

	upBtn.OnTapped()

	// After tapping Up, the view should have refreshed and show swapped order
	s.Equal("Deploy fix", view.PriorityList()[0],
		"After tapping Up on second item, it should move to first position")
	s.Equal("Write tests", view.PriorityList()[1],
		"After tapping Up on second item, first item should move to second position")
}

// =====================================================================
// Step Idle — empty state (Bug 077)
// =====================================================================

func (s *WizardViewSuite) setupIdleDefaults() {
	s.vm.On("CurrentStep").Return(presenter.StepIdle).Maybe()
	s.vm.On("AvailableTasks").Return([]presenter.TodoRow{}).Maybe()
	s.vm.On("SelectedCount").Return(0).Maybe()
	s.vm.On("Estimates").Return([]presenter.TaskEstimateRow{}).Maybe()
	s.vm.On("EstimateSummary").Return(presenter.EstimateSummary{}).Maybe()
	s.vm.On("FocusSchedule").Return((*presenter.SchedulePreview)(nil)).Maybe()
	s.vm.On("RecoverySchedule").Return((*presenter.SchedulePreview)(nil)).Maybe()
}

func (s *WizardViewSuite) TestIdleStateRendersContent() {
	s.setupIdleDefaults()

	view := ui.NewWizardView(s.vm, s.router)
	root := view.Container()

	s.Greater(len(root.Objects), 0,
		"StepIdle should render visible content in the wizard container")
}

func (s *WizardViewSuite) TestIdleStateShowsPromptLabel() {
	s.setupIdleDefaults()

	view := ui.NewWizardView(s.vm, s.router)
	root := view.Container()

	_, found := uitest.FindWidget[*widget.Label](root, func(l *widget.Label) bool {
		return l.Text != ""
	})
	s.True(found,
		"StepIdle should show a prompt label guiding the user")
}

// =====================================================================
// Step 1: Cancel resets wizard state (Bug 084)
// =====================================================================

func (s *WizardViewSuite) TestStep1CancelCallsPreviousStepBeforeNavigating() {
	s.setupStep1Defaults()
	s.vm.On("PreviousStep").Once()

	view := ui.NewWizardView(s.vm, s.router)

	cancelBtn := findNthButton(view.Container(), "Cancel", 0)
	s.Require().NotNil(cancelBtn, "Should find the Cancel button on step 1")

	cancelBtn.OnTapped()

	s.vm.AssertCalled(s.T(), "PreviousStep")
}

// filterCallsByMethod returns all mock expected calls except those for the given method.
func filterCallsByMethod(calls []*mock.Call, method string) []*mock.Call {
	var filtered []*mock.Call
	for _, c := range calls {
		if c.Method != method {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// Ensure unused imports are valid
var _ = fmt.Sprintf
