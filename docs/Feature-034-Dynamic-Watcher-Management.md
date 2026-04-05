# Feature 034: Dynamic Watcher Management

**Phase:** Phase-4-Feature-034
**Status:** Planned
**Package:** `internal/service/orchestrator/`
**Depends on:** None (parallel with Features 031-033)

---

## Overview

Modify the orchestrator to support dynamic watcher addition and removal at runtime. Currently, the orchestrator requires at least one watcher at construction time and the watcher set is immutable. This change allows the app to start with zero watchers and add/remove them as users configure accounts through the Settings UI.

## Design Decisions

### Remove Zero-Watchers Guard

Current code at `orchestrator.go:63`:
```go
if len(watchers) == 0 {
    return nil, fmt.Errorf("watchers must not be empty")
}
```

This guard is removed. The constructor accepts `nil` or empty watcher maps. `PollOnce` with zero watchers is a no-op that emits an informational activity event.

### RWMutex for Watcher Map

The `o.watchers` map is currently accessed only during `PollOnce` (which runs sequentially in a goroutine). With dynamic add/remove from the UI thread, concurrent access is possible. Solution:

- `PollOnce` takes a read lock to snapshot the watcher map
- `AddWatcher`/`RemoveWatcher` take a write lock
- Use `sync.RWMutex` (separate from existing `o.mu` which guards `stopped`)

### WatcherManager Interface

Extract an interface so the presenter layer can manage watchers without importing the full orchestrator:

```go
type WatcherManager interface {
    AddWatcher(name string, w Watcher)
    RemoveWatcher(name string)
    ListWatcherNames() []string
}
```

The `Orchestrator` struct satisfies this interface. The presenter depends on the interface, not the concrete type.

## API

### New Methods on Orchestrator

```go
// AddWatcher registers a named watcher. If a watcher with the same name exists, it is replaced.
func (o *Orchestrator) AddWatcher(name string, w Watcher)

// RemoveWatcher removes a named watcher. No-op if name doesn't exist.
func (o *Orchestrator) RemoveWatcher(name string)

// ListWatcherNames returns the names of all registered watchers.
func (o *Orchestrator) ListWatcherNames() []string
```

### Constructor Change

```go
// Before
func NewOrchestrator(cfg OrchestratorConfig, router BatchRouter, repo MessageRepository,
    watchers map[string]Watcher, eventCh chan<- ActivityEvent, alerter Alerter) (*Orchestrator, error)

// After — watchers parameter becomes optional (nil/empty allowed)
func NewOrchestrator(cfg OrchestratorConfig, router BatchRouter, repo MessageRepository,
    watchers map[string]Watcher, eventCh chan<- ActivityEvent, alerter Alerter) (*Orchestrator, error)
```

Same signature, but the `len(watchers) == 0` check is removed.

### PollOnce Behavior with Zero Watchers

When `len(o.watchers) == 0`, `PollOnce` emits an activity event: `"No watchers configured"` and returns without error. This is not an error condition — the user simply hasn't configured any accounts yet.

## Error Handling

| Scenario | Behavior |
|---|---|
| `AddWatcher` with nil watcher | No-op or panic (defensive — callers must validate) |
| `RemoveWatcher` unknown name | No-op, no error |
| `PollOnce` with zero watchers | Informational activity event, return nil |

## Integration Points

- **Feature 036** (Settings Presenter): Calls `AddWatcher`/`RemoveWatcher` via `WatcherManager` interface when user adds/removes accounts
- **Feature 038** (Main Wiring): Creates orchestrator with zero watchers, then adds watchers from DB accounts

## Test Coverage

New test cases added to existing orchestrator test suite:

- Construct orchestrator with nil watchers map — succeeds
- Construct orchestrator with empty watchers map — succeeds
- `PollOnce` with zero watchers — no error, emits activity event
- `AddWatcher` then `PollOnce` — watcher gets polled
- `AddWatcher` with duplicate name — replaces existing
- `RemoveWatcher` then `PollOnce` — removed watcher not polled
- `RemoveWatcher` unknown name — no-op
- `ListWatcherNames` — returns current names
- Concurrent `AddWatcher`/`PollOnce` — no data race (run with `-race`)

## Files

| File | Action |
|---|---|
| `internal/service/orchestrator/orchestrator.go` | Modify — remove guard, add RWMutex, add AddWatcher/RemoveWatcher/ListWatcherNames |
| `internal/service/orchestrator/orchestrator_test.go` | Modify — update existing tests, add new dynamic management tests |
