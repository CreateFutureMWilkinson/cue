# Feature 107: Fyne Client Re-wire

**Phase:** Phase-9-Feature-107
**Status:** Planning
**Depends on:** Feature 106 (API Client SDK), Feature 108 (TOFU pairing), Feature 109 (Todo Domain Restructure), Feature 110 (TOFU Client Bootstrap), Features 096-104 (server APIs)
**Enables:** Feature 111 (Sidecar Supervisor), Feature 112 (UI Single-Instance Lock)
**Packages:** `cmd/cue/`, `cmd/cue/adapters/` (new), `cmd/cue-server/`, `internal/server/runner/` (new), `internal/server/handler/`, `pkg/client/`, `internal/ui/presenter/`, `internal/ui/view/`, `internal/config/`

---

## Overview

Re-wire the Fyne GUI from a monolithic binary that creates repositories, services, and an orchestrator in-process into a thin client that connects to `cue server` and feeds HTTP/WebSocket-backed adapters into the existing presenters.

This feature is scoped to **transport rewiring only**. The work it does NOT cover, broken out into separate dependent features:

| Concern | Feature |
|---|---|
| Client-side TOFU bootstrap library | **Feature 110** (consumed by 107) |
| Sidecar supervisor + `mode=sidecar` enablement | **Feature 111** (built on top of 107) |
| Single-instance UI lock | **Feature 112** (built on top of 107) |

After 107 ships:
- `cue` is a single-binary CLI with subcommands: `cue ui`, `cue server`, `cue server reset-auth`, `cue version`. `cue` with no args prints help.
- `cue ui` is a thin Fyne client. It makes no SQLite, Ollama, or orchestrator calls and does not import `internal/repository/` or `internal/service/{buffer,decisionengine,orchestrator,queueprocessor,rulesengine,vector,watcher,planner,calendar}`.
- `cue ui` requires `[server].mode = "external"` until Feature 111 lands. Selecting `mode = "sidecar"` is rejected at config validation with a clear "not yet available — set `mode = external` and run `cue server` separately" error.
- The client config ignores `[database]`, `[ollama]`, and `[orchestrator]`; the sidecar reads those sections itself once it exists. Both `ValidateForClient` and `ValidateForServer` reject `port = 0`.
- `cmd/cue-server` is retained as a thin compatibility wrapper around the same `internal/server/runner` package the `cue server` subcommand uses. The `cue` binary is canonical.
- All UI features still work; the planner wizard is structurally preserved with two pruned steps.

---

## SDK-Only Client Code

All HTTP and WebSocket interaction with the server flows through `pkg/client`. No raw `net/http`, `coder/websocket`, or URL construction outside the SDK. Adapters are thin shims over `client.*Client` interfaces — they translate DTOs and route errors, they do not transport. Specifically:

- WebSocket reconnection is owned by `client.ActivityClient` (exponential backoff, 1s → 30s default). The activity adapter (work pkg 6) is a fan-out + DTO translator on top; it does **not** add its own retry/backoff.
- Error rendering throughout the client uses `errors.As(err, &apiErr)` + switch on `apiErr.Code` (e.g., `ErrCodeUnauthorized`, `ErrCodeNotFound`). No string-matching on error messages.
- The boot test (work pkg 13) wires `httptest.NewServer` through `client.NewAPIClient(httpServer.URL)`; it does not bypass the SDK.

If a needed primitive is missing from `pkg/client`, the work to add it is owned by whichever feature first needs it. Feature 107 itself does not add any SDK methods; Feature 111 adds `HealthInfo` for orphan-probe envelope verification.

---

## TDD Exemption

**This feature is rewire-only — no new product behavior, only swapping in-process service calls for HTTP/WebSocket adapters against an existing SDK.** By project-lead decision 2026-04-27, the strict per-behavior RED-GREEN-REFACTOR micro-loop in CLAUDE.md is waived for this feature. Tests are still written, but on the following criteria:

- **Tests written for non-trivial new logic:** activity adapter fan-out (drop-on-slow, dual consumer, `Close()` propagation), planner `CurrentFocusTask`, server runner extraction.
- **DTO-mapping adapters are covered by one integration test each** via `httptest.NewServer`, mirroring `pkg/client/*_test.go` patterns. Per-method unit tests for straight-line DTO conversion are coverage theatre and are skipped.
- **Existing test gates remain:** `just test`, `just test-ui`, `just security`, `just vulncheck`, and the ≥80% package-level coverage gate.

CLAUDE.md is unchanged. **This exemption does not extend to dependent features.** Features 110, 111, and 112 retain the full TDD discipline because they introduce genuinely new behavior (auth bootstrap, process supervision, file locking).

---

## Locked Decisions

### 1. Composition root rewrite

Current `cmd/cue/main.go` (641 lines):
```
Load config → Open DB → Create repos → Create services → Create orchestrator →
Build watchers → Create presenters(repos, services) → Create Fyne UI → Run
```

New `cue ui` action body (~280 lines):
```
Load config → ValidateForClient → Bootstrap TOFU token (Feature 110) →
Health-check server → Create APIClient → Create adapters →
Create presenters(adapters) → Create Fyne UI → Connect WebSocket → Run
```

### 2. Single binary, urfave subcommands

`cmd/cue` becomes a `urfave/cli/v3` command tree:

| Invocation | Action |
|---|---|
| `cue` | Print help (urfave default) |
| `cue ui` | Fyne client (this feature's main subject) |
| `cue server` | Headless server (today's `cmd/cue-server` body via `internal/server/runner`) |
| `cue server reset-auth` | Wipe server token DB |
| `cue version` | Standard |

The server body extracts to `internal/server/runner.Run(ctx, cfg)`. `cmd/cue-server/main.go` is retained as a thin wrapper that calls `runner.Run` so existing scripts keep working — `cue` is canonical, `cue-server` is for posterity. Both paths share one code path; binary size hit (Fyne linked into both) is accepted.

### 3. Config split — client vs server, shared TOML

Both code paths read `~/.cue/config.toml`. Each ignores sections irrelevant to it.

**Client reads:**
- `[server]` — host + port + `mode` (required; `mode ∈ {sidecar, external}`, default `external` for the duration of this feature; `sidecar` is rejected at validation until Feature 111 lands)
- `[gui]` — window dimensions, character config
- `[notification]` — audio settings (client-side playback)
- `[planner]` — timer sound, timer volume (client-side audio)
- `[logging]` — log level, log dir

**Client ignores (consumed by `cue server`):**
- `[database]`
- `[ollama]`
- `[orchestrator]`

A new `config.ValidateForClient()` enforces required sections. Both validators reject `port = 0` — the future sidecar must know the port before spawn completes, and "auto-pick" no longer makes sense in any mode.

### 4. TOFU bootstrap is delegated to Feature 110

The `cmd/cue/auth/` package is owned by Feature 110, with full TDD. 107 consumes its `Bootstrap(ctx, sdk *client.AuthClient) (token string, err error)` entry point in the `cue ui` action between config load and the health check.

### 5. Server connection — health check, no background retry

After token bootstrap, the client calls `GET /api/v1/health` with a 5-second timeout. If unhealthy or unreachable, Fyne shows an error dialog with Retry / Quit buttons. There is no silent background retry — the app is useless without the server, and a half-populated UI is worse than an explicit failure. Sidecar-aware behavior (waiting for spawn) is added in Feature 111.

### 6. WebSocket lifecycle

`ActivityClient.Connect()` is called **after** `MainWindow.Show()` returns, so a slow handshake never blocks the UI. The activity adapter (see §"Activity adapter") owns reconnection, fan-out, and `Close()` propagation. `Close()` is called from the signal handler during graceful shutdown.

### 7. Watcher add/remove → toggle

The current `cmd/cue/main.go` exposes a `WatcherFactory` closure that calls `orch.AddWatcher`. The SDK's `ServiceConfigClient` already has `ToggleSlackAccount`, `ToggleEmailAccount`, and `ToggleCalendarAccount`. The factory closure becomes `Toggle*Account(ctx, id, true)`; the `WatcherRemover` interface becomes `Toggle*Account(ctx, id, false)`. No new server endpoint required.

### 8. Validators stay client-side

`SlackValidator`, `EmailValidator`, and `CalendarValidator` (in `internal/service/validation/`) talk to Slack/IMAP/HTTP directly with user-supplied credentials. They are stateless and not server-coupled. Unchanged.

### 9. Categories — delivered by Feature 109 (Done)

Feature 109 has shipped: server endpoints under `/api/v1/todo/categories`, name-keyed model, single-FK `tasks.category_key`, and `pkg/client/categories.go` (`CategoryClient`). 107 consumes this contract directly. Specifically:

- `cmd/cue/adapters/tasks.go` maps `client.Task.Category *CategoryEmbed` (single embed `{key, name}`, never a slice) onto `presenter.TodoRow`. The legacy multi-category presenter shape (`Categories []repository.Category`) is collapsed to a single-or-nil `Category *repository.Category` view-model.
- `CategoryQuerier` adapter wraps `CategoryClient.ListCategories`.

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

## What Gets Removed from `cmd/cue/main.go` (today's body)

| Removed | Replaced by |
|---|---|
| SQLite `db.Open()` + migration | `internal/server/runner` (called only from `cue server`) |
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

## What Stays in the `cue ui` Action

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
| Centralized `*client.APIError` rendering helper | Routes `apiErr.Code == ErrCodeUnauthorized` to a "token rejected — restart and re-pair" dialog (covers stale-token mid-session, e.g., user ran `cue server reset-auth` while UI is open). All adapters' error paths flow through this helper. |

---

## Adapter Inventory (`cmd/cue/adapters/`)

A new sub-package keeps the `ui` action short and lets shims be unit-tested without booting Fyne.

| Adapter | Wraps | Satisfies | Translation Notes |
|---|---|---|---|
| `activity.go` | `client.ActivityClient` | `presenter.ActivitySource` (×2 consumers — `ActivityPresenter` + `CharacterPresenter`) | Decode `EventEnvelope` by `Type` into `presenter.ActivityEvent{Source, Message, IsError}`; reproduce today's non-blocking fan-out (`bridgeEvents`); propagate `Close()`. Surface `EventEnvelope.DroppedSinceLast > 0` as a synthetic `ActivityEvent{Source:"system", Message:"N events dropped"}` so users see when the server dropped events. Reconnection is the SDK's responsibility — adapter does not add its own retry/backoff. **Tested.** |
| `messages.go` | `client.MessagesClient` | `presenter.MessageQuerier` + `presenter.MessageUpdater` | DTO ↔ `repository.Message`; status enum, RFC3339 ↔ `time.Time`. |
| `feedback.go` | `client.FeedbackClient` | `presenter.BufferReviewer` | DTO ↔ `repository.Message` (buffered subset). |
| `rules.go` | `client.RulesClient` + `client.MessagesClient` | `repository.RoutingRuleRepository` + `repository.QueueRepository` | Two interfaces in one adapter; `RulesClient` covers rule CRUD, `MessagesClient` covers queue listing. |
| `tasks.go` | `client.TaskClient` + `client.CategoryClient` | `presenter.TodoQuerier` + `presenter.CategoryQuerier` | DTO ↔ `repository.Todo` + `repository.Category`; collapse multi-category to single `*Category`. |
| `schedule.go` | `client.ScheduleClient` | `repository.ScheduleRepository` + simplified `presenter.ScheduleGenerator` | `LoadByDate`/`Save`/`Delete` ↔ `GetSchedule`/`PutSchedule`/`DeleteSchedule`; `GenerateSchedules(ctx, date)` ↔ `POST /api/v1/planner/generate`; translate `ScheduleOption` (HH:MM strings) → `planner.DaySchedule`. |
| `service_config.go` | `client.ServiceConfigClient` | `repository.ServiceConfigRepository` + `presenter.WatcherRemover` + `presenter.WatcherFactory` closure | Toggle endpoints replace orch.AddWatcher / orch.RemoveWatcher (Decision 7). |

All adapters get one round-trip integration test using `httptest.NewServer`. The activity adapter additionally gets unit tests for fan-out semantics (drop-on-slow, dual consumer, Close propagation) per the TDD-Exemption criteria.

---

## Implementation Sequence

Per CLAUDE.md UI Feature Workflow: UI tests for the wizard trim are updated first, before any implementation, then serve as the outer verification gate.

**Pre-loop UI test commit** — `test(ui): failing acceptance tests for simplified wizard flow`
- Merged StepTodoEdit (was StepTaskSelect + StepPriority).
- StepEstimates removed.
- Current-focus-task panel in StepActive.

**Work packages** (commit at logical breakpoints, not per-RED/GREEN/REFACTOR):

| # | Work Package | Notes |
|---|---|---|
| 1 | Subcommand restructure | Extract `cmd/cue-server/main.go` body into `internal/server/runner.Run(ctx, cfg)`. `cmd/cue-server` becomes a thin wrapper. `cmd/cue/main.go` registers `urfave/cli/v3` subcommands: `ui` (stub), `server`, `server reset-auth`, `version`. Smoke test: `cue server` boots equivalently to old `cue-server`. |
| 2 | Config validation | New `config.ValidateForClient` (ignores server-only sections, requires `[server]`, validates `mode`, rejects `mode=sidecar` until Feature 111). Both validators reject `port = 0`. Table-driven tests. |
| 3 | Health-check gate | `connectServer(cfg) (*client.APIClient, error)`: poll `/api/v1/health` for up to 5s; error → Fyne dialog with Retry/Quit. |
| 4 | Planner presenter contract trim | Drop `estimator` + `cal` deps. New `ScheduleGenerator.GenerateSchedules(ctx, date)`. Add `CurrentFocusTask(ctx) (*TodoRow, error)` — **tested** (happy path + empty list). |
| 5 | Wizard UI trim | Merge `StepTaskSelect` + `StepPriority` → `StepTodoEdit`; delete `StepEstimates` view; wire current-focus-task panel into `StepActive`. UI acceptance tests from pre-loop commit must now pass. |
| 6 | Adapter: `activity.go` | Envelope decode + non-blocking dual fan-out + `Close()` propagation. **Unit-tested** for fan-out semantics + integration-tested. |
| 7 | Adapter: `messages.go` | DTO ↔ `repository.Message`. Round-trip integration test. |
| 8 | Adapter: `feedback.go` | `BufferReviewer`. Round-trip integration test. |
| 9 | Adapter: `rules.go` | `RoutingRuleRepository` + `QueueRepository`. Round-trip integration test. |
| 10 | Adapter: `tasks.go` | `TodoQuerier` + `CategoryQuerier`; single `*Category`. Round-trip integration test. |
| 11 | Adapter: `schedule.go` | `ScheduleRepository` + simplified `ScheduleGenerator` from work pkg 4; HH:MM ↔ `planner.DaySchedule`. Round-trip integration test. |
| 12 | Adapter: `service_config.go` | `ServiceConfigRepository` + toggle closures. Round-trip integration test. |
| 13 | Centralized API-error helper | Small helper that wraps any presenter-facing error: if `errors.As(err, &apiErr)` and `apiErr.Code == ErrCodeUnauthorized`, route to a "token rejected — restart and re-pair" Fyne dialog. All adapters call through this helper for terminal error rendering. Tested with table-driven cases per error code. |
| 14 | Compose `cue ui` action | Fill in stub from work pkg 1: cfg → `auth.Bootstrap` (Feature 110) → health → APIClient → adapters (each using helper from work pkg 13) → presenters → MainWindow.Show() → ActivityClient.Connect() → run. Signal handler: ActivityClient.Close. Boot test uses `httptest.NewServer` driven through `client.NewAPIClient(httpServer.URL)` — no raw HTTP in the test. |
| 15 | Dead-code purge | Unused imports, staticcheck, `go vet`. |

Each commit ends with `just fmt` immediately before its message.

---

## Imports Removed from `cmd/cue` (the `ui` action specifically)

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
"github.com/CreateFutureMWilkinson/cue/cmd/cue/auth"             // Feature 110
"github.com/CreateFutureMWilkinson/cue/internal/server/runner"   // imported by `cue server` subcommand only
```

The server-only `internal/repository`, `internal/service/{buffer,…,calendar}` imports remain reachable through `internal/server/runner` (consumed by `cue server`), but never from the `cue ui` action.

---

## Wiring Verification

After work pkg 15, before security checks:

0. `grep -rn "net/http\|coder/websocket" cmd/cue/adapters/ cmd/cue/auth/` — empty. All transport flows through `pkg/client`.

1. `grep -rn "internal/repository\|internal/service/\(buffer\|decisionengine\|orchestrator\|queueprocessor\|rulesengine\|vector\|watcher\|planner\|calendar\)" cmd/cue/` — empty (server-only deps live behind `internal/server/runner`).
2. `grep -rn ErrNotImplemented cmd/cue pkg/client` (non-test) — empty.
3. Trace each presenter dep from the `cue ui` action: every dep is an adapter, a local service (alert, clock, validators), or a Fyne primitive. No SQLite, no Ollama, no orchestrator, no calendar.
4. `ActivityClient.Close()` reachable from the signal handler.
5. `cmd/cue-server/main.go` is a thin wrapper around `internal/server/runner.Run`; both `cue server` and `cue-server` produce equivalent behavior.
6. `just test` and `just test-ui` green; `just check ./cmd/cue/... ./cmd/cue-server/...` green.

---

## Acceptance Criteria

- `cue` (no args) prints the urfave help screen.
- `cue ui` (with `[server].mode = "external"` and a separately-running `cue server`) connects, claims a TOFU token via Feature 110's `auth.Bootstrap`, opens the Fyne window, and all UI features function.
- `cue ui` with `[server].mode = "sidecar"` exits non-zero with: `"sidecar mode is not yet available — set [server].mode = \"external\" and run cue server in another terminal"`.
- `cue server` runs headless and is functionally equivalent to today's `cue-server`. `cue-server` itself is retained as a thin compat wrapper.
- `cue server reset-auth` wipes the server token DB.
- TOFU token is loaded or claimed on first run; subsequent runs skip the claim.
- All UI features work:
  - Notifications display with color-coded cards.
  - Feedback review modal works.
  - Settings UI manages service accounts (CRUD + toggle).
  - Rules tab manages routing rules.
  - Day planner wizard: edit todos → generate → pick option → run, with current-focus-task hint during focus blocks.
  - Timer countdown works with audio alerts.
  - Character animations respond to activity events from the WebSocket.
- `cue ui` shows a clear error dialog if the server is unreachable or fails health.
- SIGINT/SIGTERM/window-close on `cue ui` cleanly closes the WebSocket and exits.
- `just test`, `just test-ui`, `just security`, `just vulncheck` all pass.
- No `internal/repository/` or `internal/service/{buffer,decisionengine,orchestrator,queueprocessor,rulesengine,vector,watcher,planner,calendar}` imports anywhere under `cmd/cue/` outside of `internal/server/runner` (reached via `cue server`).
- Both validators reject `port = 0`.

---

## Risk Areas

1. **Latency.** Direct repo calls are sub-millisecond; localhost HTTP is 1–5 ms. Imperceptible for batch queries (notifications, todos) and individual operations. `GenerateSchedules` was already slow because it touches Ollama; transport is not the bottleneck.

2. **Offline operation.** The Fyne client cannot function without the server. This is intentional. Local cache/sync is a future feature.

3. **Activity adapter fan-out semantics.** Today's `bridgeEvents` uses non-blocking sends with `select default` to drop events for slow consumers. The adapter must reproduce this exactly, or character animations will block on slow notification rendering (or vice versa). Covered by the TDD-Exemption "tested" list.

4. **TOFU error UX.** Surfaced through Feature 110's API. 107's job is to render its errors clearly in the Fyne dialog (server unreachable vs token rejected vs no paired clients available).

5. **WebSocket reconnection during long Ollama calls.** A `GenerateSchedules` request can take many seconds while Ollama runs server-side. The WebSocket connection stays open the whole time; reconnection logic should not interpret a long HTTP wait as a connection problem. (This is `pkg/client/activity.go`'s responsibility, not 107's, but worth verifying in manual smoke testing.)

6. **Binary size.** Linking Fyne into `cue server` grows the headless deployment binary. Accepted explicitly; build-tag splitting is a future feature if needed.

7. **Test-only flag for development.** The `cue ui` action body needs to be testable end-to-end in the loop 14 boot test. Solution: the action is structured so that the API client and (later) supervisor can be injected via test-only construction helpers; production `main.go` uses a no-flag default. No CLI flag exists to bypass real wiring.

8. **Stale-token mid-session.** If the user runs `cue server reset-auth` while `cue ui` is running, the next authenticated request returns plain `401 UNAUTHORIZED` (no `TOKEN_ISSUED` body, since this isn't a fresh server). Mitigated by work pkg 13 — the centralized API-error helper detects `ErrCodeUnauthorized` and surfaces a "token rejected — restart and re-pair" dialog. Without this, the user sees opaque API errors per-presenter and has no clear recovery path.

9. **Server-dropped events during long disconnect.** The SDK's `ActivityClient` reconnects transparently but does not auto-call `Replay` on reconnect. Events the *server* dropped during a long disconnect (laptop closed for hours) are lost to this session. The `EventEnvelope.DroppedSinceLast` field is surfaced via the activity adapter as a system-source event so the user knows. Calling `Replay` from 107 is deferred — for Cue's 10-minute batch cadence, brief disconnects don't cause loss; long ones (laptop sleep) are an accepted limitation.

10. **`DroppedSinceLast` rendering.** The activity adapter surfaces drops as synthetic events. UX choice: drops appear in the activity log only, not as toast notifications, since they are background information rather than actionable.

---

## Estimate

- New code: ~900 LOC adapters + ~150 LOC subcommand restructure + ~280 LOC `cue ui` action body + ~80 LOC API-error helper + ~500 LOC tests.
- Removed: ~400 LOC from old `cmd/cue/main.go` + ~250 LOC deleted wizard steps/views (offset by code moving into `internal/server/runner`, not deletion).
- Net: ~+1200 LOC, mostly tests and DTO translation.
- Work packages: 15 (excluding the auth, sidecar, and uilock packages now owned by 110/111/112). With the TDD exemption, ~28–38 commits across the feature.
- Total ≈ 4.5–5.5 working days.
