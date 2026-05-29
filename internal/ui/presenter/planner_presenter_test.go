package presenter_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/service/planner"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
)

// --- Mocks ---

type mockTodoQuerier struct {
	mock.Mock
}

func (m *mockTodoQuerier) QueryFiltered(ctx context.Context, filter repository.TaskFilter) ([]*repository.Task, int, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*repository.Task), args.Int(1), args.Error(2)
}

func (m *mockTodoQuerier) Insert(ctx context.Context, todo *repository.Task) error {
	args := m.Called(ctx, todo)
	return args.Error(0)
}

func (m *mockTodoQuerier) Update(ctx context.Context, todo *repository.Task) error {
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

type mockScheduleGenerator struct {
	mock.Mock
}

func (m *mockScheduleGenerator) GenerateSchedules(
	ctx context.Context,
	targetDate time.Time,
) (*planner.DaySchedule, *planner.DaySchedule, error) {
	args := m.Called(ctx, targetDate)
	var focus, recovery *planner.DaySchedule
	if args.Get(0) != nil {
		focus = args.Get(0).(*planner.DaySchedule)
	}
	if args.Get(1) != nil {
		recovery = args.Get(1).(*planner.DaySchedule)
	}
	return focus, recovery, args.Error(2)
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
	generator  *mockScheduleGenerator
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
	s.generator = new(mockScheduleGenerator)
	s.schedRepo = new(mockScheduleRepo)
	s.clock = &mockClock{now: time.Date(2026, 3, 30, 9, 0, 0, 0, time.UTC)} // Monday 9am
	s.ctx = context.Background()

	p, err := presenter.NewPlannerPresenter(
		s.todos,
		s.categories,
		s.generator,
		s.schedRepo,
		s.clock,
	)
	s.Require().NoError(err)
	s.presenter = p
}

// --- Constructor validation ---

func (s *PlannerPresenterSuite) TestNewPlannerPresenterNilTodosReturnsError() {
	_, err := presenter.NewPlannerPresenter(nil, s.categories, s.generator, s.schedRepo, s.clock)
	s.Error(err)
}

func (s *PlannerPresenterSuite) TestNewPlannerPresenterNilCategoriesReturnsError() {
	_, err := presenter.NewPlannerPresenter(s.todos, nil, s.generator, s.schedRepo, s.clock)
	s.Error(err)
}

func (s *PlannerPresenterSuite) TestNewPlannerPresenterNilGeneratorReturnsError() {
	_, err := presenter.NewPlannerPresenter(s.todos, s.categories, nil, s.schedRepo, s.clock)
	s.Error(err)
}

func (s *PlannerPresenterSuite) TestNewPlannerPresenterNilScheduleRepoReturnsError() {
	_, err := presenter.NewPlannerPresenter(s.todos, s.categories, s.generator, nil, s.clock)
	s.Error(err)
}

func (s *PlannerPresenterSuite) TestNewPlannerPresenterNilClockReturnsError() {
	_, err := presenter.NewPlannerPresenter(s.todos, s.categories, s.generator, s.schedRepo, nil)
	s.Error(err)
}

// --- CurrentStep ---

func (s *PlannerPresenterSuite) TestCurrentStepReturnsIdleInitially() {
	s.Equal(presenter.StepIdle, s.presenter.CurrentStep())
}

// --- StartPlanning (Feature 107 WP5: directly to StepSchedule) ---

func (s *PlannerPresenterSuite) TestStartPlanningTransitionsToSchedule() {
	s.expectGenerator()

	err := s.presenter.StartPlanning(s.ctx)
	s.Require().NoError(err)
	s.Equal(presenter.StepSchedule, s.presenter.CurrentStep())
}

func (s *PlannerPresenterSuite) TestStartPlanningPopulatesPreviews() {
	s.expectGenerator()

	err := s.presenter.StartPlanning(s.ctx)
	s.Require().NoError(err)

	focus := s.presenter.FocusSchedule()
	s.Require().NotNil(focus)
	s.Equal("focus-maximized", focus.Strategy)

	recovery := s.presenter.RecoverySchedule()
	s.Require().NotNil(recovery)
	s.Equal("recovery-balanced", recovery.Strategy)
}

func (s *PlannerPresenterSuite) TestStartPlanningGeneratorErrorBubbles() {
	s.generator.On("GenerateSchedules", mock.Anything, mock.Anything).
		Return((*planner.DaySchedule)(nil), (*planner.DaySchedule)(nil), assertError())

	err := s.presenter.StartPlanning(s.ctx)
	s.Error(err)
	s.Equal(presenter.StepIdle, s.presenter.CurrentStep())
}

// --- NextStep ---

func (s *PlannerPresenterSuite) TestNextStepFromIdleIsNoop() {
	err := s.presenter.NextStep(s.ctx)
	s.NoError(err)
	s.Equal(presenter.StepIdle, s.presenter.CurrentStep())
}

func (s *PlannerPresenterSuite) TestNextStepFromScheduleReturnsError() {
	s.advanceToSchedule()
	err := s.presenter.NextStep(s.ctx)
	s.Error(err, "callers must use SelectSchedule, not NextStep, to leave StepSchedule")
}

// --- PreviousStep ---

func (s *PlannerPresenterSuite) TestPreviousStepFromScheduleReturnsToIdle() {
	s.advanceToSchedule()

	s.presenter.PreviousStep()
	s.Equal(presenter.StepIdle, s.presenter.CurrentStep())
}

func (s *PlannerPresenterSuite) TestPreviousStepFromIdleIsNoop() {
	s.presenter.PreviousStep()
	s.Equal(presenter.StepIdle, s.presenter.CurrentStep())
}

// --- SelectSchedule ---

func (s *PlannerPresenterSuite) TestSelectSchedulePersistsAndTransitionsToActive() {
	s.advanceToSchedule()

	s.schedRepo.On("Save", mock.Anything, mock.AnythingOfType("*repository.Schedule")).Return(nil)

	err := s.presenter.SelectSchedule(s.ctx, "focus-maximized")
	s.Require().NoError(err)
	s.Equal(presenter.StepActive, s.presenter.CurrentStep())
	s.schedRepo.AssertCalled(s.T(), "Save", mock.Anything, mock.AnythingOfType("*repository.Schedule"))
}

func (s *PlannerPresenterSuite) TestSelectScheduleRecoveryStrategy() {
	s.advanceToSchedule()

	s.schedRepo.On("Save", mock.Anything, mock.AnythingOfType("*repository.Schedule")).Return(nil)

	err := s.presenter.SelectSchedule(s.ctx, "recovery-balanced")
	s.Require().NoError(err)
	s.Equal(presenter.StepActive, s.presenter.CurrentStep())
}

func (s *PlannerPresenterSuite) TestSelectScheduleUnknownStrategy() {
	s.advanceToSchedule()

	err := s.presenter.SelectSchedule(s.ctx, "bogus")
	s.Error(err)
}

// --- ActiveSchedule ---

func (s *PlannerPresenterSuite) TestActiveScheduleReturnsCurrentState() {
	s.advanceToActive()

	state := s.presenter.ActiveSchedule()
	s.Require().NotNil(state)
	s.GreaterOrEqual(len(state.Blocks), 1)
	s.GreaterOrEqual(state.CurrentIndex, 0)
}

func (s *PlannerPresenterSuite) TestActiveScheduleReturnsNilWhenNotActive() {
	state := s.presenter.ActiveSchedule()
	s.Nil(state)
}

// --- CompleteCurrentTask ---

func (s *PlannerPresenterSuite) TestCompleteCurrentTaskMarksTaskDone() {
	taskID := uuid.New()
	s.advanceToActiveWithTaskID(&taskID)

	s.todos.On("Complete", mock.Anything, taskID, mock.AnythingOfType("time.Time")).Return(nil)

	err := s.presenter.CompleteCurrentTask(s.ctx)
	s.Require().NoError(err)
	s.todos.AssertCalled(s.T(), "Complete", mock.Anything, taskID, mock.AnythingOfType("time.Time"))
}

func (s *PlannerPresenterSuite) TestCompleteCurrentTaskNilTaskIDIsNoop() {
	s.advanceToActiveWithTaskID(nil)

	err := s.presenter.CompleteCurrentTask(s.ctx)
	s.Require().NoError(err)
	// No Complete call expected — block had no TaskID.
}

func (s *PlannerPresenterSuite) TestCompleteCurrentTaskNoActivePlan() {
	err := s.presenter.CompleteCurrentTask(s.ctx)
	s.Error(err)
}

// --- AbandonPlan ---

func (s *PlannerPresenterSuite) TestAbandonPlanDeletesScheduleReturnsToIdle() {
	s.advanceToActive()

	s.schedRepo.On("Delete", mock.Anything, mock.AnythingOfType("time.Time")).Return(nil)

	err := s.presenter.AbandonPlan(s.ctx)
	s.Require().NoError(err)
	s.Equal(presenter.StepIdle, s.presenter.CurrentStep())
}

// --- HasActivePlan ---

func (s *PlannerPresenterSuite) TestHasActivePlanFalseInitially() {
	s.False(s.presenter.HasActivePlan())
}

func (s *PlannerPresenterSuite) TestHasActivePlanTrueAfterSelectSchedule() {
	s.advanceToActive()
	s.True(s.presenter.HasActivePlan())
}

// --- LoadExistingPlan ---

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

func (s *PlannerPresenterSuite) TestLoadExistingPlanNoScheduleStaysIdle() {
	s.schedRepo.On("LoadByDate", mock.Anything, mock.Anything).Return(nil, nil)

	err := s.presenter.LoadExistingPlan(s.ctx)
	s.Require().NoError(err)
	s.Equal(presenter.StepIdle, s.presenter.CurrentStep())
}

// --- SetOnStepChange ---

func (s *PlannerPresenterSuite) TestSetOnStepChangeFiresOnStartPlanning() {
	s.expectGenerator()

	var received presenter.WizardStep
	s.presenter.SetOnStepChange(func(step presenter.WizardStep) {
		received = step
	})

	err := s.presenter.StartPlanning(s.ctx)
	s.Require().NoError(err)
	s.Equal(presenter.StepSchedule, received)
}

func (s *PlannerPresenterSuite) TestSetOnStepChangeFiresOnPreviousStep() {
	s.advanceToSchedule()

	var received presenter.WizardStep
	s.presenter.SetOnStepChange(func(step presenter.WizardStep) {
		received = step
	})

	s.presenter.PreviousStep()
	s.Equal(presenter.StepIdle, received)
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

// --- CurrentFocusTask (Feature 107) ---

func (s *PlannerPresenterSuite) TestCurrentFocusTaskReturnsHighestPriorityIncomplete() {
	older := time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 4, 5, 9, 0, 0, 0, time.UTC)
	tasks := []*repository.Task{
		{ID: uuid.New(), Title: "Mid prio", Priority: 2, CreatedAt: older},
		{ID: uuid.New(), Title: "Top prio", Priority: 1, CreatedAt: newer},
		{ID: uuid.New(), Title: "Low prio", Priority: 3, CreatedAt: older},
	}
	s.todos.On("QueryFiltered", mock.Anything, repository.TaskFilter{Status: "incomplete"}).
		Return(tasks, len(tasks), nil)

	row, err := s.presenter.CurrentFocusTask(s.ctx)
	s.Require().NoError(err)
	s.Require().NotNil(row)
	s.Equal("Top prio", row.Title)
	s.Equal(1, row.Priority)
}

func (s *PlannerPresenterSuite) TestCurrentFocusTaskReturnsNilWhenNoTasks() {
	s.todos.On("QueryFiltered", mock.Anything, repository.TaskFilter{Status: "incomplete"}).
		Return([]*repository.Task{}, 0, nil)

	row, err := s.presenter.CurrentFocusTask(s.ctx)
	s.Require().NoError(err)
	s.Nil(row)
}

func (s *PlannerPresenterSuite) TestCurrentFocusTaskBreaksPriorityTiesByCreatedAt() {
	first := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	second := first.Add(24 * time.Hour)
	tasks := []*repository.Task{
		{ID: uuid.New(), Title: "Newer", Priority: 1, CreatedAt: second},
		{ID: uuid.New(), Title: "Older", Priority: 1, CreatedAt: first},
	}
	s.todos.On("QueryFiltered", mock.Anything, repository.TaskFilter{Status: "incomplete"}).
		Return(tasks, len(tasks), nil)

	row, err := s.presenter.CurrentFocusTask(s.ctx)
	s.Require().NoError(err)
	s.Require().NotNil(row)
	s.Equal("Older", row.Title, "earliest CreatedAt wins on a priority tie")
}

// --- Helpers ---

func (s *PlannerPresenterSuite) expectGenerator() {
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
	s.generator.On("GenerateSchedules", mock.Anything, mock.Anything).
		Return(focusSchedule, recoverySchedule, nil)
}

func (s *PlannerPresenterSuite) advanceToSchedule() {
	s.expectGenerator()
	err := s.presenter.StartPlanning(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(presenter.StepSchedule, s.presenter.CurrentStep())
}

func (s *PlannerPresenterSuite) advanceToActive() {
	s.advanceToSchedule()
	s.schedRepo.On("Save", mock.Anything, mock.AnythingOfType("*repository.Schedule")).Return(nil)
	err := s.presenter.SelectSchedule(s.ctx, "focus-maximized")
	s.Require().NoError(err)
	s.Require().Equal(presenter.StepActive, s.presenter.CurrentStep())
}

// advanceToActiveWithTaskID configures the generator to produce a focus
// block with the given TaskID, then transitions to StepActive. Used by
// CompleteCurrentTask tests where the task ID drives the Complete call.
func (s *PlannerPresenterSuite) advanceToActiveWithTaskID(taskID *uuid.UUID) {
	focusSchedule := &planner.DaySchedule{
		ID:       uuid.New(),
		Strategy: "focus-maximized",
		Blocks: []planner.TimeBlock{
			{
				Start:    s.clock.now,
				End:      s.clock.now.Add(25 * time.Minute),
				Type:     planner.BlockFocus,
				TaskID:   taskID,
				TaskName: "Task A",
			},
		},
	}
	recoverySchedule := &planner.DaySchedule{
		ID:       uuid.New(),
		Strategy: "recovery-balanced",
		Blocks:   []planner.TimeBlock{},
	}
	s.generator.On("GenerateSchedules", mock.Anything, mock.Anything).
		Return(focusSchedule, recoverySchedule, nil)
	err := s.presenter.StartPlanning(s.ctx)
	s.Require().NoError(err)

	s.schedRepo.On("Save", mock.Anything, mock.AnythingOfType("*repository.Schedule")).Return(nil)
	err = s.presenter.SelectSchedule(s.ctx, "focus-maximized")
	s.Require().NoError(err)
}

// assertError returns a sentinel error for negative-path tests.
func assertError() error {
	return &generatorError{}
}

type generatorError struct{}

func (e *generatorError) Error() string { return "generator failure" }
