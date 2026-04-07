# Feature 084: Deterministic Routing Rules + Ollama Queue

**Phase:** Phase-8-Feature-084
**Status:** Planned
**Packages:** `internal/service/decisionengine/`, `internal/service/orchestrator/`, `internal/repository/`, `internal/ui/`
**Depends on:** Features 003, 007, 031, 032, 037

---

## Overview

Replace the current batch-score-everything approach with a two-stage routing pipeline: configurable deterministic rules run first, and only unmatched messages trickle through Ollama one at a time via a persistent FIFO queue. This prevents GPU/CPU overload from large batch scoring and gives users direct control over routing decisions.

### Problem

The current orchestrator polls all watchers every 10 minutes, then sends every fetched message through Ollama for LLM scoring in a single batch. With real-world email volumes (42+ messages per poll), this causes:

- Extended 100% GPU/CPU load (potentially 7+ minutes of continuous Ollama inference)
- PC overheating and crashes (observed)
- Re-scoring of already-seen messages on restart (watcher `lastUID` resets to 0)
- No user control over routing — all decisions delegated to LLM

### Solution

1. **Deterministic rules first** — user-configurable regex rules that match message fields and immediately route to NOTIFIED or IGNORED
2. **FIFO queue** — unmatched messages enter a persistent DB-backed queue
3. **Trickle processing** — queue is drained one message at a time with configurable cooldown between Ollama calls
4. **Startup import** — on launch, import all unseen messages as "Imported" status (record-keeping only, never scored)

## Processing Flow

```
Message arrives (from poll)
    ↓
Already in DB? (check message_id)
    → Yes: skip entirely (never re-process)
    → No: continue
    ↓
Deterministic rules (first match wins, sorted by priority)
    → NOTIFIED: store + alert user
    → IGNORED: store as record
    → No match: → enqueue for Ollama
    ↓
Queue processor (background, one at a time)
    → Dequeue oldest message
    → Score via Ollama
    → Route: NOTIFIED / BUFFERED / IGNORED
    → Wait cooldown_seconds before next
```

### Startup Behavior

On launch, before polling begins:

1. Fetch all messages from configured accounts (INBOX only for email)
2. For each message, check if `message_id` exists in DB
3. If not: insert with status "Imported" — no scoring, no routing, no alerting
4. This establishes the baseline so subsequent polls only see genuinely new messages

## Rule Model

### Schema

```go
type RoutingRule struct {
    ID       uuid.UUID
    Priority int       // 0 = highest, ascending. Controls evaluation order.
    Source   string    // "email", "slack", or "all"
    Field    string    // Field to match against (source-dependent)
    Negate   bool      // true = "not matches", false = "matches"
    Pattern  string    // Go regexp pattern
    Action   string    // "notified" or "ignored"
    Enabled  bool
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### Available Fields by Source

| Source | Field | Description |
|--------|-------|-------------|
| email | `sender` | From address |
| email | `subject` | Email subject line |
| slack | `sender` | User ID/name |
| slack | `channel` | Channel name |
| slack | `content` | Message text |
| slack | `message_type` | e.g., "channel_join" |

Rules with `source = "all"` can only match fields common to both sources (sender).

### Matching Logic

```
regex := regexp.MustCompile(rule.Pattern)
matched := regex.MatchString(fieldValue)
if rule.Negate {
    matched = !matched
}
if matched {
    return rule.Action // "notified" or "ignored"
}
```

### Actions

Only two actions are available for deterministic rules:

- **NOTIFIED** — the user is certain they want to know about this
- **IGNORED** — the user is certain they don't care

BUFFERED is exclusively an Ollama output (LLM thinks it might be important but isn't confident). Deterministic rules represent certainty, so BUFFERED doesn't apply.

### Default Rules

The existing hardcoded routing rules become default DB entries:

| Priority | Source | Field | Pattern | Negate | Action |
|----------|--------|-------|---------|--------|--------|
| 0 | slack | message_type | `^channel_join$` | false | notified |
| 1 | slack | content | `@username` | false | notified |

These are editable and deletable like any user-created rule.

### Unmatched Default

Messages not matched by any rule always go to QUEUE. This is not configurable.

## Queue Design

### Persistence

Queue is a DB table, not in-memory. Survives restarts.

```sql
CREATE TABLE ollama_queue (
    id TEXT PRIMARY KEY,
    message_id TEXT NOT NULL REFERENCES messages(id),
    enqueued_at TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending'  -- pending, processing, done, failed
);
CREATE INDEX idx_queue_status_enqueued ON ollama_queue(status, enqueued_at);
```

### Processing

- Background goroutine dequeues oldest `pending` message
- Marks as `processing`, scores via Ollama, updates message status, marks as `done`
- Waits `ollama_cooldown_seconds` (configurable in TOML, default 10) before next
- Failed Ollama calls: mark message as BUFFERED (safe default), mark queue entry as `failed`

### Queue Health Monitoring

- Track queue depth in activity log
- If queue depth exceeds threshold (configurable, default 50) after a poll cycle, emit a warning: "Ollama queue depth: N — consider adding routing rules"
- Queue depth visible in Settings UI (future: Rules tab header badge)

## Settings UI

### Rules Tab

New tab in Settings view, positioned after Calendar and before Audio:

- **Rule list**: sorted by priority, each row shows: priority, source icon, field, pattern preview, action, enabled toggle, Delete button
- **Reordering**: Up/Down buttons change priority (same pattern as wizard step 3)
- **Add Rule form**: Source dropdown, Field dropdown (updates based on source), Pattern entry, Negate checkbox, Action dropdown (Notified/Ignored)
- **Queue depth indicator**: shown at top of Rules tab

## Email: INBOX Only

The email watcher must only import from INBOX, not all folders. This is a constraint on `FetchNewMessages` — the IMAP client should be configured to only query INBOX.

## Ollama Efficiency (Future Investigation)

The current model `neural-chat` (7B) may be overkill for importance classification. Candidates for evaluation:

- `phi3:mini` (3.8B) — much faster, likely sufficient for 0-10 scoring
- `gemma2:2b` — smallest viable option
- Simplified prompts — reduce token count per request
- Structured output — constrain response format to reduce generation time

This is a separate investigation, not part of the initial implementation.

## Configuration

New TOML fields under `[orchestrator]`:

```toml
[orchestrator]
# Seconds to wait between Ollama scoring calls (thermal management).
ollama_cooldown_seconds = 10

# Warning threshold for Ollama queue depth.
queue_warning_threshold = 50
```

## Implementation Plan

### Sub-features (suggested decomposition)

1. **Rule model + DB table** — `RoutingRule` struct, SQLite DDL, CRUD operations
2. **Rules engine** — evaluate rules against a message, return first-match action
3. **Queue model + DB table** — enqueue, dequeue, status transitions
4. **Orchestrator refactor** — dedup check → deterministic rules → queue, remove batch RouteBatch
5. **Queue processor** — background goroutine, trickle Ollama scoring with cooldown
6. **Startup import** — import unseen messages as "Imported", INBOX only for email
7. **Settings UI: Rules tab** — list, add, reorder, delete
8. **Default rules migration** — seed channel_join and @mention rules on first run
9. **Queue health monitoring** — depth warnings in activity log

### Dependencies

```
1 (Rule model) → 2 (Rules engine) → 4 (Orchestrator refactor)
3 (Queue model) → 4 (Orchestrator refactor) → 5 (Queue processor)
6 (Startup import) depends on 4
7 (Rules tab UI) depends on 1
8 (Default rules) depends on 1
9 (Queue health) depends on 3, 5
```

## Migration Notes

- Existing messages in DB are unaffected — they keep their current status
- The `message_type = "mention"` detection was already removed from EmailWatcher (commit d8f3c6c)
- Existing hardcoded routing in `Router.RouteBatch` will be replaced by the rules engine + queue pipeline
