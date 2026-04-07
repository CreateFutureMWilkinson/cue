# Feature 086: Ollama Queue Model + Processor

**Phase:** Phase-8-Feature-086
**Status:** Planned
**Packages:** `internal/repository/`, `internal/repository/implementation/sqlite/`, `internal/service/orchestrator/`
**Depends on:** None (parallel with 084, 085)

---

## Overview

Implement a persistent DB-backed FIFO queue for messages that need Ollama scoring. A background processor drains the queue one message at a time with a configurable cooldown between calls, preventing GPU/CPU overload.

## Queue Model

```go
type QueueEntry struct {
    ID         uuid.UUID
    MessageID  uuid.UUID  // FK to messages table
    EnqueuedAt time.Time
    Status     string     // "pending", "processing", "done", "failed"
}
```

## Repository Interface

```go
type QueueRepository interface {
    Enqueue(ctx context.Context, messageID uuid.UUID) error
    DequeueOldest(ctx context.Context) (*QueueEntry, error)  // oldest pending → processing
    MarkDone(ctx context.Context, id uuid.UUID) error
    MarkFailed(ctx context.Context, id uuid.UUID) error
    PendingCount(ctx context.Context) (int, error)
    PurgeCompleted(ctx context.Context) error  // remove done/failed entries
}
```

`DequeueOldest` atomically sets `status = "processing"` and returns the entry. Returns nil if queue is empty.

## SQLite Schema

```sql
CREATE TABLE ollama_queue (
    id TEXT PRIMARY KEY,
    message_id TEXT NOT NULL REFERENCES messages(id),
    enqueued_at TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending'
);
CREATE INDEX idx_queue_status_enqueued ON ollama_queue(status, enqueued_at);
```

## Queue Processor

```go
type QueueProcessor struct {
    queue               QueueRepository
    messages            MessageRepository
    scorer              Scorer
    alerter             Alerter
    cooldown            time.Duration
    importanceThreshold float64
    confidenceThreshold float64
    eventCh             chan<- ActivityEvent
}

func (p *QueueProcessor) Start(ctx context.Context)
func (p *QueueProcessor) Stop()
```

The `importanceThreshold` and `confidenceThreshold` are injected from config and used for status assignment after scoring (same thresholds as the former Router). Feature 094 will add a `FewShotProvider` field to support calibration — the Scorer interface is injected, so this is a transparent enhancement.

### Processing Loop

```
loop:
    entry := queue.DequeueOldest()
    if entry == nil:
        sleep(cooldown)  // nothing to process, back off
        continue
    msg := messages.QueryByID(ctx, entry.MessageID)
    result := scorer.Score(ctx, msg)
    if error:
        msg.ImportanceScore = 7.0
        msg.ConfidenceScore = 0.0
        msg.Status = "Buffered"  // safe default
        msg.Reasoning = "Ollama scoring failed: " + error
        messages.Update(ctx, msg)
        queue.MarkFailed(entry.ID)
    else:
        msg.ImportanceScore = result.ImportanceScore
        msg.ConfidenceScore = result.ConfidenceScore
        msg.Reasoning = result.Reasoning
        // Status assignment (same thresholds as former Router.assignStatus):
        if IS >= importanceThreshold AND CS >= confidenceThreshold:
            msg.Status = "Notified"
        else if IS >= importanceThreshold:
            msg.Status = "Buffered"
        else:
            msg.Status = "Ignored"
        messages.Update(ctx, msg)
        queue.MarkDone(entry.ID)
        if msg.Status == "Notified":
            alerter.PlayNotification()
    emit activity event
    sleep(cooldown)
```

## Configuration

```toml
[orchestrator]
# Seconds to wait between Ollama scoring calls (thermal management).
ollama_cooldown_seconds = 10
```

## Failure Handling

- Ollama timeout/error: message marked BUFFERED (safe default — user reviews manually), queue entry marked `failed`
- `failed` entries are not retried automatically — user can review them in the feedback buffer
- `processing` entries from a crashed session are reset to `pending` on startup

## Relationship to Other Features

- **Feature 087** enqueues messages here after rules evaluation
- **Feature 094** enhances this processor with few-shot calibration: a `FewShotProvider` field is added, and `scorer.Score()` becomes `scorer.ScoreWithContext()` with examples. The processor's structure accommodates this — the scorer is injected via interface
