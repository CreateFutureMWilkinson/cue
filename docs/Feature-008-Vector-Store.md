# Feature 8: Vector Store with Cosine Similarity

**Phase:** Phase-1-Feature-8
**Status:** Done
**Package:** `internal/service/vector/`

---

## Overview

In-memory vector store that embeds message content via a pluggable embedding function and supports cosine similarity search. This is the foundation for the feedback learning loop — rated messages are embedded and stored so future messages can be compared against historical user preferences.

## Design Decisions

### In-Memory Storage (Not chromem-go Yet)

The current implementation stores vectors in a simple in-memory slice rather than using chromem-go directly. This was intentional:

- Keeps the vector abstraction testable without filesystem or library coupling
- The `EmbeddingFunc` type decouples embedding generation from storage
- chromem-go (or any other backend) can be swapped in behind the same interface later

### Cosine Similarity

Chosen over Euclidean distance or dot product because:

- Normalized comparison — magnitude of embeddings doesn't affect results
- Score range is naturally bounded (0.0–1.0 for non-negative embeddings)
- Standard metric for text embedding similarity

Zero-denominator case returns 0 (no similarity) rather than erroring.

### Message ID Association

Each stored vector links back to its source `Message.ID` (UUID). This enables:

- Looking up the original message context when similarity results are returned
- Connecting user feedback ratings to embedded vectors for learning

## API

### Types

```go
// Pluggable embedding function — will be backed by Ollama nomic-embed-text in production.
type EmbeddingFunc func(ctx context.Context, text string) ([]float32, error)

// Similarity search result.
type SimilarResult struct {
    MessageID uuid.UUID
    Score     float32   // 0.0–1.0, higher = more similar
}
```

### Constructor

```go
func NewVectorStore(embeddingFn EmbeddingFunc, storagePath string) (*VectorStore, error)
```

Both arguments required. Returns error on nil embedding function or empty storage path.

### Methods

```go
// Embed content and store the vector. Returns the generated VectorID.
func (vs *VectorStore) StoreEmbedding(ctx context.Context, messageID uuid.UUID, content string) (*uuid.UUID, error)

// Find topN most similar stored vectors to the query text.
func (vs *VectorStore) QuerySimilar(ctx context.Context, queryText string, topN int) ([]SimilarResult, error)
```

## Error Handling

| Scenario | Behavior |
|---|---|
| Embedding function fails during store | Returns error, no vector stored |
| Embedding function fails during query | Returns error, nil results |
| Empty store queried | Returns nil, nil (no error) |
| topN > stored vectors | Returns all stored vectors |

Aligns with CLAUDE.md Section 12: chromem-go embedding failure → store message without vector, log, continue.

## Integration Points

- **Router orchestration** (Feature 7): After routing a message, the orchestrator can call `StoreEmbedding` to persist the vector for future similarity lookups
- **Feedback buffer** (Feature 9): When a user rates a buffered message, the rating + vector can be used to improve future scoring
- **Message repository** (Feature 2): `Message.VectorID` field links to the vector store entry

## Test Coverage

15 test cases in `vector_test.go` using testify suites:

- Constructor validation (3 tests)
- StoreEmbedding behavior (2 tests)
- QuerySimilar behavior (4 tests)
- Embedding error propagation (2 tests)
- Helper function tests (2 tests via deterministic/failing embedding mocks)

Tests use a deterministic embedding function that derives 3D vectors from text length, producing reproducible similarity orderings without requiring a real model.

## TDD Agent Stats

| Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | Test Designer | 256s | 21,654 | ce3373c |
| GREEN | Implementer | 48s | 22,112 | 3b91f3c |
| REFACTOR | Refactorer | 68s | 25,874 | eefa1f9 |
