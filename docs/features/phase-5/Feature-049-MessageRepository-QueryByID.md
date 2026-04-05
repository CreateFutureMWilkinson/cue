# Feature 049: MessageRepository QueryByID

**Phase:** Phase-5-Feature-049
**Status:** Done
**Packages:** `internal/repository/`, `internal/repository/implementation/sqlite/`
**Depends on:** —

---

## Overview

Add a `QueryByID` method to the `MessageRepository` interface and implement it in the SQLite adapter. This method is required by Feature 042 (Vector-Assisted Routing) to look up user ratings for messages returned by vector similarity search, and is a general-purpose gap in the repository API.

## Motivation

The `MessageRepository` interface currently supports `Insert`, `Update`, `QueryByStatus`, `QueryAll`, `QueryOldestToNewest`, and `CountBySource` — but has no way to fetch a single message by ID. The `TodoRepository` already has `QueryByID`; the message repository should match.

Feature 042's `VectorScoreAdvisor` receives message IDs from `QuerySimilar()` and needs to look up the `UserRating` field for each matched message. Without `QueryByID`, the only option is `QueryAll()` followed by a linear scan — inefficient and wasteful.

## Design Decisions

### Interface Addition

```go
// internal/repository/message.go
type MessageRepository interface {
    // ... existing methods ...
    QueryByID(ctx context.Context, id uuid.UUID) (*Message, error)
}
```

Returns `nil, ErrNotFound` when no message matches. Define a package-level sentinel error:

```go
var ErrNotFound = errors.New("repository: not found")
```

### SQLite Implementation

```go
const querySelectByID = "SELECT " + messageColumnsStr + " FROM messages WHERE id = ?"

func (r *SQLiteMessageRepository) QueryByID(ctx context.Context, id uuid.UUID) (*Message, error) {
    row := r.db.QueryRowContext(ctx, querySelectByID, id.String())
    msg, err := scanMessageRow(row)
    if errors.Is(err, sql.ErrNoRows) {
        return nil, ErrNotFound
    }
    return msg, err
}
```

## Test Coverage

- Query existing message by ID returns correct message
- Query non-existent ID returns `ErrNotFound`
- Query after insert returns the inserted message
- Query with cancelled context returns context error

## Implementation Notes

The `QueryByID` method and interface already existed (added in Feature 042). The `ErrNotFound` sentinel already existed in `internal/repository/errors.go`. This feature's scope was narrowed to:

1. Changing `QueryByID` to return `nil, repository.ErrNotFound` instead of `nil, nil` for missing IDs
2. Adding cancelled-context test coverage
3. Removing duplicate test (`TestQueryByID_UnknownID_ReturnsNil` → merged into `TestQueryByID_UnknownID_ReturnsErrNotFound`)

The `VectorScoreAdvisor` (Feature 042) already handles `err != nil` by skipping, so the behavioral change is backward-compatible.

## Files

| File | Action |
|---|---|
| `internal/repository/implementation/sqlite/message_impl.go` | **Modified** — return `ErrNotFound` instead of `nil, nil` |
| `internal/repository/implementation/sqlite/message_impl_test.go` | **Modified** — add `ErrNotFound` and cancelled-context tests, remove duplicate |

## TDD Agent Stats

| Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | test-designer | ~48s | ~23,000 | 60181fa |
| GREEN | implementer | ~38s | ~21,000 | 8f4188f |
| REFACTOR | orchestrator | manual | — | 6ad0151 |
