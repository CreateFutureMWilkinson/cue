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

The rules engine returns lowercase action strings (`"notified"`, `"ignored"`, `"queue"`). The orchestrator maps these to capitalized message statuses (`"Notified"`, `"Ignored"`, `"Pending"`).

```
PollOnce:
    for each watcher:
        msgs = watcher.Poll()
        for each msg:
            if repo.ExistsByMessageID(ctx, msg.MessageID):
                skip (already seen)
                continue
            action, matchedRule = rulesEngine.Evaluate(msg)
            switch action:
                case "notified":
                    msg.Status = "Notified"
                    msg.ImportanceScore = 8.0  // deterministic rule default
                    msg.ConfidenceScore = 1.0
                    msg.Reasoning = "Deterministic rule: " + matchedRule summary
                    repo.Insert(ctx, msg)
                    alerter.PlayNotification()
                case "ignored":
                    msg.Status = "Ignored"
                    msg.ImportanceScore = 0.0
                    msg.ConfidenceScore = 1.0
                    msg.Reasoning = "Deterministic rule: " + matchedRule summary
                    repo.Insert(ctx, msg)
                case "queue":
                    msg.Status = "Pending"
                    repo.Insert(ctx, msg)
                    queue.Enqueue(ctx, msg.ID)
        emit summary event
```

## New Repository Method

Add to the `MessageRepository` interface in `internal/repository/message.go`:

```go
// ExistsByMessageID checks if a message with the given source-native ID already exists.
ExistsByMessageID(ctx context.Context, messageID string) (bool, error)
```

Implementation in `internal/repository/implementation/sqlite/message_impl.go`:

```sql
SELECT EXISTS(SELECT 1 FROM messages WHERE message_id = ?)
```

This replaces the current approach of attempting Insert and silently ignoring duplicate errors.

## New Message Statuses

This feature introduces one new status, and Feature 088 introduces another:

| Status | Meaning | Set by |
|--------|---------|--------|
| **Pending** | Awaiting Ollama scoring in the queue | Orchestrator (this feature) |
| **Imported** | Baseline message from startup import, never scored | Orchestrator (Feature 088) |

These join the existing statuses: Notified, Buffered, Ignored, Resolved. The `"Pending"` status replaces what was implicitly handled by the old `RouteBatch` flow (messages were scored immediately, so no pending state existed).

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

Note: the current `Router.applyDeterministicRules()` method hardcodes channel_join (IS=9) and @mention (IS=8) logic. After this refactor, those rules are DB-backed (Feature 090 seeds the defaults) and evaluated by the RulesEngine. The IS=8.0 default for rule-notified messages is a simplification — all rule-matched notifications get the same score since the rules represent user certainty.

## Router Deprecation

The `Router.RouteBatch()` and `Router.Route()` methods are no longer called from `PollOnce`. The orchestrator handles deterministic rules via the `RulesEngine`, and Ollama scoring is handled by the `QueueProcessor` (Feature 086) which calls `scorer.Score()` directly.

The Router's responsibilities are now split:
- **Deterministic rules** → `RulesEngine` (Feature 085), evaluated in the orchestrator
- **Ollama scoring** → `QueueProcessor` (Feature 086), calls `Scorer` interface directly
- **Status assignment** → `QueueProcessor` (threshold comparison after scoring)
- **Vector calibration** → `QueueProcessor` (Feature 094 adds `FewShotProvider`)

The `Router` struct, `Route()`, and `RouteBatch()` should be removed as part of this feature. The `applyDeterministicRules()` logic is superseded by DB-backed rules (Feature 090). The `assignStatus()` logic moves to the `QueueProcessor`. The `advisor` field and adjustment logic are removed in Feature 094.

The `Scorer` interface and `OllamaClient` remain unchanged — they are used directly by the `QueueProcessor`.

## Activity Events

Emit structured events for observability:

- `"fetched N messages, M new"` (after dedup)
- `"rules: N notified, M ignored, P queued"` (after rule evaluation)
- `"queue depth: N pending"` (after enqueue)
- When constructing the `RulesEngine`, any rules with invalid regex patterns are silently excluded with an `slog.Warn` in the engine itself. The orchestrator should surface these as `ActivityEvent` warnings (one per invalid rule) so the user sees them in the activity log. This only happens at construction time (startup or `RefreshRules()`), not per-evaluation.
