package server_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/server"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

type mockScheduleLoader struct {
	mu       sync.Mutex
	schedule *repository.Schedule
	loadErr  error
}

func (m *mockScheduleLoader) LoadByDate(_ context.Context, _ time.Time) (*repository.Schedule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.schedule, m.loadErr
}

type mockTickerClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *mockTickerClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *mockTickerClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// makeSchedule creates a repository.Schedule with the given blocks for today.
func makeSchedule(blocks []repository.ScheduleBlock) *repository.Schedule {
	return &repository.Schedule{
		ID:        uuid.New(),
		Date:      time.Now().Truncate(24 * time.Hour),
		Strategy:  "focus-maximized",
		Blocks:    blocks,
		CreatedAt: time.Now(),
	}
}

// readTickEvent reads a timer_tick event from the subscriber channel within timeout.
// Returns the parsed TimerTickData and true, or zero value and false on timeout.
func readTickEvent(sub *server.Subscriber, timeout time.Duration) (server.TimerTickData, bool) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case raw := <-sub.Events:
			var env server.ActivityEnvelope
			if err := json.Unmarshal(raw, &env); err != nil {
				continue
			}
			if env.Type != "timer_tick" {
				continue
			}
			dataBytes, err := json.Marshal(env.Data)
			if err != nil {
				continue
			}
			var tick server.TimerTickData
			if err := json.Unmarshal(dataBytes, &tick); err != nil {
				continue
			}
			return tick, true
		case <-timer.C:
			return server.TimerTickData{}, false
		}
	}
}

// readBlockCompleteEvent reads a timer_block_complete event from the subscriber
// channel within timeout.
func readBlockCompleteEvent(sub *server.Subscriber, timeout time.Duration) (server.TimerBlockCompleteData, bool) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case raw := <-sub.Events:
			var env server.ActivityEnvelope
			if err := json.Unmarshal(raw, &env); err != nil {
				continue
			}
			if env.Type != "timer_block_complete" {
				continue
			}
			dataBytes, err := json.Marshal(env.Data)
			if err != nil {
				continue
			}
			var bc server.TimerBlockCompleteData
			if err := json.Unmarshal(dataBytes, &bc); err != nil {
				continue
			}
			return bc, true
		case <-timer.C:
			return server.TimerBlockCompleteData{}, false
		}
	}
}

// drainEvents reads all events from a subscriber until timeout with no new events.
func drainEvents(sub *server.Subscriber, timeout time.Duration) []server.ActivityEnvelope {
	var events []server.ActivityEnvelope
	for {
		timer := time.NewTimer(timeout)
		select {
		case raw := <-sub.Events:
			timer.Stop()
			var env server.ActivityEnvelope
			if err := json.Unmarshal(raw, &env); err != nil {
				continue
			}
			events = append(events, env)
		case <-timer.C:
			return events
		}
	}
}

// ---------------------------------------------------------------------------
// Suite
// ---------------------------------------------------------------------------

type TickerSuite struct {
	suite.Suite
}

func TestTicker(t *testing.T) {
	suite.Run(t, new(TickerSuite))
}

// ---------------------------------------------------------------------------
// Behavior 11: Broadcasts ticks at configured interval
// ---------------------------------------------------------------------------

func (s *TickerSuite) TestBroadcastsTicksWhileScheduleActive() {
	now := time.Date(2026, 4, 22, 9, 0, 0, 0, time.UTC)
	clock := &mockTickerClock{now: now}

	schedule := makeSchedule([]repository.ScheduleBlock{
		{
			Start:    now,
			End:      now.Add(25 * time.Minute),
			Type:     repository.ScheduleBlockFocus,
			TaskName: "Deep work",
		},
	})

	loader := &mockScheduleLoader{schedule: schedule}
	hub := server.NewHub()
	sub, err := hub.Subscribe("test-tick")
	s.Require().NoError(err)

	ticker := server.NewTicker(loader, hub, clock, "09:00", "17:00")
	// Use a fast tick interval for testing.
	ticker.TickInterval = 10 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ticker.Start(ctx)
	defer ticker.Stop()

	tick, ok := readTickEvent(sub, 500*time.Millisecond)
	s.Require().True(ok, "expected at least one timer_tick event")
	s.True(tick.Running, "tick should indicate timer is running")
	s.Equal("focus", tick.BlockType)
	s.Equal("Deep work", tick.TaskName)
}

// ---------------------------------------------------------------------------
// Behavior 12: Detects block transition, broadcasts block_complete
// ---------------------------------------------------------------------------

func (s *TickerSuite) TestBroadcastsBlockCompleteOnTransition() {
	base := time.Date(2026, 4, 22, 9, 0, 0, 0, time.UTC)
	// Start near the end of the first block so the transition happens quickly.
	now := base.Add(24*time.Minute + 55*time.Second)
	clock := &mockTickerClock{now: now}

	schedule := makeSchedule([]repository.ScheduleBlock{
		{
			Start:    base,
			End:      base.Add(25 * time.Minute),
			Type:     repository.ScheduleBlockFocus,
			TaskName: "Deep work",
		},
		{
			Start:    base.Add(25 * time.Minute),
			End:      base.Add(30 * time.Minute),
			Type:     repository.ScheduleBlockShortBreak,
			TaskName: "",
		},
	})

	loader := &mockScheduleLoader{schedule: schedule}
	hub := server.NewHub()
	sub, err := hub.Subscribe("test-block-complete")
	s.Require().NoError(err)

	ticker := server.NewTicker(loader, hub, clock, "09:00", "17:00")
	ticker.TickInterval = 10 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ticker.Start(ctx)
	defer ticker.Stop()

	// Advance clock past the block boundary after a brief delay to let
	// the ticker goroutine start.
	time.Sleep(30 * time.Millisecond)
	clock.Advance(10 * time.Second) // now at 09:25:05 — in the short_break block

	bc, ok := readBlockCompleteEvent(sub, 500*time.Millisecond)
	s.Require().True(ok, "expected a timer_block_complete event on block transition")
	s.Equal("focus", bc.CompletedBlock)
	s.Equal("short_break", bc.NextBlock)
}

// ---------------------------------------------------------------------------
// Behavior 13: Stops after last block ends
// ---------------------------------------------------------------------------

func (s *TickerSuite) TestStopsAfterLastBlock() {
	base := time.Date(2026, 4, 22, 9, 0, 0, 0, time.UTC)
	now := base.Add(24*time.Minute + 55*time.Second)
	clock := &mockTickerClock{now: now}

	schedule := makeSchedule([]repository.ScheduleBlock{
		{
			Start:    base,
			End:      base.Add(25 * time.Minute),
			Type:     repository.ScheduleBlockFocus,
			TaskName: "Only block",
		},
	})

	loader := &mockScheduleLoader{schedule: schedule}
	hub := server.NewHub()
	sub, err := hub.Subscribe("test-stop-after-last")
	s.Require().NoError(err)

	ticker := server.NewTicker(loader, hub, clock, "09:00", "17:00")
	ticker.TickInterval = 10 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ticker.Start(ctx)
	defer ticker.Stop()

	// Let the ticker emit at least one tick while still in the block.
	tick, ok := readTickEvent(sub, 500*time.Millisecond)
	s.Require().True(ok, "expected at least one tick while block is active")
	s.True(tick.Running)

	// Advance clock past the end of the only block.
	clock.Advance(10 * time.Second) // now at 09:25:05 — past all blocks

	// Drain remaining events — we expect to see a tick with Running=false
	// (the "stopped" indication) and then no further events.
	events := drainEvents(sub, 200*time.Millisecond)

	// Among the drained events, at least one timer_tick should show Running=false.
	foundStopped := false
	for _, env := range events {
		if env.Type == "timer_tick" {
			dataBytes, err := json.Marshal(env.Data)
			if err != nil {
				continue
			}
			var td server.TimerTickData
			if err := json.Unmarshal(dataBytes, &td); err != nil {
				continue
			}
			if !td.Running {
				foundStopped = true
				break
			}
		}
	}
	s.True(foundStopped, "expected a timer_tick with Running=false after schedule ends")
}

// ---------------------------------------------------------------------------
// Behavior 14: Stops on context cancellation
// ---------------------------------------------------------------------------

func (s *TickerSuite) TestStopsOnContextCancellation() {
	now := time.Date(2026, 4, 22, 9, 0, 0, 0, time.UTC)
	clock := &mockTickerClock{now: now}

	schedule := makeSchedule([]repository.ScheduleBlock{
		{
			Start:    now,
			End:      now.Add(25 * time.Minute),
			Type:     repository.ScheduleBlockFocus,
			TaskName: "Cancelable",
		},
	})

	loader := &mockScheduleLoader{schedule: schedule}
	hub := server.NewHub()
	sub, err := hub.Subscribe("test-cancel")
	s.Require().NoError(err)

	ticker := server.NewTicker(loader, hub, clock, "09:00", "17:00")
	ticker.TickInterval = 10 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())

	ticker.Start(ctx)

	// Wait for at least one tick to confirm it is running.
	_, ok := readTickEvent(sub, 500*time.Millisecond)
	s.Require().True(ok, "expected at least one tick before cancellation")

	// Cancel the context.
	cancel()

	// Give the goroutine time to notice cancellation.
	time.Sleep(50 * time.Millisecond)

	// Drain whatever is left; no new events should appear after a brief wait.
	_ = drainEvents(sub, 100*time.Millisecond)

	// Verify no further events arrive after cancellation has settled.
	_, gotMore := readTickEvent(sub, 100*time.Millisecond)
	s.False(gotMore, "expected no further ticks after context cancellation")
}
