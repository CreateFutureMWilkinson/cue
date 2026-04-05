# Feature 032: SQLite ServiceConfig Implementation

**Phase:** Phase-4-Feature-032
**Status:** Planned
**Package:** `internal/repository/implementation/sqlite/`
**Depends on:** Feature 031

---

## Overview

Implement the `ServiceConfigRepository` interface (Feature 031) using SQLite. Creates `slack_accounts` and `email_accounts` tables with full CRUD operations. Follows the existing `message_impl.go` pattern — same DB handle, table creation in constructor, WAL mode.

## Design Decisions

### Typed Tables

Each service type gets a dedicated table rather than a single generic config table. This provides:
- Column-level type constraints (INTEGER for ports, TEXT for tokens)
- UNIQUE constraints on natural keys (workspace_id, username)
- Clean queries without type-casting or JSON parsing

### Shared DB Handle

The constructor accepts the same `*sql.DB` used by `SQLiteMessageRepository`. Both repositories share the database file. Table creation is idempotent (`CREATE TABLE IF NOT EXISTS`).

### Upsert Semantics

`UpsertSlackAccount` and `UpsertEmailAccount` use `INSERT ... ON CONFLICT(id) DO UPDATE`. This supports both initial creation and subsequent edits through a single method.

## Schema

```sql
CREATE TABLE IF NOT EXISTS slack_accounts (
    id TEXT PRIMARY KEY,
    enabled INTEGER NOT NULL DEFAULT 1,
    bot_token TEXT NOT NULL,
    workspace_id TEXT NOT NULL UNIQUE,
    poll_interval_seconds INTEGER NOT NULL DEFAULT 600,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS email_accounts (
    id TEXT PRIMARY KEY,
    enabled INTEGER NOT NULL DEFAULT 1,
    imap_host TEXT NOT NULL,
    imap_port INTEGER NOT NULL DEFAULT 993,
    username TEXT NOT NULL UNIQUE,
    password_env TEXT NOT NULL,
    poll_interval_seconds INTEGER NOT NULL DEFAULT 600,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
```

### Indexes

1. `slack_accounts(workspace_id)` UNIQUE — prevents duplicate workspace entries
2. `email_accounts(username)` UNIQUE — prevents duplicate email account entries

## API

### Constructor

```go
func NewSQLiteServiceConfigRepository(db *sql.DB) (*SQLiteServiceConfigRepository, error)
```

Creates tables if they don't exist. Returns wrapped error on failure.

### Methods

All methods from `ServiceConfigRepository` interface (Feature 031):
- `ListSlackAccounts`, `GetSlackAccount`, `UpsertSlackAccount`, `DeleteSlackAccount`
- `ListEmailAccounts`, `GetEmailAccount`, `UpsertEmailAccount`, `DeleteEmailAccount`

## Error Handling

| Scenario | Behavior |
|---|---|
| Table creation fails | Wrapped error, constructor returns nil |
| Get with unknown ID | Return `repository.ErrNotFound` |
| Upsert with duplicate natural key (workspace_id/username) | SQLite UNIQUE constraint error, wrapped |
| Delete with unknown ID | No-op, nil error |
| Scan/parse failure | Wrapped error with field context |

## Integration Points

- **Feature 031** (Interface): Implements `ServiceConfigRepository`
- **Feature 036** (Settings Presenter): Called for account CRUD operations
- **Feature 038** (Main Wiring): Instantiated in `main.go`, queries at startup

## Test Coverage

Following `message_impl_test.go` pattern (testify suite, temp directory DB):

- Table creation on fresh DB
- Slack account full round-trip (upsert + get + list)
- Email account full round-trip (upsert + get + list)
- Update existing account (upsert overwrites)
- Delete account + verify removal
- Delete unknown ID (no-op)
- Get unknown ID (ErrNotFound)
- List empty tables (returns empty slice, not nil)
- Unique constraint enforcement (duplicate workspace_id / username)
- Multiple accounts per type (multi-account)
- Enabled/disabled filtering in list (if applicable)

## Files

| File | Action |
|---|---|
| `internal/repository/implementation/sqlite/service_config_impl.go` | **New** — SQLite implementation |
| `internal/repository/implementation/sqlite/service_config_impl_test.go` | **New** — full test suite |
