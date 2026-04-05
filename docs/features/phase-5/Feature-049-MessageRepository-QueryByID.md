# Feature 049: MessageRepository QueryByID

**Phase:** Phase-5-Feature-049
**Status:** Planned
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

## Files

| File | Action |
|---|---|
| `internal/repository/message.go` | **Modify** — add `QueryByID` to interface, add `ErrNotFound` sentinel |
| `internal/repository/implementation/sqlite/message_impl.go` | **Modify** — implement `QueryByID` |
| `internal/repository/implementation/sqlite/message_impl_test.go` | **Modify** — add `QueryByID` tests (test already exists as `TestInsertAndQueryByID` but may need interface alignment) |
