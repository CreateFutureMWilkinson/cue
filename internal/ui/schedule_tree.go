package ui

import (
	"image/color"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
)

// ScheduleBlockRow represents a single rendered block row in the schedule tree.
type ScheduleBlockRow struct {
	StartTime    string
	Title        string
	DurationText string
	BarWidth     float32
	Color        color.Color
}

// ScheduleCycle represents a group of blocks forming one Pomodoro cycle.
type ScheduleCycle struct {
	Number int
	Total  int
	Blocks []ScheduleBlockRow
}

// ScheduleTree is a widget that displays the active schedule grouped by Pomodoro cycles.
type ScheduleTree struct {
	blocks []presenter.TimeBlockPreview
	now    time.Time
}

// NewScheduleTree creates a new ScheduleTree from the given blocks and current time.
func NewScheduleTree(blocks []presenter.TimeBlockPreview, now time.Time) *ScheduleTree {
	return &ScheduleTree{
		blocks: blocks,
		now:    now,
	}
}

// Cycles returns the schedule blocks grouped into Pomodoro cycles.
func (t *ScheduleTree) Cycles() []ScheduleCycle {
	return nil // TODO: implement in GREEN phase
}
