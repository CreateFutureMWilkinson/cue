# Feature 107: Fyne Client Re-wire

**Phase:** Phase-9-Feature-107
**Status:** Planning
**Depends on:** Feature 106 (API Client SDK), Features 096-105 (server APIs)
**Package:** `cmd/cue/`

---

## Overview

Re-wire the Fyne GUI application (`cmd/cue/main.go`) from a monolithic binary that directly creates repositories, services, and presenters into a thin client that connects to `cue-server` and passes HTTP/WebSocket-backed adapters (from Feature 106's `internal/client/` package) to the existing presenters.

All UI code (`internal/ui/`) is preserved unchanged. The presenters already depend on interfaces — this feature swaps the concrete implementations from direct SQLite/service access to HTTP/WS adapters. The presenter constructors, view code, character animations, and acceptance tests remain untouched.

## Design Decisions

### Composition Root Rewrite

**Decision: Replace direct service wiring with API client adapter wiring.**

Current `cmd/cue/main.go` (641 lines):
```
Load config → Open DB → Create repos → Create services → Create orchestrator →
Build watchers → Create presenters(repos, services) → Create Fyne UI → Run
```

New `cmd/cue/main.go` (~250 lines):
```
Load config → Connect to cue-server → Create API client → Create adapters →
Create presenters(adapters) → Create Fyne UI → Run
```

The reduction comes from removing all repository creation (~100 lines), service creation (~80 lines), orchestrator/queue processor setup (~60 lines), watcher building (~50 lines), and event fan-out plumbing (~50 lines).

### Config Split: Client vs Server

**Decision: Separate config scopes, shared TOML file.**

Both `cmd/cue` (client) and `cmd/cue-server` read `~/.cue/config.toml`, but each ignores sections irrelevant to it.

**Client reads:**
- `[server]` — host + port to connect to (required)
- `[gui]` — window dimensions, character config
- `[notification]` — audio settings (client-side playback)
- `[planner]` — timer sound, timer volume (client-side audio)
- `[logging]` — log level, log dir

**Client ignores (server-only):**
- `[database]` — server owns the DB
- `[ollama]` — server owns inference
- `[orchestrator]` — server owns routing/polling

**Validation change:** `cmd/cue` validates that `[server]` section is configured (host + port). It does NOT validate `[database]`, `[ollama]`, or `[orchestrator]` — those are the server's responsibility.

### Server Connection on Startup

**Decision: Health check with retry, then proceed.**

On startup, `cmd/cue`:
1. Build server URL from config: `http://{host}:{port}`
2. Call `GET /api/v1/health` with a 5-second timeout
3. If healthy: proceed with adapter creation
4. If unhealthy/unreachable: show error dialog in Fyne ("Cannot connect to cue-server at host:port"), offer Retry/Quit buttons
5. No background retry — user must explicitly retry or fix the server

Rationale: The app is useless without the server. Silently retrying in the background would show an empty, confusing UI.

### WebSocket Lifecycle

**Decision: Connect after Fyne app starts, reconnect automatically.**

The WebSocket connection to `/api/v1/ws/events` is established after the Fyne window is showing (not during startup). This prevents blocking the UI on a slow WebSocket handshake.

The `ActivityAdapter` from Feature 106 handles reconnection internally. The `ActivityPresenter` and `CharacterPresenter` receive events via the same `<-chan ActivityEvent` channel — they don't know or care about the transport.

### What Gets Removed

| Removed from cmd/cue/main.go | Reason |
|---|---|
| SQLite `db.Open()` + migration | Server owns DB |
| All `repository.New*()` calls | Server owns repos |
| `ollama.NewClient()` | Server owns inference |
| `vector.NewStore()` | Server owns embeddings |
| `buffer.NewService()` | Server owns buffer |
| `orchestrator.New()` + `Start()` | Server owns orchestration |
| `queueprocessor.New()` + `Start()` | Server owns queue |
| `rulesengine.New()` | Server owns rules |
| `buildWatchersFromDB()` | Server owns watchers |
| `bridgeEvents()` + `channelActivitySource` | Replaced by WebSocket ActivitySource |
| `osFileSystem`, `httpClient` (watcher deps) | Server-side concerns |
| Event channel fan-out plumbing | Server pushes events via WebSocket |

### What Stays

| Kept in cmd/cue/main.go | Reason |
|---|---|
| Config loading (`config.Load()`) | Client needs its own config |
| Fyne app + MainWindow creation | Client-side UI |
| All presenter creation (with new adapter deps) | Unchanged constructors |
| AppBinder (planner presenter <-> views) | Unchanged wiring |
| TimerLoop (1Hz tick) | Client-side countdown |
| Character loading + creation | Client-side animations |
| Signal handler + graceful shutdown | Clean exit |
| Alert service (audio playback) | Client-side sounds |

## Imports Removed from cmd/cue

```go
// These all move to server-only
"github.com/CreateFutureMWilkinson/cue/internal/repository/implementation/sqlite"
"github.com/CreateFutureMWilkinson/cue/internal/service/buffer"
"github.com/CreateFutureMWilkinson/cue/internal/service/decisionengine"
"github.com/CreateFutureMWilkinson/cue/internal/service/orchestrator"
"github.com/CreateFutureMWilkinson/cue/internal/service/queueprocessor"
"github.com/CreateFutureMWilkinson/cue/internal/service/rulesengine"
"github.com/CreateFutureMWilkinson/cue/internal/service/vector"
"github.com/CreateFutureMWilkinson/cue/internal/service/watcher"
```

## Imports Added to cmd/cue

```go
"github.com/CreateFutureMWilkinson/cue/internal/client"
```

## TDD Behaviors

| # | Behavior | Test Strategy |
|---|---|---|
| 1 | Server connection on startup with health check | Test with mock server: healthy, unhealthy, unreachable |
| 2 | Adapter wiring — all presenters receive correct adapter types | Verify presenter creation succeeds with adapter deps |
| 3 | Activity event stream — WebSocket connects, events flow to ActivityPresenter | Integration test with mock WS server |
| 4 | Notification flow — query + resolve/dismiss via HTTP | End-to-end with mock server |
| 5 | Feedback flow — query buffered + save rating via HTTP | End-to-end with mock server |
| 6 | Settings flow — account CRUD + watcher lifecycle via HTTP | End-to-end with mock server |
| 7 | Rules flow — rule CRUD via HTTP | End-to-end with mock server |
| 8 | Planner flow — todo CRUD + schedule generation via HTTP | End-to-end with mock server |
| 9 | Timer flow — stays local, no server dependency | Verify TimerPresenter uses local Clock + local TimerAlerter |
| 10 | Reconnection handling — WebSocket reconnect, HTTP retry | Simulate disconnect/reconnect |
| 11 | Remove dead code — no unused imports or service wiring | Compilation check |

## Acceptance Criteria

- `cmd/cue` starts and connects to a running `cue-server`
- All UI features work identically to the direct-wired version:
  - Notifications display with color-coded cards
  - Feedback review modal works
  - Settings UI manages service accounts
  - Rules tab manages routing rules
  - Day planner wizard generates schedules
  - Timer countdown works with audio alerts
  - Character animations respond to activity events
- `cmd/cue` shows a clear error if `cue-server` is not running
- `just test` passes — no existing test breaks
- `just test-ui` passes — acceptance tests unchanged
- No `internal/repository/` or `internal/service/` imports in `cmd/cue/`
- Binary size is smaller (no SQLite, Ollama client, etc. linked in)

## Risk Areas

1. **Latency perception**: Direct repo calls are <1ms. HTTP calls to localhost are 1-5ms. For batch queries (notification list, todo list), this should be imperceptible. For individual operations (resolve notification), same. The only concern is the planner's `GenerateSchedules` call which involves Ollama — but that's already slow (seconds) regardless of transport.

2. **Offline operation**: The Fyne client cannot function without `cue-server`. This is intentional — the server owns all state. If users want offline capability, that's a future feature (local cache/sync).

3. **Config migration**: Existing users have `~/.cue/config.toml` with `[database]`, `[ollama]`, etc. The client will silently ignore these sections. The server will read them. No migration needed, but documentation should clarify which binary reads which sections.

4. **Test isolation**: Presenter tests mock their interfaces and don't change. `cmd/cue/main.go` wiring is tested with `httptest.NewServer`. The only risk is integration between real server + real client, which is a manual test initially.
