package ui_test

import (
	"context"
	"strings"
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

func (m *mockPlannerViewModel) ActiveSchedule() *presenter.ActiveScheduleState {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*presenter.ActiveScheduleState)
}

func (m *mockPlannerViewModel) CurrentFocusTask(ctx context.Context) (*presenter.TodoRow, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*presenter.TodoRow), args.Error(1)
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
	s.plannerVM.On("ActiveSchedule").Return((*presenter.ActiveScheduleState)(nil)).Maybe()
	s.plannerVM.On("CurrentFocusTask", mock.Anything).Return((*presenter.TodoRow)(nil), nil).Maybe()
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
	s.plannerVM.On("ActiveSchedule").Return((*presenter.ActiveScheduleState)(nil)).Maybe()
	s.plannerVM.On("CurrentFocusTask", mock.Anything).Return((*presenter.TodoRow)(nil), nil).Maybe()
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

// --- Constructor ---

func (s *PlannerViewSuite) TestNewPlannerViewReturnsNonNil() {
	s.setupIdleDefaults()
	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)
	s.NotNil(view)
}

func (s *PlannerViewSuite) TestContainerIsNonNil() {
	s.setupIdleDefaults()
	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)
	s.NotNil(view.Container())
}

// --- Button Visibility ---

func (s *PlannerViewSuite) TestPlanButtonVisibleAtIdle() {
	s.setupIdleDefaults()
	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)
	s.True(view.PlanButton().Visible(),
		"Plan My Day button should be visible when step is Idle")
}

func (s *PlannerViewSuite) TestPlanButtonHiddenAtSchedule() {
	s.setupStepDefaults(presenter.StepSchedule)
	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)
	s.False(view.PlanButton().Visible(),
		"Plan My Day button should be hidden when step is Schedule")
}

func (s *PlannerViewSuite) TestPlanButtonHiddenAtActive() {
	s.setupStepDefaults(presenter.StepActive)
	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)
	s.False(view.PlanButton().Visible(),
		"Plan My Day button should be hidden when step is Active")
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
	s.False(view.BackButton().Visible(), "Back button should be hidden at idle")
}

func (s *PlannerViewSuite) TestCompleteTaskButtonVisibleAtActive() {
	s.setupStepDefaults(presenter.StepActive)
	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)
	s.True(view.CompleteTaskButton().Visible())
}

func (s *PlannerViewSuite) TestCompleteTaskButtonHiddenAtIdle() {
	s.setupIdleDefaults()
	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)
	s.False(view.CompleteTaskButton().Visible())
}

func (s *PlannerViewSuite) TestAbandonButtonVisibleAtActive() {
	s.setupStepDefaults(presenter.StepActive)
	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)
	s.True(view.AbandonButton().Visible())
}

func (s *PlannerViewSuite) TestAbandonButtonHiddenAtIdle() {
	s.setupIdleDefaults()
	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)
	s.False(view.AbandonButton().Visible())
}

// --- No-Plan State ---

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

func (s *PlannerViewSuite) TestNoPlanHidesScheduleTree() {
	s.setupIdleDefaults()
	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)
	s.Nil(view.ScheduleTree())
}

func (s *PlannerViewSuite) TestPlanButtonNavigatesToWizard() {
	s.setupIdleDefaults()
	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)
	view.PlanButton().OnTapped()
	s.Equal(ui.ViewWizard, s.router.CurrentView())
}

func (s *PlannerViewSuite) TestPlanButtonInvokesOnPlanMyDayCallback() {
	s.setupIdleDefaults()
	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	called := false
	view.SetOnPlanMyDay(func() { called = true })

	view.PlanButton().OnTapped()
	s.True(called)
}

// --- Active Plan State ---

func (s *PlannerViewSuite) TestActivePlanShowsScheduleTree() {
	s.setupActiveWithSchedule()
	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)
	s.NotNil(view.ScheduleTree())
}

func (s *PlannerViewSuite) TestActivePlanHidesPlaceholder() {
	s.setupActiveWithSchedule()
	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)
	s.Empty(view.PlaceholderText())
}

// --- Focus task hint (Feature 107 WP5) ---

func (s *PlannerViewSuite) TestActivePlanShowsFocusTaskHintWhenAvailable() {
	s.setupStepDefaults(presenter.StepActive)
	s.plannerVM.ExpectedCalls = filterCalls(s.plannerVM.ExpectedCalls, "ActiveSchedule")
	s.plannerVM.On("ActiveSchedule").Return(&presenter.ActiveScheduleState{
		Blocks: []presenter.TimeBlockPreview{
			{Type: "focus", TaskName: "Task A"},
		},
		CurrentIndex: 0,
	}).Maybe()
	s.plannerVM.ExpectedCalls = filterCalls(s.plannerVM.ExpectedCalls, "CurrentFocusTask")
	s.plannerVM.On("CurrentFocusTask", mock.Anything).
		Return(&presenter.TodoRow{Title: "Top priority task", Priority: 1}, nil).Maybe()

	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	s.Equal("Now: Top priority task", view.FocusTaskText(),
		"active plan with a focus task should render a 'Now: <title>' hint")

	root := view.Container()
	_, found := uitest.FindWidget[*widget.Label](root, func(l *widget.Label) bool {
		return l.Text == "Now: Top priority task"
	})
	s.True(found, "focus task hint label should appear in the widget tree")
}

func (s *PlannerViewSuite) TestActivePlanHidesFocusTaskHintWhenAbsent() {
	s.setupActiveWithSchedule()
	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	s.Empty(view.FocusTaskText(),
		"active plan without a focus task should leave the hint empty")

	root := view.Container()
	_, found := uitest.FindWidget[*widget.Label](root, func(l *widget.Label) bool {
		return strings.HasPrefix(l.Text, "Now: ")
	})
	s.False(found, "focus task hint label should not appear when there's no focus task")
}

func (s *PlannerViewSuite) TestIdleStateNoFocusTaskHint() {
	s.setupIdleDefaults()
	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	s.Empty(view.FocusTaskText())
}

// --- Behavior: Horizontal Split Layout ---

func (s *PlannerViewSuite) TestNoPlanPlaceholderLabelInWidgetTree() {
	s.setupIdleDefaults()
	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	expectedText := view.PlaceholderText()
	s.Require().NotEmpty(expectedText)

	root := view.Container()
	label, found := uitest.FindWidget[*widget.Label](root, func(l *widget.Label) bool {
		return l.Text == expectedText
	})
	s.Require().True(found)
	s.Equal(expectedText, label.Text)
}

func (s *PlannerViewSuite) TestSetOnCompleteTaskInvokesCallbackOnTap() {
	s.setupStepDefaults(presenter.StepActive)
	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	called := false
	view.SetOnCompleteTask(func() { called = true })
	view.CompleteTaskButton().OnTapped()
	s.True(called)
}

func (s *PlannerViewSuite) TestSetOnAbandonPlanInvokesCallbackOnTap() {
	s.setupStepDefaults(presenter.StepActive)
	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, nil)

	called := false
	view.SetOnAbandonPlan(func() { called = true })
	view.AbandonButton().OnTapped()
	s.True(called)
}

func (s *PlannerViewSuite) TestContainerHasHorizontalSplit() {
	s.setupIdleDefaults()
	view := ui.NewPlannerView(s.plannerVM, s.timerVM, s.router, s.todoVM)

	root := view.Container()
	split, found := uitest.FindWidget[*container.Split](root, func(sp *container.Split) bool {
		return sp.Horizontal
	})

	s.Require().True(found)
	s.NotNil(split.Leading)
	s.NotNil(split.Trailing)
}
