# Feature 099: Activity Event Stream

**Phase:** Phase-9-Feature-099
**Status:** Planning
**Package:** `internal/server/ws/`

---

## Overview

Expose the activity event stream over WebSocket so alternative UIs can display real-time updates (new messages processed, errors, watcher activity) without polling. This is the first WebSocket feature and establishes the broadcast pattern used by Features 104 (Timer) and potentially 098 (notification push).

## Architecture

### Current Event Flow (GUI)

```
Orchestrator.PollOnce()
  → orchEventCh (chan ActivityEvent, buffer 100)
    → bridgeEvents() fan-out
      → presenterEventCh → ActivityPresenter → Fyne UI
      → charPresenterEventCh → CharacterPresenter → Fyne UI
```

### Server Event Flow (Proposed)

```
Orchestrator.PollOnce()
  → orchEventCh (chan ActivityEvent, buffer 100)
    → bridgeEvents() fan-out
      → serverEventCh → WebSocket Hub → connected clients
```

The fan-out pattern already exists. The server adds one more output channel and a WebSocket hub that broadcasts to all connected clients.

## WebSocket Protocol

### Connection

```
GET /ws/events → 101 Switching Protocols
```

**Question: Should this path be under `/api/v1/` or at the root?** WebSocket paths are conventionally separate from REST paths. `/ws/events` is clear and doesn't imply REST versioning semantics. But if the API is versioned, should WebSocket be too?

**Recommendation:** `/api/v1/ws/events` — keeps everything under the versioned API namespace. If the event format changes in v2, clients on v1 still work.

### Message Format

Server sends JSON messages to connected clients:

```json
{
  "type": "activity",
  "timestamp": "2026-04-10T14:30:00Z",
  "data": {
    "source": "slack",
    "message": "Polled 5 messages from #general",
    "is_error": false
  }
}
```

The `type` field allows multiplexing different event types over the same connection in the future (e.g., `"notification_new"`, `"timer_tick"`, `"planner_step_changed"`).

**Question: Single multiplexed WebSocket or separate connections per event type?**

- **Single connection** (`/api/v1/ws/events`): Client receives all event types, filters client-side. Simpler server, fewer connections.
- **Separate connections** (`/api/v1/ws/activity`, `/api/v1/ws/timer`, etc.): Client subscribes only to what it needs. More connections but cleaner separation.
- **Single connection with subscription messages**: Client sends `{"subscribe": ["activity", "timer"]}` after connecting. Most flexible but most complex.

**Recommendation:** Single multiplexed connection for v1. Cue generates low event volume (<1 event/second in normal operation). The overhead of filtering unwanted events client-side is negligible. This keeps the server simple and avoids connection management complexity.

### Heartbeat / Keep-Alive

**Question: How to detect dead connections?**

WebSocket connections can silently die (network change, laptop sleep, client crash). Options:
- **Ping/pong frames** (WebSocket protocol level): Server sends ping every 30s, expects pong within 10s, drops connection if missing.
- **Application-level heartbeat**: Server sends `{"type": "heartbeat"}` periodically. Client must respond.
- **Both**: Protocol-level for connection health, application-level for client liveness.

**Recommendation:** Protocol-level ping/pong only. `nhooyr.io/websocket` handles this natively. Application-level heartbeat is only needed if the client needs to detect server death (the client can just reconnect on close).

## WebSocket Hub Design

The hub is a central component that:
1. Accepts new WebSocket connections (adds to subscriber set)
2. Receives events from the server event channel
3. Broadcasts events as JSON to all connected subscribers
4. Removes subscribers on disconnect or write failure
5. Handles graceful shutdown (close all connections)

**Question: What happens when a slow client can't keep up?**

If a client's write buffer fills because it's not reading fast enough, the hub has options:
- **Block**: Hub blocks until the slow client catches up. This stalls ALL clients.
- **Drop**: Hub drops messages for the slow client. Client misses events but others aren't affected.
- **Disconnect**: Hub disconnects the slow client after N dropped messages.

**Recommendation:** Per-client buffered channel (size 256). If the channel fills, drop the oldest message and increment a dropped-event counter. Include the count in the next successfully sent message so the client knows to do a full refresh.

## Behaviors to Implement

1. **WebSocket hub** — Goroutine that manages subscriber set, receives events from channel, broadcasts JSON to all subscribers.
2. **Connection handler** — HTTP upgrade to WebSocket, register with hub, read loop for close detection, deregister on disconnect.
3. **Event serialization** — Convert `ActivityEvent` to JSON wire format with type envelope.
4. **Backpressure handling** — Per-client buffered channel with drop policy for slow clients.
5. **Heartbeat** — Ping/pong at configurable interval (default 30s).
6. **Graceful shutdown** — Hub closes all connections when server shuts down, sends close frame.
7. **Composition root wiring** — Add server event channel to `bridgeEvents` fan-out, connect to hub.

## Testing Considerations

- Hub concurrency: Test with multiple simultaneous subscribers, verify all receive events.
- Slow client: Test that a blocked subscriber doesn't stall others.
- Disconnect: Test that subscriber removal is clean (no goroutine leaks).
- Reconnection: Client disconnects and reconnects — verify it starts receiving new events.
- Shutdown: Verify all connections receive close frame during graceful shutdown.

## Questions Summary

1. WebSocket path under `/api/v1/` or separate namespace?
2. Single multiplexed connection or separate per event type?
3. Subscription filtering (server-side or client-side)?
4. Heartbeat interval and timeout values?
5. Slow client policy — drop messages or disconnect?
6. Should the hub expose a REST endpoint to query "missed events since timestamp" for clients that reconnect after a gap?
7. Maximum number of concurrent WebSocket connections? (Probably unlimited for local use, but should there be a config option?)
