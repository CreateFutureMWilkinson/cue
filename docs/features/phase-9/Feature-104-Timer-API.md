# Feature 104: Timer API

**Phase:** Phase-9-Feature-104
**Status:** Complete
**Package:** `internal/service/planner/`, `internal/server/`, `internal/server/handler/`

---

## Overview

Expose the pomodoro timer as a read-only projection of the active schedule over REST and WebSocket. The timer is not an independent entity — it is derived entirely from the active day schedule. The server is the source of truth; clients display state but don't run their own clocks.

The timer has no start, stop, pause, or resume controls. It runs whenever there is an active schedule and computes its state from schedule blocks + wall clock. The only timer-specific endpoint is `GET /api/v1/timer`.

## REST Endpoint

### Get Timer Status

```
GET /api/v1/timer
```

**Response (active schedule, within a block):**
```json
{
  "running": true,
  "block": {
    "type": "focus",
    "task_name": "Review PR #42",
    "duration_seconds": 2700,
    "elapsed_seconds": 1234,
    "remaining_seconds": 1466
  },
  "display_time": "24:26",
  "elapsed_fraction": 0.457
}
```

**Response (no active schedule):**
```json
{
  "running": false
}
```

State is computed on demand from schedule start time + block durations + current clock. No tick loop needed for REST.

## WebSocket Events

Timer ticks are broadcast on the shared WebSocket connection (Feature 099) at **0.2Hz** (every 5 seconds). Clients may interpolate locally between ticks for smooth display updates.

```json
{
  "type": "timer_tick",
  "timestamp": "2026-04-10T14:30:05Z",
  "data": {
    "running": true,
    "block_type": "focus",
    "task_name": "Review PR #42",
    "elapsed_seconds": 1235,
    "remaining_seconds": 1465,
    "display_time": "24:25",
    "elapsed_fraction": 0.457
  }
}
```

```json
{
  "type": "timer_block_complete",
  "timestamp": "2026-04-10T15:15:00Z",
  "data": {
    "completed_block": "focus",
    "task_name": "Review PR #42",
    "next_block": "short_break"
  }
}
```

## Design Decisions (Resolved)

### 1. Server-Side Tick Loop

**Decision:** Ticker runs whenever there is an active schedule, not per-block and not based on client presence. The ticker is schedule-driven — it starts when a schedule becomes active and stops when the schedule ends or is abandoned.

**Rationale:** The ticker must detect block completion to broadcast `timer_block_complete` events regardless of whether clients are connected. A single goroutine doing trivial math every 5 seconds has negligible cost.

**Ticker triggers:**
- **Server startup** — if an active schedule covers `now`, compute current position within the schedule and start the ticker
- **Schedule creation** — new today-schedule created, start ticker immediately (if it covers `now`) or schedule delayed start
- **Work window boundaries** — configured work hours constrain when schedules can be active; ticker starts at window open if a schedule exists, stops at window close

**Late start / restart:** If the server starts after a schedule has begun (e.g., schedule at 9:00, server at 9:05), the ticker computes current position from persisted schedule + clock. Elapsed blocks are skipped; the ticker picks up at whichever block covers `now`. No state is lost.

### 2. Audio Alerts

**Decision:** No server-side audio. Clients handle their own alerts based on `timer_block_complete` WebSocket events.

**Rationale:** Audio is a client concern. The server may be headless or remote. Clients receive block completion events and decide locally whether to play sounds.

### 3. Timer Coupling to Schedule

**Decision:** Timer is tightly coupled to the active schedule. No standalone timer. There is no `POST /timer/start` — the only way to have a running timer is to have an active schedule.

**Rationale:** The timer is a read-only projection of schedule state. Schedule lifecycle (create, abandon, complete) controls the timer entirely. Skip-block is a schedule/planner operation, not a timer operation.

### 4. Tick Frequency

**Decision:** 0.2Hz (one tick every 5 seconds), fixed.

**Rationale:** Timer counts down in minutes — 5-second granularity is sufficient. Clients can interpolate locally for smooth display. Reduces bandwidth to ~30 bytes/sec per client. Not configurable (YAGNI).

### 5. Timer Controls

**Decision:** No start, stop, pause, or resume. Timer state is purely computed from schedule + clock.

**Rationale:** The timer has no independent lifecycle. Block transitions happen automatically within the schedule. The only user actions that affect the timer are schedule-level operations (skip block, abandon schedule) which belong to the planner API, not the timer API.

### 6. Server Restart

**Decision:** Timer state survives restart. Schedule is persisted in the DB; on startup the server recomputes position within the active schedule from persisted data + current clock.

**Rationale:** Since the timer is a projection of schedule state (which is persisted), there is no ephemeral timer state to lose.

## Behaviors Implemented

1. **Timer state computation** — `ComputeTimerState()`, `FindCurrentBlock()`, `BlockTypeString()`, `FormatDisplayTime()` in `internal/service/planner/timer.go`. Pure functions, no I/O. Edge cases: nil schedule, gaps between blocks, before/after schedule, zero-duration blocks.
2. **GET /api/v1/timer handler** — `GetTimerHandler()` in `internal/server/handler/planner.go`. Loads today's schedule, converts repo→planner types, computes state, returns JSON. Returns `{"running": false}` when no schedule or no active block.
3. **Schedule-driven ticker** — `Ticker` struct in `internal/server/ticker.go`. Goroutine ticks at configurable interval (default 5s). Detects block transitions and schedule completion.
4. **WebSocket timer_tick broadcast** — `Hub.PublishTimerTick()` in `internal/server/hub.go`. `TimerTickData` envelope type in `envelope.go`.
5. **WebSocket timer_block_complete broadcast** — `Hub.PublishTimerBlockComplete()` in `internal/server/hub.go`. `TimerBlockCompleteData` envelope type in `envelope.go`.
6. **Ticker lifecycle management** — `Start`/`Stop`/`NotifyScheduleChanged` on `Ticker`. Work window boundary enforcement. Schedule caching with reload on change. Idempotent stop.
7. **Startup schedule recovery** — Ticker computes current position from persisted schedule + clock on start. Late starts pick up at the correct block.
8. **onChange callbacks** — `PutScheduleHandler` and `DeleteScheduleHandler` accept variadic `onChange ...func()` callbacks, invoked after successful save/delete.

## Test Coverage

- 9 unit tests for pure timer computation (within block, gaps, boundaries, formatting, FindCurrentBlock)
- 3 tests for GET /api/v1/timer handler (no schedule, active, completed)
- 2 tests for Hub publish methods (timer_tick, timer_block_complete)
- 9 tests for Ticker (ticks, block transition, stop after last, context cancel, no schedule, late start, work window, schedule reload, idempotent stop)
- 4 tests for onChange callbacks (PUT success/error, DELETE success/error)

## Integration Points

- Ticker created in `server.NewComposition`, started on boot, stopped in shutdown sequence
- `GET /api/v1/timer` registered in `server.registerRoutes()`
- PUT/DELETE planner handlers wired with onChange callbacks that call `ticker.NotifyScheduleChanged()`
- `SetTicker()` on Server allows post-construction attachment (since routes are registered in constructor)
