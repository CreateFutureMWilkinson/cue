package presenter_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/service/planner"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
)

// --- Mock Clock ---

type mockTimerClock struct {
	now time.Time
}

func newMockTimerClock(t time.Time) *mockTimerClock {
	return &mockTimerClock{now: t}
}

func (c *mockTimerClock) Now() time.Time {
	return c.now
}

func (c *mockTimerClock) Advance(d time.Duration) {
	c.now = c.now.Add(d)
}

// --- Mock TimerAlerter ---

type mockTimerAlerter struct {
	calls []planner.BlockType
}

func (a *mockTimerAlerter) PlayBlockComplete(blockType planner.BlockType) {
	a.calls = append(a.calls, blockType)
}

// --- Test Suite ---

type TimerPresenterSuite struct {
	suite.Suite
	clock   *mockTimerClock
	alerter *mockTimerAlerter
}

func TestTimerPresenter(t *testing.T) {
	suite.Run(t, new(TimerPresenterSuite))
}

func (s *TimerPresenterSuite) SetupTest() {
	s.clock = newMockTimerClock(time.Date(2026, 3, 29, 9, 0, 0, 0, time.UTC))
	s.alerter = &mockTimerAlerter{}
}

func (s *TimerPresenterSuite) makeBlock(blockType planner.BlockType, duration time.Duration, taskName string) planner.TimeBlock {
	start := s.clock.Now()
	return planner.TimeBlock{
		Start:    start,
		End:      start.Add(duration),
		Type:     blockType,
		TaskName: taskName,
	}
}

// === Constructor Validation ===

func (s *TimerPresenterSuite) TestNewTimerPresenter_NilClock_ReturnsError() {
	p, err := presenter.NewTimerPresenter(nil, s.alerter)
	s.Nil(p)
	s.Error(err)
	s.Contains(err.Error(), "clock")
}

func (s *TimerPresenterSuite) TestNewTimerPresenter_NilAlerter_ReturnsError() {
	p, err := presenter.NewTimerPresenter(s.clock, nil)
	s.Nil(p)
	s.Error(err)
	s.Contains(err.Error(), "alerter")
}

func (s *TimerPresenterSuite) TestNewTimerPresenter_ValidDeps_ReturnsPresenter() {
	p, err := presenter.NewTimerPresenter(s.clock, s.alerter)
	s.NoError(err)
	s.NotNil(p)
}

// === IsRunning ===

func (s *TimerPresenterSuite) TestIsRunning_FalseInitially() {
	p, err := presenter.NewTimerPresenter(s.clock, s.alerter)
	s.Require().NoError(err)

	s.False(p.IsRunning())
}

func (s *TimerPresenterSuite) TestStart_SetsRunningTrue() {
	p, err := presenter.NewTimerPresenter(s.clock, s.alerter)
	s.Require().NoError(err)

	block := s.makeBlock(planner.BlockFocus, 25*time.Minute, "Write tests")
	p.Start(block)

	s.True(p.IsRunning())
}

func (s *TimerPresenterSuite) TestStop_SetsRunningFalse() {
	p, err := presenter.NewTimerPresenter(s.clock, s.alerter)
	s.Require().NoError(err)

	block := s.makeBlock(planner.BlockFocus, 25*time.Minute, "Write tests")
	p.Start(block)
	p.Stop()

	s.False(p.IsRunning())
}

// === ActiveSegment ===

func (s *TimerPresenterSuite) TestActiveSegment_InitialIsZero() {
	p, err := presenter.NewTimerPresenter(s.clock, s.alerter)
	s.Require().NoError(err)

	block := s.makeBlock(planner.BlockFocus, 25*time.Minute, "Write tests")
	p.Start(block)

	s.Equal(0, p.ActiveSegment())
}

func (s *TimerPresenterSuite) TestActiveSegment_AdvancesWithTime() {
	p, err := presenter.NewTimerPresenter(s.clock, s.alerter)
	s.Require().NoError(err)

	block := s.makeBlock(planner.BlockFocus, 25*time.Minute, "Write tests")
	p.Start(block)

	// Each segment = 25min / 45 = ~33.33 seconds
	// Advance past 1 segment
	s.clock.Advance(34 * time.Second)
	p.Tick()

	s.Equal(1, p.ActiveSegment())
}

func (s *TimerPresenterSuite) TestActiveSegment_MidwayThrough() {
	p, err := presenter.NewTimerPresenter(s.clock, s.alerter)
	s.Require().NoError(err)

	duration := 25 * time.Minute
	block := s.makeBlock(planner.BlockFocus, duration, "Write tests")
	p.Start(block)

	// Advance to halfway (12.5 minutes) -> segment ~22
	s.clock.Advance(duration / 2)
	p.Tick()

	s.Equal(22, p.ActiveSegment())
}

func (s *TimerPresenterSuite) TestActiveSegment_NearEnd() {
	p, err := presenter.NewTimerPresenter(s.clock, s.alerter)
	s.Require().NoError(err)

	duration := 25 * time.Minute
	block := s.makeBlock(planner.BlockFocus, duration, "Write tests")
	p.Start(block)

	// Advance to just before the end (segment 44 = last segment)
	// Segment 44 starts at 44/45 of duration
	segmentDuration := duration / 45
	s.clock.Advance(segmentDuration * 44)
	p.Tick()

	s.Equal(44, p.ActiveSegment())
}

// === ElapsedFraction ===

func (s *TimerPresenterSuite) TestElapsedFraction_InitialIsZero() {
	p, err := presenter.NewTimerPresenter(s.clock, s.alerter)
	s.Require().NoError(err)

	block := s.makeBlock(planner.BlockFocus, 25*time.Minute, "Write tests")
	p.Start(block)

	s.InDelta(0.0, p.ElapsedFraction(), 0.001)
}

func (s *TimerPresenterSuite) TestElapsedFraction_MidwayIsHalf() {
	p, err := presenter.NewTimerPresenter(s.clock, s.alerter)
	s.Require().NoError(err)

	duration := 25 * time.Minute
	block := s.makeBlock(planner.BlockFocus, duration, "Write tests")
	p.Start(block)

	s.clock.Advance(duration / 2)
	p.Tick()

	s.InDelta(0.5, p.ElapsedFraction(), 0.01)
}

func (s *TimerPresenterSuite) TestElapsedFraction_AtEndIsOne() {
	p, err := presenter.NewTimerPresenter(s.clock, s.alerter)
	s.Require().NoError(err)

	duration := 25 * time.Minute
	block := s.makeBlock(planner.BlockFocus, duration, "Write tests")
	p.Start(block)

	s.clock.Advance(duration)
	p.Tick()

	s.InDelta(1.0, p.ElapsedFraction(), 0.001)
}

// === IsFlashVisible ===

func (s *TimerPresenterSuite) TestIsFlashVisible_VisibleAtStart() {
	p, err := presenter.NewTimerPresenter(s.clock, s.alerter)
	s.Require().NoError(err)

	block := s.makeBlock(planner.BlockFocus, 25*time.Minute, "Write tests")
	p.Start(block)

	// At t=0, flash should be visible (start of 1Hz cycle)
	s.True(p.IsFlashVisible())
}

func (s *TimerPresenterSuite) TestIsFlashVisible_VisibleAt250ms() {
	p, err := presenter.NewTimerPresenter(s.clock, s.alerter)
	s.Require().NoError(err)

	block := s.makeBlock(planner.BlockFocus, 25*time.Minute, "Write tests")
	p.Start(block)

	s.clock.Advance(250 * time.Millisecond)
	p.Tick()

	// Still in the first 500ms on-phase
	s.True(p.IsFlashVisible())
}

func (s *TimerPresenterSuite) TestIsFlashVisible_InvisibleAt500ms() {
	p, err := presenter.NewTimerPresenter(s.clock, s.alerter)
	s.Require().NoError(err)

	block := s.makeBlock(planner.BlockFocus, 25*time.Minute, "Write tests")
	p.Start(block)

	s.clock.Advance(500 * time.Millisecond)
	p.Tick()

	// Now in the 500ms off-phase
	s.False(p.IsFlashVisible())
}

func (s *TimerPresenterSuite) TestIsFlashVisible_InvisibleAt750ms() {
	p, err := presenter.NewTimerPresenter(s.clock, s.alerter)
	s.Require().NoError(err)

	block := s.makeBlock(planner.BlockFocus, 25*time.Minute, "Write tests")
	p.Start(block)

	s.clock.Advance(750 * time.Millisecond)
	p.Tick()

	// Still in the off-phase
	s.False(p.IsFlashVisible())
}

func (s *TimerPresenterSuite) TestIsFlashVisible_VisibleAt1000ms() {
	p, err := presenter.NewTimerPresenter(s.clock, s.alerter)
	s.Require().NoError(err)

	block := s.makeBlock(planner.BlockFocus, 25*time.Minute, "Write tests")
	p.Start(block)

	s.clock.Advance(1000 * time.Millisecond)
	p.Tick()

	// New cycle starts, back to on-phase
	s.True(p.IsFlashVisible())
}

// === CurrentTaskName ===

func (s *TimerPresenterSuite) TestCurrentTaskName_ReturnsBlockTaskName() {
	p, err := presenter.NewTimerPresenter(s.clock, s.alerter)
	s.Require().NoError(err)

	block := s.makeBlock(planner.BlockFocus, 25*time.Minute, "Implement feature")
	p.Start(block)

	s.Equal("Implement feature", p.CurrentTaskName())
}

func (s *TimerPresenterSuite) TestCurrentTaskName_EmptyWhenNotRunning() {
	p, err := presenter.NewTimerPresenter(s.clock, s.alerter)
	s.Require().NoError(err)

	s.Equal("", p.CurrentTaskName())
}

// === BlockType ===

func (s *TimerPresenterSuite) TestBlockType_ReturnsFocusType() {
	p, err := presenter.NewTimerPresenter(s.clock, s.alerter)
	s.Require().NoError(err)

	block := s.makeBlock(planner.BlockFocus, 25*time.Minute, "Work")
	p.Start(block)

	s.Equal(planner.BlockFocus, p.BlockType())
}

func (s *TimerPresenterSuite) TestBlockType_ReturnsMeetingType() {
	p, err := presenter.NewTimerPresenter(s.clock, s.alerter)
	s.Require().NoError(err)

	block := s.makeBlock(planner.BlockMeeting, 60*time.Minute, "Standup")
	p.Start(block)

	s.Equal(planner.BlockMeeting, p.BlockType())
}

func (s *TimerPresenterSuite) TestBlockType_ReturnsShortBreakType() {
	p, err := presenter.NewTimerPresenter(s.clock, s.alerter)
	s.Require().NoError(err)

	block := s.makeBlock(planner.BlockShortBreak, 5*time.Minute, "")
	p.Start(block)

	s.Equal(planner.BlockShortBreak, p.BlockType())
}

// === SetOnTick ===

func (s *TimerPresenterSuite) TestSetOnTick_CallbackFiresOnTick() {
	p, err := presenter.NewTimerPresenter(s.clock, s.alerter)
	s.Require().NoError(err)

	called := false
	p.SetOnTick(func() { called = true })

	block := s.makeBlock(planner.BlockFocus, 25*time.Minute, "Work")
	p.Start(block)

	s.clock.Advance(1 * time.Second)
	p.Tick()

	s.True(called)
}

func (s *TimerPresenterSuite) TestSetOnTick_NilCallbackDoesNotPanic() {
	p, err := presenter.NewTimerPresenter(s.clock, s.alerter)
	s.Require().NoError(err)

	p.SetOnTick(nil)

	block := s.makeBlock(planner.BlockFocus, 25*time.Minute, "Work")
	p.Start(block)

	s.clock.Advance(1 * time.Second)
	s.NotPanics(func() { p.Tick() })
}

// === SetOnBlockComplete ===

func (s *TimerPresenterSuite) TestSetOnBlockComplete_FiresWhenBlockEnds() {
	p, err := presenter.NewTimerPresenter(s.clock, s.alerter)
	s.Require().NoError(err)

	completeCalled := false
	p.SetOnBlockComplete(func() { completeCalled = true })

	duration := 25 * time.Minute
	block := s.makeBlock(planner.BlockFocus, duration, "Work")
	p.Start(block)

	// Advance past the block end
	s.clock.Advance(duration + time.Second)
	p.Tick()

	s.True(completeCalled)
}

func (s *TimerPresenterSuite) TestSetOnBlockComplete_DoesNotFireBeforeEnd() {
	p, err := presenter.NewTimerPresenter(s.clock, s.alerter)
	s.Require().NoError(err)

	completeCalled := false
	p.SetOnBlockComplete(func() { completeCalled = true })

	duration := 25 * time.Minute
	block := s.makeBlock(planner.BlockFocus, duration, "Work")
	p.Start(block)

	// Advance to halfway
	s.clock.Advance(duration / 2)
	p.Tick()

	s.False(completeCalled)
}

// === Reset on New Block ===

func (s *TimerPresenterSuite) TestStart_ResetsSegmentAndFraction() {
	p, err := presenter.NewTimerPresenter(s.clock, s.alerter)
	s.Require().NoError(err)

	duration := 25 * time.Minute
	block1 := s.makeBlock(planner.BlockFocus, duration, "Task 1")
	p.Start(block1)

	// Advance halfway through block 1
	s.clock.Advance(duration / 2)
	p.Tick()
	s.True(p.ActiveSegment() > 0)
	s.True(p.ElapsedFraction() > 0.0)

	// Start a new block - should reset
	block2 := s.makeBlock(planner.BlockShortBreak, 5*time.Minute, "Break")
	p.Start(block2)

	s.Equal(0, p.ActiveSegment())
	s.InDelta(0.0, p.ElapsedFraction(), 0.001)
}

// === Meeting Block No Alert ===

func (s *TimerPresenterSuite) TestBlockComplete_MeetingDoesNotAlert() {
	p, err := presenter.NewTimerPresenter(s.clock, s.alerter)
	s.Require().NoError(err)

	duration := 60 * time.Minute
	block := s.makeBlock(planner.BlockMeeting, duration, "Team standup")
	p.Start(block)

	// Advance past the block end
	s.clock.Advance(duration + time.Second)
	p.Tick()

	s.Empty(s.alerter.calls)
}

// === Non-Meeting Block Fires Alert ===

func (s *TimerPresenterSuite) TestBlockComplete_FocusBlockFiresAlert() {
	p, err := presenter.NewTimerPresenter(s.clock, s.alerter)
	s.Require().NoError(err)

	duration := 25 * time.Minute
	block := s.makeBlock(planner.BlockFocus, duration, "Deep work")
	p.Start(block)

	// Advance past the block end
	s.clock.Advance(duration + time.Second)
	p.Tick()

	s.Require().Len(s.alerter.calls, 1)
	s.Equal(planner.BlockFocus, s.alerter.calls[0])
}

func (s *TimerPresenterSuite) TestBlockComplete_ShortBreakFiresAlert() {
	p, err := presenter.NewTimerPresenter(s.clock, s.alerter)
	s.Require().NoError(err)

	duration := 5 * time.Minute
	block := s.makeBlock(planner.BlockShortBreak, duration, "")
	p.Start(block)

	s.clock.Advance(duration + time.Second)
	p.Tick()

	s.Require().Len(s.alerter.calls, 1)
	s.Equal(planner.BlockShortBreak, s.alerter.calls[0])
}

func (s *TimerPresenterSuite) TestBlockComplete_LongBreakFiresAlert() {
	p, err := presenter.NewTimerPresenter(s.clock, s.alerter)
	s.Require().NoError(err)

	duration := 20 * time.Minute
	block := s.makeBlock(planner.BlockLongBreak, duration, "")
	p.Start(block)

	s.clock.Advance(duration + time.Second)
	p.Tick()

	s.Require().Len(s.alerter.calls, 1)
	s.Equal(planner.BlockLongBreak, s.alerter.calls[0])
}

// === Stopped Timer Does Not Progress ===

func (s *TimerPresenterSuite) TestTick_WhenStopped_DoesNotFireCallbacks() {
	p, err := presenter.NewTimerPresenter(s.clock, s.alerter)
	s.Require().NoError(err)

	tickCalled := false
	completeCalled := false
	p.SetOnTick(func() { tickCalled = true })
	p.SetOnBlockComplete(func() { completeCalled = true })

	duration := 25 * time.Minute
	block := s.makeBlock(planner.BlockFocus, duration, "Work")
	p.Start(block)
	p.Stop()

	s.clock.Advance(duration + time.Second)
	p.Tick()

	s.False(tickCalled)
	s.False(completeCalled)
}

// === Alert Only Fires Once ===

func (s *TimerPresenterSuite) TestBlockComplete_AlertFiresOnlyOnce() {
	p, err := presenter.NewTimerPresenter(s.clock, s.alerter)
	s.Require().NoError(err)

	p.SetOnBlockComplete(func() {})

	duration := 25 * time.Minute
	block := s.makeBlock(planner.BlockFocus, duration, "Work")
	p.Start(block)

	// Advance past end and tick multiple times
	s.clock.Advance(duration + time.Second)
	p.Tick()
	s.clock.Advance(time.Second)
	p.Tick()
	s.clock.Advance(time.Second)
	p.Tick()

	s.Len(s.alerter.calls, 1)
}
