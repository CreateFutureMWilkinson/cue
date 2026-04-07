# Feature 087: Orchestrator Refactor

**Phase:** Phase-8-Feature-087
**Status:** Planned
**Packages:** `internal/service/orchestrator/`
**Depends on:** Features 085, 086

---

## Overview

Refactor the orchestrator's `PollOnce` to use the new two-stage pipeline: dedup check → deterministic rules → queue. Replace the current batch `RouteBatch` call that sends all messages through Ollama in one go.

## Current Flow (to be replaced)

```
PollOnce:
    for each watcher:
        msgs = watcher.Poll()
        routed = router.RouteBatch(msgs)   ← ALL go through Ollama
        for each routed msg:
            repo.Insert(msg)               ← fails silently on duplicate message_id
```

## New Flow

```
PollOnce:
    for each watcher:
        msgs = watcher.Poll()
        for each msg:
            if repo.ExistsByMessageID(msg.MessageID):
                skip (already seen)
                continue
            action = rulesEngine.Evaluate(msg)
            switch action:
                case "notified":
                    msg.Status = "Notified"
                    msg.ImportanceScore = 8.0  // deterministic rule default
                    msg.ConfidenceScore = 1.0
                    repo.Insert(msg)
                    alerter.PlayNotification()
                case "ignored":
                    msg.Status = "Ignored"
                    repo.Insert(msg)
                case "queue":
                    msg.Status = "Pending"
                    repo.Insert(msg)
                    queue.Enqueue(msg.ID)
        emit summary event
```

## New Repository Method

```go
// ExistsByMessageID checks if a message with the given source-native ID already exists.
ExistsByMessageID(ctx context.Context, messageID string) (bool, error)
```

This replaces the current approach of attempting Insert and silently ignoring duplicate errors.

## Rules Engine Integration

The orchestrator holds a `RulesEngine` that is reconstructed when rules change:

```go
type Orchestrator struct {
    // ... existing fields ...
    rulesEngine *decisionengine.RulesEngine
    queue       repository.QueueRepository
}
```

The rules engine is rebuilt from the DB at startup and when the Settings UI modifies rules (via a `RefreshRules()` method).

## Scoring Assignment for Deterministic Rules

Messages routed by deterministic rules get fixed scores (not Ollama-scored):

- NOTIFIED by rule: `IS = 8.0, CS = 1.0` (high confidence, deterministic)
- IGNORED by rule: `IS = 0.0, CS = 1.0`

The `Reasoning` field records which rule matched: `"Deterministic rule: [field] matches [pattern]"`.

## Batch RouteBatch Removal

The `Router.RouteBatch` method is no longer called from `PollOnce`. It may be retained for direct Ollama scoring (used by the queue processor) or deprecated.

## Activity Events

Emit structured events for observability:

- `"fetched N messages, M new"` (after dedup)
- `"rules: N notified, M ignored, P queued"` (after rule evaluation)
- `"queue depth: N pending"` (after enqueue)
