package planner

import (
	"fmt"
	"time"
)

// TimerState represents the computed state of the timer at a point in time.
type TimerState struct {
	Running          bool
	BlockType        string // "focus", "short_break", "long_break", "meeting"
	TaskName         string
	DurationSeconds  int
	ElapsedSeconds   int
	RemainingSeconds int
	DisplayTime      string  // "MM:SS"
	ElapsedFraction  float64 // 0.0-1.0
	BlockIndex       int
}

// ComputeTimerState computes the timer state for a schedule at the given time.
func ComputeTimerState(schedule *DaySchedule, now time.Time) TimerState {
	if schedule == nil {
		return TimerState{Running: false}
	}

	idx := FindCurrentBlock(schedule.Blocks, now)
	if idx == -1 {
		return TimerState{Running: false}
	}

	block := schedule.Blocks[idx]
	totalSeconds := int(block.End.Sub(block.Start).Seconds())
	elapsedSeconds := int(now.Sub(block.Start).Seconds())
	remainingSeconds := totalSeconds - elapsedSeconds

	remainingSeconds = max(remainingSeconds, 0)

	// Calculate elapsed fraction, avoiding division by zero
	var elapsedFraction float64
	if totalSeconds > 0 {
		elapsedFraction = float64(elapsedSeconds) / float64(totalSeconds)
	}

	return TimerState{
		Running:          true,
		BlockType:        BlockTypeString(block.Type),
		TaskName:         block.TaskName,
		DurationSeconds:  totalSeconds,
		ElapsedSeconds:   elapsedSeconds,
		RemainingSeconds: remainingSeconds,
		DisplayTime:      FormatDisplayTime(remainingSeconds),
		ElapsedFraction:  elapsedFraction,
		BlockIndex:       idx,
	}
}

// FindCurrentBlock returns the index of the block containing now, or -1 if none.
func FindCurrentBlock(blocks []TimeBlock, now time.Time) int {
	if blocks == nil {
		return -1
	}

	for i, b := range blocks {
		if !now.Before(b.Start) && now.Before(b.End) {
			return i
		}
	}
	return -1
}

// BlockTypeString converts a BlockType to its string representation.
func BlockTypeString(bt BlockType) string {
	switch bt {
	case BlockFocus:
		return "focus"
	case BlockShortBreak:
		return "short_break"
	case BlockLongBreak:
		return "long_break"
	case BlockMeeting:
		return "meeting"
	default:
		return ""
	}
}

// FormatDisplayTime formats seconds as "MM:SS".
func FormatDisplayTime(seconds int) string {
	if seconds < 0 {
		seconds = 0
	}
	m := seconds / 60
	s := seconds % 60
	return fmt.Sprintf("%02d:%02d", m, s)
}
