# Feature 104: Timer API

**Phase:** Phase-9-Feature-104
**Status:** Planning
**Package:** `internal/server/handler/`, `internal/server/ws/`

---

## Overview

Expose the pomodoro timer over REST (control) and WebSocket (real-time state). The timer runs server-side — the server is the source of truth for elapsed time, block transitions, and alerts. Clients display the timer state but don't run their own clocks.

## REST Endpoints

### Get Timer Status

```
GET /api/v1/timer
```

**Response (running):**
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

**Response (stopped):**
```json
{
  "running": false
}
```

### Start Timer

```
POST /api/v1/timer/start
```

**Request:**
```json
{
  "block_type": "focus",
  "duration_seconds": 2700,
  "task_name": "Review PR #42"
}
```

Normally called automatically when the planner advances to a new block. Manual start is for standalone timer use outside the planner.

Returns 409 if timer is already running.

### Stop Timer

```
POST /api/v1/timer/stop
```

Stops the running timer. Idempotent — returns 200 even if already stopped.

### Skip Block

```
POST /api/v1/timer/skip
```

Stops the current timer and signals the planner to advance to the next block (if a plan is active). Returns 200 with the new timer state.

## WebSocket Events

Timer ticks are broadcast on the shared WebSocket connection (Feature 099):

```json
{
  "type": "timer_tick",
  "timestamp": "2026-04-10T14:30:01Z",
  "data": {
    "running": true,
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

## Design Decisions to Make

### Server-Side Tick Loop

The GUI timer uses `TimerPresenter.Tick()` called at 1Hz by Fyne's animation loop. The server needs its own tick source.

**Question: How should the server drive the tick loop?**

- **`time.Ticker` goroutine**: Simple 1Hz ticker that calls `TimerPresenter.Tick()` (or equivalent) and broadcasts state. Straightforward.
- **Computed on demand**: No tick loop. Timer stores start time and duration. State is computed on `GET /timer` and on WebSocket heartbeat. Reduces CPU load to zero when no clients are connected.

**Recommendation:** Computed on demand for REST. Ticker goroutine only when WebSocket clients are subscribed to timer events. No clients = no ticker = no CPU waste.

### Audio Alerts in Headless Mode

The GUI plays audio alerts when a block completes (`TimerAlertService.PlayBlockComplete()`). In headless mode there's no audio device (or the server might be running on a remote machine).

**Question: Should the server play audio alerts?**

- **Yes**: If running on the same machine as the user, audio still works. Keep `beeep` alerts.
- **No**: Headless means no audio. Clients handle their own alerts based on `timer_block_complete` events.
- **Configurable**: `[server] audio_enabled = false` (default off for headless).

**Recommendation:** Configurable, default off. A desktop user running cue-server locally might want audio. A remote server definitely doesn't. Let config decide.

### Timer Without Planner

**Question: Should the timer work standalone (not tied to an active plan)?**

The GUI timer is tightly coupled to the planner — it starts when a plan block begins. But a standalone timer ("I want a 25-minute focus block right now") is useful.

**Recommendation:** Support both. The `POST /timer/start` endpoint accepts a block definition directly. The planner auto-starts the timer when advancing blocks. They're not mutually exclusive.

### Tick Frequency

1Hz matches the GUI. For a WebSocket client, 1Hz means one message per second per connected client.

**Question: Is 1Hz appropriate for the API?**

- Too slow: Timer display might feel laggy.
- Too fast: Wastes bandwidth for a countdown that changes by 1 second each tick.
- 1Hz is standard for countdown timers and matches user expectations.

**Recommendation:** 1Hz. The payload is tiny (~150 bytes). Even 10 connected clients at 1Hz is 1.5 KB/s — negligible.

## Behaviors to Implement

1. **Get timer status handler** — Compute current state from start time + duration.
2. **Start timer handler** — Initialize timer, start tick broadcaster if WS clients exist.
3. **Stop timer handler** — Stop timer, broadcast final state.
4. **Skip block handler** — Stop timer, signal planner to advance, start next block.
5. **WebSocket tick broadcaster** — 1Hz broadcast of timer state to subscribed clients.
6. **Block complete event** — Detect block completion, broadcast event, optionally play audio.
7. **Lazy tick loop** — Start/stop ticker based on WebSocket client presence.

## Questions Summary

1. Server-side tick loop always running, or only when clients are connected?
2. Audio alerts in headless mode?
3. Standalone timer without active plan?
4. Tick frequency — 1Hz, 0.5Hz, or configurable?
5. Should the timer support pause/resume, or only start/stop?
6. What happens if the server restarts while a timer is running? (Lost — acceptable?)
