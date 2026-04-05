package planner_test

import (
	"testing"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
	"github.com/CreateFutureMWilkinson/cue/internal/service/planner"
	"github.com/stretchr/testify/suite"
)

// mockClock is a test double for the Clock interface.
type mockClock struct {
	now time.Time
}

func (m *mockClock) Now() time.Time {
	return m.now
}

// validConfig returns a PlannerConfig with all fields populated correctly.
func validConfig() config.PlannerConfig {
	return config.PlannerConfig{
		WorkdayStart:           "09:00",
		WorkdayEnd:             "17:00",
		PlanningCutoff:         "16:00",
		PomodoroMinutes:        25,
		ShortBreakMinutes:      5,
		LongBreakMinutes:       20,
		LongBreakAfterCycles:   4,
		MeetingMergeGapMinutes: 5,
		LunchWindowStart:       "12:00",
		LunchWindowEnd:         "14:00",
	}
}

type PlannerSuite struct {
	suite.Suite
}

func TestPlanner(t *testing.T) {
	suite.Run(t, new(PlannerSuite))
}

// ---------------------------------------------------------------------------
// 1. TestNewPlannerValidConfig — constructor succeeds with valid config and mock clock
// ---------------------------------------------------------------------------

func (s *PlannerSuite) TestNewPlannerValidConfig() {
	cfg := validConfig()
	clock := &mockClock{now: time.Now()}

	p, err := planner.NewPlanner(cfg, nil, clock)
	s.Require().NoError(err)
	s.Require().NotNil(p, "NewPlanner must return a non-nil Planner with valid config")
}

// ---------------------------------------------------------------------------
// 2. TestNewPlannerNilClock — constructor returns error when clock is nil
// ---------------------------------------------------------------------------

func (s *PlannerSuite) TestNewPlannerNilClock() {
	cfg := validConfig()

	p, err := planner.NewPlanner(cfg, nil, nil)
	s.Require().Error(err, "NewPlanner must return error when clock is nil")
	s.Nil(p)
	s.Contains(err.Error(), "clock")
}

// ---------------------------------------------------------------------------
// 3. TestNewPlannerNilEstimator — constructor succeeds (estimator is optional)
// ---------------------------------------------------------------------------

func (s *PlannerSuite) TestNewPlannerNilEstimator() {
	cfg := validConfig()
	clock := &mockClock{now: time.Now()}

	p, err := planner.NewPlanner(cfg, nil, clock)
	s.Require().NoError(err)
	s.Require().NotNil(p, "NewPlanner must succeed when estimator is nil")
}

// ---------------------------------------------------------------------------
// 4. TestNewPlannerInvalidConfig — constructor returns error for zero-value config
// ---------------------------------------------------------------------------

func (s *PlannerSuite) TestNewPlannerInvalidConfig() {
	clock := &mockClock{now: time.Now()}

	tests := []struct {
		name   string
		modify func(cfg *config.PlannerConfig)
		errMsg string
	}{
		{
			name:   "zero pomodoro_minutes",
			modify: func(cfg *config.PlannerConfig) { cfg.PomodoroMinutes = 0 },
			errMsg: "pomodoro_minutes",
		},
		{
			name:   "zero short_break_minutes",
			modify: func(cfg *config.PlannerConfig) { cfg.ShortBreakMinutes = 0 },
			errMsg: "short_break_minutes",
		},
		{
			name:   "zero long_break_minutes",
			modify: func(cfg *config.PlannerConfig) { cfg.LongBreakMinutes = 0 },
			errMsg: "long_break_minutes",
		},
		{
			name:   "zero long_break_after_cycles",
			modify: func(cfg *config.PlannerConfig) { cfg.LongBreakAfterCycles = 0 },
			errMsg: "long_break_after_cycles",
		},
		{
			name:   "empty workday_start",
			modify: func(cfg *config.PlannerConfig) { cfg.WorkdayStart = "" },
			errMsg: "workday_start",
		},
		{
			name:   "empty workday_end",
			modify: func(cfg *config.PlannerConfig) { cfg.WorkdayEnd = "" },
			errMsg: "workday_end",
		},
		{
			name:   "empty planning_cutoff",
			modify: func(cfg *config.PlannerConfig) { cfg.PlanningCutoff = "" },
			errMsg: "planning_cutoff",
		},
		{
			name:   "workday_end before workday_start",
			modify: func(cfg *config.PlannerConfig) { cfg.WorkdayStart = "17:00"; cfg.WorkdayEnd = "09:00" },
			errMsg: "workday_end",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			cfg := validConfig()
			tc.modify(&cfg)

			p, err := planner.NewPlanner(cfg, nil, clock)
			s.Require().Error(err, "expected error for %s", tc.name)
			s.Nil(p)
			s.Contains(err.Error(), tc.errMsg)
		})
	}
}

// ---------------------------------------------------------------------------
// 5. TestTargetDateBeforeCutoff — returns today (Monday 10:00 with cutoff 16:00)
// ---------------------------------------------------------------------------

func (s *PlannerSuite) TestTargetDateBeforeCutoff() {
	cfg := validConfig()
	// Monday 2026-03-30 10:00
	now := time.Date(2026, time.March, 30, 10, 0, 0, 0, time.UTC)
	clock := &mockClock{now: now}

	p, err := planner.NewPlanner(cfg, nil, clock)
	s.Require().NoError(err)
	s.Require().NotNil(p)

	target := p.TargetDate(now)
	expected := time.Date(2026, time.March, 30, 0, 0, 0, 0, time.UTC)
	s.Equal(expected, target, "before cutoff on weekday should return today")
}

// ---------------------------------------------------------------------------
// 6. TestTargetDateAfterCutoff — returns next working day (Monday 17:00 -> Tuesday)
// ---------------------------------------------------------------------------

func (s *PlannerSuite) TestTargetDateAfterCutoff() {
	cfg := validConfig()
	// Monday 2026-03-30 17:00
	now := time.Date(2026, time.March, 30, 17, 0, 0, 0, time.UTC)
	clock := &mockClock{now: now}

	p, err := planner.NewPlanner(cfg, nil, clock)
	s.Require().NoError(err)
	s.Require().NotNil(p)

	target := p.TargetDate(now)
	expected := time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC)
	s.Equal(expected, target, "after cutoff on Monday should return Tuesday")
}

// ---------------------------------------------------------------------------
// 7. TestTargetDateAtCutoff — at exactly cutoff time -> next working day
// ---------------------------------------------------------------------------

func (s *PlannerSuite) TestTargetDateAtCutoff() {
	cfg := validConfig()
	// Monday 2026-03-30 16:00 (exactly at cutoff)
	now := time.Date(2026, time.March, 30, 16, 0, 0, 0, time.UTC)
	clock := &mockClock{now: now}

	p, err := planner.NewPlanner(cfg, nil, clock)
	s.Require().NoError(err)
	s.Require().NotNil(p)

	target := p.TargetDate(now)
	expected := time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC)
	s.Equal(expected, target, "at exactly cutoff should return next working day")
}

// ---------------------------------------------------------------------------
// 8. TestTargetDateFridayAfterCutoff — Friday after cutoff -> Monday
// ---------------------------------------------------------------------------

func (s *PlannerSuite) TestTargetDateFridayAfterCutoff() {
	cfg := validConfig()
	// Friday 2026-03-27 17:00
	now := time.Date(2026, time.March, 27, 17, 0, 0, 0, time.UTC)
	clock := &mockClock{now: now}

	p, err := planner.NewPlanner(cfg, nil, clock)
	s.Require().NoError(err)
	s.Require().NotNil(p)

	target := p.TargetDate(now)
	expected := time.Date(2026, time.March, 30, 0, 0, 0, 0, time.UTC)
	s.Equal(expected, target, "Friday after cutoff should return Monday")
}

// ---------------------------------------------------------------------------
// 9. TestTargetDateSaturday — Saturday -> Monday
// ---------------------------------------------------------------------------

func (s *PlannerSuite) TestTargetDateSaturday() {
	cfg := validConfig()
	// Saturday 2026-03-28 12:00
	now := time.Date(2026, time.March, 28, 12, 0, 0, 0, time.UTC)
	clock := &mockClock{now: now}

	p, err := planner.NewPlanner(cfg, nil, clock)
	s.Require().NoError(err)
	s.Require().NotNil(p)

	target := p.TargetDate(now)
	expected := time.Date(2026, time.March, 30, 0, 0, 0, 0, time.UTC)
	s.Equal(expected, target, "Saturday should return Monday")
}

// ---------------------------------------------------------------------------
// 10. TestTargetDateSunday — Sunday -> Monday
// ---------------------------------------------------------------------------

func (s *PlannerSuite) TestTargetDateSunday() {
	cfg := validConfig()
	// Sunday 2026-03-29 14:00
	now := time.Date(2026, time.March, 29, 14, 0, 0, 0, time.UTC)
	clock := &mockClock{now: now}

	p, err := planner.NewPlanner(cfg, nil, clock)
	s.Require().NoError(err)
	s.Require().NotNil(p)

	target := p.TargetDate(now)
	expected := time.Date(2026, time.March, 30, 0, 0, 0, 0, time.UTC)
	s.Equal(expected, target, "Sunday should return Monday")
}
