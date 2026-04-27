# Feature 099: Activity Event Stream

**Phase:** Phase-9-Feature-099
**Status:** Done
**Package:** `internal/server/`, `internal/server/handler/`
**Depends on:** Feature 097 (Server Infrastructure), Feature 098 (Message & Notification API)
**Blocks:** Feature 099A (Server Orchestrator Wiring), Feature 107 (Fyne Client Re-wire)

---

## Overview

Expose an activity event stream over WebSocket so alternative UIs can display real-time updates (watcher activity, errors, orchestrator ticks) without polling. Establishes the broadcast pattern reused by later features (timer ticks, notification push).

**Scope boundary:** This feature ships only the *protocol and hub*: WebSocket upgrade handler, JSON envelope, in-memory ring buffer for replay, per-subscriber drop counter, heartbeat, connection cap. It does **not** connect a publisher — no events flow through the hub yet. The publisher wiring (orchestrator → hub) is owned by **Feature 099A**, which must also relocate the orchestrator from `cmd/cue` into `cue-server`.

## Architecture

### Current Event Flow (GUI)

```
Orchestrator.PollOnce()
  → orchEventCh (chan ActivityEvent, buffer 100)
    → bridgeEvents() fan-out
      → presenterEventCh → ActivityPresenter → Fyne UI
      → charPresenterEventCh → CharacterPresenter → Fyne UI
```

The orchestrator currently lives in `cmd/cue/main.go`. `cmd/cue-server/main.go` has a hub but no orchestrator. Relocation is out of scope for 099 — see Feature 099A.

### Target Event Flow (after 099 + 099A)

```
Orchestrator.PollOnce()              [lives in cue-server after 099A]
  → hub.Publish(ActivityEvent)       [hub.Publish added in 099]
    → assigns seq + timestamp
    → stores in 500-event ring
    → broadcasts to all subscribers
      → WebSocket clients receive JSON envelopes
```

## Locked Design Decisions

### Scope

Protocol + hub + replay + connection cap. No publisher wiring. No events flow until 099A lands. The hub exposes `Publish(event)` so 099A only needs to call it.

### WebSocket Library

**`github.com/coder/websocket`** — maintained successor to `nhooyr.io/websocket` (same author, same API). Pure Go, context-first, built-in ping/pong.

### Paths

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/api/v1/websocket/events` | WebSocket upgrade |
| `GET` | `/api/v1/events?since=<seq>` | REST replay of recent events |

Both versioned under `/api/v1`. WebSocket grouped under `/websocket` sub-namespace; spelled out (not `/ws`) to match the package naming.

### Event Envelope

Every broadcast is a single JSON object:

```json
{
  "seq": 42,
  "type": "activity",
  "timestamp": "2026-04-15T14:30:00Z",
  "data": {
    "source": "slack",
    "message": "Polled 5 messages from #general",
    "is_error": false
  }
}
```

- `seq` is a monotonic counter assigned by the hub, starting at 1 on server startup.
- `type` is `"activity"` for now; future event kinds (notification push, timer tick) will multiplex over the same connection via additional `type` values.
- `timestamp` is RFC 3339 UTC, stamped by the hub at publish time.
- `data` is the type-specific payload. For `"activity"` it mirrors `orchestrator.ActivityEvent`.
- `dropped_since_last` (optional int) — see "Slow-Client Policy" below. Omitted when zero.

Clients resume with `?since=<seq>`. Seq is opaque to clients; a server restart resets it to 1 and clients must treat the stream as fresh (seq numbers aren't persistent across server runs).

### Replay: In-Memory Ring Buffer

Hub retains the **last 500 envelopes** in a fixed-size ring buffer. At typical event rates (<1/s in normal operation) this covers ~8+ minutes of history — enough for transient disconnects (network change, laptop wake, brief server restart).

Events before the current server run are **not** recoverable. This matches the orchestrator's behaviour (no backfill on restart) and keeps the feature free of schema/retention complexity.

### Replay Endpoint: `GET /api/v1/events?since=<seq>`

**Success (200):**

```json
{
  "events": [
    {"seq": 43, "type": "activity", "timestamp": "...", "data": {...}},
    {"seq": 44, "type": "activity", "timestamp": "...", "data": {...}}
  ],
  "truncated": false,
  "oldest_seq": 10,
  "latest_seq": 44
}
```

**Truncation** — when `since` is older than the ring's `oldest_seq`, the response starts at `oldest_seq` and sets `truncated: true`. Clients know earlier events are permanently lost.

```json
{
  "events": [/* from oldest_seq onward */],
  "truncated": true,
  "oldest_seq": 10,
  "latest_seq": 44
}
```

**Empty buffer / future since** — when `since >= latest_seq`, returns `events: []`, `truncated: false`, and whatever `oldest_seq`/`latest_seq` currently hold (zero if no events yet).

**400 Bad Request** — missing `since`, non-numeric, or negative. Body: `{"error": "invalid since parameter"}`.

### Slow-Client Policy

- Per-subscriber buffered channel, depth **64**.
- On overflow: drop the oldest queued envelope, increment the subscriber's `dropped_since_last` counter.
- On the next successful delivery: the envelope is re-serialized with `dropped_since_last: N` attached, counter resets to zero.
- Shared JSON serialization is used for the common (no-drops) path; per-subscriber re-serialization happens only when that subscriber has outstanding drops.

Clients can detect the same condition themselves via seq gaps; `dropped_since_last` is an explicit signal to simplify client logic.

### Heartbeat

Protocol-level **ping every 30s**, **pong timeout 10s**. Handled by `github.com/coder/websocket` natively. No application-level heartbeat.

If no pong arrives within the timeout, the hub closes the connection and deregisters the subscriber. Clients reconnect and use `?since=<last-seq>` to catch up.

### Connection Cap

**Hardcoded maximum of 16 concurrent WebSocket connections.** Additional upgrade attempts receive **HTTP 503 Service Unavailable** with a `Retry-After: 5` header. No TOML knob — can be added later if real deployments hit the cap.

This is a single-user local-first app. 16 covers CLI + GUI + browser tab + mobile app + a few spare slots.

### Origin Policy

**Same-origin only, with missing-Origin allowed.** This is the `coder/websocket` default (`OriginPatterns` unset, `InsecureSkipVerify` false).

- Native mobile apps typically send no `Origin` header → accepted.
- Browser-based UIs served from the Cue server itself → `Origin` matches `Host` → accepted.
- Cross-origin browser UIs (served from a different host) → rejected.

If cross-origin deployments become a real need, add a TOML allowlist in a later hotfix.

## Package Layout

| File | Purpose | New? |
|---|---|---|
| `internal/server/envelope.go` | `ActivityEnvelope` type + JSON tags | new |
| `internal/server/hub.go` | Extend existing Hub with `Publish`, `History`, seq counter, ring buffer, per-sub drop counter | modify |
| `internal/server/handler/websocket.go` | WebSocket upgrade handler | new |
| `internal/server/handler/events.go` | REST replay handler | new |
| `internal/server/server.go` | Register `/api/v1/websocket/events` and `/api/v1/events` routes | modify |

The existing Hub's `Broadcast([]byte)` / `Subscribe` / `Unsubscribe` / `SubscriberCount` API stays. `Broadcast` becomes an internal detail used by `Publish`; callers external to the server package use `Publish(event)`.

## API Surface (Go)

```go
// internal/server/envelope.go
package server

type ActivityEnvelope struct {
    Seq              uint64    `json:"seq"`
    Type             string    `json:"type"`       // "activity"
    Timestamp        time.Time `json:"timestamp"`
    Data             any       `json:"data"`
    DroppedSinceLast int       `json:"dropped_since_last,omitempty"`
}

type ActivityData struct {
    Source  string `json:"source"`
    Message string `json:"message"`
    IsError bool   `json:"is_error"`
}

// internal/server/hub.go (additions)
func (h *Hub) Publish(data ActivityData) ActivityEnvelope
func (h *Hub) History(sinceSeq uint64) HistoryResponse

type HistoryResponse struct {
    Events     []ActivityEnvelope `json:"events"`
    Truncated  bool               `json:"truncated"`
    OldestSeq  uint64             `json:"oldest_seq"`
    LatestSeq  uint64             `json:"latest_seq"`
}
```

## Behaviors to TDD

Each behavior is a full RED → GREEN → REFACTOR micro-loop with its own commits. Run `just fmt` before each commit.

| # | Behavior |
|---|---|
| 1 | `ActivityEnvelope` type marshals to stable JSON (seq, type, timestamp, data, optional dropped_since_last) |
| 2 | `Hub.Publish` assigns monotonic seq starting at 1, stamps UTC timestamp, retains envelope in 500-event ring |
| 3 | `Hub.History(since)` semantics: returns events with `seq > since`, signals truncation when `since < oldest_seq`, handles empty and future `since` cleanly |
| 4 | `Hub.Publish` broadcasts serialized envelope to all subscribers via 64-deep per-subscriber buffer |
| 5 | Slow subscriber: on overflow drop oldest and track per-sub counter; next successful delivery re-serializes with `dropped_since_last: N` and resets counter |
| 6 | WebSocket upgrade handler (happy path): upgrade, subscribe, receive envelope, clean disconnect (no goroutine leak) |
| 7 | Upgrade handler origin policy: accept same-origin, accept missing-Origin, reject cross-origin |
| 8 | Upgrade handler connection cap: rejects upgrade attempt 17 with HTTP 503 + `Retry-After: 5` |
| 9 | Upgrade handler heartbeat: sends protocol ping every interval, closes connection on pong timeout (test uses short interval) |
| 10 | Upgrade handler graceful shutdown: when server shuts down, all live connections receive a close frame |
| 11 | REST replay handler: wrapped response shape, truncation flag, 400 on missing/invalid `since` |
| 12 | Route registration in `server.go`: `/api/v1/websocket/events` and `/api/v1/events` mounted and reachable |

## Testing Considerations

- **Hub concurrency:** multiple simultaneous subscribers, verify all receive events in order.
- **Slow subscriber:** one blocked consumer must not stall the hub or other subscribers.
- **Disconnect cleanliness:** subscriber removal completes; no leaked goroutines or channel writers.
- **Reconnection:** client disconnects, reconnects with `?since=<seq>`, resumes without gaps.
- **Shutdown:** `Server.Shutdown(ctx)` closes all live WebSocket connections with a close frame before returning.
- **Heartbeat testing:** inject a configurable interval so tests run in ~1s rather than real 30s.
- **Origin tests:** exercise same-origin, missing Origin, and cross-origin explicitly.

## Error Handling

| Condition | HTTP Status | Response |
|---|---|---|
| Upgrade: cross-origin | 403 | library default |
| Upgrade: 16-connection cap reached | 503 | `{"error": "too many connections"}` + `Retry-After: 5` |
| Upgrade: bad request | 400 | library default |
| Replay: missing `since` | 400 | `{"error": "invalid since parameter"}` |
| Replay: non-numeric `since` | 400 | `{"error": "invalid since parameter"}` |
| Replay: negative `since` | 400 | `{"error": "invalid since parameter"}` |

## Integration Points

- `internal/config/` — no changes. Connection cap, ring size, heartbeat interval, and buffer depth are all hardcoded constants.
- `internal/server/server.go` — register two new routes; inject hub into both handlers.
- `cmd/cue/main.go` + `cmd/cue-server/main.go` — **no changes**. Publisher wiring is Feature 099A.

## Out of Scope (Owned by 099A)

- Relocating orchestrator / watchers / queue processor / ollama / buffer from `cmd/cue` into `cue-server`.
- Calling `hub.Publish(...)` from the event fan-out.
- End-to-end integration tests that require a live publisher.

## Future Extensions

- Additional event types multiplexed over the same WebSocket (notification push in Feature 098.x, timer tick in Feature 104).
- Configurable connection cap / ring size via TOML (likely in a small hotfix once real deployments exist).
- Cross-origin allowlist for browser clients hosted separately.
- SQLite-backed replay for gap recovery across server restarts (only if clients report problems).

## Implementation Summary

Shipped as 12 behaviors on `develop` between 2026-04-15 and 2026-04-16. All TDD micro-loops used the go-test-designer / go-implementer / go-refactorer agent team.

### Files Added / Modified

| File | Purpose |
|---|---|
| `internal/server/envelope.go` | `ActivityEnvelope` + `ActivityData` with canonical JSON tags |
| `internal/server/hub.go` | `Publish` (seq + timestamp + ring), `History`, fanout with drop-oldest + per-sub drop counter, 64-deep buffer |
| `internal/server/publisher.go` | `hubPublisher` adapter implementing both `handler.Publisher` and `handler.HistoryProvider` |
| `internal/server/server.go` | `wsManager` field; routes `/api/v1/websocket/events` and `/api/v1/events`; `Shutdown` closes live WS conns |
| `internal/server/middleware.go` | `Unwrap()` on `statusWriter` so `http.NewResponseController` can find the underlying `Hijacker` |
| `internal/server/handler/websocket.go` | `Manager` (conn registry + `CloseAll`), `WebSocketHandler`, `WebSocketHandlerWithHeartbeat`, 16-connection cap, same-origin-only |
| `internal/server/handler/events.go` | `EventsHandler` (REST replay), `HistoryProvider` interface |
| `go.mod` | Added `github.com/coder/websocket v1.8.14` |

### Design Notes Confirmed During Implementation

- **Import-cycle wall:** `handler` cannot import `server` because `server` imports `handler`. Resolved by defining handler-local mirrors (`handler.Subscription`, `handler.HistoryJSON`) and placing the adapter in the `server` package (`publisher.go`).
- **Hijacked-connection shutdown:** `net/http.Server.Shutdown` does not close upgraded connections. Added `handler.Manager.CloseAll` and called it before `server.Shutdown` on the inner `*http.Server`.
- **Middleware + Hijacker:** The `statusWriter` middleware wrapper needed `Unwrap() http.ResponseWriter` so the `coder/websocket` upgrade could walk up to the underlying `Hijacker`; otherwise the upgrade returned 501.
- **Close vs CloseNow in heartbeat:** `conn.Close` tries to acquire the read lock held by `CloseRead`, causing a 5s stall. The heartbeat goroutine uses `CloseNow` on ping-timeout.
- **gosec G115:** The ring-buffer index math does `uint64 ↔ int` conversions bounded by `ringCapacity = 500`; suppressed with `#nosec G115` and a justification.

### TDD Agent Stats

See `docs/agent-log.md` under `Phase-9-Feature-099`.
