package ui_test

import (
	"image/color"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/ui"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
)

// --- Suite ---

type ScheduleTreeSuite struct {
	suite.Suite
}

func TestScheduleTree(t *testing.T) {
	suite.Run(t, new(ScheduleTreeSuite))
}

// helper to build a time on a fixed date for testing.
func blockTime(hour, minute int) time.Time {
	return time.Date(2026, 3, 31, hour, minute, 0, 0, time.UTC)
}

// sampleBlocks returns a schedule with two cycles separated by a long break.
// Cycle 1: focus(09:00-09:25), short_break(09:25-09:30), focus(09:30-09:55), long_break(09:55-10:10)
// Cycle 2: focus(10:10-10:35), short_break(10:35-10:40)
func sampleBlocks() []presenter.TimeBlockPreview {
	return []presenter.TimeBlockPreview{
		{Start: blockTime(9, 0), End: blockTime(9, 25), Type: "focus", TaskName: "Task A"},
		{Start: blockTime(9, 25), End: blockTime(9, 30), Type: "short_break"},
		{Start: blockTime(9, 30), End: blockTime(9, 55), Type: "focus", TaskName: "Task B"},
		{Start: blockTime(9, 55), End: blockTime(10, 10), Type: "long_break"},
		{Start: blockTime(10, 10), End: blockTime(10, 35), Type: "focus", TaskName: "Task C"},
		{Start: blockTime(10, 35), End: blockTime(10, 40), Type: "short_break"},
	}
}

// --- Cycle Grouping Tests ---

func (s *ScheduleTreeSuite) TestGroupBlocksIntoCycles() {
	blocks := sampleBlocks()
	now := blockTime(8, 0) // all blocks in the future

	tree := ui.NewScheduleTree(blocks, now)
	cycles := tree.Cycles()

	s.Equal(2, len(cycles),
		"Blocks separated by a long break should form 2 cycles")

	// Cycle 1 has: focus, short_break, focus, long_break = 4 blocks
	s.Equal(4, len(cycles[0].Blocks),
		"Cycle 1 should contain 4 blocks (up to and including the long break)")

	// Cycle 2 has: focus, short_break = 2 blocks
	s.Equal(2, len(cycles[1].Blocks),
		"Cycle 2 should contain 2 blocks (after the long break)")
}

func (s *ScheduleTreeSuite) TestSingleCycleWhenNoLongBreak() {
	blocks := []presenter.TimeBlockPreview{
		{Start: blockTime(9, 0), End: blockTime(9, 25), Type: "focus", TaskName: "Task A"},
		{Start: blockTime(9, 25), End: blockTime(9, 30), Type: "short_break"},
		{Start: blockTime(9, 30), End: blockTime(9, 55), Type: "focus", TaskName: "Task B"},
	}
	now := blockTime(8, 0)

	tree := ui.NewScheduleTree(blocks, now)
	cycles := tree.Cycles()

	s.Equal(1, len(cycles),
		"When there is no long break, all blocks should be in a single cycle")
	s.Equal(3, len(cycles[0].Blocks),
		"The single cycle should contain all 3 blocks")
}

func (s *ScheduleTreeSuite) TestCycleHeadersShowCorrectNumbering() {
	blocks := sampleBlocks()
	now := blockTime(8, 0)

	tree := ui.NewScheduleTree(blocks, now)
	cycles := tree.Cycles()

	s.Equal(2, len(cycles), "Should have 2 cycles")

	s.Equal(1, cycles[0].Number, "First cycle Number should be 1")
	s.Equal(2, cycles[0].Total, "First cycle Total should be 2")
	s.Equal(2, cycles[1].Number, "Second cycle Number should be 2")
	s.Equal(2, cycles[1].Total, "Second cycle Total should be 2")
}

// --- Block Rendering Tests ---

func (s *ScheduleTreeSuite) TestBlockRowShowsStartTime() {
	blocks := []presenter.TimeBlockPreview{
		{Start: blockTime(9, 0), End: blockTime(9, 25), Type: "focus", TaskName: "Task A"},
		{Start: blockTime(14, 5), End: blockTime(14, 30), Type: "focus", TaskName: "Task B"},
	}
	now := blockTime(8, 0)

	tree := ui.NewScheduleTree(blocks, now)
	cycles := tree.Cycles()

	s.Require().Equal(1, len(cycles))
	s.Require().Equal(2, len(cycles[0].Blocks))

	s.Equal("09:00", cycles[0].Blocks[0].StartTime,
		"Block start time should be formatted as HH:MM")
	s.Equal("14:05", cycles[0].Blocks[1].StartTime,
		"Block start time should be zero-padded HH:MM")
}

func (s *ScheduleTreeSuite) TestFocusBlockShowsFocusTitle() {
	blocks := []presenter.TimeBlockPreview{
		{Start: blockTime(9, 0), End: blockTime(9, 25), Type: "focus", TaskName: "My Important Task"},
	}
	now := blockTime(8, 0)

	tree := ui.NewScheduleTree(blocks, now)
	cycles := tree.Cycles()

	s.Require().Equal(1, len(cycles))
	s.Require().Equal(1, len(cycles[0].Blocks))
	s.Equal("Focus", cycles[0].Blocks[0].Title,
		"Focus blocks should be titled 'Focus', not the task name")
}

func (s *ScheduleTreeSuite) TestMeetingBlockShowsMeetingTitle() {
	blocks := []presenter.TimeBlockPreview{
		{Start: blockTime(10, 0), End: blockTime(10, 30), Type: "meeting", TaskName: "Sprint Planning"},
	}
	now := blockTime(8, 0)

	tree := ui.NewScheduleTree(blocks, now)
	cycles := tree.Cycles()

	s.Require().Equal(1, len(cycles))
	s.Require().Equal(1, len(cycles[0].Blocks))
	s.Equal("Meeting: Sprint Planning", cycles[0].Blocks[0].Title,
		"Meeting blocks should be titled 'Meeting: {event name}'")
}

func (s *ScheduleTreeSuite) TestShortBreakBlockTitle() {
	blocks := []presenter.TimeBlockPreview{
		{Start: blockTime(9, 25), End: blockTime(9, 30), Type: "short_break"},
	}
	now := blockTime(8, 0)

	tree := ui.NewScheduleTree(blocks, now)
	cycles := tree.Cycles()

	s.Require().Equal(1, len(cycles))
	s.Require().Equal(1, len(cycles[0].Blocks))
	s.Equal("Short Break", cycles[0].Blocks[0].Title,
		"Short break blocks should be titled 'Short Break'")
}

func (s *ScheduleTreeSuite) TestLongBreakBlockTitle() {
	blocks := []presenter.TimeBlockPreview{
		{Start: blockTime(9, 55), End: blockTime(10, 10), Type: "long_break"},
	}
	now := blockTime(8, 0)

	tree := ui.NewScheduleTree(blocks, now)
	cycles := tree.Cycles()

	s.Require().Equal(1, len(cycles))
	s.Require().Equal(1, len(cycles[0].Blocks))
	s.Equal("Long Break", cycles[0].Blocks[0].Title,
		"Long break blocks should be titled 'Long Break'")
}

// --- Elapsed Pruning Tests ---

func (s *ScheduleTreeSuite) TestElapsedBlocksPruned() {
	blocks := []presenter.TimeBlockPreview{
		{Start: blockTime(9, 0), End: blockTime(9, 25), Type: "focus", TaskName: "Task A"},  // elapsed
		{Start: blockTime(9, 25), End: blockTime(9, 30), Type: "short_break"},               // elapsed
		{Start: blockTime(9, 30), End: blockTime(9, 55), Type: "focus", TaskName: "Task B"}, // future
		{Start: blockTime(9, 55), End: blockTime(10, 10), Type: "short_break"},              // future
	}
	now := blockTime(9, 30) // 09:25 end and 09:30 end blocks are elapsed

	tree := ui.NewScheduleTree(blocks, now)
	cycles := tree.Cycles()

	s.Require().Equal(1, len(cycles))
	// Only the two future blocks should remain (09:30-09:55 and 09:55-10:10)
	s.Equal(2, len(cycles[0].Blocks),
		"Elapsed blocks (end time <= now) should be pruned")
}

func (s *ScheduleTreeSuite) TestFullyElapsedCyclesPruned() {
	blocks := sampleBlocks()
	// Set now to after cycle 1 ends (10:10) but before cycle 2 ends (10:40)
	now := blockTime(10, 10)

	tree := ui.NewScheduleTree(blocks, now)
	cycles := tree.Cycles()

	s.Equal(1, len(cycles),
		"Fully elapsed cycles should be removed entirely")
	// Remaining cycle should be cycle 2's blocks
	s.Equal(2, cycles[0].Number, "Remaining cycle should retain its original numbering")
}

func (s *ScheduleTreeSuite) TestPartiallyElapsedCycleKeptWithRemainingBlocks() {
	blocks := sampleBlocks()
	// Set now to after first focus block of cycle 1, but before the rest
	now := blockTime(9, 25)

	tree := ui.NewScheduleTree(blocks, now)
	cycles := tree.Cycles()

	s.Require().GreaterOrEqual(len(cycles), 1,
		"A partially elapsed cycle should still be present")

	// Cycle 1 started with 4 blocks; after pruning the first one (09:00-09:25),
	// 3 remain: short_break(09:25-09:30), focus(09:30-09:55), long_break(09:55-10:10)
	s.Equal(3, len(cycles[0].Blocks),
		"Partially elapsed cycle should only contain remaining (non-elapsed) blocks")
}

// --- Bar Scaling Tests ---

func (s *ScheduleTreeSuite) TestBarWidthProportionalToDuration() {
	blocks := []presenter.TimeBlockPreview{
		{Start: blockTime(9, 0), End: blockTime(9, 25), Type: "focus", TaskName: "Task A"},  // 25min (longest)
		{Start: blockTime(9, 25), End: blockTime(9, 30), Type: "short_break"},               // 5min
		{Start: blockTime(9, 30), End: blockTime(9, 45), Type: "focus", TaskName: "Task B"}, // 15min
	}
	now := blockTime(8, 0)

	tree := ui.NewScheduleTree(blocks, now)
	cycles := tree.Cycles()

	s.Require().Equal(1, len(cycles))
	rows := cycles[0].Blocks
	s.Require().Equal(3, len(rows))

	// Longest block (25min) should have BarWidth 1.0
	s.InDelta(1.0, float64(rows[0].BarWidth), 0.01,
		"Longest block should have BarWidth 1.0")

	// 5min block: 5/25 = 0.2
	s.InDelta(0.2, float64(rows[1].BarWidth), 0.01,
		"5-minute block should have BarWidth proportional to longest (5/25=0.2)")

	// 15min block: 15/25 = 0.6
	s.InDelta(0.6, float64(rows[2].BarWidth), 0.01,
		"15-minute block should have BarWidth proportional to longest (15/25=0.6)")
}

func (s *ScheduleTreeSuite) TestDurationTextFormatted() {
	blocks := []presenter.TimeBlockPreview{
		{Start: blockTime(9, 0), End: blockTime(9, 25), Type: "focus", TaskName: "Task A"},  // 25min
		{Start: blockTime(9, 25), End: blockTime(9, 30), Type: "short_break"},               // 5min
		{Start: blockTime(9, 30), End: blockTime(10, 0), Type: "focus", TaskName: "Task B"}, // 30min
	}
	now := blockTime(8, 0)

	tree := ui.NewScheduleTree(blocks, now)
	cycles := tree.Cycles()

	s.Require().Equal(1, len(cycles))
	rows := cycles[0].Blocks
	s.Require().Equal(3, len(rows))

	s.Equal("25m", rows[0].DurationText, "Duration should be formatted as '25m'")
	s.Equal("5m", rows[1].DurationText, "Duration should be formatted as '5m'")
	s.Equal("30m", rows[2].DurationText, "Duration should be formatted as '30m'")
}

// --- Block Color Tests ---

func (s *ScheduleTreeSuite) TestBlockColorByType() {
	blocks := []presenter.TimeBlockPreview{
		{Start: blockTime(9, 0), End: blockTime(9, 25), Type: "focus", TaskName: "Task A"},
		{Start: blockTime(9, 25), End: blockTime(9, 30), Type: "short_break"},
		{Start: blockTime(9, 30), End: blockTime(9, 45), Type: "long_break"},
		{Start: blockTime(9, 45), End: blockTime(10, 15), Type: "meeting", TaskName: "Standup"},
	}
	now := blockTime(8, 0)

	tree := ui.NewScheduleTree(blocks, now)
	cycles := tree.Cycles()

	s.Require().Equal(1, len(cycles))
	rows := cycles[0].Blocks
	s.Require().Equal(4, len(rows))

	// Focus = green
	s.Equal(color.RGBA{R: 76, G: 175, B: 80, A: 255}, rows[0].Color,
		"Focus blocks should be green")

	// Short break = light blue
	s.Equal(color.RGBA{R: 129, G: 212, B: 250, A: 255}, rows[1].Color,
		"Short break blocks should be light blue")

	// Long break = blue
	s.Equal(color.RGBA{R: 66, G: 165, B: 245, A: 255}, rows[2].Color,
		"Long break blocks should be blue")

	// Meeting = amber
	s.Equal(color.RGBA{R: 255, G: 193, B: 7, A: 255}, rows[3].Color,
		"Meeting blocks should be amber")
}
