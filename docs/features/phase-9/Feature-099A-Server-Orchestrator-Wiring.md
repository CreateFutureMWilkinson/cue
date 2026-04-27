# Feature 099A: Server Orchestrator Wiring

**Phase:** Phase-9-Feature-099A
**Status:** Done
**Package:** `cmd/cue-server/`, `internal/server/`, `internal/config/`
**Depends on:** Feature 097 (Server Infrastructure), Feature 099 (Activity Event Stream)
**Blocks:** Feature 102 (Service Configuration API), Feature 107 (Fyne Client Re-wire)

---

## Overview

Wire the orchestrator — and everything it depends on (watchers, queue processor, rules engine, Ollama client, vector store, service config repo) — into `cmd/cue-server/` so that activity events produced by the orchestrator flow out over the WebSocket hub established in Feature 099. Introduces `server.Composition` as the single composition root used by both `cmd/cue-server/main.go` and the end-to-end integration test.

This feature closes the publisher gap left by Feature 099: the hub existed but no events flowed through it. After 099A, `orchestrator.ActivityEvent`s reach WebSocket clients end-to-end, and `cue-server` is a fully functional headless runtime.

Also introduces a parallel alert pathway: instead of playing local audio (which is nonsense on a headless server), alerts broadcast over the same WebSocket with a new `type: "alert"` envelope. Clients decide how to render.

## Motivation

Three downstream features are unblocked:

1. **Feature 099 Event Stream** — the hub can now broadcast real orchestrator events, not just accept subscribers.
2. **Feature 102 Service Configuration API** — the handler has a live orchestrator reference available (`comp.Orchestrator`) to register/deregister watchers at runtime.
3. **Feature 107 Fyne Client Re-wire** — `cue-server` now owns all state (DB, orchestrator, services, watchers), so `cmd/cue` can become a thin HTTP/WebSocket client.

## Locked Design Decisions

### `cmd/cue` is untouched

`cmd/cue` is abandoned until Feature 107 re-wires it. 099A does **not** relocate the orchestrator; it *adds* an independent composition inside `cue-server`. Both binaries could in principle run simultaneously (and would each open their own SQLite handle), but in practice `cmd/cue` is not exercised during the 099A→107 window. This sidesteps the dual-DB-handle and transitional-activity-source questions entirely.

### Composition root lives in `internal/server/`

`server.NewComposition(ctx, cfg) (*Composition, error)` opens repos, constructs services, starts the orchestrator, registers watchers from the DB, builds the HTTP/WebSocket server, and wires the publisher goroutine. `cmd/cue-server/main.go` is now a thin wrapper: load config → validate → `NewComposition` → start HTTP → wait for SIGINT → `Shutdown`.

Both `cmd/cue-server/main.go` and the end-to-end integration test consume the same `NewComposition` — no drift possible.

### Shutdown is ordered, blocking, unbounded, and logged

`Composition.Shutdown(ctx)` runs these steps in order, logging progress via `slog` at every stage so an operator watching stdout/logs never wonders whether the process is hung:

1. `slog.Info("shutdown initiated")`
2. `Orchestrator.Stop()` — blocks until in-flight poll batches + Ollama scoring complete (each subcomponent has its own timeout, so shutdown is bounded in practice but unbounded by design).
3. `QueueProcessor.Stop()` — blocks until background scoring loop exits.
4. `close(EventCh)` — publisher goroutine exits naturally on range-over-closed-channel.
5. `HTTP.Shutdown(ctx)` — closes live WebSocket connections first (via `wsManager.CloseAll`), then drains in-flight REST requests.
6. `MessageRepo.DB().Close()` — closes the shared SQLite connection.

`sync.Once` guards the sequence so repeat calls are idempotent (no double-close panics on `EventCh`).

### Alerts broadcast over WebSocket with a new `type: "alert"` envelope

The orchestrator and queue processor require an `orchestrator.Alerter` (`PlayNotification(ctx) error`). In cue-server, `server.HubAlerter` implements this interface by publishing an envelope with `type: "alert"` + `AlertData{Kind: "notification"}` through the same hub. Feature 099's envelope multiplexing was designed for exactly this ("future event kinds … will multiplex over the same connection via additional `type` values"). Clients render audio or visual cues as they see fit.

This is coordinated with Feature 104 (Timer API), which owns its own `timer_block_complete` envelope type for timer-specific alerts. 099A's `alert` envelope is for generic orchestrator notifications only.

### `cue-server` requires `[server]` section; `cmd/cue` does not

`Config.ValidateForServer()` delegates to `Config.Validate()` for the standard rules, then additionally requires `Server.isConfigured()` to be true (host, port, timeouts all populated). `cmd/cue-server/main.go` calls `ValidateForServer()`; `cmd/cue` continues to call `Validate()`, so it keeps running without a `[server]` section.

### Testing: in-process `httptest.NewServer`

The end-to-end test (`TestCompositionEndToEndActivityBroadcast`) wraps `comp.HTTP.Handler()` in `httptest.NewServer`, dials the WebSocket with `coder/websocket`, injects an `orchestrator.ActivityEvent` through `comp.EventCh`, and reads the envelope back. Pure in-process, no subprocess or build artifacts, runs under `just test`.

## Architecture

### Composition wiring

```
cmd/cue-server/main.go
  → config.Load + ValidateForServer (requires [server])
  → server.NewComposition(ctx, cfg)
      → openRepositories(cfg)
          → MessageRepo (owns *sql.DB)
          → QueueRepo, RuleRepo, ServiceConfigRepo (share DB)
          → keyfile encryptor (sibling of DB file)
      → constructServices(ctx, cfg, ruleRepo)
          → OllamaClient
          → ChromemVectorStore (with Ollama embedding func)
          → RulesEngine (loaded from ruleRepo.ListRules)
      → constructHTTPServer(cfg.Server, msgRepo, hub)
          → server.New(cfg.Server, Deps{Messages: msgRepo, Hub: hub})
      → startOrchestration(ctx, cfg, msgRepo, queueRepo, rulesEngine, ollamaClient, serviceConfigRepo)
          → NewHub + NewHubAlerter(hub)
          → make(chan orchestrator.ActivityEvent, 100)
          → NewOrchestrator(cfg, rules, queueRepo, msgRepo, nil, eventCh, alerter)
          → NewQueueProcessor(...)
          → orch.Start(ctx)
          → queueProcessor.Start(ctx)
          → go publishEventsToHub(eventCh, hub)    [sole consumer of eventCh]
          → registerWatchersFromDB(ctx, orch, serviceConfigRepo)
              → enabled Slack accounts → SlackWebClient → SlackWatcher → orch.AddWatcher("slack:"+WorkspaceID)
              → enabled Email accounts → IMAPClient → EmailWatcher → orch.AddWatcher("email:"+Username)
              → errors logged via slog, never fail composition
  → go comp.HTTP.Start()
  → wait on SIGINT/SIGTERM
  → comp.Shutdown(ctx) [ordered, logged, idempotent]
```

### Event flow

```
Orchestrator / QueueProcessor
  → comp.EventCh (chan ActivityEvent, buf 100)
    → publishEventsToHub goroutine
      → hub.Publish(ActivityData{Source, Message, IsError})
        → type: "activity" envelope
          → ring buffer (500 events)
          → broadcast to WS subscribers

HubAlerter.PlayNotification(ctx)
  → hub.PublishAlert(AlertData{Kind: "notification"})
    → type: "alert" envelope
      → ring buffer + WS broadcast
```

Both envelope types share the same monotonic `seq` counter, so clients can process them in strict order.

## API Changes

### `internal/server/envelope.go`

```go
// AlertData is the payload for type="alert" envelopes.
type AlertData struct {
    Kind string `json:"kind"`  // "notification" for generic orchestrator alerts
}
```

### `internal/server/hub.go`

```go
// PublishAlert emits a type="alert" envelope with AlertData payload.
func (h *Hub) PublishAlert(data AlertData) ActivityEnvelope
```

Internally refactored to share a `publishEnvelope(envelopeType, data)` helper between `Publish` (type="activity") and `PublishAlert`.

### `internal/server/alerter.go` (new)

```go
// HubAlerter is an orchestrator.Alerter that broadcasts alerts to WS clients.
type HubAlerter struct { /* ... */ }
func NewHubAlerter(hub *Hub) *HubAlerter
func (a *HubAlerter) PlayNotification(ctx context.Context) error

var _ orchestrator.Alerter = (*HubAlerter)(nil)
```

### `internal/server/server.go`

```go
// Deps now accepts an optional shared Hub.
type Deps struct {
    Messages handler.MessageQuerier
    Hub      *Hub  // optional; if nil, New creates its own
}
```

### `internal/server/composition.go` (new)

```go
// Composition holds all long-lived cue-server components.
type Composition struct {
    MessageRepo        repository.MessageRepository
    QueueRepo          repository.QueueRepository
    RuleRepo           repository.RoutingRuleRepository
    ServiceConfigRepo  repository.ServiceConfigRepository
    OllamaClient       *decisionengine.OllamaClient
    VectorStore        *vector.ChromemVectorStore
    RulesEngine        *decisionengine.RulesEngine
    Hub                *Hub
    HTTP               *Server
    Alerter            *HubAlerter
    Orchestrator       *orchestrator.Orchestrator
    QueueProcessor     *orchestrator.QueueProcessor
    EventCh            chan orchestrator.ActivityEvent
    // ...
}

func NewComposition(ctx context.Context, cfg config.Config) (*Composition, error)
func (c *Composition) Shutdown(ctx context.Context) error
```

### `internal/config/config.go`

```go
// ValidateForServer runs Validate then requires [server] to be configured.
func (c *Config) ValidateForServer() error
```

## Integration Points

- `cmd/cue-server/main.go` — rewritten to use `NewComposition` + `Shutdown` + `ValidateForServer`. Slimmed from ~90 lines of inline wiring to ~85 lines of thin boot/shutdown.
- `cmd/cue/main.go` — **no changes**. Abandoned until Feature 107.
- `internal/server/server.go` — `Deps.Hub` field added; `New` uses it when non-nil, preserving back-compat.
- No new config sections. Existing `[server]`, `[ollama]`, `[orchestrator]`, `[database]` are all required.

## Testing

11 new tests across 10 TDD micro-loops:

| Test | Purpose |
|---|---|
| `TestHub/TestPublishAlertEmitsAlertEnvelopeWithMonotonicSeq` | B1 — Hub.PublishAlert stores/broadcasts type="alert" envelopes |
| `TestAlerter/TestPlayNotificationPublishesAlertEnvelopes` | B2 — HubAlerter publishes via hub, satisfies orchestrator.Alerter |
| `TestComposition/TestNewCompositionOpensRepositories` | B3 — message, queue, rule, service config repos populated |
| `TestComposition/TestNewCompositionConstructsServices` | B4 — Ollama client, vector store, rules engine populated |
| `TestComposition/TestNewCompositionStartsOrchestrator` | B5 — Hub, Alerter, Orchestrator, QueueProcessor, EventCh populated |
| `TestComposition/TestNewCompositionBuildsWatchersFromDB` | B6 — seeded Slack + Email accounts produce matching watcher names |
| `TestComposition/TestNewCompositionPublishesOrchestratorEventsToHub` | B7 — events sent on EventCh reach hub.History |
| `TestComposition/TestCompositionShutdownClosesEventChannel` | B8 — Shutdown closes EventCh, is idempotent |
| `TestConfig/TestValidateForServerRejectsMissingServer` | B9 — ValidateForServer rejects zero-value [server] |
| `TestCompositionIntegration/TestCompositionEndToEndActivityBroadcast` | B10 — WS client receives envelope after orchestrator emits event |

All tests run under `just test` (no new just target). The end-to-end test uses `httptest.NewServer` + `coder/websocket`.

## Error Handling

| Condition | Behavior |
|---|---|
| `[server]` section missing | `ValidateForServer` returns error mentioning "server configuration required" |
| SQLite open fails | `NewComposition` returns wrapped error; binary exits with non-zero |
| Ollama unreachable at boot | No-op — construction is pure; queue processor retries on next tick with BUFFERED fallback (IS=7, CS=0) |
| Slack/Email account credentials invalid at boot | `registerWatchersFromDB` logs warning via `slog`, skips that account, continues |
| `ListSlackAccounts` / `ListEmailAccounts` errors | Logged, skipped — composition succeeds with zero watchers for that source |
| Orchestrator.Stop error during shutdown | Logged via `slog.Error`, shutdown continues (don't block DB close) |
| HTTP.Shutdown error during shutdown | Logged + captured in `shutdownErr` return value |
| DB close error during shutdown | Wrapped + captured in `shutdownErr` return value |
| `Shutdown` called twice | Idempotent via `sync.Once`; second call returns the cached result |

## Wiring Verification

Post-implementation verification performed per CLAUDE.md §17:

- ✅ No `ErrNotImplemented` remaining in 099A production code.
- ✅ No empty function bodies in new code.
- ✅ `cmd/cue-server/main.go` instantiates `Composition` and passes it to consumer (SIGINT handler, shutdown).
- ✅ `HubAlerter` has concrete implementation AND is injected into orchestrator + queue processor.
- ✅ Event channel producers (orchestrator, queueProcessor) and consumer (`publishEventsToHub` goroutine) both connected.
- ✅ Shared Hub instance threaded through `server.New` via `Deps.Hub` so WS subscribers see the same hub the publisher writes to.

## TDD Agent Stats

See `docs/agent-log.md` under `Phase-9-Feature-099A`.

## Follow-ups

- **Feature 107** — `cmd/cue` becomes a thin HTTP/WebSocket client consuming `/api/v1/*` REST + the event stream.
- **Feature 102** — Service Configuration API builds on `comp.Orchestrator.AddWatcher` / `RemoveWatcher` for hot-add/remove of accounts.
- **Few-shot calibration wiring** — `QueueProcessor.SetFewShotProvider` path from `cmd/cue` has not been ported to cue-server. If calibration is enabled in cue-server's config, the queue processor currently runs without examples. Intentionally deferred — revisit after Feature 107 when `cmd/cue-server` owns the full runtime.
- **`cfg.Ollama` model validation** — `cmd/cue` calls `decisionengine.ValidateOllamaModels` at startup to verify the inference + embedding models are pulled. `cue-server` does not; the queue processor's runtime fallback (IS=7, CS=0 → BUFFERED) covers unreachable Ollama gracefully. Consider adding an optional readiness probe in a follow-up if operators want early fail-fast.
