package planner

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
)

// BlockType represents the type of a time block in a day schedule.
type BlockType int

const (
	BlockFocus      BlockType = iota // Pomodoro focus window
	BlockShortBreak                  // Short break (default 5min)
	BlockLongBreak                   // Long break (default 20min)
	BlockMeeting                     // Calendar event (fixed)
)

// TimeBlock represents a single scheduled block of time.
type TimeBlock struct {
	Start    time.Time
	End      time.Time
	Type     BlockType
	TaskID   *uuid.UUID
	TaskName string
}

// DaySchedule represents a full day's schedule of time blocks.
type DaySchedule struct {
	ID        uuid.UUID
	Date      time.Time
	Strategy  string // "focus-maximized" | "recovery-balanced"
	Blocks    []TimeBlock
	CreatedAt time.Time
}

// TaskEstimate represents a task with its estimated pomodoro count.
type TaskEstimate struct {
	TodoID         uuid.UUID
	Title          string
	EstimatedPomos int
	UserOverride   *int
}

// EffectivePomos returns the user override if set, otherwise the estimated count.
func (te TaskEstimate) EffectivePomos() int {
	if te.UserOverride != nil {
		return *te.UserOverride
	}
	return te.EstimatedPomos
}

// Clock provides the current time, allowing test injection.
type Clock interface {
	Now() time.Time
}

// TaskEstimator estimates the time needed for a task.
type TaskEstimator interface {
	EstimateMinutes(ctx context.Context, title string, description string) (int, error)
}

// Planner manages day schedule generation and task estimation.
type Planner struct {
	cfg       config.PlannerConfig
	estimator TaskEstimator
	clock     Clock
}

// NewPlanner creates a new Planner with the given configuration, estimator, and clock.
func NewPlanner(cfg config.PlannerConfig, estimator TaskEstimator, clock Clock) (*Planner, error) {
	if clock == nil {
		return nil, fmt.Errorf("clock must not be nil")
	}
	if err := validatePlannerConfig(cfg); err != nil {
		return nil, err
	}
	return &Planner{
		cfg:       cfg,
		estimator: estimator,
		clock:     clock,
	}, nil
}

func validatePlannerConfig(cfg config.PlannerConfig) error {
	if cfg.PomodoroMinutes <= 0 {
		return fmt.Errorf("pomodoro_minutes must be greater than 0")
	}
	if cfg.ShortBreakMinutes <= 0 {
		return fmt.Errorf("short_break_minutes must be greater than 0")
	}
	if cfg.LongBreakMinutes <= 0 {
		return fmt.Errorf("long_break_minutes must be greater than 0")
	}
	if cfg.LongBreakAfterCycles <= 0 {
		return fmt.Errorf("long_break_after_cycles must be greater than 0")
	}
	if err := validateTimeFormat(cfg.WorkdayStart); err != nil {
		return fmt.Errorf("workday_start must be a valid HH:MM time: %w", err)
	}
	if err := validateTimeFormat(cfg.WorkdayEnd); err != nil {
		return fmt.Errorf("workday_end must be a valid HH:MM time: %w", err)
	}
	if err := validateTimeFormat(cfg.PlanningCutoff); err != nil {
		return fmt.Errorf("planning_cutoff must be a valid HH:MM time: %w", err)
	}
	ws, _ := parseTimeFormat(cfg.WorkdayStart)
	we, _ := parseTimeFormat(cfg.WorkdayEnd)
	if !we.After(ws) {
		return fmt.Errorf("workday_end must be after workday_start")
	}
	return nil
}

// TargetDate returns the date that planning should target given the current time.
// If now is before the planning cutoff on a weekday, returns today.
// If now is at or after the cutoff, or on a weekend, returns the next working day.
func (p *Planner) TargetDate(now time.Time) time.Time {
	cutoff, _ := parseTimeFormat(p.cfg.PlanningCutoff)
	cutoffToday := time.Date(now.Year(), now.Month(), now.Day(),
		cutoff.Hour(), cutoff.Minute(), 0, 0, now.Location())

	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	if isWeekday(now) && now.Before(cutoffToday) {
		return today
	}
	return nextWorkingDay(today)
}

func isWeekday(t time.Time) bool {
	day := t.Weekday()
	return day != time.Saturday && day != time.Sunday
}

func nextWorkingDay(date time.Time) time.Time {
	next := date.AddDate(0, 0, 1)
	for !isWeekday(next) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

func validateTimeFormat(timeStr string) error {
	_, err := time.Parse("15:04", timeStr)
	return err
}

func parseTimeFormat(timeStr string) (time.Time, error) {
	return time.Parse("15:04", timeStr)
}
