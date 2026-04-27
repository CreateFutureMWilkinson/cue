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
