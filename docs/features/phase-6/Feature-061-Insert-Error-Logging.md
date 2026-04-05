# Feature 061: Database Insert Error Logging

**Phase:** Phase-6-Feature-061
**Type:** Bugfix
**Severity:** Low
**Status:** Done
**Packages:** `internal/service/orchestrator/`
**Related:** Feature 007 (Orchestrator)

---

## Bug Description

When `repo.Insert()` fails for a routed message, the error is silently discarded with `continue`. No event is emitted to the activity log and no log entry is written, making database insertion failures completely invisible.

## Expected Behavior

Failed inserts should emit an error event to the activity log (consistent with how poll errors and routing errors are already handled in the orchestrator) so users and operators can see that messages are being lost.

## Root Cause

Error handling for the insert loop was not implemented — likely an oversight during initial Feature 007 implementation.

## Fix

Added `emitEvent` call on insert failure, consistent with existing error event patterns in the orchestrator:

```go
if err := o.repo.Insert(ctx, msg); err != nil {
    o.emitEvent(name, fmt.Sprintf("failed to store %s message: %v", msg.Source, err), true)
    continue
}
```

The event uses `IsError: true` and includes the message source and error text, matching the pattern used for poll errors (line 142) and routing errors (line 150).

## Test Coverage

- `TestStoreErrorEmitsErrorEvent` — verifies that a failing `repo.Insert()` emits an `ActivityEvent` with `IsError: true`, source matching the watcher name, and message containing "failed to store" and the error text. Also verifies remaining messages in the batch are still stored.
- `TestStoreErrorDoesNotAbortBatch` — pre-existing test verifying batch continuation on insert failure.

## TDD Agent Stats

| TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | Test Designer | ~56s | ~25,500 | c6a8f0a |
| GREEN | Implementer | ~27s | ~20,400 | 07ad261 |
| REFACTOR | Refactorer | ~24s | ~32,900 | (no changes) |
