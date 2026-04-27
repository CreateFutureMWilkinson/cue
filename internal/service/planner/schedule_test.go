package planner_test

import (
	"context"
	"testing"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
	"github.com/CreateFutureMWilkinson/cue/internal/service/calendar"
	"github.com/CreateFutureMWilkinson/cue/internal/service/planner"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type ScheduleGenerationSuite struct {
	suite.Suite
	cfg   config.PlannerConfig
	clock *mockClock
	date  time.Time // 2026-03-30 (Monday)
}

func TestScheduleGeneration(t *testing.T) {
	suite.Run(t, new(ScheduleGenerationSuite))
}

func (s *ScheduleGenerationSuite) SetupTest() {
	s.cfg = validConfig()
	s.date = time.Date(2026, time.March, 30, 0, 0, 0, 0, time.UTC)
	s.clock = &mockClock{now: time.Date(2026, time.March, 30, 9, 0, 0, 0, time.UTC)}
}

func (s *ScheduleGenerationSuite) newPlanner() *planner.Planner {
	p, err := planner.NewPlanner(s.cfg, nil, s.clock)
	s.Require().NoError(err)
	return p
}

// helper to create a time on the test date
func (s *ScheduleGenerationSuite) at(hour, min int) time.Time {
	return time.Date(2026, time.March, 30, hour, min, 0, 0, time.UTC)
}

// ---------------------------------------------------------------------------
// 1. Pure Pomodoro day (no meetings, no tasks) — only focus/break blocks
// ---------------------------------------------------------------------------

func (s *ScheduleGenerationSuite) TestPurePomodoroDay_NoMeetings() {
	p := s.newPlanner()
	ctx := context.Background()

	focus, recovery, err := p.GenerateSchedules(ctx, nil, nil, s.date)
	s.Require().NoError(err)
	s.Require().NotNil(focus)
	s.Require().NotNil(recovery)

	s.Equal("focus-maximized", focus.Strategy)
	s.Equal("recovery-balanced", recovery.Strategy)
	s.Equal(s.date, focus.Date)
	s.Equal(s.date, recovery.Date)

	// Both should have blocks that span 09:00-17:00
	s.NotEmpty(focus.Blocks)
	s.NotEmpty(recovery.Blocks)

	// First block should start at workday start
	s.Equal(s.at(9, 0), focus.Blocks[0].Start)
	s.Equal(s.at(9, 0), recovery.Blocks[0].Start)

	// Last block should end at or before workday end
	s.True(!focus.Blocks[len(focus.Blocks)-1].End.After(s.at(17, 0)))
	s.True(!recovery.Blocks[len(recovery.Blocks)-1].End.After(s.at(17, 0)))
}

// ---------------------------------------------------------------------------
// 2. Pure Pomodoro day has correct block structure (focus + short breaks + long break)
// ---------------------------------------------------------------------------

func (s *ScheduleGenerationSuite) TestPurePomodoroDay_BlockStructure() {
	p := s.newPlanner()
	ctx := context.Background()

	_, recovery, err := p.GenerateSchedules(ctx, nil, nil, s.date)
	s.Require().NoError(err)

	// Count block types in recovery schedule
	var focusCount, shortBreakCount, longBreakCount int
	for _, b := range recovery.Blocks {
		switch b.Type {
		case planner.BlockFocus:
			focusCount++
			s.Equal(25*time.Minute, b.End.Sub(b.Start), "focus block should be 25 minutes")
		case planner.BlockShortBreak:
			shortBreakCount++
			s.Equal(5*time.Minute, b.End.Sub(b.Start), "short break should be 5 minutes")
		case planner.BlockLongBreak:
			longBreakCount++
			s.Equal(20*time.Minute, b.End.Sub(b.Start), "long break should be 20 minutes")
		}
	}

	s.Greater(focusCount, 0, "should have focus blocks")
	s.Greater(shortBreakCount, 0, "should have short breaks")
	s.Greater(longBreakCount, 0, "should have at least one long break")
}

// ---------------------------------------------------------------------------
// 3. Long break after every N cycles (default 4)
// ---------------------------------------------------------------------------

func (s *ScheduleGenerationSuite) TestLongBreakAfterNCycles() {
	p := s.newPlanner()
	ctx := context.Background()

	_, recovery, err := p.GenerateSchedules(ctx, nil, nil, s.date)
	s.Require().NoError(err)

	// In recovery-balanced, after every 4 focus blocks there should be a long break
	focusSinceLong := 0
	for _, b := range recovery.Blocks {
		if b.Type == planner.BlockFocus {
			focusSinceLong++
		}
		if b.Type == planner.BlockLongBreak {
			s.Equal(4, focusSinceLong, "long break should come after 4 focus blocks")
			focusSinceLong = 0
		}
	}
}

// ---------------------------------------------------------------------------
// 4. Meetings-only day — no focus blocks when meetings fill the day
// ---------------------------------------------------------------------------

func (s *ScheduleGenerationSuite) TestMeetingsOnlyDay() {
	p := s.newPlanner()
	ctx := context.Background()

	events := []calendar.CalendarEvent{
		{ID: "m1", Title: "All-day meeting", Start: s.at(9, 0), End: s.at(17, 0)},
	}

	focus, recovery, err := p.GenerateSchedules(ctx, nil, events, s.date)
	s.Require().NoError(err)

	// Should have meeting block(s) but no focus blocks
	for _, sched := range []*planner.DaySchedule{focus, recovery} {
		var focusCount, meetingCount int
		for _, b := range sched.Blocks {
			if b.Type == planner.BlockFocus {
				focusCount++
			}
			if b.Type == planner.BlockMeeting {
				meetingCount++
			}
		}
		s.Equal(0, focusCount, "all-day meeting leaves no room for focus blocks")
		s.Greater(meetingCount, 0, "should have meeting blocks")
	}
}

// ---------------------------------------------------------------------------
// 5. Mixed day — meetings + focus blocks around them
// ---------------------------------------------------------------------------

func (s *ScheduleGenerationSuite) TestMixedDay() {
	p := s.newPlanner()
	ctx := context.Background()

	events := []calendar.CalendarEvent{
		{ID: "m1", Title: "Standup", Start: s.at(10, 0), End: s.at(10, 30)},
	}

	focus, recovery, err := p.GenerateSchedules(ctx, nil, events, s.date)
	s.Require().NoError(err)

	for _, sched := range []*planner.DaySchedule{focus, recovery} {
		var hasMeeting, hasFocus bool
		for _, b := range sched.Blocks {
			if b.Type == planner.BlockMeeting {
				hasMeeting = true
				s.Equal("Standup", b.TaskName)
			}
			if b.Type == planner.BlockFocus {
				hasFocus = true
			}
		}
		s.True(hasMeeting, "should include meeting block")
		s.True(hasFocus, "should include focus blocks around meeting")
	}
}

// ---------------------------------------------------------------------------
// 6. Meeting merging — gaps < 5 minutes merged into single block
// ---------------------------------------------------------------------------

func (s *ScheduleGenerationSuite) TestMeetingMerging() {
	p := s.newPlanner()
	ctx := context.Background()

	events := []calendar.CalendarEvent{
		{ID: "m1", Title: "Meeting A", Start: s.at(10, 0), End: s.at(10, 30)},
		{ID: "m2", Title: "Meeting B", Start: s.at(10, 33), End: s.at(11, 0)}, // 3 min gap < 5 min
	}

	focus, _, err := p.GenerateSchedules(ctx, nil, events, s.date)
	s.Require().NoError(err)

	var meetingBlocks []planner.TimeBlock
	for _, b := range focus.Blocks {
		if b.Type == planner.BlockMeeting {
			meetingBlocks = append(meetingBlocks, b)
		}
	}

	// Should be merged into 1 meeting block
	s.Equal(1, len(meetingBlocks), "meetings with <5 min gap should be merged")
	s.Equal(s.at(10, 0), meetingBlocks[0].Start)
	s.Equal(s.at(11, 0), meetingBlocks[0].End)
}

// ---------------------------------------------------------------------------
// 7. Meetings NOT merged when gap >= meeting_merge_gap_minutes
// ---------------------------------------------------------------------------

func (s *ScheduleGenerationSuite) TestMeetingsNotMerged() {
	p := s.newPlanner()
	ctx := context.Background()

	events := []calendar.CalendarEvent{
		{ID: "m1", Title: "Meeting A", Start: s.at(10, 0), End: s.at(10, 30)},
		{ID: "m2", Title: "Meeting B", Start: s.at(10, 40), End: s.at(11, 0)}, // 10 min gap >= 5 min
	}

	focus, _, err := p.GenerateSchedules(ctx, nil, events, s.date)
	s.Require().NoError(err)

	var meetingBlocks []planner.TimeBlock
	for _, b := range focus.Blocks {
		if b.Type == planner.BlockMeeting {
			meetingBlocks = append(meetingBlocks, b)
		}
	}

	s.Equal(2, len(meetingBlocks), "meetings with >=5 min gap should remain separate")
}

// ---------------------------------------------------------------------------
// 8. Post-meeting recovery break (<=30min meeting -> 5min break) — recovery-balanced
// ---------------------------------------------------------------------------

func (s *ScheduleGenerationSuite) TestPostMeetingBreakShort() {
	p := s.newPlanner()
	ctx := context.Background()

	events := []calendar.CalendarEvent{
		{ID: "m1", Title: "Quick sync", Start: s.at(10, 0), End: s.at(10, 25)}, // 25 min
	}

	_, recovery, err := p.GenerateSchedules(ctx, nil, events, s.date)
	s.Require().NoError(err)

	// Find the meeting block, then check next block is a break
	for i, b := range recovery.Blocks {
		if b.Type == planner.BlockMeeting && i+1 < len(recovery.Blocks) {
			nextBlock := recovery.Blocks[i+1]
			isBreak := nextBlock.Type == planner.BlockShortBreak || nextBlock.Type == planner.BlockLongBreak
			s.True(isBreak, "block after <=30min meeting should be a break in recovery-balanced")
			s.Equal(5*time.Minute, nextBlock.End.Sub(nextBlock.Start), "post-meeting break for <=30min meeting should be 5 minutes")
			break
		}
	}
}

// ---------------------------------------------------------------------------
// 9. Post-meeting recovery break (>30min meeting -> 20min break) — recovery-balanced
// ---------------------------------------------------------------------------

func (s *ScheduleGenerationSuite) TestPostMeetingBreakLong() {
	p := s.newPlanner()
	ctx := context.Background()

	events := []calendar.CalendarEvent{
		{ID: "m1", Title: "Long planning", Start: s.at(10, 0), End: s.at(11, 0)}, // 60 min
	}

	_, recovery, err := p.GenerateSchedules(ctx, nil, events, s.date)
	s.Require().NoError(err)

	for i, b := range recovery.Blocks {
		if b.Type == planner.BlockMeeting && i+1 < len(recovery.Blocks) {
			nextBlock := recovery.Blocks[i+1]
			isBreak := nextBlock.Type == planner.BlockShortBreak || nextBlock.Type == planner.BlockLongBreak
			s.True(isBreak, "block after >30min meeting should be a break in recovery-balanced")
			s.Equal(20*time.Minute, nextBlock.End.Sub(nextBlock.Start), "post-meeting break for >30min meeting should be 20 minutes")
			break
		}
	}
}

// ---------------------------------------------------------------------------
// 10. Focus-maximized skips post-meeting recovery breaks
// ---------------------------------------------------------------------------

func (s *ScheduleGenerationSuite) TestFocusMaximizedSkipsPostMeetingBreaks() {
	p := s.newPlanner()
	ctx := context.Background()

	events := []calendar.CalendarEvent{
		{ID: "m1", Title: "Long planning", Start: s.at(10, 0), End: s.at(11, 0)},
	}

	focus, _, err := p.GenerateSchedules(ctx, nil, events, s.date)
	s.Require().NoError(err)

	for i, b := range focus.Blocks {
		if b.Type == planner.BlockMeeting && i+1 < len(focus.Blocks) {
			nextBlock := focus.Blocks[i+1]
			// In focus-maximized, the next block after a meeting should be focus, not a recovery break
			s.Equal(planner.BlockFocus, nextBlock.Type,
				"focus-maximized should not add post-meeting recovery breaks")
			break
		}
	}
}

// ---------------------------------------------------------------------------
// 11. Task assignment by priority — lower priority number = higher priority
// ---------------------------------------------------------------------------

func (s *ScheduleGenerationSuite) TestTaskAssignmentByPriority() {
	p := s.newPlanner()
	ctx := context.Background()

	id1 := uuid.New()
	id2 := uuid.New()
	tasks := []planner.TaskEstimate{
		{TodoID: id1, Title: "High priority", EstimatedPomos: 2},
		{TodoID: id2, Title: "Low priority", EstimatedPomos: 2},
	}
	// id1 has lower index = higher priority (design says sort by priority)

	focus, _, err := p.GenerateSchedules(ctx, tasks, nil, s.date)
	s.Require().NoError(err)

	// Find the first focus block with a task assigned
	var firstTaskID *uuid.UUID
	for _, b := range focus.Blocks {
		if b.Type == planner.BlockFocus && b.TaskID != nil {
			firstTaskID = b.TaskID
			break
		}
	}
	s.Require().NotNil(firstTaskID, "should have at least one task assigned")
	s.Equal(id1, *firstTaskID, "first assigned task should be the highest priority")
}

// ---------------------------------------------------------------------------
// 12. Task overflow — tasks that don't fit are not assigned
// ---------------------------------------------------------------------------

func (s *ScheduleGenerationSuite) TestTaskOverflow() {
	// Use a very short workday
	s.cfg.WorkdayStart = "09:00"
	s.cfg.WorkdayEnd = "10:00" // Only 1 hour = 2 pomodoros max
	p := s.newPlanner()
	ctx := context.Background()

	tasks := []planner.TaskEstimate{
		{TodoID: uuid.New(), Title: "Task A", EstimatedPomos: 1},
		{TodoID: uuid.New(), Title: "Task B", EstimatedPomos: 1},
		{TodoID: uuid.New(), Title: "Task C", EstimatedPomos: 1}, // Won't fit
	}

	focus, _, err := p.GenerateSchedules(ctx, tasks, nil, s.date)
	s.Require().NoError(err)

	// Count how many unique tasks got assigned
	assigned := make(map[uuid.UUID]bool)
	for _, b := range focus.Blocks {
		if b.Type == planner.BlockFocus && b.TaskID != nil {
			assigned[*b.TaskID] = true
		}
	}

	// Not all 3 tasks should fit in 1 hour
	s.Less(len(assigned), 3, "some tasks should not fit in a 1-hour workday")
}

// ---------------------------------------------------------------------------
// 13. No tasks — schedules have blocks but no task assignments
// ---------------------------------------------------------------------------

func (s *ScheduleGenerationSuite) TestNoTasks() {
	p := s.newPlanner()
	ctx := context.Background()

	focus, recovery, err := p.GenerateSchedules(ctx, nil, nil, s.date)
	s.Require().NoError(err)

	for _, sched := range []*planner.DaySchedule{focus, recovery} {
		for _, b := range sched.Blocks {
			if b.Type == planner.BlockFocus {
				s.Nil(b.TaskID, "focus blocks with no tasks should have nil TaskID")
			}
		}
	}
}

// ---------------------------------------------------------------------------
// 13b. Empty task slice — focus blocks have nil TaskID and empty TaskName
// ---------------------------------------------------------------------------

func (s *ScheduleGenerationSuite) TestEmptyTaskSlice_FocusBlocksUnassigned() {
	p := s.newPlanner()
	ctx := context.Background()

	focus, recovery, err := p.GenerateSchedules(ctx, []planner.TaskEstimate{}, nil, s.date)
	s.Require().NoError(err)
	s.Require().NotNil(focus)
	s.Require().NotNil(recovery)

	for _, sched := range []*planner.DaySchedule{focus, recovery} {
		s.NotEmpty(sched.Blocks, "schedule should still have blocks")
		for i, b := range sched.Blocks {
			if b.Type == planner.BlockFocus {
				s.Nil(b.TaskID, "focus block %d in %s should have nil TaskID with empty task list", i, sched.Strategy)
				s.Empty(b.TaskName, "focus block %d in %s should have empty TaskName with empty task list", i, sched.Strategy)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// 14. No overlapping blocks — blocks must be contiguous or sequential
// ---------------------------------------------------------------------------

func (s *ScheduleGenerationSuite) TestNoOverlappingBlocks() {
	p := s.newPlanner()
	ctx := context.Background()

	events := []calendar.CalendarEvent{
		{ID: "m1", Title: "Standup", Start: s.at(10, 0), End: s.at(10, 30)},
	}

	focus, recovery, err := p.GenerateSchedules(ctx, nil, events, s.date)
	s.Require().NoError(err)

	for _, sched := range []*planner.DaySchedule{focus, recovery} {
		for i := 1; i < len(sched.Blocks); i++ {
			prev := sched.Blocks[i-1]
			curr := sched.Blocks[i]
			s.True(!curr.Start.Before(prev.End),
				"block %d starts at %v but previous block ends at %v — overlap detected",
				i, curr.Start, prev.End)
		}
	}
}

// ---------------------------------------------------------------------------
// 15. All-day event — treated as a meeting consuming the full workday
// ---------------------------------------------------------------------------

func (s *ScheduleGenerationSuite) TestAllDayEvent() {
	p := s.newPlanner()
	ctx := context.Background()

	events := []calendar.CalendarEvent{
		{ID: "allday", Title: "Company offsite", Start: s.at(0, 0), End: s.at(0, 0).AddDate(0, 0, 1), AllDay: true},
	}

	focus, _, err := p.GenerateSchedules(ctx, nil, events, s.date)
	s.Require().NoError(err)

	var focusCount int
	for _, b := range focus.Blocks {
		if b.Type == planner.BlockFocus {
			focusCount++
		}
	}
	s.Equal(0, focusCount, "all-day event should leave no focus blocks")
}

// ---------------------------------------------------------------------------
// 16. User override on task estimate
// ---------------------------------------------------------------------------

func (s *ScheduleGenerationSuite) TestUserOverrideApplied() {
	p := s.newPlanner()
	ctx := context.Background()

	override := 1
	tasks := []planner.TaskEstimate{
		{TodoID: uuid.New(), Title: "Overridden task", EstimatedPomos: 5, UserOverride: &override},
	}

	focus, _, err := p.GenerateSchedules(ctx, tasks, nil, s.date)
	s.Require().NoError(err)

	// Count focus blocks assigned to this task — should be 1, not 5
	var taskFocusCount int
	for _, b := range focus.Blocks {
		if b.Type == planner.BlockFocus && b.TaskID != nil && *b.TaskID == tasks[0].TodoID {
			taskFocusCount++
		}
	}
	s.Equal(1, taskFocusCount, "should use user override (1) not estimated (5)")
}

// ---------------------------------------------------------------------------
// 17. Schedule IDs are unique UUIDs
// ---------------------------------------------------------------------------

func (s *ScheduleGenerationSuite) TestScheduleIDsUnique() {
	p := s.newPlanner()
	ctx := context.Background()

	focus, recovery, err := p.GenerateSchedules(ctx, nil, nil, s.date)
	s.Require().NoError(err)

	s.NotEqual(uuid.Nil, focus.ID, "focus schedule should have a non-nil UUID")
	s.NotEqual(uuid.Nil, recovery.ID, "recovery schedule should have a non-nil UUID")
	s.NotEqual(focus.ID, recovery.ID, "schedules should have different UUIDs")
}
