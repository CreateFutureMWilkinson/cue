package ui_test

import (
	"testing"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/service/planner"
	"github.com/CreateFutureMWilkinson/cue/internal/ui"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/uitest"
)

// --- Mock PlannerViewModel ---

type mockPlannerViewModel struct {
	mock.Mock
}

func (m *mockPlannerViewModel) CurrentStep() presenter.WizardStep {
	args := m.Called()
	return args.Get(0).(presenter.WizardStep)
}

func (m *mockPlannerViewModel) HasActivePlan() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *mockPlannerViewModel) AvailableTasks() []presenter.TodoRow {
	args := m.Called()
	return args.Get(0).([]presenter.TodoRow)
}

func (m *mockPlannerViewModel) Estimates() []presenter.TaskEstimateRow {
	args := m.Called()
	return args.Get(0).([]presenter.TaskEstimateRow)
}

func (m *mockPlannerViewModel) EstimateSummary() presenter.EstimateSummary {
	args := m.Called()
	return args.Get(0).(presenter.EstimateSummary)
}

func (m *mockPlannerViewModel) FocusSchedule() *presenter.SchedulePreview {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*presenter.SchedulePreview)
}

func (m *mockPlannerViewModel) RecoverySchedule() *presenter.SchedulePreview {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*presenter.SchedulePreview)
}

func (m *mockPlannerViewModel) ActiveSchedule() *presenter.ActiveScheduleState {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*presenter.ActiveScheduleState)
}

// --- Mock TimerViewModel ---

type mockTimerViewModel struct {
	mock.Mock
}

func (m *mockTimerViewModel) IsRunning() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *mockTimerViewModel) ActiveSegment() int {
	args := m.Called()
	return args.Int(0)
}

func (m *mockTimerViewModel) ElapsedFraction() float64 {
	args := m.Called()
	return args.Get(0).(float64)
}

func (m *mockTimerViewModel) IsFlashVisible() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *mockTimerViewModel) CurrentTaskName() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockTimerViewModel) BlockType() planner.BlockType {
	args := m.Called()
	return args.Get(0).(planner.BlockType)
}

// --- Suite ---

type PlannerViewSuite struct {
	suite.Suite
	plannerVM *mockPlannerViewModel
	timerVM   *mockTimerViewModel
	router    *ui.CenterViewRouter
	todoVM    *mockTodoListViewModel
}

func TestPlannerView(t *testing.T) {
	suite.Run(t, new(PlannerViewSuite))
}

func (s *PlannerViewSuite) SetupTest() {
	s.plannerVM = new(mockPlannerViewModel)
	s.timerVM = new(mockTimerViewModel)
	s.router = ui.NewCenterViewRouter()
	s.todoVM = new(mockTodoListViewModel)
	s.todoVM.On("AllTodos").Return([]ui.TodoListRow{}).Maybe()
}

// setupIdleDefaults configures mock expectations for the idle state.
func (s *PlannerViewSuite) setupIdleDefaults() {
	s.plannerVM.On("CurrentStep").Return(presenter.StepIdle).Maybe()
	s.plannerVM.On("HasActivePlan").Return(false).Maybe()
	s.plannerVM.On("AvailableTasks").Return([]presenter.TodoRow{}).Maybe()
	s.plannerVM.On("Estimates").Return([]presenter.TaskEstimateRow{}).Maybe()
	s.plannerVM.On("EstimateSummary").Return(presenter.EstimateSummary{}).Maybe()
	s.plannerVM.On("FocusSchedule").Return((*presenter.SchedulePreview)(nil)).Maybe()
	s.plannerVM.On("RecoverySchedule").Return((*presenter.SchedulePreview)(nil)).Maybe()
	s.plannerVM.On("ActiveSchedule").Return((*presenter.ActiveScheduleState)(nil)).Maybe()
	s.timerVM.On("IsRunning").Return(false).Maybe()
	s.timerVM.On("ActiveSegment").Return(0).Maybe()
	s.timerVM.On("ElapsedFraction").Return(0.0).Maybe()
	s.timerVM.On("IsFlashVisible").Return(false).Maybe()
	s.timerVM.On("CurrentTaskName").Return("").Maybe()
	s.timerVM.On("BlockType").Return(planner.BlockFocus).Maybe()
}

// setupStepDefaults configures mock expectations for a specific wizard step.
func (s *PlannerViewSuite) setupStepDefaults(step presenter.WizardStep) {
	s.plannerVM.On("CurrentStep").Return(step).Maybe()
	s.plannerVM.On("HasActivePlan").Return(step == presenter.StepActive).Maybe()
	s.plannerVM.On("AvailableTasks").Return([]presenter.TodoRow{}).Maybe()
	s.plannerVM.On("Estimates").Return([]presenter.TaskEstimateRow{}).Maybe()
	s.plannerVM.On("EstimateSummary").Return(presenter.EstimateSummary{}).Maybe()
	s.plannerVM.On("FocusSchedule").Return((*presenter.SchedulePreview)(nil)).Maybe()
	s.plannerVM.On("RecoverySchedule").Return((*presenter.SchedulePreview)(nil)).Maybe()
	s.plannerVM.On("ActiveSchedule").Return((*presenter.ActiveScheduleState)(nil)).Maybe()
	s.timerVM.On("IsRunning").Return(false).Maybe()
	s.timerVM.On("ActiveSegment").Return(0).Maybe()
	s.timerVM.On("ElapsedFraction").Return(0.0).Maybe()
	s.timerVM.On("IsFlashVisible").Return(false).Maybe()
	s.timerVM.On("CurrentTaskName").Return("").Maybe()
	s.timerVM.On("BlockType").Return(planner.BlockFocus).Maybe()
}

// setupActiveWithSchedule configures mock for active state with a schedule.
func (s *PlannerViewSuite) setupActiveWithSchedule() {
	s.setupStepDefaults(presenter.StepActive)
	// Override ActiveSchedule to return real data.
	s.plannerVM.ExpectedCalls = filterCalls(s.plannerVM.ExpectedCalls, "ActiveSchedule")
	s.plannerVM.On("ActiveSchedule").Return(&presenter.ActiveScheduleState{
		Blocks: []presenter.TimeBlockPreview{
			{Type: "focus", TaskName: "Task A"},
		},
		CurrentIndex: 0,
	}).Maybe()
}

// filterCalls removes expected calls for the given method name.
func filterCalls(calls []*mock.Call, method string) []*mock.Call {
	var filtered []*mock.Call
	for _, c := range calls {
		if c.Method != method {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// --- Existing Tests (Button Visibility) ---

func (s *PlannerViewSuite) TestNewPlannerViewReturnsNonNil() {
	s.setupIdleDefaults()

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	s.NotNil(view, "NewPlannerView should return a non-nil component")
}

func (s *PlannerViewSuite) TestContainerIsNonNil() {
	s.setupIdleDefaults()

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	s.NotNil(view.Container(), "Container should return a non-nil fyne.Container")
}

func (s *PlannerViewSuite) TestPlanButtonVisibleAtIdle() {
	s.setupIdleDefaults()

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	s.True(view.PlanButton().Visible(),
		"Plan My Day button should be visible when step is Idle")
}

func (s *PlannerViewSuite) TestPlanButtonHiddenAtTaskSelect() {
	s.setupStepDefaults(presenter.StepTaskSelect)

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	s.False(view.PlanButton().Visible(),
		"Plan My Day button should be hidden when step is TaskSelect")
}

func (s *PlannerViewSuite) TestPlanButtonHiddenAtActive() {
	s.setupStepDefaults(presenter.StepActive)

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	s.False(view.PlanButton().Visible(),
		"Plan My Day button should be hidden when step is Active")
}

func (s *PlannerViewSuite) TestNextButtonVisibleAtTaskSelect() {
	s.setupStepDefaults(presenter.StepTaskSelect)

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	s.True(view.NextButton().Visible(),
		"Next button should be visible during TaskSelect step")
}

func (s *PlannerViewSuite) TestNextButtonVisibleAtEstimates() {
	s.setupStepDefaults(presenter.StepEstimates)

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	s.True(view.NextButton().Visible(),
		"Next button should be visible during Estimates step")
}

func (s *PlannerViewSuite) TestNextButtonVisibleAtPriority() {
	s.setupStepDefaults(presenter.StepPriority)

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	s.True(view.NextButton().Visible(),
		"Next button should be visible during Priority step")
}

func (s *PlannerViewSuite) TestNextButtonHiddenAtIdle() {
	s.setupIdleDefaults()

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	s.False(view.NextButton().Visible(),
		"Next button should be hidden when step is Idle")
}

func (s *PlannerViewSuite) TestNextButtonHiddenAtActive() {
	s.setupStepDefaults(presenter.StepActive)

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	s.False(view.NextButton().Visible(),
		"Next button should be hidden when step is Active")
}

func (s *PlannerViewSuite) TestNextButtonHiddenAtSchedule() {
	s.setupStepDefaults(presenter.StepSchedule)

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	s.False(view.NextButton().Visible(),
		"Next button should be hidden when step is Schedule (use schedule selection instead)")
}

func (s *PlannerViewSuite) TestBackButtonVisibleAtTaskSelect() {
	s.setupStepDefaults(presenter.StepTaskSelect)

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	s.True(view.BackButton().Visible(),
		"Back button should be visible during TaskSelect step")
}

func (s *PlannerViewSuite) TestBackButtonVisibleAtEstimates() {
	s.setupStepDefaults(presenter.StepEstimates)

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	s.True(view.BackButton().Visible(),
		"Back button should be visible during Estimates step")
}

func (s *PlannerViewSuite) TestBackButtonVisibleAtPriority() {
	s.setupStepDefaults(presenter.StepPriority)

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	s.True(view.BackButton().Visible(),
		"Back button should be visible during Priority step")
}

func (s *PlannerViewSuite) TestBackButtonVisibleAtSchedule() {
	s.setupStepDefaults(presenter.StepSchedule)

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	s.True(view.BackButton().Visible(),
		"Back button should be visible during Schedule step")
}

func (s *PlannerViewSuite) TestBackButtonHiddenAtIdle() {
	s.setupIdleDefaults()

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	s.False(view.BackButton().Visible(),
		"Back button should be hidden when step is Idle")
}

func (s *PlannerViewSuite) TestBackButtonHiddenAtActive() {
	s.setupStepDefaults(presenter.StepActive)

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	s.False(view.BackButton().Visible(),
		"Back button should be hidden when step is Active")
}

func (s *PlannerViewSuite) TestCompleteTaskButtonVisibleAtActive() {
	s.setupStepDefaults(presenter.StepActive)

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	s.True(view.CompleteTaskButton().Visible(),
		"Complete Task button should be visible during Active step")
}

func (s *PlannerViewSuite) TestCompleteTaskButtonHiddenAtIdle() {
	s.setupIdleDefaults()

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	s.False(view.CompleteTaskButton().Visible(),
		"Complete Task button should be hidden when step is Idle")
}

func (s *PlannerViewSuite) TestCompleteTaskButtonHiddenAtTaskSelect() {
	s.setupStepDefaults(presenter.StepTaskSelect)

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	s.False(view.CompleteTaskButton().Visible(),
		"Complete Task button should be hidden during wizard steps")
}

func (s *PlannerViewSuite) TestAbandonButtonVisibleAtActive() {
	s.setupStepDefaults(presenter.StepActive)

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	s.True(view.AbandonButton().Visible(),
		"Abandon Plan button should be visible during Active step")
}

func (s *PlannerViewSuite) TestAbandonButtonHiddenAtIdle() {
	s.setupIdleDefaults()

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	s.False(view.AbandonButton().Visible(),
		"Abandon Plan button should be hidden when step is Idle")
}

func (s *PlannerViewSuite) TestAbandonButtonHiddenAtEstimates() {
	s.setupStepDefaults(presenter.StepEstimates)

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	s.False(view.AbandonButton().Visible(),
		"Abandon Plan button should be hidden during wizard steps")
}

func (s *PlannerViewSuite) TestRefreshUpdatesButtonVisibility() {
	s.setupIdleDefaults()

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	// Verify idle state: Plan visible, Next/Back/Complete/Abandon hidden.
	s.True(view.PlanButton().Visible(), "Plan button should be visible at idle")
	s.False(view.NextButton().Visible(), "Next button should be hidden at idle")
	s.False(view.BackButton().Visible(), "Back button should be hidden at idle")
	s.False(view.CompleteTaskButton().Visible(), "Complete button should be hidden at idle")
	s.False(view.AbandonButton().Visible(), "Abandon button should be hidden at idle")

	// Reconfigure mock for TaskSelect step.
	s.plannerVM = new(mockPlannerViewModel)
	s.timerVM = new(mockTimerViewModel)
	s.setupStepDefaults(presenter.StepTaskSelect)
	view.SetPlannerModel(s.plannerVM)
	view.SetTimerModel(s.timerVM)

	view.Refresh()

	// After refresh with TaskSelect: Plan hidden, Next/Back visible, Complete/Abandon hidden.
	s.False(view.PlanButton().Visible(), "Plan button should be hidden after refresh to TaskSelect")
	s.True(view.NextButton().Visible(), "Next button should be visible after refresh to TaskSelect")
	s.True(view.BackButton().Visible(), "Back button should be visible after refresh to TaskSelect")
	s.False(view.CompleteTaskButton().Visible(), "Complete button should be hidden after refresh to TaskSelect")
	s.False(view.AbandonButton().Visible(), "Abandon button should be hidden after refresh to TaskSelect")
}

// --- New Tests: No-Plan State ---

func (s *PlannerViewSuite) TestNoPlanShowsPlaceholderMessage() {
	s.setupIdleDefaults()

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	validMessages := []string{
		"Who even knows",
		"It's your time you're wasting",
		"A goal without a plan is just a wish",
		"Winging it, are we?",
		"The plan is there is no plan",
		"Chaos is also a strategy, I suppose",
		"Bold of you to go planless",
	}

	text := view.PlaceholderText()
	s.Contains(validMessages, text,
		"PlaceholderText should return one of the 7 valid placeholder messages, got: %q", text)
}

func (s *PlannerViewSuite) TestNoPlanShowsPlanButton() {
	s.setupIdleDefaults()

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	s.True(view.PlanButton().Visible(),
		"Plan My Day button should be visible in no-plan state")
}

func (s *PlannerViewSuite) TestNoPlanHidesAbandonButton() {
	s.setupIdleDefaults()

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	s.False(view.AbandonButton().Visible(),
		"Abandon button should be hidden in no-plan state")
}

func (s *PlannerViewSuite) TestNoPlanHidesScheduleTree() {
	s.setupIdleDefaults()

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	s.Nil(view.ScheduleTree(),
		"ScheduleTree should be nil when there is no active plan")
}

func (s *PlannerViewSuite) TestPlanButtonNavigatesToWizard() {
	s.setupIdleDefaults()

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	// Tap the Plan button
	view.PlanButton().OnTapped()

	s.Equal(ui.ViewWizard, s.router.CurrentView(),
		"Tapping Plan My Day should navigate to ViewWizard via the router")
}

func (s *PlannerViewSuite) TestPlanButtonInvokesOnPlanMyDayCallback() {
	s.setupIdleDefaults()

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	called := false
	view.SetOnPlanMyDay(func() { called = true })

	view.PlanButton().OnTapped()

	s.True(called,
		"Tapping Plan My Day should invoke the onPlanMyDay callback")
}

// --- New Tests: Active Plan State ---

func (s *PlannerViewSuite) TestActivePlanShowsScheduleTree() {
	s.setupActiveWithSchedule()

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	s.NotNil(view.ScheduleTree(),
		"ScheduleTree should be non-nil when there is an active plan")
}

func (s *PlannerViewSuite) TestActivePlanHidesPlanButton() {
	s.setupStepDefaults(presenter.StepActive)

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	s.False(view.PlanButton().Visible(),
		"Plan My Day button should be hidden when there is an active plan")
}

func (s *PlannerViewSuite) TestActivePlanShowsAbandonButton() {
	s.setupStepDefaults(presenter.StepActive)

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	s.True(view.AbandonButton().Visible(),
		"Abandon Plan button should be visible when there is an active plan")
}

func (s *PlannerViewSuite) TestActivePlanHidesPlaceholder() {
	s.setupActiveWithSchedule()

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	s.Empty(view.PlaceholderText(),
		"PlaceholderText should return empty string when there is an active plan")
}

// --- Refresh: Idle to Active ---

func (s *PlannerViewSuite) TestRefreshUpdatesContent() {
	s.setupIdleDefaults()

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	// Initially idle: placeholder visible, schedule tree nil.
	s.NotEmpty(view.PlaceholderText(), "Should have placeholder text in idle state")
	s.Nil(view.ScheduleTree(), "Schedule tree should be nil in idle state")

	// Transition to active state.
	s.plannerVM = new(mockPlannerViewModel)
	s.timerVM = new(mockTimerViewModel)
	s.setupStepDefaults(presenter.StepActive)
	// Override ActiveSchedule to return real data.
	s.plannerVM.ExpectedCalls = filterCalls(s.plannerVM.ExpectedCalls, "ActiveSchedule")
	s.plannerVM.On("ActiveSchedule").Return(&presenter.ActiveScheduleState{
		Blocks: []presenter.TimeBlockPreview{
			{Type: "focus", TaskName: "Task A"},
		},
		CurrentIndex: 0,
	}).Maybe()
	view.SetPlannerModel(s.plannerVM)
	view.SetTimerModel(s.timerVM)

	view.Refresh()

	// After refresh to active: placeholder gone, schedule tree present.
	s.Empty(view.PlaceholderText(), "Placeholder should be empty after transitioning to active")
	s.NotNil(view.ScheduleTree(), "Schedule tree should be non-nil after transitioning to active")
}

// --- Behavior 3: Horizontal Split Layout ---

func (s *PlannerViewSuite) TestNoPlanPlaceholderLabelInWidgetTree() {
	s.setupIdleDefaults()

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	expectedText := view.PlaceholderText()
	s.Require().NotEmpty(expectedText, "PlaceholderText should be non-empty in idle state")

	root := view.Container()
	label, found := uitest.FindWidget[*widget.Label](root, func(l *widget.Label) bool {
		return l.Text == expectedText
	})

	s.Require().True(found,
		"Container widget tree should contain a *widget.Label with the placeholder text %q", expectedText)
	s.Equal(expectedText, label.Text,
		"Found label text should match PlaceholderText()")
}

// --- Bug 073: SetOnNext / SetOnBack callback wiring ---

func (s *PlannerViewSuite) TestSetOnNextInvokesCallbackOnTap() {
	s.setupStepDefaults(presenter.StepTaskSelect)

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	called := false
	view.SetOnNext(func() { called = true })

	view.NextButton().OnTapped()

	s.True(called,
		"SetOnNext callback should be invoked when NextButton is tapped")
}

func (s *PlannerViewSuite) TestSetOnBackInvokesCallbackOnTap() {
	s.setupStepDefaults(presenter.StepTaskSelect)

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	called := false
	view.SetOnBack(func() { called = true })

	view.BackButton().OnTapped()

	s.True(called,
		"SetOnBack callback should be invoked when BackButton is tapped")
}

func (s *PlannerViewSuite) TestSetOnCompleteTaskInvokesCallbackOnTap() {
	s.setupStepDefaults(presenter.StepActive)

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	called := false
	view.SetOnCompleteTask(func() { called = true })

	view.CompleteTaskButton().OnTapped()

	s.True(called,
		"SetOnCompleteTask callback should be invoked when CompleteTaskButton is tapped")
}

func (s *PlannerViewSuite) TestSetOnAbandonPlanInvokesCallbackOnTap() {
	s.setupStepDefaults(presenter.StepActive)

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	called := false
	view.SetOnAbandonPlan(func() { called = true })

	view.AbandonButton().OnTapped()

	s.True(called,
		"SetOnAbandonPlan callback should be invoked when AbandonButton is tapped")
}

func (s *PlannerViewSuite) TestContainerHasHorizontalSplit() {
	s.setupIdleDefaults()

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, s.todoVM)

	root := view.Container()
	split, found := uitest.FindWidget[*container.Split](root, func(sp *container.Split) bool {
		return sp.Horizontal
	})

	s.Require().True(found,
		"PlannerView container should contain a horizontal *container.Split")
	s.NotNil(split.Leading,
		"Split leading pane (buttons) should not be nil")
	s.NotNil(split.Trailing,
		"Split trailing pane (todo list) should not be nil")
}
