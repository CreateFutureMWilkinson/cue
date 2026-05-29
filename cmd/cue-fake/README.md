# cue-fake

UI-testing harness. A drop-in fake of `cue-server` that exposes the same client-facing HTTP/WebSocket API but is backed entirely by in-memory state — no DB, no Ollama, no Slack, no IMAP, no calendar.

Point a real `cue` GUI at `cue-fake` and exercise the UI by POSTing fake events at the `/_fake/*` control surface. State resets every time the process restarts.

## Build & run

```bash
just build-fake          # produces _build/cue-fake
_build/cue-fake -addr :8765
```

On startup the harness prints the seeded state (account names, message counts) to stdout.

**Auth:** disabled. Any (or no) bearer token is accepted. Pairing endpoints are not registered.

## Pre-seeded fixtures

| Kind | Count | Examples |
|---|---|---|
| Slack accounts | 3 | `personal-workspace`, `work-acme`, `open-source-club` (disabled) |
| Email accounts | 2 | `personal-mail` (alice@example.com), `work-mail` (alice@work.example.com) |
| Calendar accounts | 2 | `personal-google`, `work-outlook` |
| Categories | 3 | `work`, `personal`, `urgent` |
| Tasks | 3 | sample todos in `work`/`personal` |
| Messages | 6 | mix of Notified / Buffered / Ignored across slack + email |
| Routing rules | 1 | `Always notify on incident channel` |

`GET /_fake/state` returns the current snapshot as JSON for inspection.

## Control endpoints (`/_fake/*`)

All endpoints accept JSON, return JSON, and do NOT require auth. Each injecting endpoint:
1. Mutates in-memory state so the next `GET /api/v1/...` reflects it.
2. Broadcasts the matching envelope on the WebSocket hub so a connected UI updates live.

### `POST /_fake/inject/slack-message`

Adds a Slack message. Importance/confidence are auto-derived: channel-join → IS=9 CS=1.0, mention → IS=8 CS=1.0, otherwise IS=5 CS=0.7 (Buffered).

```bash
curl -X POST localhost:8765/_fake/inject/slack-message \
  -H 'Content-Type: application/json' \
  -d '{
    "accountId": "work-acme",
    "channel":   "#incident-room",
    "sender":    "@oncall",
    "content":   "@alice prod DB is degraded",
    "isMention": true
  }'
```

Fields: `accountId`, `channel`, `sender`, `content`, `isMention` (bool, default false), `isChannelJoin` (bool, default false).

### `POST /_fake/inject/email`

Adds an email. Always lands as Notified (IS=7, CS=0.85).

```bash
curl -X POST localhost:8765/_fake/inject/email \
  -H 'Content-Type: application/json' \
  -d '{
    "accountId": "work-mail",
    "sender":    "boss@work.example.com",
    "subject":   "Re: weekly report",
    "body":      "please send today"
  }'
```

Fields: `accountId`, `sender`, `subject`, `body`.

### `POST /_fake/inject/buffered`

Pushes a message into the feedback buffer for review.

```bash
curl -X POST localhost:8765/_fake/inject/buffered \
  -H 'Content-Type: application/json' \
  -d '{
    "source":    "slack",
    "accountId": "open-source-club",
    "channel":   "#general",
    "sender":    "carol",
    "content":   "Could you take a look at PR #42?",
    "reason":    "uncertain importance",
    "importance": 7,
    "confidence": 0.4
  }'
```

Fields: `source` (default `slack`), `accountId`, `channel`, `sender`, `content`, `reason`, `importance` (default 7), `confidence` (default 0.5).

### `POST /_fake/inject/notification`

Force-injects a Notified message regardless of routing thresholds. Use when you want a notification card to appear immediately.

```bash
curl -X POST localhost:8765/_fake/inject/notification \
  -H 'Content-Type: application/json' \
  -d '{
    "source":    "slack",
    "accountId": "work-acme",
    "channel":   "#incident-room",
    "sender":    "alerts",
    "content":   "API error rate spiked to 12%",
    "importance": 9,
    "confidence": 0.95
  }'
```

Fields: `source` (default `slack`), `accountId`, `sender`, `channel`, `content`, `importance` (default 8), `confidence` (default 0.9).

### `POST /_fake/inject/activity`

Broadcasts an activity-log entry without creating a message. Useful for testing the activity drawer.

```bash
curl -X POST localhost:8765/_fake/inject/activity \
  -H 'Content-Type: application/json' \
  -d '{"source": "ollama", "message": "scored 3 messages", "isError": false}'
```

Fields: `source`, `message`, `isError` (bool).

### `POST /_fake/reset`

Clears all in-memory state and re-seeds fixtures.

```bash
curl -X POST localhost:8765/_fake/reset
```

### `GET /_fake/state`

Dumps the current in-memory store (messages, accounts, tasks, categories, rules) as JSON.

```bash
curl localhost:8765/_fake/state | jq
```

## Typical workflow

1. `_build/cue-fake -addr :8765 &`
2. Launch the GUI pointed at `http://localhost:8765`.
3. Verify the seeded notifications/buffer items render.
4. Fire one of the `/_fake/inject/*` endpoints; watch the UI update live over WebSocket.
5. Iterate. `POST /_fake/reset` to start clean without restarting.

## Out of scope

- No tests in this package.
- Calendar accounts exist as fixtures but no calendar event injection endpoint — the harness only verifies the UI's interaction with calendar account configuration, not event rendering.
- `ScheduleGenerator` and `CalendarFetcher` are stubbed (empty results); planner-generation flows in the UI will return no options.
- Auth/pairing flows are bypassed; the harness is not suitable for testing the pairing UX.
