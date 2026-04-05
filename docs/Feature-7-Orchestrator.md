# Feature 7: Router Orchestration

**Phase:** Phase-1-Feature-7
**Status:** Done
**Package:** `internal/service/orchestrator/`

---

## Overview

Central coordinator that drives the poll → route → store cycle. Manages multiple source watchers (Slack, Email) with per-source goroutines, performs an immediate first poll on startup, runs on configurable intervals, emits activity events for the UI, and shuts down gracefully with idempotent Stop().

## Design Decisions

### Sequential Watchers Within a Single Goroutine

Each watcher is processed serially within a single polling goroutine rather than spawning per-watcher goroutines. This simplifies synchronization and ensures predictable ordering of activity events. The batch processing latency target (<500ms p95) is achievable with sequential execution.

### Immediate First Poll

`Start()` executes `PollOnce()` immediately before entering the ticker loop. This ensures the user sees results on application startup rather than waiting for the first interval to elapse.

### Store Errors Don't Abort Batch

If `repo.Insert()` fails for one message, the remaining messages in the batch are still stored. This prevents a single database hiccup from losing an entire batch of routed messages.

### Idempotent Shutdown

`Stop()` is protected by a mutex and a `stopped` flag. Multiple calls are safe — the second and subsequent calls return immediately. This prevents double-cancel panics and simplifies cleanup in defer chains.

### Activity Events via Channel

Events are sent on a `chan<- ActivityEvent` rather than logged directly. This decouples the orchestrator from the UI — the activity log pane reads from the channel while the orchestrator writes to it.

## API

### Interfaces

```go
type Watcher interface {
    Poll(ctx context.Context) ([]*repository.Message, error)
}

type BatchRouter interface {
    RouteBatch(ctx context.Context, msgs []*repository.Message) ([]*repository.Message, error)
}
```

### Types

```go
type ActivityEvent struct {
    Source  string // "slack", "email"
    Message string // Human-readable event description
    IsError bool   // Error events shown differently in UI
}

type OrchestratorConfig struct {
    PollIntervalSeconds int
}
```

### Constructor

```go
func NewOrchestrator(cfg OrchestratorConfig, router BatchRouter, repo MessageRepository,
    watchers map[string]Watcher, eventCh chan<- ActivityEvent) (*Orchestrator, error)
```

Requires non-nil router, repo, and non-empty watchers map.

### Methods

```go
func (o *Orchestrator) PollOnce(ctx context.Context)  // Single poll cycle
func (o *Orchestrator) Start(ctx context.Context)      // Background polling loop
func (o *Orchestrator) Stop()                           // Graceful shutdown
```

## Poll Cycle Flow

For each watcher (sequentially):

1. `watcher.Poll(ctx)` — fetch new messages
2. On error: emit error event, skip to next watcher
3. Emit fetch event: "fetched N messages"
4. `router.RouteBatch(ctx, msgs)` — route entire batch
5. On error: emit error event, skip to next watcher
6. For each routed message: `repo.Insert(ctx, msg)` (errors silently skipped)
7. Emit summary: "Routed X NOTIFIED, Y BUFFERED, Z IGNORED"

## Activity Events

| Event | Source | Message | IsError |
|---|---|---|---|
| Fetch complete | watcher name | "fetched N messages" | false |
| Route complete | watcher name | "Routed X NOTIFIED, Y BUFFERED, Z IGNORED" | false |
| Poll error | watcher name | "poll error: ..." | true |
| Route error | watcher name | "routing error: ..." | true |

Store failures are silent — no event emitted.

## Error Handling

| Scenario | Behavior |
|---|---|
| Watcher poll fails | Emit error event, skip batch, continue next watcher |
| Router batch fails | Emit error event, skip storage, continue next watcher |
| Single message store fails | Skip message, continue storing remaining |
| Context cancelled | Exit polling loop gracefully |
| Multiple Stop() calls | Second+ calls return immediately |

## Integration Points

- **Watchers** (Features 5, 6): Consumed via the Watcher interface
- **Router** (Feature 3): Consumed via the BatchRouter interface
- **Repository** (Feature 2): Consumed via MessageRepository for persistence
- **GUI** (Feature 11): Activity events consumed by the activity log pane

## Test Coverage

10 test cases in `orchestrator_test.go` using testify suites:

- Constructor validation (3): nil router, nil repo, nil/empty watchers
- Single poll cycle (1): routes and stores messages
- Activity events (1): correct event counts and status summary
- Error handling (2): watcher error doesn't crash, store error doesn't abort batch
- Multiple watchers (1): separate batches per watcher
- Lifecycle (2): start/stop with idempotent shutdown, immediate first poll

## TDD Agent Stats

| Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | Test Designer | 135s | 24,332 | 2dd21b4 |
| GREEN | Implementer | 45s | 25,667 | 9900806 |
| REFACTOR | Refactorer | 60s | 28,182 | 8aa1d7e |
