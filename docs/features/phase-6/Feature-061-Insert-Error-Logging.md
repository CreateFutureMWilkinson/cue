# Feature 061: Database Insert Error Logging

**Phase:** Phase-6-Feature-061
**Type:** Bugfix
**Severity:** Low
**Status:** Planned
**Packages:** `internal/service/orchestrator/`
**Related:** Feature 007 (Orchestrator)

---

## Bug Description

When `repo.Insert()` fails for a routed message, the error is silently discarded with `continue`. No event is emitted to the activity log and no log entry is written, making database insertion failures completely invisible.

## Expected Behavior

Failed inserts should emit a warning event to the activity log (consistent with how poll errors and routing errors are already handled in the orchestrator) so users and operators can see that messages are being lost.

## Actual Behavior

`orchestrator.go:155-159`:
```go
for _, msg := range routed {
    if err := o.repo.Insert(ctx, msg); err != nil {
        continue  // Silent failure
    }
}
```

## Root Cause

Error handling for the insert loop was not implemented — likely an oversight during initial Feature 007 implementation.

## Proposed Fix

Emit an activity event on insert failure, consistent with existing error event patterns in the orchestrator:

```go
if err := o.repo.Insert(ctx, msg); err != nil {
    o.emitEvent(ActivityEvent{
        Type:    "error",
        Message: fmt.Sprintf("failed to store %s message: %v", msg.Source, err),
    })
    continue
}
```

## Test Strategy

- RED: Test that a failing repository emits an error activity event during the insert loop
- GREEN: Add the event emission
- REFACTOR: Clean up if needed
