# Feature 091: Queue Health Monitoring

**Phase:** Phase-8-Feature-091
**Status:** Done
**Packages:** `internal/service/orchestrator/`
**Depends on:** Features 086, 087

---

## Overview

Monitor the Ollama queue depth and warn the user when it grows beyond a threshold, signaling that more deterministic routing rules are needed.

## Warning Logic

After each poll cycle completes (messages deduped, rules evaluated, unmatched enqueued):

1. Query `queue.PendingCount()`
2. If count exceeds `queue_warning_threshold` (default 50):
   - Emit activity event: `"⚠ Ollama queue depth: N — consider adding routing rules"`
   - Event is visible in the activity log drawer
3. If count is at or below threshold:
   - Emit activity event: `"Ollama queue depth: N"`
4. If threshold is 0, the check is disabled entirely (no events emitted).

## Configuration

```toml
[orchestrator]
# Warning threshold for Ollama queue depth.
queue_warning_threshold = 50
```

The threshold is sourced from `config.Orchestrator.Router.QueueWarningThreshold` and passed to `orchestrator.OrchestratorConfig.QueueWarningThreshold` at construction in `main.go`.

## Queue Depth in UI

The Rules tab in Settings (Feature 089) displays the current queue depth as a header indicator. This feature provides the runtime activity log events; the UI rendering is part of Feature 089.

## Design Decisions

- **Always emit**: Both warning and ok events are emitted so the activity log shows queue status each cycle, not just when things go wrong.
- **Threshold 0 disables**: A zero threshold skips the check entirely, avoiding unnecessary DB queries if the user doesn't want monitoring.
- **Non-blocking**: Uses the existing `emitEvent` pattern which drops events on a full channel rather than blocking the poll loop.

## Error Handling

If `PendingCount()` fails, an error event is emitted (`isError: true`) and the poll cycle continues normally.

## Test Coverage

| Test | Behavior |
|---|---|
| `TestPollOnceEmitsQueueWarningWhenThresholdExceeded` | Warning event when depth > threshold |
| `TestPollOnceEmitsQueueOkWhenBelowThreshold` | Ok event when depth <= threshold |
| `TestPollOnceSkipsQueueCheckWhenThresholdZero` | No queue events when threshold = 0 |

## TDD Agent Stats

| Phase | TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|---|
| Phase-8-Feature-091 | RED | orchestrator | ~2min | ~45,000 | 7af146c |
| Phase-8-Feature-091 | GREEN | orchestrator | ~1min | ~10,000 | 16b8805 |

## Continuous Growth Detection (Future)

A more sophisticated approach could track queue depth over time and detect trends:

- If depth increases for 3 consecutive poll cycles, escalate the warning
- If depth stabilizes or decreases, suppress warnings

This is deferred — the simple threshold check is sufficient initially.
