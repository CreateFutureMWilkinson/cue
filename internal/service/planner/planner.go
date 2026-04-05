package planner

import (
	"context"
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

// TaskEstimator estimates the number of pomodoros for a task.
type TaskEstimator interface {
	EstimatePomodoros(ctx context.Context, title string, description string) (int, error)
}

// Planner manages day schedule generation and task estimation.
type Planner struct {
	cfg       config.PlannerConfig
	estimator TaskEstimator
	clock     Clock
}

// NewPlanner creates a new Planner with the given configuration, estimator, and clock.
func NewPlanner(cfg config.PlannerConfig, estimator TaskEstimator, clock Clock) (*Planner, error) {
	return nil, nil
}

// TargetDate returns the date that planning should target given the current time.
func (p *Planner) TargetDate(now time.Time) time.Time {
	return time.Time{}
}
