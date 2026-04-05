# Feature 047: MessageType SQLite Persistence

**Phase:** Phase-5-Feature-047
**Status:** Done
**Packages:** `internal/repository/implementation/sqlite/`
**Depends on:** —

---

## Overview

Add `message_type` to the SQLite schema and wire it through `Insert`, `Update`, and `scanMessage` in the `SQLiteMessageRepository`. The `Message.MessageType` field (defined in `internal/repository/message.go:18`) exists on the struct but was not persisted — the SQLite `messages` table had no `message_type` column, and the field was omitted from all SQL queries.

## Motivation

The router's deterministic rule for `channel_join` detection (`router.go:109`) checks `msg.MessageType == "channel_join"`. This works at initial routing time because the watcher sets the field in memory. However, the type was lost when the message was stored and later retrieved from SQLite — any code that loads persisted messages and re-checks `MessageType` found an empty string.

This becomes relevant for:
- Activity log displaying message type icons
- Re-routing after config threshold changes
- Analytics and reporting on message type distribution
- Debugging and audit trails

## Design Decisions

### Schema Migration

Added the column with a default empty string to avoid breaking existing databases:

```sql
ALTER TABLE messages ADD COLUMN message_type TEXT NOT NULL DEFAULT '';
```

Migration runs at startup after table creation. Idempotency is ensured by catching the "duplicate column name" error from SQLite.

### Column Position

Added `message_type` after `message_id` in `messageColumnsStr` to keep related fields together.

### All CRUD Paths

- **Insert**: `message_type` included in INSERT and ON CONFLICT SET clauses
- **Update**: `message_type` included in UPDATE SET clause
- **scanMessage**: `message_type` scanned into `msg.MessageType`

## Error Handling

| Scenario | Behavior |
|---|---|
| Migration fails (non-duplicate error) | Log error, exit (schema integrity required) |
| Existing rows with empty message_type | Acceptable — historical messages have no type, functions that check type handle empty string |

## Integration Points

- **Feature 003** (Deterministic Routing): `channel_join` check now survives persist/reload
- **Feature 005/006** (Watchers): Set `MessageType` on messages — already done, now persisted
- **Activity log**: Can display message type in UI

## Test Coverage

| Test | Description |
|---|---|
| `TestMessageTypePersisted` | Insert with `MessageType: "channel_join"` → query returns same value |
| `TestMessageTypeEmptyStringPersisted` | Insert with empty `MessageType` → query returns empty string |
| `TestMessageTypeUpdated` | Update `MessageType` from `"message"` to `"channel_join"` → query returns updated value |
| `TestMessageTypeMigrationIdempotent` | Open repo twice on same DB → second open succeeds, MessageType round-trips |

## TDD Agent Stats

| TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | test-designer | ~40s | ~30,000 | 2d2be7b |
| GREEN | implementer | ~63s | ~27,000 | 9aa37f1 |
| REFACTOR | refactorer | ~23s | ~33,000 | (no changes) |

## Files

| File | Action |
|---|---|
| `internal/repository/implementation/sqlite/message_impl.go` | **Modified** — added `message_type` to schema migration, columns, Insert, Update, scanMessage |
| `internal/repository/implementation/sqlite/message_impl_test.go` | **Modified** — added 4 MessageType round-trip tests |
