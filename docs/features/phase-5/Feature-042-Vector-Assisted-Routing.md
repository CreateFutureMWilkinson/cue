# Feature 042: Vector-Assisted Routing

**Phase:** Phase-5-Feature-042
**Status:** Planned
**Packages:** `internal/service/decisionengine/`, `internal/service/vector/`, `internal/service/buffer/`, `cmd/cue/`
**Depends on:** Feature 043 (chromem-go Vector Database), Feature 044 (Ollama Scorer Wiring)

---

## Overview

Wire the existing `VectorStore.QuerySimilar()` into the routing decision path so that historical user feedback influences future scoring. When a message arrives that the Ollama scorer evaluates, the router queries the vector store for similar previously-rated messages and uses the aggregate user rating to adjust the final importance score. This closes the feedback loop that Features 008 (Vector Store) and 009 (Feedback Buffer) were designed to enable.

## Motivation

The vector store (`internal/service/vector/`) and feedback buffer embedding path (`internal/service/buffer/buffer.go:82-88`) were built as scaffolding for adaptive routing. Currently:

- Users rate buffered messages 0–10 in the feedback review UI
- The `BufferService.SaveRating()` method has a code path to embed rated messages via `VectorEmbedder`
- The `VectorStore` supports `QuerySimilar()` with cosine similarity search
- **None of this feeds back into routing decisions** — the router is purely threshold-based

This feature connects the output of user feedback to the input of routing, making the system learn from corrections over time.

## Design Decisions

### Score Adjustment, Not Replacement

Vector similarity provides a **bias signal**, not a replacement for Ollama scoring. The router continues to use Ollama as the primary scorer. After Ollama returns its result, the router optionally queries the vector store for similar rated messages and applies an adjustment:

```
final_importance = ollama_importance + vector_adjustment
```

This is conservative: if the vector store is empty, has no similar matches, or is unavailable, routing behaves exactly as it does today.

### Adjustment Calculation

Given the top-N similar messages (by cosine similarity) and their user ratings:

1. Filter results to those above a similarity threshold (e.g., cosine ≥ 0.75)
2. Compute a weighted average of user ratings, weighted by similarity score
3. Compare to the Ollama importance score:
   - If users consistently rated similar messages **higher** → positive adjustment
   - If users consistently rated similar messages **lower** → negative adjustment
4. Cap the adjustment to ±2.0 to prevent wild swings

```go
adjustment = clamp(weightedAvgRating - ollamaImportance, -2.0, 2.0) * dampingFactor
```

The `dampingFactor` (0.0–1.0, default 0.5) controls how aggressively the system adapts. Configurable in `config.toml`.

### VectorScoreAdvisor Interface

A new interface keeps the router decoupled from the vector store implementation:

```go
// VectorScoreAdvisor provides score adjustments based on historical feedback.
type VectorScoreAdvisor interface {
    Advise(ctx context.Context, content string) (*ScoreAdvice, error)
}

type ScoreAdvice struct {
    Adjustment      float64  // Score adjustment to apply (-2.0 to +2.0)
    SimilarCount    int      // Number of similar rated messages found
    AvgUserRating   float64  // Weighted average user rating of similar messages
    TopSimilarity   float32  // Highest cosine similarity score
}
```

The router accepts an optional `VectorScoreAdvisor` (nil-safe, like the existing alerter pattern). When nil, no adjustment is applied.

### Embedding Pipeline Activation

This feature also wires the currently-dormant embedding path:

1. **`VectorStore` instantiation in `main.go`** — Create a real `VectorStore` using the Ollama embedding model
2. **`EmbeddingFunc` implementation** — New function that calls Ollama's `/api/embeddings` endpoint using the configured `embedding_model`
3. **Pass `VectorStore` to `BufferService`** — Replace the current `nil` embedder
4. **Pass `VectorScoreAdvisor` to `Router`** — New optional constructor parameter

### New Config Fields

```toml
[orchestrator.router]
# Existing fields...
importance_threshold = 7
confidence_threshold = 0.8
buffer_size_per_source = 100

# New fields for vector-assisted routing
vector_similarity_threshold = 0.75    # Minimum cosine similarity to consider
vector_top_n = 5                      # Max similar messages to consider
vector_damping_factor = 0.5           # How aggressively to adjust (0.0-1.0)
vector_enabled = false                # Opt-in; disabled by default
```

`vector_enabled = false` by default ensures no behavior change for existing users until they opt in and have built up a sufficient feedback corpus.

### MessageRepository Extension

The adjustment calculation needs to look up user ratings for messages returned by `QuerySimilar()`. This requires a `QueryByID` method on `MessageRepository`:

```go
QueryByID(ctx context.Context, id uuid.UUID) (*Message, error)
```

This method already exists on `TodoRepository` but is missing from `MessageRepository`.

## Architecture

```
Message arrives
    ↓
Router.Route()
    ├→ Deterministic rules (channel_join, @mention) — skip vector step
    └→ Scorer.Score() (Ollama)
         ↓
    VectorScoreAdvisor.Advise(msg.RawContent)
         ├→ VectorStore.QuerySimilar(content, topN)
         ├→ MessageRepository.QueryByID(matchID) for each match
         ├→ Filter by similarity threshold
         ├→ Compute weighted avg of UserRating
         └→ Return ScoreAdvice{Adjustment: ...}
         ↓
    Apply: msg.ImportanceScore += advice.Adjustment
         ↓
    assignStatus() (existing threshold logic)
```

### Embedding Pipeline (Already Scaffolded)

```
User rates buffered message
    ↓
BufferService.SaveRating()
    ↓
VectorEmbedder.StoreEmbedding(messageID, content)  ← currently nil, now wired
    ↓
VectorStore stores embedding + messageID
```

## Ollama Embedding Function

New file implementing the `EmbeddingFunc` signature that the `VectorStore` expects:

```go
// internal/service/decisionengine/ollama_embeddings.go

func NewOllamaEmbeddingFunc(baseURL, model string, timeout time.Duration) vector.EmbeddingFunc {
    return func(ctx context.Context, text string) ([]float32, error) {
        // POST to baseURL/api/embeddings with {"model": model, "prompt": text}
        // Parse response.embedding []float64, convert to []float32
    }
}
```

This uses the `embedding_model` config field that is already loaded and validated but currently unused.

## Error Handling

| Scenario | Behavior |
|---|---|
| Vector store empty (no feedback yet) | `Advise()` returns nil advice, no adjustment |
| No similar messages above threshold | `Advise()` returns zero adjustment |
| Similar messages found but none have ratings | Skip unrated, use only rated matches |
| Ollama embedding endpoint fails | Log warning, return nil advice (no adjustment) |
| `VectorScoreAdvisor` is nil | Router skips vector step entirely |
| Adjustment would push IS below 0 or above 10 | Clamp to [0, 10] |

**Philosophy:** Vector assistance is always best-effort. Any failure degrades gracefully to the existing threshold-based routing with no user-visible error.

## Integration Points

- **`internal/service/decisionengine/router.go`**: Accept optional `VectorScoreAdvisor`, call after Ollama scoring
- **`internal/service/vector/vector.go`**: Already implements `QuerySimilar()` — no changes needed
- **`internal/service/buffer/buffer.go`**: Already has `VectorEmbedder` support — just needs non-nil embedder
- **`internal/config/config.go`**: Add vector routing config fields to `RouterConfig`
- **`cmd/cue/main.go`**: Instantiate `VectorStore`, wire to both `BufferService` and `Router`
- **Feature 038** (Main Wiring): This feature extends the wiring further
- **Feature 039** (Ollama Model Validation): Validates `embedding_model` is available

## Test Coverage

### VectorScoreAdvisor Tests
- No similar messages → zero adjustment
- Similar messages with ratings → weighted adjustment calculated correctly
- All matches below similarity threshold → zero adjustment
- Mixed rated/unrated matches → unrated excluded from calculation
- Adjustment capped at ±2.0
- Damping factor applied correctly
- Empty vector store → nil advice

### Router Integration Tests
- Router with nil advisor → routing unchanged (backward compatible)
- Router with advisor → adjustment applied to Ollama score
- Adjustment does not affect deterministic rules (channel_join, @mention)
- Adjusted score correctly changes status (e.g., IS 6.5 + adjustment 1.0 = 7.5 → NOTIFIED)
- Adjusted score clamped to [0, 10]

### Embedding Pipeline Tests
- `OllamaEmbeddingFunc` returns valid embedding from mock server
- `OllamaEmbeddingFunc` handles Ollama errors gracefully
- `BufferService.SaveRating()` with real embedder stores embedding
- End-to-end: rate message → embedding stored → new similar message → adjustment applied

### Config Tests
- New fields have sensible defaults
- `vector_enabled = false` disables advisor creation
- Validation: damping factor in [0.0, 1.0], top_n > 0, threshold in [0.0, 1.0]

## Files

| File | Action |
|---|---|
| `internal/service/decisionengine/vector_advisor.go` | **New** — `VectorScoreAdvisor` interface + implementation |
| `internal/service/decisionengine/vector_advisor_test.go` | **New** — advisor unit tests |
| `internal/service/decisionengine/ollama_embeddings.go` | **New** — `OllamaEmbeddingFunc` for Ollama `/api/embeddings` |
| `internal/service/decisionengine/ollama_embeddings_test.go` | **New** — embedding function tests with httptest |
| `internal/service/decisionengine/router.go` | **Modify** — accept optional `VectorScoreAdvisor`, apply after scoring |
| `internal/service/decisionengine/router_test.go` | **Modify** — add advisor integration tests |
| `internal/config/config.go` | **Modify** — add vector routing fields to `RouterConfig` |
| `internal/config/config_test.go` | **Modify** — validate new config fields |
| `internal/repository/message.go` | **Modify** — add `QueryByID` to `MessageRepository` interface |
| `internal/repository/implementation/sqlite/message_impl.go` | **Modify** — implement `QueryByID` |
| `internal/repository/implementation/sqlite/message_impl_test.go` | **Modify** — test `QueryByID` |
| `cmd/cue/main.go` | **Modify** — instantiate `VectorStore`, wire embedder + advisor |

---

## Appendix: Unwired Scaffolding Inventory

This feature completes scaffolding that was built in earlier phases but never connected to the runtime. Below is a full inventory of unwired scaffolding across completed features, as of Phase 4 Feature 041:

### Scaffolding Wired by This Feature

| Scaffold | Built In | Location | Current State | Wired By |
|---|---|---|---|---|
| `VectorStore.QuerySimilar()` | Feature 008 | `internal/service/vector/vector.go:67` | Implemented, never called | This feature (042) |
| `VectorStore.StoreEmbedding()` | Feature 008 | `internal/service/vector/vector.go:50` | Implemented, never called | This feature (042) |
| `BufferService.VectorEmbedder` param | Feature 009 | `internal/service/buffer/buffer.go:20-23` | Interface defined, always `nil` | This feature (042) |
| `SaveRating()` embedding path | Feature 009 | `internal/service/buffer/buffer.go:82-88` | Code exists, skipped (nil check) | This feature (042) |
| `Message.VectorID` field | Feature 002 | `internal/repository/message.go:26` | Column in SQLite, never populated | This feature (042) |
| `OllamaConfig.EmbeddingModel` | Feature 001 | `internal/config/config.go:81` | Loaded and validated, never used | This feature (042) |

### Scaffolding NOT Addressed by This Feature

| Scaffold | Built In | Location | Current State | Notes |
|---|---|---|---|---|
| `placeholderScorer` | Feature 004 | `cmd/cue/main.go:253-261` | Returns hardcoded 5.0/0.5 for all messages | Needs real `OllamaClient` wiring (Feature 038) |
| `placeholderSlackAPI` | Feature 005 | `cmd/cue/main.go:263-275` | Returns `nil` for all API calls | Needs real Slack API client (not yet planned) |
| `placeholderEmailAPI` | Feature 006 | `cmd/cue/main.go:284-288` | Returns `nil` for all fetches | Needs real IMAP client (not yet planned) |
| `Message.MessageType` not persisted | Feature 002 | `sqlite/message_impl.go` | Field exists on struct but not in SQLite schema columns | `channel_join` rule works at routing time but type is lost on persist/reload |
| `RouterConfig.BufferSizePerSource` | Feature 001 | `internal/config/config.go:74` | Validated but ignored; `maxMessagesPerSource=100` hardcoded in SQLite impl | Config value should replace hardcoded constant |
| `NotificationConfig.BatchProcess` | Feature 001 | `internal/config/config.go:87` | Loaded, never checked | System always batches regardless |
| `Slack.Enabled` / `Email.Enabled` flags | Feature 001 | `internal/config/config.go:52,59` | Loaded, never checked | Watchers always created regardless of flag |
| `Orchestrator.WatcherManager` methods | Feature 034 | `orchestrator/orchestrator.go:16-20` | `AddWatcher`/`RemoveWatcher` implemented, never called from main | Dormant until Feature 038 wires settings UI to orchestrator |
| chromem-go dependency | — | `go.mod` | **Not present** despite CLAUDE.md listing it | In-memory `VectorStore` is the actual implementation |
