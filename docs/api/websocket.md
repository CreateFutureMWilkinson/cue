# Cue WebSocket Reference

Cue exposes a single bi-directional WebSocket channel for real-time activity
updates. The REST surface is documented separately in
[`openapi.yaml`](./openapi.yaml).

| Property | Value |
|----------|-------|
| URL      | `ws[s]://<host>/api/v1/websocket/events` |
| Protocol | RFC 6455 WebSocket |
| Subprotocols | none |
| Direction | server → client (clients should not send application messages) |
| Max concurrent connections | 16 (the 17th upgrade attempt receives `503`) |
| Heartbeat | server-initiated ping every 30 s, expects pong within 10 s |

---

## Connection

```
GET /api/v1/websocket/events HTTP/1.1
Host: cue.local:8080
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Key: ...
Sec-WebSocket-Version: 13
```

### Authentication

If the server is started with auth enabled (`server.auth_enabled = true`),
the upgrade handshake validates a paired bearer token supplied as a query
parameter:

```
GET /api/v1/websocket/events?token=<bearer-token> HTTP/1.1
```

Tokens are obtained via the pairing flow (`POST /api/v1/auth/pair` →
approve on the desktop UI → `GET /api/v1/auth/pair/{id}` returns the
plaintext token). Invalid or revoked tokens cause the upgrade to fail with
`401 Unauthorized` before the WebSocket handshake completes.

When auth is disabled, the `token` parameter is ignored and any client may
connect.

### Disconnection

Either side may close the connection at any time. The server closes
connections during shutdown (closing all hijacked sockets is sequenced
before `http.Server.Shutdown`). Clients should reconnect with backoff;
the SDK at `pkg/client` implements exponential backoff with jitter.

---

## Resume by sequence

Every envelope-wrapped event carries a monotonically increasing `seq`
field (assigned at publish time, not connection time). The server retains
the most recent **500** envelopes in an in-memory ring buffer.

After reconnecting, clients replay missed events via the REST endpoint:

```
GET /api/v1/events?since=<last_seq>
```

The response is a JSON object:

```json
{
  "events":      [{"seq": 4321, "type": "activity", ...}, ...],
  "truncated":   false,
  "oldest_seq":  4001,
  "latest_seq":  4500
}
```

`truncated: true` indicates that some events were evicted from the ring
before the client could replay them — the gap between the client's last
seq and `oldest_seq` is unrecoverable.

The next dispatched envelope on a freshly connected socket may include a
`dropped_since_last` field, indicating how many envelopes that subscriber
missed because its outbound channel was full. This is per-subscriber slow
consumer protection and is independent of `truncated`.

---

## Envelope schema

Most events on the channel use the `ActivityEnvelope` shape:

```json
{
  "seq":                 1234,
  "type":                "activity",
  "timestamp":           "2026-04-25T08:14:22.918Z",
  "data":                { /* type-specific payload */ },
  "dropped_since_last":  0
}
```

| Field | Type | Notes |
|-------|------|-------|
| `seq` | uint64 | Monotonic sequence; use with `/api/v1/events?since=` to replay. |
| `type` | string | One of `activity`, `alert`, `timer_tick`, `timer_block_complete`. |
| `timestamp` | RFC 3339 | UTC. Set by the server at publish time. |
| `data` | object | See per-type sections below. |
| `dropped_since_last` | int | Omitted when zero. Counts subscriber-side drops. |

Pairing events (`pairing_request`, `pairing_resolved`) are broadcast as
raw bytes outside the envelope and use the simpler shape documented in
their own sections below.

---

## Event reference

### `activity`

General-purpose activity log entry — orchestrator, watcher, scoring, or
buffer events surfaced for the UI activity drawer.

```json
{
  "seq": 91,
  "type": "activity",
  "timestamp": "2026-04-25T08:14:22.918Z",
  "data": {
    "source":   "Slack",
    "message":  "5 new messages from #incidents",
    "is_error": false
  }
}
```

| Field | Type | Notes |
|-------|------|-------|
| `data.source` | string | Free-text origin label (`Slack`, `Email`, `Router`, `Buffer`, ...). |
| `data.message` | string | Human-readable description. |
| `data.is_error` | bool | `true` for failure conditions (red entry in UI). |

### `alert`

System notification cue — currently used to trigger the desktop audio
alerter when a new high-priority message lands.

```json
{
  "seq": 92,
  "type": "alert",
  "timestamp": "2026-04-25T08:14:23.040Z",
  "data": { "kind": "notification" }
}
```

| Field | Type | Notes |
|-------|------|-------|
| `data.kind` | string | Alert classifier; `notification` is the only kind currently emitted. |

### `timer_tick`

Per-second update from the focus-block ticker while a Pomodoro session is
running. Equivalent to a live `GET /api/v1/timer` snapshot.

```json
{
  "seq": 93,
  "type": "timer_tick",
  "timestamp": "2026-04-25T08:14:24.000Z",
  "data": {
    "running":            true,
    "block_type":         "focus",
    "task_name":          "Write Phase 9 docs",
    "elapsed_seconds":    742,
    "remaining_seconds":  758,
    "display_time":       "12:38",
    "elapsed_fraction":   0.4944
  }
}
```

When no block is active the ticker still emits `running: false` ticks so
clients can clear stale UI state.

### `timer_block_complete`

Emitted once when a focus or break block ends. Useful for surfacing
"block finished" toasts or chimes.

```json
{
  "seq": 94,
  "type": "timer_block_complete",
  "timestamp": "2026-04-25T08:39:00.000Z",
  "data": {
    "completed_block":  "focus",
    "task_name":        "Write Phase 9 docs",
    "next_block":       "short_break"
  }
}
```

`next_block` is empty string when the schedule has no following block.

### `pairing_request`

Broadcast when a new device calls `POST /api/v1/auth/pair`. Already-paired
clients use this to display an approve/deny prompt.

```json
{
  "event": "pairing_request",
  "data": {
    "request_id": "8f4c91d4-9c33-4a8c-8aaa-3dd2e91b2d7c",
    "label":      "MacBook Air",
    "code":       "742193"
  }
}
```

| Field | Type | Notes |
|-------|------|-------|
| `data.request_id` | uuid | Pass to `/api/v1/auth/pair/{id}/approve` or `.../deny`. |
| `data.label` | string | User-supplied device label; may be empty. |
| `data.code` | string | 6-digit pairing code shown to the user. |

This event is broadcast as raw bytes and is NOT wrapped in the
`ActivityEnvelope`; it has no `seq`, `type`, or `timestamp` field. It is
also not retained in the ring buffer, so it cannot be replayed via
`/api/v1/events`.

### `pairing_resolved`

Broadcast when an outstanding pairing request is approved or denied.

```json
{
  "event": "pairing_resolved",
  "data": {
    "request_id": "8f4c91d4-9c33-4a8c-8aaa-3dd2e91b2d7c",
    "status":     "approved"
  }
}
```

`status` is `approved` or `denied`. Like `pairing_request`, this event is
broadcast outside the `ActivityEnvelope` and is not replayable.

---

## Backpressure and slow consumers

Each subscriber has a bounded outbound channel. If a client is slow to
read and the channel fills, the server drops the oldest queued envelope
for that subscriber (not for others) and increments a per-subscriber
`droppedSinceLast` counter. The next successfully delivered envelope to
that subscriber carries the `dropped_since_last` field set to the drop
count, after which the counter is reset.

Clients that observe `dropped_since_last > 0` should call
`GET /api/v1/events?since=<last_seq>` to replay the missing range from
the ring buffer.

---

## Reference implementation

A pure-Go reference client is available at `pkg/client`. It implements
reconnection with exponential backoff, automatic resume by `seq`, and
typed callbacks for each event listed above.
