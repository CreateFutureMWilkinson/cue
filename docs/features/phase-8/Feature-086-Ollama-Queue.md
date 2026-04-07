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
    queue    QueueRepository
    messages MessageRepository
    scorer   Scorer
    alerter  Alerter
    cooldown time.Duration
    eventCh  chan<- ActivityEvent
}

func (p *QueueProcessor) Start(ctx context.Context)
func (p *QueueProcessor) Stop()
```

### Processing Loop

```
loop:
    entry := queue.DequeueOldest()
    if entry == nil:
        sleep(cooldown)  // nothing to process, back off
        continue
    msg := messages.QueryByID(entry.MessageID)
    result := scorer.Score(ctx, msg)
    if error:
        msg.Status = "Buffered"  // safe default
        queue.MarkFailed(entry.ID)
    else:
        apply score + status to msg
        messages.Update(msg)
        queue.MarkDone(entry.ID)
        if status == "Notified":
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
