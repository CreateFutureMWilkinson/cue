package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ScheduleBlockType represents the type of a time block.
type ScheduleBlockType int

const (
	ScheduleBlockFocus ScheduleBlockType = iota
	ScheduleBlockShortBreak
	ScheduleBlockLongBreak
	ScheduleBlockMeeting
)

// ScheduleBlock represents a single block of time in a day schedule.
type ScheduleBlock struct {
	Start    time.Time
	End      time.Time
	Type     ScheduleBlockType
	TaskID   *uuid.UUID
	TaskName string
}

// Schedule represents a persisted day schedule.
type Schedule struct {
	ID        uuid.UUID
	Date      time.Time
	Strategy  string // "focus-maximized" | "recovery-balanced"
	Blocks    []ScheduleBlock
	CreatedAt time.Time
}

// ScheduleRepository defines the contract for schedule persistence.
type ScheduleRepository interface {
	Save(ctx context.Context, schedule *Schedule) error
	LoadByDate(ctx context.Context, date time.Time) (*Schedule, error)
	Delete(ctx context.Context, date time.Time) error
}
