package ui_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/ui"
)

// --- Mocks ---

type mockTickableTimer struct {
	mock.Mock
}

func (m *mockTickableTimer) Tick()                    { m.Called() }
func (m *mockTickableTimer) ElapsedFraction() float64 { return m.Called().Get(0).(float64) }
func (m *mockTickableTimer) IsFlashVisible() bool     { return m.Called().Bool(0) }
func (m *mockTickableTimer) CurrentTaskName() string  { return m.Called().String(0) }

type mockTimerWidget struct {
	mock.Mock
}

func (m *mockTimerWidget) SetProgress(p float64)  { m.Called(p) }
func (m *mockTimerWidget) SetFlashVisible(v bool) { m.Called(v) }

type mockTaskUpdater struct {
	mock.Mock
}

func (m *mockTaskUpdater) SetCurrentTask(task string) { m.Called(task) }

// --- Suite ---

type TimerLoopSuite struct {
	suite.Suite
	timer    *mockTickableTimer
	widget   *mockTimerWidget
	taskView *mockTaskUpdater
}

func TestTimerLoop(t *testing.T) {
	suite.Run(t, new(TimerLoopSuite))
}

func (s *TimerLoopSuite) SetupTest() {
	s.timer = new(mockTickableTimer)
	s.widget = new(mockTimerWidget)
	s.taskView = new(mockTaskUpdater)
}

// --- Constructor ---

func (s *TimerLoopSuite) TestNewTimerLoopReturnsNonNil() {
	loop, err := ui.NewTimerLoop(s.timer, s.widget, s.taskView)
	s.NoError(err)
	s.NotNil(loop)
}

func (s *TimerLoopSuite) TestTimerLoopNilTimerReturnsError() {
	loop, err := ui.NewTimerLoop(nil, s.widget, s.taskView)
	s.Error(err)
	s.Nil(loop)
}

func (s *TimerLoopSuite) TestTimerLoopNilWidgetReturnsError() {
	loop, err := ui.NewTimerLoop(s.timer, nil, s.taskView)
	s.Error(err)
	s.Nil(loop)
}

func (s *TimerLoopSuite) TestTimerLoopNilTaskViewReturnsError() {
	loop, err := ui.NewTimerLoop(s.timer, s.widget, nil)
	s.Error(err)
	s.Nil(loop)
}

// --- TickOnce behavior ---

func (s *TimerLoopSuite) TestTimerLoopTickCallsTimerTick() {
	s.timer.On("Tick").Return()
	s.timer.On("ElapsedFraction").Return(0.0)
	s.timer.On("IsFlashVisible").Return(false)
	s.timer.On("CurrentTaskName").Return("")
	s.widget.On("SetProgress", 0.0).Return()
	s.widget.On("SetFlashVisible", false).Return()
	s.taskView.On("SetCurrentTask", "").Return()

	loop, err := ui.NewTimerLoop(s.timer, s.widget, s.taskView)
	s.Require().NoError(err)

	loop.TickOnce()

	s.timer.AssertCalled(s.T(), "Tick")
}

func (s *TimerLoopSuite) TestTimerLoopUpdatesProgressAfterTick() {
	s.timer.On("Tick").Return()
	s.timer.On("ElapsedFraction").Return(0.5)
	s.timer.On("IsFlashVisible").Return(false)
	s.timer.On("CurrentTaskName").Return("")
	s.widget.On("SetProgress", 0.5).Return()
	s.widget.On("SetFlashVisible", false).Return()
	s.taskView.On("SetCurrentTask", "").Return()

	loop, err := ui.NewTimerLoop(s.timer, s.widget, s.taskView)
	s.Require().NoError(err)

	loop.TickOnce()

	s.widget.AssertCalled(s.T(), "SetProgress", 0.5)
}

func (s *TimerLoopSuite) TestTimerLoopUpdatesFlashAfterTick() {
	s.timer.On("Tick").Return()
	s.timer.On("ElapsedFraction").Return(0.0)
	s.timer.On("IsFlashVisible").Return(true)
	s.timer.On("CurrentTaskName").Return("")
	s.widget.On("SetProgress", 0.0).Return()
	s.widget.On("SetFlashVisible", true).Return()
	s.taskView.On("SetCurrentTask", "").Return()

	loop, err := ui.NewTimerLoop(s.timer, s.widget, s.taskView)
	s.Require().NoError(err)

	loop.TickOnce()

	s.widget.AssertCalled(s.T(), "SetFlashVisible", true)
}

func (s *TimerLoopSuite) TestTimerLoopUpdatesTaskLabelAfterTick() {
	s.timer.On("Tick").Return()
	s.timer.On("ElapsedFraction").Return(0.0)
	s.timer.On("IsFlashVisible").Return(false)
	s.timer.On("CurrentTaskName").Return("Write code")
	s.widget.On("SetProgress", 0.0).Return()
	s.widget.On("SetFlashVisible", false).Return()
	s.taskView.On("SetCurrentTask", "Write code").Return()

	loop, err := ui.NewTimerLoop(s.timer, s.widget, s.taskView)
	s.Require().NoError(err)

	loop.TickOnce()

	s.taskView.AssertCalled(s.T(), "SetCurrentTask", "Write code")
}

func (s *TimerLoopSuite) TestTimerLoopTickOnceUsesUIScheduler() {
	s.timer.On("Tick").Return()
	s.timer.On("ElapsedFraction").Return(0.5)
	s.timer.On("IsFlashVisible").Return(true)
	s.timer.On("CurrentTaskName").Return("Write code")
	s.widget.On("SetProgress", 0.5).Return()
	s.widget.On("SetFlashVisible", true).Return()
	s.taskView.On("SetCurrentTask", "Write code").Return()

	loop, err := ui.NewTimerLoop(s.timer, s.widget, s.taskView)
	s.Require().NoError(err)

	schedulerCalled := false
	loop.SetUIScheduler(func(fn func()) {
		schedulerCalled = true
		fn()
	})

	loop.TickOnce()

	s.True(schedulerCalled, "TickOnce should dispatch widget updates through UIScheduler")
}

func (s *TimerLoopSuite) TestTimerLoopStopPreventsMoreTicks() {
	s.timer.On("Tick").Return()
	s.timer.On("ElapsedFraction").Return(0.0)
	s.timer.On("IsFlashVisible").Return(false)
	s.timer.On("CurrentTaskName").Return("")
	s.widget.On("SetProgress", 0.0).Return()
	s.widget.On("SetFlashVisible", false).Return()
	s.taskView.On("SetCurrentTask", "").Return()

	loop, err := ui.NewTimerLoop(s.timer, s.widget, s.taskView)
	s.Require().NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	loop.Start(ctx)
	loop.Stop()
	cancel()

	// After stop, TickOnce should be a no-op (timer.Tick not called again)
	// Reset mock call counts by creating fresh expectations
	tickCountBefore := len(s.timer.Calls)

	loop.TickOnce()

	// Tick should not have been called after Stop
	s.Equal(tickCountBefore, len(s.timer.Calls),
		"TickOnce should not call Tick after Stop")
}
