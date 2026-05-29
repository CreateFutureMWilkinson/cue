package adapters

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/service/planner"
	"github.com/CreateFutureMWilkinson/cue/pkg/client"
)

// ScheduleAdapter satisfies repository.ScheduleRepository AND
// presenter.ScheduleGenerator on top of client.ScheduleClient.
//
// The wire format uses "HH:MM" strings for block boundaries and
// "YYYY-MM-DD" for dates; the repository uses time.Time. The adapter
// performs both translations.
//
// On a 404 from GetSchedule the adapter returns (nil, nil) so the
// presenter's "if schedule == nil { /* nothing yet */ }" path works
// without inspecting error values.
type ScheduleAdapter struct {
	client client.ScheduleClient
}

// NewScheduleAdapter wraps the given SDK schedule client.
func NewScheduleAdapter(c client.ScheduleClient) *ScheduleAdapter {
	return &ScheduleAdapter{client: c}
}

// === repository.ScheduleRepository ===

// Save persists the supplied schedule for its Date. The wire shape
// only carries Strategy + Blocks, so the adapter encodes the blocks
// as HH:MM strings and lets the server stamp the canonical CreatedAt
// on its response.
func (a *ScheduleAdapter) Save(ctx context.Context, schedule *repository.Schedule) error {
	if schedule == nil {
		return fmt.Errorf("schedule adapter: cannot save nil schedule")
	}
	dto, err := a.client.PutSchedule(ctx, schedule.Date, client.PutScheduleRequest{
		Strategy: schedule.Strategy,
		Blocks:   repoBlocksToWire(schedule.Blocks, schedule.Date),
	})
	if err != nil {
		return fmt.Errorf("put schedule for %s: %w", schedule.Date.Format("2006-01-02"), err)
	}
	if dto != nil {
		schedule.CreatedAt = parseRFC3339OrZero(dto.CreatedAt)
	}
	return nil
}

// LoadByDate returns the schedule for the given date or (nil, nil)
// when none exists yet. Other server errors are wrapped and returned.
func (a *ScheduleAdapter) LoadByDate(ctx context.Context, date time.Time) (*repository.Schedule, error) {
	dto, err := a.client.GetSchedule(ctx, date)
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.Code == client.ErrCodeNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get schedule for %s: %w", date.Format("2006-01-02"), err)
	}
	return wireScheduleToRepo(dto, date)
}

// Delete removes the schedule for the given date. Server-reported 404
// is treated as a no-op (the date had no schedule, so nothing to do).
func (a *ScheduleAdapter) Delete(ctx context.Context, date time.Time) error {
	if err := a.client.DeleteSchedule(ctx, date); err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.Code == client.ErrCodeNotFound {
			return nil
		}
		return fmt.Errorf("delete schedule for %s: %w", date.Format("2006-01-02"), err)
	}
	return nil
}

// === presenter.ScheduleGenerator ===

// GenerateSchedules requests two candidate schedules from the server
// and translates the response options into planner.DaySchedule values
// for the focus-maximized and recovery-balanced strategies.
//
// If a strategy is missing from the response the corresponding return
// pointer is nil; the presenter handles partial responses by checking
// for nil before previewing.
func (a *ScheduleAdapter) GenerateSchedules(ctx context.Context, targetDate time.Time) (*planner.DaySchedule, *planner.DaySchedule, error) {
	resp, err := a.client.GenerateSchedules(ctx, client.GenerateSchedulesRequest{
		Date: targetDate.Format("2006-01-02"),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("generate schedules for %s: %w", targetDate.Format("2006-01-02"), err)
	}
	day, err := parseDate(resp.Date, targetDate)
	if err != nil {
		return nil, nil, err
	}
	var focus, recovery *planner.DaySchedule
	for i := range resp.Options {
		opt := resp.Options[i]
		blocks, err := optionBlocksToPlanner(opt.Blocks, day)
		if err != nil {
			return nil, nil, err
		}
		ds := &planner.DaySchedule{
			ID:       uuid.New(),
			Date:     day,
			Strategy: opt.Strategy,
			Blocks:   blocks,
		}
		switch opt.Strategy {
		case "focus-maximized":
			focus = ds
		case "recovery-balanced":
			recovery = ds
		}
	}
	return focus, recovery, nil
}

// === translation helpers ===

func wireScheduleToRepo(dto *client.Schedule, fallbackDate time.Time) (*repository.Schedule, error) {
	if dto == nil {
		return nil, nil
	}
	day, err := parseDate(dto.Date, fallbackDate)
	if err != nil {
		return nil, err
	}
	blocks, err := wireBlocksToRepo(dto.Blocks, day)
	if err != nil {
		return nil, err
	}
	return &repository.Schedule{
		ID:        uuid.New(), // server does not expose an opaque ID; date is the key
		Date:      day,
		Strategy:  dto.Strategy,
		Blocks:    blocks,
		CreatedAt: parseRFC3339OrZero(dto.CreatedAt),
	}, nil
}

func wireBlocksToRepo(blocks []client.ScheduleBlock, day time.Time) ([]repository.ScheduleBlock, error) {
	out := make([]repository.ScheduleBlock, 0, len(blocks))
	for i := range blocks {
		b := blocks[i]
		start, end, err := parseBlockTimes(b.Start, b.End, day)
		if err != nil {
			return nil, err
		}
		bt, err := wireBlockType(b.Type)
		if err != nil {
			return nil, err
		}
		var taskID *uuid.UUID
		if b.TaskID != nil && *b.TaskID != "" {
			id, err := uuid.Parse(*b.TaskID)
			if err != nil {
				return nil, fmt.Errorf("parse block task_id %q: %w", *b.TaskID, err)
			}
			taskID = &id
		}
		out = append(out, repository.ScheduleBlock{
			Start:    start,
			End:      end,
			Type:     bt,
			TaskID:   taskID,
			TaskName: b.TaskName,
		})
	}
	return out, nil
}

func optionBlocksToPlanner(blocks []client.ScheduleBlock, day time.Time) ([]planner.TimeBlock, error) {
	out := make([]planner.TimeBlock, 0, len(blocks))
	for i := range blocks {
		b := blocks[i]
		start, end, err := parseBlockTimes(b.Start, b.End, day)
		if err != nil {
			return nil, err
		}
		bt, err := wirePlannerBlockType(b.Type)
		if err != nil {
			return nil, err
		}
		var taskID *uuid.UUID
		if b.TaskID != nil && *b.TaskID != "" {
			id, err := uuid.Parse(*b.TaskID)
			if err != nil {
				return nil, fmt.Errorf("parse block task_id %q: %w", *b.TaskID, err)
			}
			taskID = &id
		}
		out = append(out, planner.TimeBlock{
			Start:    start,
			End:      end,
			Type:     bt,
			TaskID:   taskID,
			TaskName: b.TaskName,
		})
	}
	return out, nil
}

func repoBlocksToWire(blocks []repository.ScheduleBlock, _ time.Time) []client.ScheduleBlock {
	out := make([]client.ScheduleBlock, 0, len(blocks))
	for i := range blocks {
		b := blocks[i]
		var taskID *string
		if b.TaskID != nil && *b.TaskID != uuid.Nil {
			s := b.TaskID.String()
			taskID = &s
		}
		out = append(out, client.ScheduleBlock{
			Start:    b.Start.Format("15:04"),
			End:      b.End.Format("15:04"),
			Type:     repoBlockTypeToWire(b.Type),
			TaskID:   taskID,
			TaskName: b.TaskName,
		})
	}
	return out
}

func parseDate(s string, fallback time.Time) (time.Time, error) {
	if s == "" {
		return time.Date(fallback.Year(), fallback.Month(), fallback.Day(), 0, 0, 0, 0, fallback.Location()), nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse schedule date %q: %w", s, err)
	}
	return t, nil
}

func parseBlockTimes(startHM, endHM string, day time.Time) (time.Time, time.Time, error) {
	start, err := parseHHMMOnDay(startHM, day)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("parse block start: %w", err)
	}
	end, err := parseHHMMOnDay(endHM, day)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("parse block end: %w", err)
	}
	return start, end, nil
}

func parseHHMMOnDay(hhmm string, day time.Time) (time.Time, error) {
	t, err := time.Parse("15:04", hhmm)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse HH:MM %q: %w", hhmm, err)
	}
	return time.Date(day.Year(), day.Month(), day.Day(), t.Hour(), t.Minute(), 0, 0, day.Location()), nil
}

func wireBlockType(s string) (repository.ScheduleBlockType, error) {
	switch s {
	case "focus":
		return repository.ScheduleBlockFocus, nil
	case "short_break":
		return repository.ScheduleBlockShortBreak, nil
	case "long_break":
		return repository.ScheduleBlockLongBreak, nil
	case "meeting":
		return repository.ScheduleBlockMeeting, nil
	default:
		return 0, fmt.Errorf("unknown schedule block type %q", s)
	}
}

func wirePlannerBlockType(s string) (planner.BlockType, error) {
	switch s {
	case "focus":
		return planner.BlockFocus, nil
	case "short_break":
		return planner.BlockShortBreak, nil
	case "long_break":
		return planner.BlockLongBreak, nil
	case "meeting":
		return planner.BlockMeeting, nil
	default:
		return 0, fmt.Errorf("unknown schedule block type %q", s)
	}
}

func repoBlockTypeToWire(t repository.ScheduleBlockType) string {
	switch t {
	case repository.ScheduleBlockFocus:
		return "focus"
	case repository.ScheduleBlockShortBreak:
		return "short_break"
	case repository.ScheduleBlockLongBreak:
		return "long_break"
	case repository.ScheduleBlockMeeting:
		return "meeting"
	default:
		return "focus"
	}
}
