# Feature 047: MessageType SQLite Persistence

**Phase:** Phase-5-Feature-047
**Status:** Planned
**Packages:** `internal/repository/implementation/sqlite/`
**Depends on:** —

---

## Overview

Add `message_type` to the SQLite schema and wire it through `Insert`, `Update`, and `scanMessage` in the `SQLiteMessageRepository`. The `Message.MessageType` field (defined in `internal/repository/message.go:18`) exists on the struct but is not persisted — the SQLite `messages` table has no `message_type` column, and the field is omitted from all SQL queries.

## Motivation

The router's deterministic rule for `channel_join` detection (`router.go:109`) checks `msg.MessageType == "channel_join"`. This works at initial routing time because the watcher sets the field in memory. However, the type is lost when the message is stored and later retrieved from SQLite — any code that loads persisted messages and re-checks `MessageType` will find an empty string.

This becomes relevant for:
- Activity log displaying message type icons
- Re-routing after config threshold changes
- Analytics and reporting on message type distribution
- Debugging and audit trails

## Design Decisions

### Schema Migration

Add the column with a default empty string to avoid breaking existing databases:

```sql
ALTER TABLE messages ADD COLUMN message_type TEXT NOT NULL DEFAULT '';
```

Run this migration at startup after table creation. `ALTER TABLE ... ADD COLUMN` is idempotent-safe when wrapped in a check for column existence, or by catching the "duplicate column" error.

### Column Position

Add `message_type` after `message_id` in `messageColumnsStr` to keep related fields together:

```go
const messageColumnsStr = "id, source, source_account, channel, sender, message_id, message_type, " +
    "raw_content, importance_score, confidence_score, status, reasoning, " +
    "user_rating, user_feedback, vector_id, created_at, updated_at, resolved_at"
```

### All CRUD Paths

- **Insert**: Include `message_type` in the INSERT statement
- **Update**: Include `message_type` in the UPDATE statement
- **scanMessage**: Scan `message_type` into `msg.MessageType`

## Error Handling

| Scenario | Behavior |
|---|---|
| Migration fails | Log error, exit (schema integrity required) |
| Existing rows with empty message_type | Acceptable — historical messages have no type, functions that check type handle empty string |

## Integration Points

- **Feature 003** (Deterministic Routing): `channel_join` check now survives persist/reload
- **Feature 005/006** (Watchers): Set `MessageType` on messages — already done, now persisted
- **Activity log**: Can display message type in UI

## Test Coverage

- Insert message with `MessageType` set → query returns same value
- Insert message with empty `MessageType` → query returns empty string
- Update message `MessageType` → query returns updated value
- Migration on fresh database succeeds
- Migration on existing database (no column) adds column
- Migration on already-migrated database is idempotent

## Files

| File | Action |
|---|---|
| `internal/repository/implementation/sqlite/message_impl.go` | **Modify** — add `message_type` to schema, columns, Insert, Update, scanMessage, migration |
| `internal/repository/implementation/sqlite/message_impl_test.go` | **Modify** — add MessageType round-trip tests |
