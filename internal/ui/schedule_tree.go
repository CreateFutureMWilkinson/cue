package ui

import (
	"fmt"
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
	rawCycles := t.groupBlocksIntoCycles()
	totalCycles := len(rawCycles)
	maxDuration := t.findMaxRemainingDuration(rawCycles)

	return t.buildPrunedCycles(rawCycles, totalCycles, maxDuration)
}

// groupBlocksIntoCycles groups blocks into cycles. A long_break ends its cycle,
// and a new cycle begins at the next focus block.
func (t *ScheduleTree) groupBlocksIntoCycles() [][]presenter.TimeBlockPreview {
	var rawCycles [][]presenter.TimeBlockPreview
	var current []presenter.TimeBlockPreview
	sawLongBreak := false

	for _, b := range t.blocks {
		if sawLongBreak && b.Type == "focus" {
			// Start a new cycle at the next focus block after a long break.
			rawCycles = append(rawCycles, current)
			current = nil
			sawLongBreak = false
		}
		current = append(current, b)
		if b.Type == "long_break" {
			sawLongBreak = true
		}
	}
	if len(current) > 0 {
		rawCycles = append(rawCycles, current)
	}

	return rawCycles
}

// findMaxRemainingDuration finds the longest remaining (non-elapsed) block duration for bar scaling.
func (t *ScheduleTree) findMaxRemainingDuration(rawCycles [][]presenter.TimeBlockPreview) time.Duration {
	var maxDuration time.Duration
	for _, cycle := range rawCycles {
		for _, b := range cycle {
			if !b.End.After(t.now) {
				continue // elapsed
			}
			d := b.End.Sub(b.Start)
			if d > maxDuration {
				maxDuration = d
			}
		}
	}
	return maxDuration
}

// buildPrunedCycles builds the final cycles, pruning elapsed blocks and fully-elapsed cycles.
func (t *ScheduleTree) buildPrunedCycles(rawCycles [][]presenter.TimeBlockPreview, totalCycles int, maxDuration time.Duration) []ScheduleCycle {
	var result []ScheduleCycle
	for i, cycle := range rawCycles {
		var rows []ScheduleBlockRow
		for _, b := range cycle {
			if !b.End.After(t.now) {
				continue // elapsed: End <= now
			}
			rows = append(rows, renderBlock(b, maxDuration))
		}
		if len(rows) == 0 {
			continue // fully elapsed cycle
		}
		result = append(result, ScheduleCycle{
			Number: i + 1,
			Total:  totalCycles,
			Blocks: rows,
		})
	}
	return result
}

// renderBlock converts a TimeBlockPreview into a ScheduleBlockRow.
func renderBlock(b presenter.TimeBlockPreview, maxDuration time.Duration) ScheduleBlockRow {
	duration := b.End.Sub(b.Start)
	minutes := int(duration.Minutes())

	var barWidth float32
	if maxDuration > 0 {
		barWidth = float32(float64(duration) / float64(maxDuration))
	}

	return ScheduleBlockRow{
		StartTime:    b.Start.Format("15:04"),
		Title:        blockTitle(b),
		DurationText: fmt.Sprintf("%dm", minutes),
		BarWidth:     barWidth,
		Color:        blockColor(b.Type),
	}
}

// blockTitle returns the display title for a block based on its type.
func blockTitle(b presenter.TimeBlockPreview) string {
	switch b.Type {
	case "focus":
		return "Focus"
	case "meeting":
		return "Meeting: " + b.TaskName
	case "short_break":
		return "Short Break"
	case "long_break":
		return "Long Break"
	default:
		return b.Type
	}
}

// blockColor returns the color for a block type.
func blockColor(blockType string) color.Color {
	switch blockType {
	case "focus":
		return color.RGBA{R: 76, G: 175, B: 80, A: 255}
	case "short_break":
		return color.RGBA{R: 129, G: 212, B: 250, A: 255}
	case "long_break":
		return color.RGBA{R: 66, G: 165, B: 245, A: 255}
	case "meeting":
		return color.RGBA{R: 255, G: 193, B: 7, A: 255}
	default:
		return color.RGBA{R: 128, G: 128, B: 128, A: 255}
	}
}
