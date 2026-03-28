# Feature 9: Feedback Buffer Service

**Phase:** Phase-1-Feature-9
**Status:** Done
**Package:** `internal/service/buffer/`

---

## Overview

Service layer that manages the feedback review workflow for BUFFERED messages (importance >= 7, confidence < 0.8). Sits between the SQLite repository (Feature 2) and the future GUI (Feature 11). Users review buffered messages oldest-first, rate them 0-10, optionally add notes, and can delete unwanted messages. Rated messages are optionally embedded via the vector store (Feature 8) for future learning.

## Design Decisions

### Consumer-Side Interfaces

The buffer package defines its own narrow interfaces rather than importing the full `repository.MessageRepository`:

- `MessageRepository` — only `QueryByStatus` and `Update` (2 of 6 methods)
- `VectorEmbedder` — only `StoreEmbedding`

This keeps the buffer package loosely coupled and easily testable.

### Optional Vector Embedder

The `VectorEmbedder` dependency is nullable. When nil, the service skips embedding entirely. When present but failing, the save still succeeds — embedding is best-effort. This aligns with CLAUDE.md Section 12: "chromem-go embedding failure → store message without vector, log, continue."

### Fetch-All-Scan for Message Lookup

`SaveRating` and `DeleteMessage` need to find a specific message by ID. Rather than adding `QueryByID` to the repository interface (which would require updating all implementations and mocks), the service fetches all buffered messages and scans linearly. With max 200 messages (100 per source), this is negligible overhead and keeps Feature 9 self-contained.

### Skip is GUI-Only

The "Skip" action advances a cursor in the review queue. Since the service is stateless (no cursor), skip requires no service method — the GUI tracks position locally.

## API

### Constructor

```go
func NewBufferService(repo MessageRepository, embedder VectorEmbedder) (*BufferService, error)
```

`repo` required (error if nil). `embedder` optional (nil OK).

### Methods

```go
// All buffered messages, sorted oldest-first by CreatedAt.
func (bs *BufferService) GetBufferedMessages(ctx context.Context) ([]*repository.Message, error)

// Count of buffered messages (for "X of Y" counter).
func (bs *BufferService) CountBuffered(ctx context.Context) (int, error)

// Rate a buffered message 0-10 with optional notes. Transitions to Resolved.
// Optionally embeds via vector store for learning.
func (bs *BufferService) SaveRating(ctx context.Context, messageID uuid.UUID, rating int, feedback *string) error

// Discard a buffered message. Transitions to Resolved without rating or embedding.
func (bs *BufferService) DeleteMessage(ctx context.Context, messageID uuid.UUID) error
```

## Error Handling

| Scenario | Behavior |
|---|---|
| Repository query fails | Error propagated with "buffer:" prefix |
| Message not found in buffer | Error: "message {id} not found in buffer" |
| Rating out of range (< 0 or > 10) | Error: "rating must be 0-10, got {n}" |
| Repository update fails | Error propagated |
| Embedding fails | Silently ignored — save still succeeds |
| Nil embedder | Embedding skipped entirely |

## Integration Points

- **Repository (Feature 2):** Uses `QueryByStatus("Buffered")` and `Update` — both existing methods, no changes needed
- **Vector Store (Feature 8):** Calls `StoreEmbedding` after successful rating save to embed message for future similarity lookups
- **Router (Feature 3):** Creates buffered messages (IS >= 7, CS < 0.8) — no code change needed
- **GUI (Feature 11, future):** Will instantiate `BufferService` and use its methods for the Feedback Buffer Review pane

## Test Coverage

23 test cases (25 with subtests) in `buffer_test.go` using testify suites:

- Constructor validation (3 tests)
- GetBufferedMessages behavior (4 tests)
- CountBuffered behavior (2 tests)
- SaveRating behavior (8 tests)
- DeleteMessage behavior (4 tests)
- Edge cases / boundary values (2 subtests)

Tests use mock implementations of `MessageRepository` and `VectorEmbedder` with call tracking.

## TDD Agent Stats

| Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | Implementer (as Test Designer) | 145s | 40,239 | acf2bc3 |
| GREEN | Implementer | 39s | 26,805 | 7f879ad |
| REFACTOR | Refactorer | 88s | 32,843 | 38ce22a |
