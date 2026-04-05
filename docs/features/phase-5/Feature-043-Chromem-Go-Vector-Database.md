# Feature 043: chromem-go Vector Database

**Phase:** Phase-5-Feature-043
**Status:** Planned
**Packages:** `internal/service/vector/`, `cmd/cue/`
**Depends on:** Feature 044 (Ollama Scorer Wiring — shares Ollama HTTP client patterns)

---

## Overview

Replace the in-memory `VectorStore` implementation with `github.com/rengensheng/chromem-go`, a flat-file pure-Go vector database with BERT embeddings. The current `VectorStore` in `internal/service/vector/vector.go` stores embeddings in a `[]storedVector` slice with no persistence — all vectors are lost on restart. chromem-go provides durable, file-backed storage with built-in similarity search.

## Motivation

The in-memory `VectorStore` was scaffolded in Feature 008 as a placeholder. The project's technology stack (documented in `.claude/CLAUDE.md` §2) specifies chromem-go as the vector database, but it was never added to `go.mod` or integrated. For Feature 042 (Vector-Assisted Routing) to be useful, the vector corpus must persist across restarts.

## Design Decisions

### Adapter Pattern

The existing `VectorEmbedder` interface in `internal/service/buffer/buffer.go` defines:

```go
type VectorEmbedder interface {
    StoreEmbedding(ctx context.Context, messageID uuid.UUID, content string) (*uuid.UUID, error)
}
```

And `VectorStore` exposes `QuerySimilar()`. Rather than forcing chromem-go's API through these existing interfaces directly, create a `ChromemVectorStore` adapter that implements both `VectorEmbedder` and a new `VectorQuerier` interface:

```go
// VectorQuerier supports similarity search over stored embeddings.
type VectorQuerier interface {
    QuerySimilar(ctx context.Context, queryText string, topN int) ([]SimilarResult, error)
}
```

This lets the buffer service use it as a `VectorEmbedder` and the router's `VectorScoreAdvisor` (Feature 042) use it as a `VectorQuerier`.

### Storage Location

chromem-go stores its data as flat files. The storage path derives from the database path in config:

```
~/.cue/vectors/    (sibling to ~/.cue/messages.db)
```

No new config field needed — derive from `cfg.Database.Path` by replacing the filename with `vectors/`.

### Embedding Function

chromem-go supports pluggable embedding functions. Wire it to use the Ollama `/api/embeddings` endpoint with the configured `embedding_model`. This shares the embedding function created in Feature 042's `ollama_embeddings.go`, but since Feature 043 is a prerequisite, the embedding function is created here and reused by Feature 042.

### Migration from In-Memory Store

The current `VectorStore` struct and its tests are replaced entirely. The `EmbeddingFunc` type signature and `SimilarResult` struct are preserved as they form the contract used by other packages.

## API

```go
// internal/service/vector/chromem_store.go

type ChromemVectorStore struct {
    db         *chromem.DB
    collection *chromem.Collection
}

func NewChromemVectorStore(storagePath string, embeddingFn EmbeddingFunc) (*ChromemVectorStore, error)

// Implements VectorEmbedder
func (s *ChromemVectorStore) StoreEmbedding(ctx context.Context, messageID uuid.UUID, content string) (*uuid.UUID, error)

// Implements VectorQuerier
func (s *ChromemVectorStore) QuerySimilar(ctx context.Context, queryText string, topN int) ([]SimilarResult, error)

// Close persists any pending data and releases resources.
func (s *ChromemVectorStore) Close() error
```

## Error Handling

| Scenario | Behavior |
|---|---|
| Storage directory creation fails | Return error from constructor — fatal at startup |
| chromem-go DB open fails | Return error from constructor — fatal at startup |
| Embedding fails during store | Return error to caller (BufferService logs and continues) |
| Embedding fails during query | Return error to caller (VectorScoreAdvisor returns nil advice) |
| Corrupt vector file on disk | chromem-go handles internally; log warning if detected |

## Integration Points

- **`internal/service/buffer/buffer.go`**: Receives `ChromemVectorStore` as `VectorEmbedder` (replaces current `nil`)
- **Feature 042** (Vector-Assisted Routing): Receives `ChromemVectorStore` as `VectorQuerier`
- **Feature 044** (Ollama Scorer Wiring): Shares Ollama HTTP patterns for embedding endpoint
- **`cmd/cue/main.go`**: Instantiates `ChromemVectorStore`, passes to buffer service and router advisor

## Test Coverage

- Constructor validates storage path and embedding function
- `StoreEmbedding` persists and returns valid UUID
- `QuerySimilar` returns results sorted by similarity
- `QuerySimilar` with empty store returns nil
- Round-trip: store embedding, query similar, get match
- Storage persists across close/reopen (durability test using `s.T().TempDir()`)
- Embedding function errors propagate correctly

## Files

| File | Action |
|---|---|
| `go.mod` | **Modify** — add `github.com/rengensheng/chromem-go` dependency |
| `internal/service/vector/chromem_store.go` | **New** — chromem-go adapter |
| `internal/service/vector/chromem_store_test.go` | **New** — adapter tests |
| `internal/service/vector/vector.go` | **Delete** — replaced by chromem adapter |
| `internal/service/vector/vector_test.go` | **Delete** — replaced by chromem adapter tests |
| `internal/service/vector/interfaces.go` | **New** — `VectorQuerier` interface, shared types (`SimilarResult`, `EmbeddingFunc`) |
| `cmd/cue/main.go` | **Modify** — instantiate `ChromemVectorStore`, pass to buffer service |
