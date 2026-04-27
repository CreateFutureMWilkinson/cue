package planner_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/service/planner"
)

type TimerSuite struct {
	suite.Suite
}

func TestTimer(t *testing.T) {
	suite.Run(t, new(TimerSuite))
}

func (s *TimerSuite) TestComputeTimerStateWithinBlock() {
	blockStart := time.Date(2026, 4, 22, 9, 0, 0, 0, time.UTC)
	blockEnd := time.Date(2026, 4, 22, 9, 25, 0, 0, time.UTC)
	now := time.Date(2026, 4, 22, 9, 10, 0, 0, time.UTC)

	taskID := uuid.New()
	schedule := &planner.DaySchedule{
		ID:       uuid.New(),
		Date:     time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC),
		Strategy: "focus-maximized",
		Blocks: []planner.TimeBlock{
			{
				Start:    blockStart,
				End:      blockEnd,
				Type:     planner.BlockFocus,
				TaskID:   &taskID,
				TaskName: "Write timer API",
			},
		},
		CreatedAt: time.Now(),
	}

	state := planner.ComputeTimerState(schedule, now)

	s.True(state.Running, "should be running when now is within a block")
	s.Equal("focus", state.BlockType)
	s.Equal("Write timer API", state.TaskName)
	s.Equal(1500, state.DurationSeconds, "25 minutes = 1500 seconds")
	s.Equal(600, state.ElapsedSeconds, "10 minutes = 600 seconds elapsed")
	s.Equal(900, state.RemainingSeconds, "15 minutes = 900 seconds remaining")
	s.Equal("15:00", state.DisplayTime)
	s.InDelta(0.4, state.ElapsedFraction, 0.001, "10/25 = 0.4 elapsed fraction")
	s.Equal(0, state.BlockIndex)
}

func (s *TimerSuite) TestTimerStateBetweenBlocksReturnsNotRunning() {
	taskID := uuid.New()
	schedule := &planner.DaySchedule{
		ID:       uuid.New(),
		Date:     time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC),
		Strategy: "focus-maximized",
		Blocks: []planner.TimeBlock{
			{
				Start:    time.Date(2026, 4, 22, 9, 0, 0, 0, time.UTC),
				End:      time.Date(2026, 4, 22, 9, 25, 0, 0, time.UTC),
				Type:     planner.BlockFocus,
				TaskID:   &taskID,
				TaskName: "Deep work",
			},
			{
				Start:    time.Date(2026, 4, 22, 9, 30, 0, 0, time.UTC),
				End:      time.Date(2026, 4, 22, 9, 35, 0, 0, time.UTC),
				Type:     planner.BlockShortBreak,
				TaskName: "Short break",
			},
		},
		CreatedAt: time.Now(),
	}

	now := time.Date(2026, 4, 22, 9, 27, 0, 0, time.UTC)
	state := planner.ComputeTimerState(schedule, now)

	s.False(state.Running, "should not be running in gap between blocks")
}

func (s *TimerSuite) TestTimerStateBeforeAndAfterScheduleReturnsNotRunning() {
	schedule := &planner.DaySchedule{
		ID:       uuid.New(),
		Date:     time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC),
		Strategy: "focus-maximized",
		Blocks: []planner.TimeBlock{
			{
				Start:    time.Date(2026, 4, 22, 9, 0, 0, 0, time.UTC),
				End:      time.Date(2026, 4, 22, 9, 25, 0, 0, time.UTC),
				Type:     planner.BlockFocus,
				TaskName: "Morning focus",
			},
		},
		CreatedAt: time.Now(),
	}

	beforeSchedule := time.Date(2026, 4, 22, 8, 50, 0, 0, time.UTC)
	state := planner.ComputeTimerState(schedule, beforeSchedule)
	s.False(state.Running, "should not be running before schedule starts")

	afterSchedule := time.Date(2026, 4, 22, 9, 30, 0, 0, time.UTC)
	state = planner.ComputeTimerState(schedule, afterSchedule)
	s.False(state.Running, "should not be running after schedule ends")
}

func (s *TimerSuite) TestFormatDisplayTimeEdgeCases() {
	s.Equal("00:00", planner.FormatDisplayTime(0))
	s.Equal("00:59", planner.FormatDisplayTime(59))
	s.Equal("60:00", planner.FormatDisplayTime(3600))
	s.Equal("00:00", planner.FormatDisplayTime(-5), "negative seconds should clamp to 00:00")
}

func (s *TimerSuite) TestFindCurrentBlockReturnsCorrectIndex() {
	blocks := []planner.TimeBlock{
		{
			Start: time.Date(2026, 4, 22, 9, 0, 0, 0, time.UTC),
			End:   time.Date(2026, 4, 22, 9, 25, 0, 0, time.UTC),
			Type:  planner.BlockFocus,
		},
		{
			Start: time.Date(2026, 4, 22, 9, 30, 0, 0, time.UTC),
			End:   time.Date(2026, 4, 22, 9, 55, 0, 0, time.UTC),
			Type:  planner.BlockFocus,
		},
		{
			Start: time.Date(2026, 4, 22, 10, 0, 0, 0, time.UTC),
			End:   time.Date(2026, 4, 22, 10, 25, 0, 0, time.UTC),
			Type:  planner.BlockShortBreak,
		},
	}

	// Now in second block → returns index 1
	nowInSecond := time.Date(2026, 4, 22, 9, 40, 0, 0, time.UTC)
	s.Equal(1, planner.FindCurrentBlock(blocks, nowInSecond))

	// Now outside all blocks → returns -1
	nowOutside := time.Date(2026, 4, 22, 11, 0, 0, 0, time.UTC)
	s.Equal(-1, planner.FindCurrentBlock(blocks, nowOutside))

	// Nil blocks → returns -1
	s.Equal(-1, planner.FindCurrentBlock(nil, nowInSecond))
}
