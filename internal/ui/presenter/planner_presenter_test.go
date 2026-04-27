package presenter_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/service/calendar"
	"github.com/CreateFutureMWilkinson/cue/internal/service/planner"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
)

// --- Interfaces ---
// NOTE: These interfaces (TodoQuerier, CategoryQuerier, ScheduleGenerator)
// should live in the presenter package (e.g., interfaces.go or planner_presenter.go).
// They are defined here for test compilation reference only.

// --- Mocks ---

type mockTodoQuerier struct {
	mock.Mock
}

func (m *mockTodoQuerier) QueryFiltered(ctx context.Context, filter repository.TodoFilter) ([]*repository.Todo, int, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*repository.Todo), args.Int(1), args.Error(2)
}

func (m *mockTodoQuerier) Insert(ctx context.Context, todo *repository.Todo) error {
	args := m.Called(ctx, todo)
	return args.Error(0)
}

func (m *mockTodoQuerier) Update(ctx context.Context, todo *repository.Todo) error {
	args := m.Called(ctx, todo)
	return args.Error(0)
}

func (m *mockTodoQuerier) Complete(ctx context.Context, id uuid.UUID, completedAt time.Time) error {
	args := m.Called(ctx, id, completedAt)
	return args.Error(0)
}

type mockCategoryQuerier struct {
	mock.Mock
}

func (m *mockCategoryQuerier) QueryAll(ctx context.Context, withCounts bool) ([]*repository.CategoryWithCount, error) {
	args := m.Called(ctx, withCounts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*repository.CategoryWithCount), args.Error(1)
}

type mockCalendarProvider struct {
	mock.Mock
}

func (m *mockCalendarProvider) FetchEvents(ctx context.Context, date time.Time) ([]calendar.CalendarEvent, error) {
	args := m.Called(ctx, date)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]calendar.CalendarEvent), args.Error(1)
}

type mockScheduleGenerator struct {
	mock.Mock
}

func (m *mockScheduleGenerator) GenerateSchedules(
	ctx context.Context,
	tasks []planner.TaskEstimate,
	events []calendar.CalendarEvent,
	targetDate time.Time,
) (*planner.DaySchedule, *planner.DaySchedule, error) {
	args := m.Called(ctx, tasks, events, targetDate)
	var focus, recovery *planner.DaySchedule
	if args.Get(0) != nil {
		focus = args.Get(0).(*planner.DaySchedule)
	}
	if args.Get(1) != nil {
		recovery = args.Get(1).(*planner.DaySchedule)
	}
	return focus, recovery, args.Error(2)
}

type mockTaskEstimator struct {
	mock.Mock
}

func (m *mockTaskEstimator) EstimateMinutes(ctx context.Context, title string, description string) (int, error) {
	args := m.Called(ctx, title, description)
	return args.Int(0), args.Error(1)
}

type mockScheduleRepo struct {
	mock.Mock
}

func (m *mockScheduleRepo) Save(ctx context.Context, schedule *repository.Schedule) error {
	args := m.Called(ctx, schedule)
	return args.Error(0)
}

func (m *mockScheduleRepo) LoadByDate(ctx context.Context, date time.Time) (*repository.Schedule, error) {
	args := m.Called(ctx, date)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.Schedule), args.Error(1)
}

func (m *mockScheduleRepo) Delete(ctx context.Context, date time.Time) error {
	args := m.Called(ctx, date)
	return args.Error(0)
}

type mockClock struct {
	now time.Time
}

func (m *mockClock) Now() time.Time {
	return m.now
}

// --- Suite ---

type PlannerPresenterSuite struct {
	suite.Suite
	todos      *mockTodoQuerier
	categories *mockCategoryQuerier
	cal        *mockCalendarProvider
	generator  *mockScheduleGenerator
	estimator  *mockTaskEstimator
	schedRepo  *mockScheduleRepo
	clock      *mockClock
	presenter  *presenter.PlannerPresenter
	ctx        context.Context
}

func TestPlannerPresenter(t *testing.T) {
	suite.Run(t, new(PlannerPresenterSuite))
}

func (s *PlannerPresenterSuite) SetupTest() {
	s.todos = new(mockTodoQuerier)
	s.categories = new(mockCategoryQuerier)
	s.cal = new(mockCalendarProvider)
	s.generator = new(mockScheduleGenerator)
	s.estimator = new(mockTaskEstimator)
	s.schedRepo = new(mockScheduleRepo)
	s.clock = &mockClock{now: time.Date(2026, 3, 30, 9, 0, 0, 0, time.UTC)} // Monday 9am
	s.ctx = context.Background()

	p, err := presenter.NewPlannerPresenter(
		s.todos,
		s.categories,
		s.cal,
		s.generator,
		s.estimator,
		s.schedRepo,
		s.clock,
	)
	s.Require().NoError(err)
	s.presenter = p
}

// --- 1. Constructor validation ---

func (s *PlannerPresenterSuite) TestNewPlannerPresenterNilTodosReturnsError() {
	_, err := presenter.NewPlannerPresenter(nil, s.categories, s.cal, s.generator, s.estimator, s.schedRepo, s.clock)
	s.Error(err)
}

func (s *PlannerPresenterSuite) TestNewPlannerPresenterNilCategoriesReturnsError() {
	_, err := presenter.NewPlannerPresenter(s.todos, nil, s.cal, s.generator, s.estimator, s.schedRepo, s.clock)
	s.Error(err)
}

func (s *PlannerPresenterSuite) TestNewPlannerPresenterNilCalendarReturnsError() {
	_, err := presenter.NewPlannerPresenter(s.todos, s.categories, nil, s.generator, s.estimator, s.schedRepo, s.clock)
	s.Error(err)
}

func (s *PlannerPresenterSuite) TestNewPlannerPresenterNilGeneratorReturnsError() {
	_, err := presenter.NewPlannerPresenter(s.todos, s.categories, s.cal, nil, s.estimator, s.schedRepo, s.clock)
	s.Error(err)
}

func (s *PlannerPresenterSuite) TestNewPlannerPresenterNilEstimatorReturnsError() {
	_, err := presenter.NewPlannerPresenter(s.todos, s.categories, s.cal, s.generator, nil, s.schedRepo, s.clock)
	s.Error(err)
}

func (s *PlannerPresenterSuite) TestNewPlannerPresenterNilScheduleRepoReturnsError() {
	_, err := presenter.NewPlannerPresenter(s.todos, s.categories, s.cal, s.generator, s.estimator, nil, s.clock)
	s.Error(err)
}

func (s *PlannerPresenterSuite) TestNewPlannerPresenterNilClockReturnsError() {
	_, err := presenter.NewPlannerPresenter(s.todos, s.categories, s.cal, s.generator, s.estimator, s.schedRepo, nil)
	s.Error(err)
}

// --- 2. StartPlanning ---

func (s *PlannerPresenterSuite) TestStartPlanningTransitionsToTaskSelect() {
	todos := []*repository.Todo{
		{ID: uuid.New(), Title: "Task A", Priority: 1},
		{ID: uuid.New(), Title: "Task B", Priority: 2},
	}
	s.todos.On("QueryFiltered", mock.Anything, mock.Anything).Return(todos, len(todos), nil)

	err := s.presenter.StartPlanning(s.ctx)
	s.Require().NoError(err)
	s.Equal(presenter.StepTaskSelect, s.presenter.CurrentStep())
	s.todos.AssertCalled(s.T(), "QueryFiltered", mock.Anything, mock.Anything)
}

func (s *PlannerPresenterSuite) TestStartPlanningLoadsTodos() {
	todo1 := &repository.Todo{ID: uuid.New(), Title: "Write report", Priority: 1}
	todo2 := &repository.Todo{ID: uuid.New(), Title: "Code review", Priority: 2}
	s.todos.On("QueryFiltered", mock.Anything, mock.Anything).Return([]*repository.Todo{todo1, todo2}, 2, nil)

	err := s.presenter.StartPlanning(s.ctx)
	s.Require().NoError(err)

	tasks := s.presenter.AvailableTasks()
	s.Len(tasks, 2)
	s.Equal("Write report", tasks[0].Title)
	s.Equal("Code review", tasks[1].Title)
}

// --- 3. CurrentStep ---

func (s *PlannerPresenterSuite) TestCurrentStepReturnsIdleInitially() {
	s.Equal(presenter.StepIdle, s.presenter.CurrentStep())
}

// --- 4. NextStep from TaskSelect ---

func (s *PlannerPresenterSuite) TestNextStepFromTaskSelectGeneratesEstimates() {
	todo := &repository.Todo{ID: uuid.New(), Title: "Design API", Priority: 1, Description: "REST endpoints"}
	s.todos.On("QueryFiltered", mock.Anything, mock.Anything).Return([]*repository.Todo{todo}, 1, nil)
	s.estimator.On("EstimateMinutes", mock.Anything, "Design API", "REST endpoints").Return(3, nil)

	err := s.presenter.StartPlanning(s.ctx)
	s.Require().NoError(err)

	s.presenter.SelectTask(todo.ID, true)

	err = s.presenter.NextStep(s.ctx)
	s.Require().NoError(err)
	s.Equal(presenter.StepEstimates, s.presenter.CurrentStep())

	estimates := s.presenter.Estimates()
	s.Require().Len(estimates, 1)
	s.Equal(3, estimates[0].EstimatedPomos)
}

// --- 5. NextStep from Estimates ---

func (s *PlannerPresenterSuite) TestNextStepFromEstimatesTransitionsToPriority() {
	s.advanceToEstimates()

	err := s.presenter.NextStep(s.ctx)
	s.Require().NoError(err)
	s.Equal(presenter.StepPriority, s.presenter.CurrentStep())
}

// --- 6. NextStep from Priority ---

func (s *PlannerPresenterSuite) TestNextStepFromPriorityGeneratesSchedules() {
	s.advanceToPriority()

	focusSchedule := &planner.DaySchedule{
		ID:       uuid.New(),
		Strategy: "focus-maximized",
		Blocks: []planner.TimeBlock{
			{Start: s.clock.now, End: s.clock.now.Add(25 * time.Minute), Type: planner.BlockFocus, TaskName: "Task A"},
		},
	}
	recoverySchedule := &planner.DaySchedule{
		ID:       uuid.New(),
		Strategy: "recovery-balanced",
		Blocks: []planner.TimeBlock{
			{Start: s.clock.now, End: s.clock.now.Add(25 * time.Minute), Type: planner.BlockFocus, TaskName: "Task A"},
			{Start: s.clock.now.Add(25 * time.Minute), End: s.clock.now.Add(30 * time.Minute), Type: planner.BlockShortBreak},
		},
	}
	s.cal.On("FetchEvents", mock.Anything, mock.Anything).Return([]calendar.CalendarEvent{}, nil)
	s.generator.On("GenerateSchedules", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(focusSchedule, recoverySchedule, nil)

	err := s.presenter.NextStep(s.ctx)
	s.Require().NoError(err)
	s.Equal(presenter.StepSchedule, s.presenter.CurrentStep())

	focus := s.presenter.FocusSchedule()
	s.Require().NotNil(focus)
	s.Equal("focus-maximized", focus.Strategy)

	recovery := s.presenter.RecoverySchedule()
	s.Require().NotNil(recovery)
	s.Equal("recovery-balanced", recovery.Strategy)
}

// --- 7. NextStep from Schedule ---

func (s *PlannerPresenterSuite) TestNextStepFromScheduleReturnsError() {
	s.advanceToSchedule()

	err := s.presenter.NextStep(s.ctx)
	s.Error(err)
}

// --- 8. NextStep with no tasks selected ---

func (s *PlannerPresenterSuite) TestNextStepWithNoTasksSelectedReturnsError() {
	s.todos.On("QueryFiltered", mock.Anything, mock.Anything).Return([]*repository.Todo{
		{ID: uuid.New(), Title: "Task A", Priority: 1},
	}, 1, nil)

	err := s.presenter.StartPlanning(s.ctx)
	s.Require().NoError(err)
	// Do not select any tasks

	err = s.presenter.NextStep(s.ctx)
	s.Error(err)
}

// --- 9. PreviousStep ---

func (s *PlannerPresenterSuite) TestPreviousStepFromEstimatesToTaskSelect() {
	s.advanceToEstimates()

	s.presenter.PreviousStep()
	s.Equal(presenter.StepTaskSelect, s.presenter.CurrentStep())
}

func (s *PlannerPresenterSuite) TestPreviousStepFromPriorityToEstimates() {
	s.advanceToPriority()

	s.presenter.PreviousStep()
	s.Equal(presenter.StepEstimates, s.presenter.CurrentStep())
}

func (s *PlannerPresenterSuite) TestPreviousStepFromScheduleToPriority() {
	s.advanceToSchedule()

	s.presenter.PreviousStep()
	s.Equal(presenter.StepPriority, s.presenter.CurrentStep())
}

// --- 10. PreviousStep from TaskSelect ---

func (s *PlannerPresenterSuite) TestPreviousStepFromTaskSelectReturnsToIdle() {
	s.todos.On("QueryFiltered", mock.Anything, mock.Anything).Return([]*repository.Todo{}, 0, nil)

	err := s.presenter.StartPlanning(s.ctx)
	s.Require().NoError(err)
	s.Equal(presenter.StepTaskSelect, s.presenter.CurrentStep())

	s.presenter.PreviousStep()
	s.Equal(presenter.StepIdle, s.presenter.CurrentStep())
}

// --- 11. AvailableTasks ---

func (s *PlannerPresenterSuite) TestAvailableTasksReturnsLoadedTodosWithSelectionState() {
	dueDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	cat := repository.Category{NameKey: "work"}
	todo := &repository.Todo{
		ID:         uuid.New(),
		Title:      "Review PR",
		Priority:   2,
		DueDate:    &dueDate,
		Categories: []repository.Category{cat},
	}
	s.todos.On("QueryFiltered", mock.Anything, mock.Anything).Return([]*repository.Todo{todo}, 1, nil)

	err := s.presenter.StartPlanning(s.ctx)
	s.Require().NoError(err)

	tasks := s.presenter.AvailableTasks()
	s.Require().Len(tasks, 1)
	s.Equal(todo.ID, tasks[0].ID)
	s.Equal("Review PR", tasks[0].Title)
	s.Equal(2, tasks[0].Priority)
	s.Equal(&dueDate, tasks[0].DueDate)
	s.Len(tasks[0].Categories, 1)
	s.False(tasks[0].Selected)
}

// --- 12. SelectTask ---

func (s *PlannerPresenterSuite) TestSelectTaskTogglesSelection() {
	todoID := uuid.New()
	s.todos.On("QueryFiltered", mock.Anything, mock.Anything).Return([]*repository.Todo{
		{ID: todoID, Title: "Task A", Priority: 1},
	}, 1, nil)

	err := s.presenter.StartPlanning(s.ctx)
	s.Require().NoError(err)

	s.presenter.SelectTask(todoID, true)
	tasks := s.presenter.AvailableTasks()
	s.True(tasks[0].Selected)

	s.presenter.SelectTask(todoID, false)
	tasks = s.presenter.AvailableTasks()
	s.False(tasks[0].Selected)
}

// --- 13. AddTask ---

func (s *PlannerPresenterSuite) TestAddTaskCreatesAndAppears() {
	s.todos.On("QueryFiltered", mock.Anything, mock.Anything).Return([]*repository.Todo{}, 0, nil)
	s.todos.On("Insert", mock.Anything, mock.AnythingOfType("*repository.Todo")).Return(nil)

	err := s.presenter.StartPlanning(s.ctx)
	s.Require().NoError(err)

	err = s.presenter.AddTask(s.ctx, "New Task", 3)
	s.Require().NoError(err)

	tasks := s.presenter.AvailableTasks()
	s.Require().Len(tasks, 1)
	s.Equal("New Task", tasks[0].Title)
	s.Equal(3, tasks[0].Priority)
	s.todos.AssertCalled(s.T(), "Insert", mock.Anything, mock.AnythingOfType("*repository.Todo"))
}

// --- 14. Estimates ---

func (s *PlannerPresenterSuite) TestEstimatesReturnsSelectedTasksWithPomodoros() {
	s.advanceToEstimates()

	estimates := s.presenter.Estimates()
	s.Require().Len(estimates, 1)
	s.Equal("Task A", estimates[0].Title)
	s.Equal(2, estimates[0].EstimatedPomos)
	s.Nil(estimates[0].UserOverride)
	s.Equal(2, estimates[0].EffectivePomos)
}

// --- 15. OverrideEstimate ---

func (s *PlannerPresenterSuite) TestOverrideEstimateSetsUserOverride() {
	s.advanceToEstimates()

	todoID := s.presenter.Estimates()[0].TodoID
	s.presenter.OverrideEstimate(todoID, 5)

	estimates := s.presenter.Estimates()
	s.Require().Len(estimates, 1)
	s.Require().NotNil(estimates[0].UserOverride)
	s.Equal(5, *estimates[0].UserOverride)
	s.Equal(5, estimates[0].EffectivePomos)
}

// --- 16. EstimateSummary ---

func (s *PlannerPresenterSuite) TestEstimateSummaryCalculatesTotals() {
	s.advanceToEstimates()

	summary := s.presenter.EstimateSummary()
	s.Equal(2, summary.TotalPomos) // Task A estimated at 2 pomos
	s.Greater(summary.AvailableBlocks, 0)
}

func (s *PlannerPresenterSuite) TestEstimateSummaryDetectsOverloaded() {
	todoID := uuid.New()
	todo := &repository.Todo{ID: todoID, Title: "Huge Task", Priority: 1}
	s.todos.On("QueryFiltered", mock.Anything, mock.Anything).Return([]*repository.Todo{todo}, 1, nil)
	// Estimate a very high number of pomodoros
	s.estimator.On("EstimateMinutes", mock.Anything, "Huge Task", "").Return(100, nil)

	err := s.presenter.StartPlanning(s.ctx)
	s.Require().NoError(err)
	s.presenter.SelectTask(todoID, true)

	err = s.presenter.NextStep(s.ctx)
	s.Require().NoError(err)

	summary := s.presenter.EstimateSummary()
	s.True(summary.Overloaded)
}

// --- 17. ReorderTask ---

func (s *PlannerPresenterSuite) TestReorderTaskMovesTaskPosition() {
	todoA := &repository.Todo{ID: uuid.New(), Title: "Task A", Priority: 1}
	todoB := &repository.Todo{ID: uuid.New(), Title: "Task B", Priority: 2}
	s.todos.On("QueryFiltered", mock.Anything, mock.Anything).Return([]*repository.Todo{todoA, todoB}, 2, nil)
	s.estimator.On("EstimateMinutes", mock.Anything, mock.Anything, mock.Anything).Return(1, nil)

	err := s.presenter.StartPlanning(s.ctx)
	s.Require().NoError(err)
	s.presenter.SelectTask(todoA.ID, true)
	s.presenter.SelectTask(todoB.ID, true)

	err = s.presenter.NextStep(s.ctx) // -> Estimates
	s.Require().NoError(err)
	err = s.presenter.NextStep(s.ctx) // -> Priority
	s.Require().NoError(err)

	// Move Task B (index 1) to index 0
	s.presenter.ReorderTask(1, 0)

	estimates := s.presenter.Estimates()
	s.Require().Len(estimates, 2)
	s.Equal("Task B", estimates[0].Title)
	s.Equal("Task A", estimates[1].Title)
}

// --- 18. FocusSchedule / RecoverySchedule ---

func (s *PlannerPresenterSuite) TestFocusAndRecoveryScheduleReturnPreviews() {
	s.advanceToSchedule()

	focus := s.presenter.FocusSchedule()
	s.Require().NotNil(focus)
	s.Equal("focus-maximized", focus.Strategy)
	s.Greater(len(focus.Blocks), 0)

	recovery := s.presenter.RecoverySchedule()
	s.Require().NotNil(recovery)
	s.Equal("recovery-balanced", recovery.Strategy)
}

// --- 19. SelectSchedule ---

func (s *PlannerPresenterSuite) TestSelectSchedulePersistsAndTransitionsToActive() {
	s.advanceToSchedule()

	s.schedRepo.On("Save", mock.Anything, mock.AnythingOfType("*repository.Schedule")).Return(nil)

	err := s.presenter.SelectSchedule(s.ctx, "focus-maximized")
	s.Require().NoError(err)
	s.Equal(presenter.StepActive, s.presenter.CurrentStep())
	s.schedRepo.AssertCalled(s.T(), "Save", mock.Anything, mock.AnythingOfType("*repository.Schedule"))
}

// --- 20. ActiveSchedule ---

func (s *PlannerPresenterSuite) TestActiveScheduleReturnsCurrentState() {
	s.advanceToActive()

	state := s.presenter.ActiveSchedule()
	s.Require().NotNil(state)
	s.GreaterOrEqual(len(state.Blocks), 1)
	s.GreaterOrEqual(state.CurrentIndex, 0)
}

// --- 21. CompleteCurrentTask ---

func (s *PlannerPresenterSuite) TestCompleteCurrentTaskMarksTaskDone() {
	s.advanceToActive()

	s.todos.On("Complete", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("time.Time")).Return(nil)

	err := s.presenter.CompleteCurrentTask(s.ctx)
	s.Require().NoError(err)
}

// --- 22. AbandonPlan ---

func (s *PlannerPresenterSuite) TestAbandonPlanDeletesScheduleReturnsToIdle() {
	s.advanceToActive()

	s.schedRepo.On("Delete", mock.Anything, mock.AnythingOfType("time.Time")).Return(nil)

	err := s.presenter.AbandonPlan(s.ctx)
	s.Require().NoError(err)
	s.Equal(presenter.StepIdle, s.presenter.CurrentStep())
}

// --- 23. HasActivePlan ---

func (s *PlannerPresenterSuite) TestHasActivePlanFalseInitially() {
	s.False(s.presenter.HasActivePlan())
}

func (s *PlannerPresenterSuite) TestHasActivePlanTrueAfterSelectSchedule() {
	s.advanceToActive()
	s.True(s.presenter.HasActivePlan())
}

// --- 24. LoadExistingPlan ---

func (s *PlannerPresenterSuite) TestLoadExistingPlanTransitionsToActive() {
	scheduleID := uuid.New()
	schedule := &repository.Schedule{
		ID:       scheduleID,
		Date:     time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC),
		Strategy: "focus-maximized",
		Blocks: []repository.ScheduleBlock{
			{
				Start:    s.clock.now,
				End:      s.clock.now.Add(25 * time.Minute),
				Type:     repository.ScheduleBlockFocus,
				TaskName: "Task A",
			},
		},
	}
	s.schedRepo.On("LoadByDate", mock.Anything, mock.Anything).Return(schedule, nil)

	err := s.presenter.LoadExistingPlan(s.ctx)
	s.Require().NoError(err)
	s.Equal(presenter.StepActive, s.presenter.CurrentStep())
	s.True(s.presenter.HasActivePlan())
}

// --- 25. LoadExistingPlan with no schedule ---

func (s *PlannerPresenterSuite) TestLoadExistingPlanNoScheduleStaysIdle() {
	s.schedRepo.On("LoadByDate", mock.Anything, mock.Anything).Return(nil, nil)

	err := s.presenter.LoadExistingPlan(s.ctx)
	s.Require().NoError(err)
	s.Equal(presenter.StepIdle, s.presenter.CurrentStep())
}

// --- 26. Calendar failure during wizard ---

func (s *PlannerPresenterSuite) TestCalendarFailureProceedsWithoutEvents() {
	s.advanceToPriorityWithCalendarFailure()

	// Should still transition to Schedule step without error
	s.Equal(presenter.StepSchedule, s.presenter.CurrentStep())
}

// --- 27. Estimation failure falls back to 1 pomodoro ---

func (s *PlannerPresenterSuite) TestEstimationFailureFallsBackToOnePomo() {
	todoID := uuid.New()
	todo := &repository.Todo{ID: todoID, Title: "Failing Task", Priority: 1}
	s.todos.On("QueryFiltered", mock.Anything, mock.Anything).Return([]*repository.Todo{todo}, 1, nil)
	s.estimator.On("EstimateMinutes", mock.Anything, "Failing Task", "").Return(0, errors.New("ollama down"))

	err := s.presenter.StartPlanning(s.ctx)
	s.Require().NoError(err)
	s.presenter.SelectTask(todoID, true)

	err = s.presenter.NextStep(s.ctx)
	s.Require().NoError(err)

	estimates := s.presenter.Estimates()
	s.Require().Len(estimates, 1)
	s.Equal(1, estimates[0].EstimatedPomos)
}

// --- Helpers ---

// advanceToEstimates sets up a presenter at StepEstimates with one selected task.
func (s *PlannerPresenterSuite) advanceToEstimates() {
	todoID := uuid.New()
	todo := &repository.Todo{ID: todoID, Title: "Task A", Priority: 1}
	s.todos.On("QueryFiltered", mock.Anything, mock.Anything).Return([]*repository.Todo{todo}, 1, nil)
	s.estimator.On("EstimateMinutes", mock.Anything, "Task A", "").Return(2, nil)

	err := s.presenter.StartPlanning(s.ctx)
	s.Require().NoError(err)
	s.presenter.SelectTask(todoID, true)

	err = s.presenter.NextStep(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(presenter.StepEstimates, s.presenter.CurrentStep())
}

// advanceToPriority sets up a presenter at StepPriority.
func (s *PlannerPresenterSuite) advanceToPriority() {
	s.advanceToEstimates()
	err := s.presenter.NextStep(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(presenter.StepPriority, s.presenter.CurrentStep())
}

// advanceToSchedule sets up a presenter at StepSchedule.
func (s *PlannerPresenterSuite) advanceToSchedule() {
	s.advanceToPriority()

	focusSchedule := &planner.DaySchedule{
		ID:       uuid.New(),
		Strategy: "focus-maximized",
		Blocks: []planner.TimeBlock{
			{Start: s.clock.now, End: s.clock.now.Add(25 * time.Minute), Type: planner.BlockFocus, TaskName: "Task A"},
		},
	}
	recoverySchedule := &planner.DaySchedule{
		ID:       uuid.New(),
		Strategy: "recovery-balanced",
		Blocks: []planner.TimeBlock{
			{Start: s.clock.now, End: s.clock.now.Add(25 * time.Minute), Type: planner.BlockFocus, TaskName: "Task A"},
			{Start: s.clock.now.Add(25 * time.Minute), End: s.clock.now.Add(30 * time.Minute), Type: planner.BlockShortBreak},
		},
	}
	s.cal.On("FetchEvents", mock.Anything, mock.Anything).Return([]calendar.CalendarEvent{}, nil)
	s.generator.On("GenerateSchedules", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(focusSchedule, recoverySchedule, nil)

	err := s.presenter.NextStep(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(presenter.StepSchedule, s.presenter.CurrentStep())
}

// advanceToActive sets up a presenter at StepActive with a saved schedule.
func (s *PlannerPresenterSuite) advanceToActive() {
	s.advanceToSchedule()

	s.schedRepo.On("Save", mock.Anything, mock.AnythingOfType("*repository.Schedule")).Return(nil)

	err := s.presenter.SelectSchedule(s.ctx, "focus-maximized")
	s.Require().NoError(err)
	s.Require().Equal(presenter.StepActive, s.presenter.CurrentStep())
}

// --- 28. SetOnStepChange ---

func (s *PlannerPresenterSuite) TestSetOnStepChangeFiresOnStartPlanning() {
	todos := []*repository.Todo{
		{ID: uuid.New(), Title: "Task A", Priority: 1},
	}
	s.todos.On("QueryFiltered", mock.Anything, mock.Anything).Return(todos, len(todos), nil)

	var received presenter.WizardStep
	s.presenter.SetOnStepChange(func(step presenter.WizardStep) {
		received = step
	})

	err := s.presenter.StartPlanning(s.ctx)
	s.Require().NoError(err)
	s.Equal(presenter.StepTaskSelect, received)
}

func (s *PlannerPresenterSuite) TestSetOnStepChangeFiresOnNextStep() {
	s.advanceToTaskSelect()

	var received presenter.WizardStep
	s.presenter.SetOnStepChange(func(step presenter.WizardStep) {
		received = step
	})

	todoID := s.presenter.AvailableTasks()[0].ID
	s.presenter.SelectTask(todoID, true)
	s.estimator.On("EstimateMinutes", mock.Anything, mock.Anything, mock.Anything).Return(2, nil)

	err := s.presenter.NextStep(s.ctx)
	s.Require().NoError(err)
	s.Equal(presenter.StepEstimates, received)
}

func (s *PlannerPresenterSuite) TestSetOnStepChangeFiresOnPreviousStep() {
	s.advanceToEstimates()

	var received presenter.WizardStep
	s.presenter.SetOnStepChange(func(step presenter.WizardStep) {
		received = step
	})

	s.presenter.PreviousStep()
	s.Equal(presenter.StepTaskSelect, received)
}

func (s *PlannerPresenterSuite) TestSetOnStepChangeFiresOnSelectSchedule() {
	s.advanceToSchedule()

	var received presenter.WizardStep
	s.presenter.SetOnStepChange(func(step presenter.WizardStep) {
		received = step
	})

	s.schedRepo.On("Save", mock.Anything, mock.AnythingOfType("*repository.Schedule")).Return(nil)

	err := s.presenter.SelectSchedule(s.ctx, "focus-maximized")
	s.Require().NoError(err)
	s.Equal(presenter.StepActive, received)
}

func (s *PlannerPresenterSuite) TestSetOnStepChangeFiresOnAbandonPlan() {
	s.advanceToActive()

	var received presenter.WizardStep
	s.presenter.SetOnStepChange(func(step presenter.WizardStep) {
		received = step
	})

	s.schedRepo.On("Delete", mock.Anything, mock.AnythingOfType("time.Time")).Return(nil)

	err := s.presenter.AbandonPlan(s.ctx)
	s.Require().NoError(err)
	s.Equal(presenter.StepIdle, received)
}

func (s *PlannerPresenterSuite) TestSetOnStepChangeFiresOnLoadExistingPlan() {
	scheduleID := uuid.New()
	schedule := &repository.Schedule{
		ID:       scheduleID,
		Date:     time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC),
		Strategy: "focus-maximized",
		Blocks: []repository.ScheduleBlock{
			{
				Start:    s.clock.now,
				End:      s.clock.now.Add(25 * time.Minute),
				Type:     repository.ScheduleBlockFocus,
				TaskName: "Task A",
			},
		},
	}
	s.schedRepo.On("LoadByDate", mock.Anything, mock.Anything).Return(schedule, nil)

	var received presenter.WizardStep
	s.presenter.SetOnStepChange(func(step presenter.WizardStep) {
		received = step
	})

	err := s.presenter.LoadExistingPlan(s.ctx)
	s.Require().NoError(err)
	s.Equal(presenter.StepActive, received)
}

// --- Helpers (continued) ---

// advanceToTaskSelect sets up a presenter at StepTaskSelect with one task.
func (s *PlannerPresenterSuite) advanceToTaskSelect() {
	todo := &repository.Todo{ID: uuid.New(), Title: "Task A", Priority: 1}
	s.todos.On("QueryFiltered", mock.Anything, mock.Anything).Return([]*repository.Todo{todo}, 1, nil)

	err := s.presenter.StartPlanning(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(presenter.StepTaskSelect, s.presenter.CurrentStep())
}

// advanceToPriorityWithCalendarFailure tests that calendar failures are handled gracefully.
func (s *PlannerPresenterSuite) advanceToPriorityWithCalendarFailure() {
	s.advanceToPriority()

	focusSchedule := &planner.DaySchedule{
		ID:       uuid.New(),
		Strategy: "focus-maximized",
		Blocks: []planner.TimeBlock{
			{Start: s.clock.now, End: s.clock.now.Add(25 * time.Minute), Type: planner.BlockFocus, TaskName: "Task A"},
		},
	}
	recoverySchedule := &planner.DaySchedule{
		ID:       uuid.New(),
		Strategy: "recovery-balanced",
		Blocks: []planner.TimeBlock{
			{Start: s.clock.now, End: s.clock.now.Add(25 * time.Minute), Type: planner.BlockFocus, TaskName: "Task A"},
		},
	}
	// Calendar fails but schedule generation proceeds with empty events
	s.cal.On("FetchEvents", mock.Anything, mock.Anything).Return(nil, errors.New("calendar unavailable"))
	s.generator.On("GenerateSchedules", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(focusSchedule, recoverySchedule, nil)

	err := s.presenter.NextStep(s.ctx)
	s.Require().NoError(err)
}

// --- SelectedCount ---

func (s *PlannerPresenterSuite) TestSelectedCountReturnsCountOfSelectedTasks() {
	id1 := uuid.New()
	id2 := uuid.New()
	id3 := uuid.New()
	s.todos.On("QueryFiltered", mock.Anything, mock.Anything).Return([]*repository.Todo{
		{ID: id1, Title: "Task A", Priority: 1},
		{ID: id2, Title: "Task B", Priority: 2},
		{ID: id3, Title: "Task C", Priority: 3},
	}, 3, nil)

	err := s.presenter.StartPlanning(s.ctx)
	s.Require().NoError(err)

	// No tasks selected initially
	s.Equal(0, s.presenter.SelectedCount())

	// Select two of three tasks
	s.presenter.SelectTask(id1, true)
	s.presenter.SelectTask(id3, true)

	s.Equal(2, s.presenter.SelectedCount())
}
