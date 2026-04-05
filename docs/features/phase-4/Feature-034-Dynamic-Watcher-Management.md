# Feature 034: Dynamic Watcher Management

**Phase:** Phase-4-Feature-034
**Status:** Done
**Package:** `internal/service/orchestrator/`
**Depends on:** None (parallel with Features 031-033)

---

## Overview

Modify the orchestrator to support dynamic watcher addition and removal at runtime. Previously, the orchestrator required at least one watcher at construction time and the watcher set was immutable. This change allows the app to start with zero watchers and add/remove them as users configure accounts through the Settings UI.

## Design Decisions

### Remove Zero-Watchers Guard

The constructor guard `if len(watchers) == 0 { return error }` was removed. The constructor accepts `nil` or empty watcher maps. `nil` maps are initialized to an empty map internally.

### RWMutex for Watcher Map

A dedicated `sync.RWMutex` (`watcherMu`) protects the watcher map, separate from the existing `o.mu` which guards `stopped`:

- `PollOnce` takes a read lock to snapshot the watcher map, then releases before doing work
- `AddWatcher`/`RemoveWatcher` take a write lock
- Snapshot-then-release pattern prevents blocking `AddWatcher` during long poll cycles

### WatcherManager Interface

Extracted an interface so the presenter layer can manage watchers without importing the full orchestrator:

```go
type WatcherManager interface {
    AddWatcher(name string, w Watcher)
    RemoveWatcher(name string)
    ListWatcherNames() []string
}
```

The `Orchestrator` struct satisfies this interface (compile-time check in tests).

### Non-blocking emitEvent

Changed `emitEvent` to use `select`/`default` to drop events if the channel is full. This prevents deadlock when concurrent goroutines emit events during the race-safety test.

## API

### New Methods on Orchestrator

```go
func (o *Orchestrator) AddWatcher(name string, w Watcher)
func (o *Orchestrator) RemoveWatcher(name string)
func (o *Orchestrator) ListWatcherNames() []string
```

### Constructor Change

Same signature, but `nil`/empty watcher maps are accepted. `nil` is promoted to an empty map.

### PollOnce with Zero Watchers

Emits an activity event `"No watchers configured"` (source: `"system"`, non-error) and returns immediately.

## Error Handling

| Scenario | Behavior |
|---|---|
| `AddWatcher` with nil watcher | Stores nil — callers must validate |
| `RemoveWatcher` unknown name | No-op, no error |
| `PollOnce` with zero watchers | Informational activity event, return nil |

## Integration Points

- **Feature 036** (Settings Presenter): Calls `AddWatcher`/`RemoveWatcher` via `WatcherManager` interface
- **Feature 038** (Main Wiring): Creates orchestrator with zero watchers, then adds from DB accounts

## Test Coverage

| Test Case | Description |
|---|---|
| `TestNewOrchestratorRequiresWatchers` | nil/empty watchers accepted (updated from error assertion) |
| `TestConstructorAcceptsNilWatchers` | nil watchers map → succeeds |
| `TestConstructorAcceptsEmptyWatchers` | empty watchers map → succeeds |
| `TestPollOnceZeroWatchersEmitsEvent` | Emits "No watchers configured", no error |
| `TestAddWatcherThenPoll` | Dynamically added watcher gets polled |
| `TestAddWatcherDuplicateReplaces` | Second AddWatcher replaces first |
| `TestRemoveWatcherThenPoll` | Removed watcher not polled |
| `TestRemoveWatcherUnknownNoOp` | No panic on unknown name |
| `TestListWatcherNames` | Returns sorted names, updates after add/remove |
| `TestConcurrentAddAndPoll` | Race-safe concurrent access (50 iterations, `-race`) |

### TDD Agent Stats

| TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | Test Designer | 80s | 33,087 | 39fe00a |
| GREEN | Implementer | 153s | 36,640 | 635f1df |
| REFACTOR | Refactorer | 93s | 36,203 | 5e71a54 |

## Files

| File | Action |
|---|---|
| `internal/service/orchestrator/orchestrator.go` | Modified — removed guard, added RWMutex, WatcherManager interface, AddWatcher/RemoveWatcher/ListWatcherNames, non-blocking emitEvent, maps.Copy |
| `internal/service/orchestrator/orchestrator_test.go` | Modified — updated existing test, added 9 new test cases, compile-time interface check, mustNewOrchestrator helper |
