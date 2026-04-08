# Feature 094: Calibration Loop Redesign

**Phase:** Phase-8-Feature-094
**Status:** Done
**Packages:** `internal/service/decisionengine/`, `internal/service/orchestrator/`, `internal/service/vector/`, `internal/repository/implementation/sqlite/`, `internal/config/`
**Depends on:** Feature 042 (replaces), Feature 086 (QueueProcessor), Feature 087 (Orchestrator), Feature 092 (prompt optimization)

---

## Overview

Replaced the arithmetic `VectorScoreAdvisor` (Feature 042) with few-shot prompt injection. Similar previously-rated messages are included directly in the Ollama scoring prompt as examples, so the LLM reasons about them and produces a single importance/confidence score informed by real user feedback. No post-processing arithmetic applied.

## Why the Old Implementation Was Wrong

The old `VectorScoreAdvisor`:
1. Computed a weighted average of user ratings vs Ollama scores
2. Derived an arithmetic adjustment (`avgUserRating - avgImportanceScore`)
3. Clamped to ±2.0 and applied a damping factor
4. Added the adjustment to the Ollama score after the fact

This bypassed the LLM entirely, couldn't reason about context, introduced arbitrary constraints, and was coupled to score distributions (model transitions invalidated stored deltas).

## Design: Few-Shot Prompt Injection

When `calibration_enabled = true`, the `QueueProcessor` calls `FewShotProvider.GetExamples` before scoring. Up to `calibration_max_examples` (default 5) similar rated messages are retrieved from the vector store. Their content and user ratings are injected into the Ollama prompt:

```
Score this message's importance for an ADHD user who needs to catch critical
items (deadlines, outages, @mentions) without noise.

The user has rated similar messages in the past:

- "API latency alert resolved, all clear" → User rated: 1/10
- "Reminder: quarterly review is tomorrow at 2pm" → User rated: 8/10

Now score this message:

Source: slack | Sender: ops-bot | Channel: incidents
Content: database connection pool exhausted

{"importance_score": 0-10, "confidence_score": 0.0-1.0, "reasoning": "one sentence"}
```

When no rated examples exist, the prompt is identical to the base prompt (graceful degradation).

## API

### FewShotExample

```go
type FewShotExample struct {
    Content    string  // Truncated to 200 chars
    UserRating int     // 0-10
    Similarity float32 // For logging; not sent to LLM
}
```

### FewShotProvider interface

```go
type FewShotProvider interface {
    GetExamples(ctx context.Context, content string) ([]FewShotExample, error)
}
```

Constructor: `NewFewShotProvider(querier VectorQuerier, msgQuerier MessageQuerier, cfg FewShotProviderConfig)`

### Scorer interface (breaking change)

```go
type Scorer interface {
    ScoreWithContext(ctx context.Context, msg *repository.Message, examples []FewShotExample) (*ScorerResult, error)
}
```

`Score` is gone. All callers pass `examples` explicitly (nil = no calibration).

### ScorerResult

```go
type ScorerResult struct {
    ImportanceScore float64
    ConfidenceScore float64
    Reasoning       string
    ScoringModel    string // NEW: model that produced this score
}
```

### QueueProcessor

`SetFewShotProvider(fsp FewShotProvider)` wires calibration into the processor. When nil, scoring is unchanged (backward compatible). `ExamplesUsed` and `ScoringModel` are recorded on each scored message.

### ChromemVectorStore

`NewChromemVectorStore(storagePath, embeddingFn, embeddingModelName)` — new third parameter. Vectors are tagged with `embedding_model` metadata. `QuerySimilar` filters out vectors from a different embedding model, preventing cross-model similarity comparisons.

## Configuration

```toml
[orchestrator.router]
calibration_enabled = false              # Enable few-shot calibration
calibration_similarity_threshold = 0.75  # Min cosine similarity
calibration_max_examples = 5             # Max examples per scoring call
```

Removed fields: `vector_enabled`, `vector_similarity_threshold`, `vector_top_n`, `vector_damping_factor`.

## Error Handling

| Scenario | Behaviour |
|---|---|
| No rated examples in vector store | No examples block in prompt; normal scoring |
| FewShotProvider returns error | Log suppressed; score without examples (graceful) |
| FewShotProvider is nil | Score without examples (backward compatible) |
| Ollama fails with examples in prompt | IS=7, CS=0.0, BUFFERED (unchanged fallback) |
| Different embedding model in store | Filtered by `embedding_model` metadata |

## SQLite Schema Migrations

```sql
ALTER TABLE messages ADD COLUMN scoring_model TEXT NOT NULL DEFAULT '';
ALTER TABLE messages ADD COLUMN examples_used INTEGER NOT NULL DEFAULT 0;
```

Both migrations are idempotent (duplicate column name errors are ignored).

## Vector Store Invariant

Only Ollama-scored messages with user ratings enter the vector store (embedded at rating time by `BufferService.SaveRating`). Deterministic-rule messages are excluded by design — the calibration examples are always relevant to the LLM's actual task.

## Test Coverage Summary

| Component | Tests |
|---|---|
| FewShotProvider | Empty store, threshold filtering, content truncation, unrated excluded, max cap, query error, nil guards |
| OllamaClient.ScoreWithContext | 0 examples (base prompt identical), N examples injected, rating formatted as N/10, ScoringModel in result |
| QueueProcessor | Nil provider (backward compat), provider error (graceful), examples forwarded to scorer, ExamplesUsed set, ScoringModel set |
| ChromemVectorStore | Model stored in metadata, same model returns results, different model filtered, empty model name allowed |
| SQLite | ScoringModel + ExamplesUsed round-trip |

## TDD Agent Stats

See `docs/agent-log.md` — Phase-8-Feature-094 entries.
