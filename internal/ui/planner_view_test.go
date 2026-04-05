package ui_test

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/service/planner"
	"github.com/CreateFutureMWilkinson/cue/internal/ui"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
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
}

func TestPlannerView(t *testing.T) {
	suite.Run(t, new(PlannerViewSuite))
}

func (s *PlannerViewSuite) SetupTest() {
	s.plannerVM = new(mockPlannerViewModel)
	s.timerVM = new(mockTimerViewModel)
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

// --- Tests ---

func (s *PlannerViewSuite) TestNewPlannerViewReturnsNonNil() {
	s.setupIdleDefaults()

	view := ui.NewPlannerView(s.plannerVM, s.timerVM)

	s.NotNil(view, "NewPlannerView should return a non-nil component")
}

func (s *PlannerViewSuite) TestContainerIsNonNil() {
	s.setupIdleDefaults()

	view := ui.NewPlannerView(s.plannerVM, s.timerVM)

	s.NotNil(view.Container(), "Container should return a non-nil fyne.Container")
}

func (s *PlannerViewSuite) TestPlanButtonVisibleAtIdle() {
	s.setupIdleDefaults()

	view := ui.NewPlannerView(s.plannerVM, s.timerVM)

	s.True(view.PlanButton().Visible(),
		"Plan My Day button should be visible when step is Idle")
}

func (s *PlannerViewSuite) TestPlanButtonHiddenAtTaskSelect() {
	s.setupStepDefaults(presenter.StepTaskSelect)

	view := ui.NewPlannerView(s.plannerVM, s.timerVM)

	s.False(view.PlanButton().Visible(),
		"Plan My Day button should be hidden when step is TaskSelect")
}

func (s *PlannerViewSuite) TestPlanButtonHiddenAtActive() {
	s.setupStepDefaults(presenter.StepActive)

	view := ui.NewPlannerView(s.plannerVM, s.timerVM)

	s.False(view.PlanButton().Visible(),
		"Plan My Day button should be hidden when step is Active")
}

func (s *PlannerViewSuite) TestNextButtonVisibleAtTaskSelect() {
	s.setupStepDefaults(presenter.StepTaskSelect)

	view := ui.NewPlannerView(s.plannerVM, s.timerVM)

	s.True(view.NextButton().Visible(),
		"Next button should be visible during TaskSelect step")
}

func (s *PlannerViewSuite) TestNextButtonVisibleAtEstimates() {
	s.setupStepDefaults(presenter.StepEstimates)

	view := ui.NewPlannerView(s.plannerVM, s.timerVM)

	s.True(view.NextButton().Visible(),
		"Next button should be visible during Estimates step")
}

func (s *PlannerViewSuite) TestNextButtonVisibleAtPriority() {
	s.setupStepDefaults(presenter.StepPriority)

	view := ui.NewPlannerView(s.plannerVM, s.timerVM)

	s.True(view.NextButton().Visible(),
		"Next button should be visible during Priority step")
}

func (s *PlannerViewSuite) TestNextButtonHiddenAtIdle() {
	s.setupIdleDefaults()

	view := ui.NewPlannerView(s.plannerVM, s.timerVM)

	s.False(view.NextButton().Visible(),
		"Next button should be hidden when step is Idle")
}

func (s *PlannerViewSuite) TestNextButtonHiddenAtActive() {
	s.setupStepDefaults(presenter.StepActive)

	view := ui.NewPlannerView(s.plannerVM, s.timerVM)

	s.False(view.NextButton().Visible(),
		"Next button should be hidden when step is Active")
}

func (s *PlannerViewSuite) TestNextButtonHiddenAtSchedule() {
	s.setupStepDefaults(presenter.StepSchedule)

	view := ui.NewPlannerView(s.plannerVM, s.timerVM)

	s.False(view.NextButton().Visible(),
		"Next button should be hidden when step is Schedule (use schedule selection instead)")
}

func (s *PlannerViewSuite) TestBackButtonVisibleAtTaskSelect() {
	s.setupStepDefaults(presenter.StepTaskSelect)

	view := ui.NewPlannerView(s.plannerVM, s.timerVM)

	s.True(view.BackButton().Visible(),
		"Back button should be visible during TaskSelect step")
}

func (s *PlannerViewSuite) TestBackButtonVisibleAtEstimates() {
	s.setupStepDefaults(presenter.StepEstimates)

	view := ui.NewPlannerView(s.plannerVM, s.timerVM)

	s.True(view.BackButton().Visible(),
		"Back button should be visible during Estimates step")
}

func (s *PlannerViewSuite) TestBackButtonVisibleAtPriority() {
	s.setupStepDefaults(presenter.StepPriority)

	view := ui.NewPlannerView(s.plannerVM, s.timerVM)

	s.True(view.BackButton().Visible(),
		"Back button should be visible during Priority step")
}

func (s *PlannerViewSuite) TestBackButtonVisibleAtSchedule() {
	s.setupStepDefaults(presenter.StepSchedule)

	view := ui.NewPlannerView(s.plannerVM, s.timerVM)

	s.True(view.BackButton().Visible(),
		"Back button should be visible during Schedule step")
}

func (s *PlannerViewSuite) TestBackButtonHiddenAtIdle() {
	s.setupIdleDefaults()

	view := ui.NewPlannerView(s.plannerVM, s.timerVM)

	s.False(view.BackButton().Visible(),
		"Back button should be hidden when step is Idle")
}

func (s *PlannerViewSuite) TestBackButtonHiddenAtActive() {
	s.setupStepDefaults(presenter.StepActive)

	view := ui.NewPlannerView(s.plannerVM, s.timerVM)

	s.False(view.BackButton().Visible(),
		"Back button should be hidden when step is Active")
}

func (s *PlannerViewSuite) TestCompleteTaskButtonVisibleAtActive() {
	s.setupStepDefaults(presenter.StepActive)

	view := ui.NewPlannerView(s.plannerVM, s.timerVM)

	s.True(view.CompleteTaskButton().Visible(),
		"Complete Task button should be visible during Active step")
}

func (s *PlannerViewSuite) TestCompleteTaskButtonHiddenAtIdle() {
	s.setupIdleDefaults()

	view := ui.NewPlannerView(s.plannerVM, s.timerVM)

	s.False(view.CompleteTaskButton().Visible(),
		"Complete Task button should be hidden when step is Idle")
}

func (s *PlannerViewSuite) TestCompleteTaskButtonHiddenAtTaskSelect() {
	s.setupStepDefaults(presenter.StepTaskSelect)

	view := ui.NewPlannerView(s.plannerVM, s.timerVM)

	s.False(view.CompleteTaskButton().Visible(),
		"Complete Task button should be hidden during wizard steps")
}

func (s *PlannerViewSuite) TestAbandonButtonVisibleAtActive() {
	s.setupStepDefaults(presenter.StepActive)

	view := ui.NewPlannerView(s.plannerVM, s.timerVM)

	s.True(view.AbandonButton().Visible(),
		"Abandon Plan button should be visible during Active step")
}

func (s *PlannerViewSuite) TestAbandonButtonHiddenAtIdle() {
	s.setupIdleDefaults()

	view := ui.NewPlannerView(s.plannerVM, s.timerVM)

	s.False(view.AbandonButton().Visible(),
		"Abandon Plan button should be hidden when step is Idle")
}

func (s *PlannerViewSuite) TestAbandonButtonHiddenAtEstimates() {
	s.setupStepDefaults(presenter.StepEstimates)

	view := ui.NewPlannerView(s.plannerVM, s.timerVM)

	s.False(view.AbandonButton().Visible(),
		"Abandon Plan button should be hidden during wizard steps")
}

func (s *PlannerViewSuite) TestRefreshUpdatesButtonVisibility() {
	s.setupIdleDefaults()

	view := ui.NewPlannerView(s.plannerVM, s.timerVM)

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
