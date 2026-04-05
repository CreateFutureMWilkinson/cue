package planner

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/CreateFutureMWilkinson/cue/internal/service/calendar"
)

// GenerateSchedules produces two candidate schedules from tasks and calendar events.
func (p *Planner) GenerateSchedules(
	ctx context.Context,
	tasks []TaskEstimate,
	events []calendar.CalendarEvent,
	targetDate time.Time,
) (focusMaximized *DaySchedule, recoveryBalanced *DaySchedule, err error) {
	ws := p.parseTime(p.cfg.WorkdayStart, targetDate)
	we := p.parseTime(p.cfg.WorkdayEnd, targetDate)

	meetings := p.buildMeetingBlocks(events, ws, we)
	meetings = p.mergeMeetings(meetings)

	focusBlocks := p.generateFocusMaximized(meetings, ws, we)
	recoveryBlocks := p.generateRecoveryBalanced(meetings, ws, we)

	focusBlocks = p.assignTasks(focusBlocks, tasks)
	recoveryBlocks = p.assignTasks(recoveryBlocks, tasks)

	now := p.clock.Now()
	focusMaximized = &DaySchedule{
		ID:        uuid.New(),
		Date:      targetDate,
		Strategy:  "focus-maximized",
		Blocks:    focusBlocks,
		CreatedAt: now,
	}
	recoveryBalanced = &DaySchedule{
		ID:        uuid.New(),
		Date:      targetDate,
		Strategy:  "recovery-balanced",
		Blocks:    recoveryBlocks,
		CreatedAt: now,
	}
	return focusMaximized, recoveryBalanced, nil
}

func (p *Planner) parseTime(hhmm string, date time.Time) time.Time {
	t, _ := parseTimeFormat(hhmm)
	return time.Date(date.Year(), date.Month(), date.Day(),
		t.Hour(), t.Minute(), 0, 0, date.Location())
}

func (p *Planner) buildMeetingBlocks(events []calendar.CalendarEvent, ws, we time.Time) []TimeBlock {
	var meetings []TimeBlock
	for _, e := range events {
		start := e.Start
		end := e.End

		// Clamp all-day events and events extending beyond workday to workday bounds
		if e.AllDay || start.Before(ws) {
			start = ws
		}
		if e.AllDay || end.After(we) {
			end = we
		}
		if !start.Before(end) {
			continue
		}

		meetings = append(meetings, TimeBlock{
			Start:    start,
			End:      end,
			Type:     BlockMeeting,
			TaskName: e.Title,
		})
	}

	sort.Slice(meetings, func(i, j int) bool {
		return meetings[i].Start.Before(meetings[j].Start)
	})
	return meetings
}

func (p *Planner) mergeMeetings(meetings []TimeBlock) []TimeBlock {
	if len(meetings) == 0 {
		return meetings
	}

	gapThreshold := time.Duration(p.cfg.MeetingMergeGapMinutes) * time.Minute
	merged := []TimeBlock{meetings[0]}

	for i := 1; i < len(meetings); i++ {
		lastMeeting := &merged[len(merged)-1]
		currentMeeting := meetings[i]
		gap := currentMeeting.Start.Sub(lastMeeting.End)

		if gap < gapThreshold {
			// Merge: extend last meeting to cover this one
			if currentMeeting.End.After(lastMeeting.End) {
				lastMeeting.End = currentMeeting.End
			}
			lastMeeting.TaskName = lastMeeting.TaskName + " + " + currentMeeting.TaskName
		} else {
			merged = append(merged, currentMeeting)
		}
	}
	return merged
}

// generateFocusMaximized fills all gaps between meetings with focus blocks,
// using shortest possible breaks (short break between pomodoros, one long break at lunch).
func (p *Planner) generateFocusMaximized(meetings []TimeBlock, ws, we time.Time) []TimeBlock {
	gaps := p.findGaps(meetings, ws, we)
	var blocks []TimeBlock

	// Add meeting blocks
	blocks = append(blocks, meetings...)

	pomoDur := time.Duration(p.cfg.PomodoroMinutes) * time.Minute
	shortDur := time.Duration(p.cfg.ShortBreakMinutes) * time.Minute
	longDur := time.Duration(p.cfg.LongBreakMinutes) * time.Minute
	cycleLen := p.cfg.LongBreakAfterCycles
	lunchStart := p.parseTime(p.cfg.LunchWindowStart, ws)
	lunchEnd := p.parseTime(p.cfg.LunchWindowEnd, ws)

	totalFocus := 0
	longBreakPlaced := false

	for _, gap := range gaps {
		cursor := gap.start
		for p.fitsInGap(cursor, gap.end, pomoDur) {
			// Focus block
			blocks = append(blocks, TimeBlock{
				Start: cursor,
				End:   cursor.Add(pomoDur),
				Type:  BlockFocus,
			})
			cursor = cursor.Add(pomoDur)
			totalFocus++

			// Check if we need a long break (place at lunch window if possible)
			if totalFocus%cycleLen == 0 && !longBreakPlaced &&
				!cursor.Before(lunchStart) && cursor.Before(lunchEnd) &&
				p.fitsInGap(cursor, gap.end, longDur) {
				blocks = append(blocks, TimeBlock{
					Start: cursor,
					End:   cursor.Add(longDur),
					Type:  BlockLongBreak,
				})
				cursor = cursor.Add(longDur)
				longBreakPlaced = true
			} else if p.fitsInGap(cursor, gap.end, pomoDur) {
				// Short break only if another focus block fits after it
				if p.fitsInGap(cursor, gap.end, shortDur+pomoDur) {
					blocks = append(blocks, TimeBlock{
						Start: cursor,
						End:   cursor.Add(shortDur),
						Type:  BlockShortBreak,
					})
					cursor = cursor.Add(shortDur)
				}
			}
		}
	}

	sort.Slice(blocks, func(i, j int) bool {
		return blocks[i].Start.Before(blocks[j].Start)
	})
	return blocks
}

// generateRecoveryBalanced places post-meeting breaks and standard pomodoro cycles.
func (p *Planner) generateRecoveryBalanced(meetings []TimeBlock, ws, we time.Time) []TimeBlock {
	var blocks []TimeBlock
	blocks = append(blocks, meetings...)

	pomoDur := time.Duration(p.cfg.PomodoroMinutes) * time.Minute
	shortDur := time.Duration(p.cfg.ShortBreakMinutes) * time.Minute
	longDur := time.Duration(p.cfg.LongBreakMinutes) * time.Minute
	cycleLen := p.cfg.LongBreakAfterCycles

	// Build post-meeting breaks
	type postMeetingBreak struct {
		after    time.Time
		duration time.Duration
	}
	var postMeetingBreaks []postMeetingBreak
	for _, meeting := range meetings {
		meetingLength := meeting.End.Sub(meeting.Start)
		if meetingLength <= 30*time.Minute {
			postMeetingBreaks = append(postMeetingBreaks, postMeetingBreak{after: meeting.End, duration: shortDur})
		} else {
			postMeetingBreaks = append(postMeetingBreaks, postMeetingBreak{after: meeting.End, duration: longDur})
		}
	}

	gaps := p.findGaps(meetings, ws, we)
	focusCount := 0

	for _, gap := range gaps {
		cursor := gap.start

		// Check if this gap starts right after a meeting — place post-meeting break
		for _, pmBreak := range postMeetingBreaks {
			if pmBreak.after.Equal(cursor) && p.fitsInGap(cursor, gap.end, pmBreak.duration) {
				blocks = append(blocks, TimeBlock{
					Start: cursor,
					End:   cursor.Add(pmBreak.duration),
					Type:  breakTypeForDuration(pmBreak.duration, shortDur),
				})
				cursor = cursor.Add(pmBreak.duration)
				break
			}
		}

		// Fill with pomodoro cycles
		for p.fitsInGap(cursor, gap.end, pomoDur) {
			blocks = append(blocks, TimeBlock{
				Start: cursor,
				End:   cursor.Add(pomoDur),
				Type:  BlockFocus,
			})
			cursor = cursor.Add(pomoDur)
			focusCount++

			// After N focus blocks, place a long break
			if focusCount%cycleLen == 0 {
				if p.fitsInGap(cursor, gap.end, longDur) {
					blocks = append(blocks, TimeBlock{
						Start: cursor,
						End:   cursor.Add(longDur),
						Type:  BlockLongBreak,
					})
					cursor = cursor.Add(longDur)
				}
			} else if p.fitsInGap(cursor, gap.end, pomoDur) {
				// Short break if another focus fits
				if p.fitsInGap(cursor, gap.end, shortDur+pomoDur) {
					blocks = append(blocks, TimeBlock{
						Start: cursor,
						End:   cursor.Add(shortDur),
						Type:  BlockShortBreak,
					})
					cursor = cursor.Add(shortDur)
				}
			}
		}
	}

	sort.Slice(blocks, func(i, j int) bool {
		return blocks[i].Start.Before(blocks[j].Start)
	})
	return blocks
}

func breakTypeForDuration(dur, shortDur time.Duration) BlockType {
	if dur <= shortDur {
		return BlockShortBreak
	}
	return BlockLongBreak
}

type timeGap struct {
	start time.Time
	end   time.Time
}

func (p *Planner) findGaps(meetings []TimeBlock, ws, we time.Time) []timeGap {
	if len(meetings) == 0 {
		return []timeGap{{start: ws, end: we}}
	}

	var gaps []timeGap
	cursor := ws

	for _, m := range meetings {
		if cursor.Before(m.Start) {
			gaps = append(gaps, timeGap{start: cursor, end: m.Start})
		}
		if m.End.After(cursor) {
			cursor = m.End
		}
	}

	if cursor.Before(we) {
		gaps = append(gaps, timeGap{start: cursor, end: we})
	}
	return gaps
}

func (p *Planner) fitsInGap(cursor, gapEnd time.Time, duration time.Duration) bool {
	return !cursor.Add(duration).After(gapEnd)
}

// assignTasks assigns tasks to focus blocks in priority order.
func (p *Planner) assignTasks(blocks []TimeBlock, tasks []TaskEstimate) []TimeBlock {
	if len(tasks) == 0 {
		return blocks
	}

	focusIdx := 0
	for _, task := range tasks {
		pomos := task.EffectivePomos()
		assigned := 0
		for assigned < pomos && focusIdx < len(blocks) {
			if blocks[focusIdx].Type == BlockFocus {
				id := task.TodoID
				blocks[focusIdx].TaskID = &id
				blocks[focusIdx].TaskName = task.Title
				assigned++
			}
			focusIdx++
		}
	}
	return blocks
}
