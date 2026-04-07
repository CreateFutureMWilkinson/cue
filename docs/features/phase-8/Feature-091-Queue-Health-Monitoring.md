# Feature 091: Queue Health Monitoring

**Phase:** Phase-8-Feature-091
**Status:** Planned
**Packages:** `internal/service/orchestrator/`, `internal/ui/presenter/`
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

## Configuration

```toml
[orchestrator]
# Warning threshold for Ollama queue depth.
queue_warning_threshold = 50
```

## Queue Depth in UI

The Rules tab in Settings (Feature 089) displays the current queue depth as a header indicator. This feature provides the data; the UI rendering is part of Feature 089.

## Continuous Growth Detection (Future)

A more sophisticated approach could track queue depth over time and detect trends:

- If depth increases for 3 consecutive poll cycles, escalate the warning
- If depth stabilizes or decreases, suppress warnings

This is deferred — the simple threshold check is sufficient initially.
