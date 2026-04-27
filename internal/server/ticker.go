package server

import (
	"context"
	"sync"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/service/planner"
)

// ScheduleLoader loads a schedule for a given date.
type ScheduleLoader interface {
	LoadByDate(ctx context.Context, date time.Time) (*repository.Schedule, error)
}

// Ticker broadcasts timer events at 0.2 Hz while a schedule is active.
type Ticker struct {
	store     ScheduleLoader
	hub       *Hub
	clock     planner.Clock
	workStart string
	workEnd   string

	mu        sync.Mutex
	cancel    context.CancelFunc
	lastBlock int

	// TickInterval controls how often the ticker emits events.
	// Defaults to 5 seconds. Tests may set a smaller value.
	TickInterval time.Duration
}

// NewTicker creates a new Ticker.
func NewTicker(store ScheduleLoader, hub *Hub, clock planner.Clock, workStart, workEnd string) *Ticker {
	return &Ticker{
		store:        store,
		hub:          hub,
		clock:        clock,
		workStart:    workStart,
		workEnd:      workEnd,
		lastBlock:    -1,
		TickInterval: 5 * time.Second,
	}
}

// Start begins the ticker loop. It loads today's schedule and ticks at the configured interval.
func (t *Ticker) Start(ctx context.Context) {
	today := t.clock.Now().Truncate(24 * time.Hour)
	sched, err := t.store.LoadByDate(ctx, today)
	if err != nil || sched == nil || len(sched.Blocks) == 0 {
		return
	}

	ds := repoToPlannerSchedule(sched)

	childCtx, cancel := context.WithCancel(ctx)
	t.mu.Lock()
	t.cancel = cancel
	state := planner.ComputeTimerState(&ds, t.clock.Now())
	t.lastBlock = state.BlockIndex
	t.mu.Unlock()

	go t.tickLoop(childCtx, ds)
}

// tickLoop runs the periodic tick emission in a goroutine.
func (t *Ticker) tickLoop(ctx context.Context, schedule planner.DaySchedule) {
	ticker := time.NewTicker(t.TickInterval)
	defer ticker.Stop()

	endParsed, _ := time.Parse("15:04", t.workEnd)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := t.clock.Now()

			// Stop if the clock has passed the work window end time.
			workEndToday := time.Date(now.Year(), now.Month(), now.Day(), endParsed.Hour(), endParsed.Minute(), 0, 0, time.UTC)
			if !now.Before(workEndToday) {
				t.hub.PublishTimerTick(TimerTickData{
					Running: false,
				})
				return
			}

			state := planner.ComputeTimerState(&schedule, now)

			if !state.Running {
				t.hub.PublishTimerTick(TimerTickData{
					Running: false,
				})
				return
			}

			t.mu.Lock()
			lastBlock := t.lastBlock
			if state.BlockIndex != lastBlock {
				if lastBlock != -1 && lastBlock < len(schedule.Blocks) {
					completedType := planner.BlockTypeString(schedule.Blocks[lastBlock].Type)
					nextType := planner.BlockTypeString(schedule.Blocks[state.BlockIndex].Type)
					t.hub.PublishTimerBlockComplete(TimerBlockCompleteData{
						CompletedBlock: completedType,
						TaskName:       schedule.Blocks[lastBlock].TaskName,
						NextBlock:      nextType,
					})
				}
				t.lastBlock = state.BlockIndex
			}
			t.mu.Unlock()

			t.hub.PublishTimerTick(TimerTickData{
				Running:          true,
				BlockType:        state.BlockType,
				TaskName:         state.TaskName,
				ElapsedSeconds:   state.ElapsedSeconds,
				RemainingSeconds: state.RemainingSeconds,
				DisplayTime:      state.DisplayTime,
				ElapsedFraction:  state.ElapsedFraction,
			})
		}
	}
}

// Stop cancels the ticker loop.
func (t *Ticker) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.cancel != nil {
		t.cancel()
		t.cancel = nil
	}
}

// NotifyScheduleChanged reloads the schedule and restarts the ticker.
func (t *Ticker) NotifyScheduleChanged(ctx context.Context) {
	t.Stop()
	t.Start(ctx)
}

// repoToPlannerSchedule converts a repository.Schedule to a planner.DaySchedule.
func repoToPlannerSchedule(s *repository.Schedule) planner.DaySchedule {
	blocks := make([]planner.TimeBlock, len(s.Blocks))
	for i, b := range s.Blocks {
		blocks[i] = planner.TimeBlock{
			Start:    b.Start,
			End:      b.End,
			Type:     planner.BlockType(b.Type),
			TaskID:   b.TaskID,
			TaskName: b.TaskName,
		}
	}
	return planner.DaySchedule{
		ID:       s.ID,
		Date:     s.Date,
		Strategy: s.Strategy,
		Blocks:   blocks,
	}
}
