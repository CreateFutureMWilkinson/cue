# Feature 107: Fyne Client Re-wire

**Phase:** Phase-9-Feature-107
**Status:** Planning
**Depends on:** Feature 106 (API Client SDK), Feature 108 (TOFU pairing), Feature 109 (Todo Domain Restructure), Features 096-104 (server APIs)
**Packages:** `cmd/cue/`, `cmd/cue/adapters/` (new), `cmd/cue/auth/` (new), `internal/server/handler/`, `pkg/client/`, `internal/ui/presenter/`, `internal/ui/view/`

---

## Overview

Re-wire the Fyne GUI application from a monolithic binary that creates repositories, services, and an orchestrator in-process into a thin client that connects to `cue-server` and feeds HTTP/WebSocket-backed adapters into the existing presenters.

The work is mostly composition-root rewrite plus translating adapters between the SDK's DTOs (`pkg/client/`) and the presenters' repository-shaped interfaces (`internal/ui/presenter/`). One presenter — the planner — also gets a small contract simplification, since the planner's previous design pulled `Ollama` and `CalendarProvider` into the client process and that no longer makes sense once the server owns them.

After 107 ships:
- `cmd/cue` makes no SQLite, Ollama, or orchestrator calls.
- `cmd/cue` does not import `internal/repository/`, `internal/service/{buffer,decisionengine,orchestrator,queueprocessor,rulesengine,vector,watcher,planner,calendar}`.
- The client config no longer requires `[ollama]`, `[database]`, or `[orchestrator]`.
- All UI features still work; the planner wizard is structurally preserved with two pruned steps.

---

## Locked Decisions

### 1. One feature, no split

The planner's contract trim is small enough to ride inside 107. There is no 107A.

### 2. Composition root rewrite

Current `cmd/cue/main.go` (641 lines):
```
Load config → Open DB → Create repos → Create services → Create orchestrator →
Build watchers → Create presenters(repos, services) → Create Fyne UI → Run
```

New `cmd/cue/main.go` (~280 lines):
```
Load config → Load/claim TOFU token → Health-check server → Create APIClient →
Create adapters → Create presenters(adapters) → Create Fyne UI →
Connect WebSocket → Run
```

### 3. Config split — client vs server, shared TOML

Both binaries read `~/.cue/config.toml`. Each ignores sections irrelevant to it.

**Client reads:**
- `[server]` — host + port (required)
- `[gui]` — window dimensions, character config
- `[notification]` — audio settings (client-side playback)
- `[planner]` — timer sound, timer volume (client-side audio)
- `[logging]` — log level, log dir

**Client ignores (server-only):**
- `[database]`
- `[ollama]`
- `[orchestrator]`

A new `config.ValidateForClient()` enforces this. The existing `config.ValidateForServer()` (used by `cmd/cue-server`) is unchanged.

### 4. TOFU bootstrap

Feature 108 (TOFU pairing) is the auth source of truth. On startup, the client:

1. Loads the bearer token from disk (path managed by the new `cmd/cue/auth/` helper).
2. If absent, performs the first-run pairing claim against the server and persists the result.
3. Calls `APIClient.SetToken(...)` and configures `ActivityClient` with the same token.
4. Only then proceeds to the health check.

### 5. Server connection — health check, no background retry

After token bootstrap, the client calls `GET /api/v1/health` with a 5-second timeout. If unhealthy or unreachable, Fyne shows an error dialog with Retry / Quit buttons. There is no silent background retry — the app is useless without the server, and a half-populated UI is worse than an explicit failure.

### 6. WebSocket lifecycle

`ActivityClient.Connect()` is called **after** `MainWindow.Show()` returns, so a slow handshake never blocks the UI. The activity adapter (see §"Activity adapter") owns reconnection, fan-out, and `Close()` propagation. `Close()` is called from the signal handler during graceful shutdown.

### 7. Watcher add/remove → toggle

The current `cmd/cue/main.go` exposes a `WatcherFactory` closure that calls `orch.AddWatcher`. The SDK's `ServiceConfigClient` already has `ToggleSlackAccount`, `ToggleEmailAccount`, and `ToggleCalendarAccount`. The factory closure becomes `Toggle*Account(ctx, id, true)`; the `WatcherRemover` interface becomes `Toggle*Account(ctx, id, false)`. No new server endpoint required.

### 8. Validators stay client-side

`SlackValidator`, `EmailValidator`, and `CalendarValidator` (in `internal/service/validation/`) talk to Slack/IMAP/HTTP directly with user-supplied credentials. They are stateless and not server-coupled. Unchanged.

### 9. Categories — delivered by Feature 109

The planner's todo list view groups by category (`CategoryQuerier.QueryAll`). Feature 109 (Todo Domain Restructure) delivers the server endpoint (`GET /api/v1/todo/categories`) and the SDK (`pkg/client/categories.go` with `CategoryClient`) before 107 starts. 107 only consumes them via its tasks adapter — no server-side or SDK loop is required inside 107 itself.

### 10. Planner contract simplification

The planner has no client/server gap, but the existing `PlannerPresenter` constructor pulls in `planner.TaskEstimator` and `calendar.CalendarProvider` so the client can do per-task LLM estimation and ICS fetching in-process. Both are unnecessary:

- The server-side `GenerateSchedulesHandler` already fetches the calendar (`internal/server/handler/planner.go:327`).
- Schedules don't take user-selected tasks as input. Tasks are managed in their own list view; during a focus block the active view shows the top-priority incomplete todo as a hint. There is no coupling between todo state and schedule structure.

The presenter contract therefore changes:

- `NewPlannerPresenter` drops `estimator planner.TaskEstimator` and `cal calendar.CalendarProvider`.
- `ScheduleGenerator.GenerateSchedules` simplifies from `(ctx, tasks, events, date) → (focus, recovery, error)` to `(ctx, date) → (focus, recovery, error)`.
- A new `CurrentFocusTask(ctx) (*TodoRow, error)` method calls `TaskClient.ListTasks(filter={status:pending})` and returns the highest-priority incomplete todo. The active-schedule view consumes it as a single-task hint.

### 11. Wizard step trim

The wizard structure is preserved with two trims:

| Today | After 107 |
|---|---|
| StepIdle | StepIdle |
| StepTaskSelect (select tasks for plan) | merged into **StepTodoEdit** — todo list with reorder + priority controls; no per-plan selection |
| StepPriority (reorder selected tasks) | merged into **StepTodoEdit** |
| StepEstimates (per-task pomo override) | **deleted** |
| StepSchedule (review options + pick) | StepSchedule — unchanged |
| StepActive (running schedule) | StepActive — adds current-focus-task panel |

The Generate Plan button placement is unchanged.

### 12. No server-side planner work

The server's `GenerateSchedulesHandler` is correct as-is. Empty `[]TaskEstimate{}` is intentional, not a bug. No server changes for the planner.

If the user has no calendar account configured, the server's `noopCalendarProvider` returns no events and the schedule contains only focus + break blocks. The presenter does not handle this specially.

---

## What Gets Removed from `cmd/cue/main.go`

| Removed | Replaced by |
|---|---|
| SQLite `db.Open()` + migration | server-only |
| All `repository.New*()` calls | server-only |
| `decisionengine.NewClient()` (Ollama) | server-only |
| `vector.NewStore()` | server-only |
| `buffer.NewService()` | server-only |
| `orchestrator.New()` + `Start()` | server-only |
| `queueprocessor.New()` + `Start()` | server-only |
| `rulesengine.New()` | server-only |
| `buildWatchersFromDB()` | server-only |
| `bridgeEvents()` + `channelActivitySource` | activity adapter (WebSocket fan-out) |
| `osFileSystem`, `httpClient` (watcher deps) | server-only |
| `planner.NewPlanner` + `OllamaTaskEstimator` | simplified `ScheduleGenerator` adapter |
| `calendar.NewICSProvider` | server-side calendar fetch (already done) |

## What Stays in `cmd/cue/main.go`

| Kept | Why |
|---|---|
| `config.Load()` + `config.ValidateForClient()` | client owns its own config |
| Fyne app + MainWindow creation | client-side UI |
| All presenter constructors (with new adapter deps) | presenters unchanged except planner contract |
| AppBinder (planner presenter ↔ views) | unchanged |
| TimerLoop (1Hz tick) | client-side countdown |
| Character loading + creation + animator | client-side animations |
| Signal handler + graceful shutdown (incl. `ActivityClient.Close()`) | clean exit |
| Alert service (audio playback) | client-side sounds |
| Validators (`SlackValidator`, `EmailValidator`, `CalendarValidator`) | per Decision 8 |

---

## Adapter Inventory (`cmd/cue/adapters/`)

A new sub-package keeps `main.go` short and lets shims be unit-tested without booting Fyne.

| Adapter | Wraps | Satisfies | Translation Notes |
|---|---|---|---|
| `activity.go` | `client.ActivityClient` | `presenter.ActivitySource` (×2 consumers — `ActivityPresenter` + `CharacterPresenter`) | Decode `EventEnvelope` by `Type` into `presenter.ActivityEvent{Source, Message, IsError}`; reproduce today's non-blocking fan-out (`bridgeEvents`); propagate `Close()` |
| `messages.go` | `client.MessagesClient` | `presenter.MessageQuerier` + `presenter.MessageUpdater` | DTO ↔ `repository.Message`; status enum, RFC3339 ↔ `time.Time` |
| `feedback.go` | `client.FeedbackClient` | `presenter.BufferReviewer` | DTO ↔ `repository.Message` (buffered subset) |
| `rules.go` | `client.RulesClient` + `client.MessagesClient` | `repository.RoutingRuleRepository` + `repository.QueueRepository` | Two interfaces in one adapter; `RulesClient` covers rule CRUD, `MessagesClient` covers queue listing |
| `tasks.go` | `client.TaskClient` + `client.CategoryClient` | `presenter.TodoQuerier` + `presenter.CategoryQuerier` | DTO ↔ `repository.Todo` + `repository.Category` |
| `schedule.go` | `client.ScheduleClient` | `repository.ScheduleRepository` + simplified `presenter.ScheduleGenerator` | `LoadByDate`/`Save`/`Delete` ↔ `GetSchedule`/`PutSchedule`/`DeleteSchedule`; `GenerateSchedules(ctx, date)` ↔ `POST /api/v1/planner/generate`; translate `ScheduleOption` (HH:MM strings) → `planner.DaySchedule` |
| `service_config.go` | `client.ServiceConfigClient` | `repository.ServiceConfigRepository` + `presenter.WatcherRemover` + `presenter.WatcherFactory` closure | Toggle endpoints replace orch.AddWatcher / orch.RemoveWatcher (Decision 7) |

Plus a small new package `cmd/cue/auth/` for TOFU token persistence and first-run pairing claim.

---

## TDD Sequence

Per CLAUDE.md UI Feature Workflow: UI tests for the wizard trim are updated first, before any RED phase, then serve as the outer verification gate.

**Pre-loop UI test commit** — `test(ui): failing acceptance tests for simplified wizard flow`
- Merged StepTodoEdit (was StepTaskSelect + StepPriority).
- StepEstimates removed.
- Current-focus-task panel in StepActive.

**Micro-loops (RED → GREEN → REFACTOR, three commits per loop):**

| # | Loop | Scope |
|---|---|---|
| 1 | `config.ValidateForClient` | drops `[ollama]`, `[database]`, `[orchestrator]` requirements; requires `[server]` |
| 2 | TOFU client bootstrap | new `cmd/cue/auth/` package: token load, first-run pairing claim, persistence |
| 3 | Health-check gate | `connectServer(cfg) (*client.APIClient, error)`; 5s timeout; error surfaces to Fyne dialog |
| 4 | Planner presenter contract change | drop `estimator` + `cal` deps; new `ScheduleGenerator` signature; `CurrentFocusTask` method; collapse step set |
| 5 | Wizard UI trim | merge StepTaskSelect + StepPriority into StepTodoEdit; delete StepEstimates view; wire current-focus-task panel into StepActive |
| 6 | `cmd/cue/adapters/activity.go` | envelope decode + dual-consumer fan-out + `Close()` propagation |
| 7 | `cmd/cue/adapters/messages.go` | DTO ↔ `repository.Message` |
| 8 | `cmd/cue/adapters/feedback.go` | DTO ↔ `repository.Message` (buffered) |
| 9 | `cmd/cue/adapters/rules.go` | `RulesClient` + `MessagesClient` → `RoutingRuleRepository` + `QueueRepository` |
| 10 | `cmd/cue/adapters/tasks.go` | `TaskClient` → `TodoQuerier`; `CategoryClient` → `CategoryQuerier` (both delivered by Feature 109) |
| 11 | `cmd/cue/adapters/schedule.go` | satisfies the simplified contract from loop 4 |
| 12 | `cmd/cue/adapters/service_config.go` | `ServiceConfigClient` → `ServiceConfigRepository`; toggle replaces watcher factory |
| 13 | Composition root rewrite | new `cmd/cue/main.go`; WS connect post-`MainWindow.Show()`; `ActivityClient.Close()` in cleanup |
| 14 | Dead-code purge | remove unused imports; `go vet`; staticcheck clean |

Each loop ends with `just fmt` immediately before its commit.

---

## Imports Removed from `cmd/cue`

```go
"github.com/CreateFutureMWilkinson/cue/internal/repository/implementation/sqlite"
"github.com/CreateFutureMWilkinson/cue/internal/service/buffer"
"github.com/CreateFutureMWilkinson/cue/internal/service/decisionengine"
"github.com/CreateFutureMWilkinson/cue/internal/service/orchestrator"
"github.com/CreateFutureMWilkinson/cue/internal/service/queueprocessor"
"github.com/CreateFutureMWilkinson/cue/internal/service/rulesengine"
"github.com/CreateFutureMWilkinson/cue/internal/service/vector"
"github.com/CreateFutureMWilkinson/cue/internal/service/watcher"
"github.com/CreateFutureMWilkinson/cue/internal/service/planner"
"github.com/CreateFutureMWilkinson/cue/internal/service/calendar"
```

## Imports Added to `cmd/cue`

```go
"github.com/CreateFutureMWilkinson/cue/pkg/client"
"github.com/CreateFutureMWilkinson/cue/cmd/cue/adapters"
"github.com/CreateFutureMWilkinson/cue/cmd/cue/auth"
```

---

## Wiring Verification

After loop 16, before security checks:

1. `grep -n "internal/repository\|internal/service/\(buffer\|decisionengine\|orchestrator\|queueprocessor\|rulesengine\|vector\|watcher\|planner\|calendar\)" cmd/cue/main.go` — empty.
2. `grep -rn ErrNotImplemented cmd/cue pkg/client` (non-test) — empty.
3. Trace each presenter dep from `main.go`: every dep is an adapter, a local service (alert, clock, validators), or a Fyne primitive. No SQLite, no Ollama, no orchestrator, no calendar.
4. `ActivityClient.Close()` reachable from the signal handler.
5. `cmd/cue-server/main.go` unchanged; both binaries build green.
6. `just test` and `just test-ui` green.

---

## Acceptance Criteria

- `cmd/cue` starts and connects to a running `cue-server`.
- TOFU token is loaded or claimed on first run; subsequent runs skip the claim.
- All UI features work:
  - Notifications display with color-coded cards.
  - Feedback review modal works.
  - Settings UI manages service accounts (CRUD + toggle).
  - Rules tab manages routing rules.
  - Day planner wizard: edit todos → generate → pick option → run, with current-focus-task hint during focus blocks.
  - Timer countdown works with audio alerts.
  - Character animations respond to activity events from the WebSocket.
- `cmd/cue` shows a clear error dialog if `cue-server` is unreachable.
- `just test`, `just test-ui`, `just security`, `just vulncheck` all pass.
- No `internal/repository/` or `internal/service/{buffer,decisionengine,orchestrator,queueprocessor,rulesengine,vector,watcher,planner,calendar}` imports in `cmd/cue/`.
- Client config genuinely no longer needs `[ollama]`, `[database]`, or `[orchestrator]`.

---

## Risk Areas

1. **Latency.** Direct repo calls are sub-millisecond; localhost HTTP is 1–5 ms. Imperceptible for batch queries (notifications, todos) and individual operations (resolve notification). `GenerateSchedules` was already slow because it touches Ollama; transport is not the bottleneck.

2. **Offline operation.** The Fyne client cannot function without `cue-server`. This is intentional. Local cache/sync is a future feature.

3. **Config migration.** Existing users have full TOML files with `[database]`, `[ollama]`, etc. The client silently ignores those sections; the server reads them. Documentation must clarify which binary reads which sections.

4. **Activity adapter fan-out semantics.** Today's `bridgeEvents` uses non-blocking sends with `select default` to drop events for slow consumers. The adapter must reproduce this exactly, or character animations will block on slow notification rendering (or vice versa).

5. **TOFU error UX.** First-run pairing requires the server to be reachable. If pairing fails (server down, network hiccup), the error dialog must distinguish "no token, server unreachable" from "token rejected" so the user knows whether to retry or run `cue-server --reset-auth`.

6. **WebSocket reconnection during long Ollama calls.** A `GenerateSchedules` request can take many seconds while Ollama runs server-side. The WebSocket connection stays open the whole time; reconnection logic should not interpret a long HTTP wait as a connection problem. (This is `pkg/client/activity.go`'s responsibility, not 107's, but worth verifying.)

7. **Test isolation.** Adapter tests use `httptest.NewServer` (HTTP) or `coder/websocket` upgrader (WS), mirroring `pkg/client/*_test.go`. The boot test for loop 15 uses an in-process `httptest` server plus headless Fyne (`test.NewApp()`); manual integration testing against a real `cue-server` happens once before merge.

---

## Estimate

- New code: ~900 LOC adapters + ~70 LOC categories endpoint + ~280 LOC main.go + ~600 LOC tests.
- Removed: ~400 LOC from main.go + ~250 LOC of deleted wizard steps/views.
- Net: ~+1300 LOC, mostly tests and DTO translation.
- Loops: 14, each one half-day TDD micro-cycle. Loops 4 + 5 + 11 are the heaviest because they touch presenter contract + UI + adapter together. Total ≈ 7–9 working days.
