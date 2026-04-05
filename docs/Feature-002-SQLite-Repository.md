# Feature 2: SQLite Message Repository

**Phase:** Phase-1-Feature-2
**Status:** Done
**Package:** `internal/repository/` (interface), `internal/repository/implementation/sqlite/` (implementation)

---

## Overview

Pure Go SQLite storage for message events using `modernc.org/sqlite` (no CGO). Supports CRUD operations, FIFO eviction (100 messages per source), upsert by MessageID for idempotency, and WAL mode for concurrent access. The repository is the persistence layer that all other components write to and read from.

## Design Decisions

### Pure Go SQLite (modernc.org/sqlite)

Hard constraint from CLAUDE.md: never use CGO drivers (`mattn/go-sqlite3`). The pure Go driver enables cross-compilation and eliminates C toolchain dependencies.

### WAL Mode

Write-ahead logging is enabled on initialization via `PRAGMA journal_mode=WAL`. This improves concurrent read/write performance and crash resilience — important since the orchestrator writes while the GUI reads.

### FIFO Eviction Per Source

Each source (Slack, Email) independently maintains up to 100 messages. When a 101st message arrives for a source, the oldest message for that source is deleted within the same transaction as the insert. This prevents unbounded growth while keeping sources isolated.

### Upsert by MessageID

`INSERT ... ON CONFLICT(message_id) DO UPDATE SET ...` ensures that retrying a message (e.g., after a crash mid-batch) overwrites with fresh routing scores rather than duplicating. The `message_id` column has a UNIQUE index.

## API

### Interface

```go
type MessageRepository interface {
    Insert(ctx context.Context, msg *Message) error
    Update(ctx context.Context, msg *Message) error
    QueryByStatus(ctx context.Context, status string) ([]*Message, error)
    QueryAll(ctx context.Context) ([]*Message, error)
    QueryOldestToNewest(ctx context.Context, limit int) ([]*Message, error)
    CountBySource(ctx context.Context, source string) (int, error)
}
```

### Constructor

```go
func NewSQLiteMessageRepository(dbPath string) (*SQLiteMessageRepository, error)
```

Opens or creates the database, enables WAL mode, creates the messages table and indexes (idempotent).

## Schema

```sql
CREATE TABLE IF NOT EXISTS messages (
    id TEXT PRIMARY KEY,
    source TEXT NOT NULL,
    source_account TEXT NOT NULL,
    channel TEXT NOT NULL,
    sender TEXT NOT NULL,
    message_id TEXT NOT NULL UNIQUE,
    raw_content TEXT NOT NULL,
    importance_score REAL NOT NULL,
    confidence_score REAL NOT NULL,
    status TEXT NOT NULL,
    reasoning TEXT NOT NULL DEFAULT '',
    user_rating INTEGER,
    user_feedback TEXT,
    vector_id TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    resolved_at TEXT
);
```

### Indexes

1. `idx_messages_status` — speeds QueryByStatus
2. `idx_messages_source_created` — supports FIFO eviction ordering
3. `idx_messages_message_id` (UNIQUE) — enforces MessageID uniqueness for upsert

## Error Handling

| Scenario | Behavior |
|---|---|
| Database open fails | Wrapped error, no cleanup needed |
| WAL pragma fails | Close DB, wrapped error |
| Table/index creation fails | Close DB, wrapped error |
| Insert transaction fails | Rollback, wrapped error |
| FIFO eviction fails | Wrapped error within transaction |
| Row scan/parse fails | Wrapped error with field context |

All errors use `fmt.Errorf("context: %w", err)` for wrapping.

## Integration Points

- **Orchestrator** (Feature 7): Calls `Insert()` after routing each message batch
- **GUI** (Feature 11): Calls `QueryByStatus("Notified")` for notification queue, `QueryOldestToNewest()` for feedback buffer
- **Feedback buffer** (Feature 9): Calls `Update()` to save user ratings and feedback

## Test Coverage

12 test cases in `message_impl_test.go` using testify suites:

- Database creation on disk
- Full round-trip insert + retrieve (all fields including nullables)
- Status-based query filtering
- Update with nullable field mutations
- FIFO eviction at capacity (101st message triggers eviction)
- FIFO source isolation (Slack eviction doesn't affect Email)
- Chronological ordering (QueryOldestToNewest)
- Full table scan (QueryAll with mixed statuses)
- Per-source counting (CountBySource)
- WAL mode verification
- Nullable field handling (partial updates)
- Upsert by MessageID (duplicate detection)

## TDD Agent Stats

| Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | orchestrator | — | 10,984 | fa9b574 |
| GREEN | orchestrator | — | 12,589 | 30e317e |
| REFACTOR | orchestrator | — | — | 2ae7b0c |
