# Feature 094: Calibration Loop Redesign

**Phase:** Phase-8-Feature-094
**Status:** Planned
**Packages:** `internal/service/decisionengine/`, `internal/service/buffer/`, `internal/service/vector/`, `internal/config/`
**Depends on:** Feature 042 (replaces current implementation), Feature 086 (queue processor), Feature 087 (orchestrator refactor), Feature 092 (prompt optimization)

---

## Overview

Replace the current arithmetic-based `VectorScoreAdvisor` (Feature 042) with the originally intended design: few-shot prompt injection. Instead of computing a post-hoc score adjustment, similar previously-rated messages are included directly in the Ollama scoring prompt as context examples. The LLM reasons about the examples and produces a single importance/confidence score informed by real user feedback.

## Why the Current Implementation Is Wrong

Feature 042 was intended to close the feedback loop by feeding similar rated messages back into the LLM as context. The implementation drifted from this design and instead:

1. Queries the vector store for similar rated messages
2. Computes a weighted average of user ratings vs Ollama scores
3. Derives an arithmetic adjustment (`avgUserRating - avgImportanceScore`)
4. Clamps to ±2.0, applies a damping factor
5. Adds the result to the Ollama score after the fact

This is fundamentally wrong because:

- **It bypasses the LLM entirely.** The adjustment is pure arithmetic — cosine similarity + weighted averages. The LLM never sees the user feedback.
- **It can't reason about context.** A numeric adjustment of -1.3 can't distinguish "the user ignores all bot messages" from "the user ignores messages from #random." The LLM can.
- **It introduces artificial constraints.** The ±2.0 clamp and damping factor are arbitrary guardrails needed because the arithmetic approach is crude. Few-shot prompting doesn't need them — the LLM produces the score directly.
- **It couples calibration to score distributions.** Changing the model or prompt shifts importance scores, invalidating the `avgUserRating - avgImportanceScore` delta. Few-shot examples are model-agnostic — the LLM reads the examples fresh each time.

## Intended Design: Few-Shot Prompt Injection

### Flow

After Feature 087, the full message pipeline is:

```
Watcher.Poll() → new messages
    ↓
Orchestrator.PollOnce():
    ExistsByMessageID → dedup
    RulesEngine.Evaluate → "notified"/"ignored" (deterministic, stored immediately)
                         → "queue" (enqueued for Ollama)
    ↓
QueueProcessor (background, one at a time):
    DequeueOldest → message
    FewShotProvider.GetExamples → 0-5 similar rated messages
    ↓
    Build prompt:
        - Scoring instruction (from Feature 092)
        - Few-shot examples: [content, user_rating] × 0-5
        - New message to score
    ↓
    Scorer.ScoreWithContext → single Ollama request
    ↓
    LLM returns importance/confidence/reasoning
        (informed by examples — no post-processing)
    ↓
    Assign status on thresholds, update message, alert if Notified
```

### Prompt Structure

```
Score this message's importance for an ADHD user who needs to catch critical
items (deadlines, outages, @mentions) without noise.

The user has rated similar messages in the past:

- "API latency alert resolved, all clear" → User rated: 1/10
- "Reminder: quarterly review is tomorrow at 2pm" → User rated: 8/10
- "Updated the README with new setup instructions" → User rated: 2/10

Now score this message:

Source: %s | Sender: %s | Channel: %s
Content: %s

{"importance_score": 0-10, "confidence_score": 0.0-1.0, "reasoning": "one sentence"}
```

When no rated similar messages exist, the examples section is omitted entirely and the prompt is identical to the base prompt from Feature 092. Graceful degradation — zero examples means current behavior, more examples means better calibration.

### Example Selection

- Query the vector store for the top 5 most similar messages to the incoming message content
- Only messages with a user rating are in the vector store (explicit design constraint — see "Vector Store Invariant" below)
- Include all results returned (0-5), ordered by descending similarity
- Each example includes only: truncated content (first 200 chars) and user rating (0-10)
- **User rating only — do not include the original Ollama score.** The LLM's job is to understand what the user considers important, not to reason about its own past mistakes. Keeping examples simple also reduces token count, which matters for smaller models.

### Why User Rating Only

Including the Ollama score ("Ollama scored this 4, user rated it 8") would ask the model to reason about the correction direction. This is problematic for smaller models:

- Extra tokens per example (doubles the context cost)
- Requires the model to understand meta-reasoning ("I should score higher because the user disagreed with a previous score")
- Couples the examples to the scoring model's distribution — if the model changes, the Ollama scores in examples become misleading

User rating alone is clean signal: "messages like this are important to this user" or "messages like this are not."

## Vector Store Invariant

**Only Ollama-scored messages with user ratings enter the vector store.** This is enforced by the existing `BufferService.SaveRating()` code path — embedding happens at rating time, not at message arrival. This means:

- Messages routed by deterministic rules (channel_join, @mention) → NOTIFIED → never enter the buffer → never rated → never embedded
- Messages scored by Ollama → NOTIFIED/BUFFERED/IGNORED → only BUFFERED messages appear in the feedback buffer → only rated messages get embedded
- The few-shot examples are always messages that Ollama would actually see in production

This invariant ensures that calibration examples are relevant to the LLM's actual task. Deterministic-rule messages are excluded by design because Ollama never evaluates them.

## What Gets Removed

The current `VectorScoreAdvisor` and its associated infrastructure are replaced:

| Remove | Reason |
|---|---|
| `VectorScoreAdvisor` interface | No longer needed — calibration happens in the prompt |
| `vectorScoreAdvisor` implementation | Replaced by few-shot prompt building |
| `ScoreAdvice` struct | No adjustment step |
| `VectorAdvisorConfig` (similarity threshold, damping, top-N) | Replaced by simpler config |
| `MessageQuerier` interface in vector_advisor.go | Message lookup moves to the `FewShotProvider` |
| Config fields: `vector_damping_factor` | Not applicable to few-shot |

Note: The `Router` struct and its `advisor` field are already removed by Feature 087. This feature removes the `VectorScoreAdvisor` infrastructure from `vector_advisor.go` and rewrites the file to contain the `FewShotProvider` instead.

## What Gets Added

| Add | Purpose |
|---|---|
| `FewShotProvider` interface | Retrieves similar rated messages for prompt injection |
| `fewShotProvider` implementation | Queries vector store + message repo, returns examples |
| `buildPromptWithExamples()` function | Constructs the few-shot prompt |
| Modified `OllamaClient.Score()` | Accepts optional few-shot examples |
| Config field: `vector_max_examples` | Max few-shot examples (default 5) |

### FewShotProvider Interface

```go
// FewShotExample is a previously-rated message used as LLM context.
type FewShotExample struct {
    Content    string // Truncated to 200 chars
    UserRating int    // 0-10
    Similarity float32 // For logging/debugging, not sent to LLM
}

// FewShotProvider retrieves similar rated messages for prompt injection.
type FewShotProvider interface {
    GetExamples(ctx context.Context, content string, maxN int) ([]FewShotExample, error)
}
```

The provider queries the vector store, looks up each message's user rating, truncates content, and returns up to `maxN` examples. It replaces `VectorScoreAdvisor` and is injected into the `QueueProcessor` (Feature 086).

### QueueProcessor Integration

After Feature 087, the `Router` struct is removed. Deterministic rules are evaluated by the `RulesEngine` in the orchestrator, and Ollama scoring is performed by the `QueueProcessor` (Feature 086). The few-shot logic is added to the `QueueProcessor`, which already holds the `Scorer` interface.

```go
type QueueProcessor struct {
    queue               QueueRepository
    messages            MessageRepository
    scorer              Scorer
    fewShot             FewShotProvider  // NEW — replaces Router's advisor
    alerter             Alerter
    cooldown            time.Duration
    importanceThreshold float64
    confidenceThreshold float64
    maxExamples         int
    eventCh             chan<- ActivityEvent
}
```

The processing loop becomes:

```go
// In the QueueProcessor's scoring step:
msg := messages.QueryByID(ctx, entry.MessageID)

// Get few-shot examples (nil-safe, best-effort)
var examples []FewShotExample
if p.fewShot != nil {
    examples, _ = p.fewShot.GetExamples(ctx, msg.RawContent, p.maxExamples)
}

// Score with examples included in prompt
result, err := p.scorer.ScoreWithContext(ctx, msg, examples)
if err != nil {
    // Fallback: IS=7, CS=0.0, BUFFERED
    msg.ImportanceScore = 7.0
    msg.ConfidenceScore = 0.0
    msg.Status = "Buffered"
    msg.Reasoning = "Ollama scoring failed: " + err.Error()
} else {
    msg.ImportanceScore = result.ImportanceScore
    msg.ConfidenceScore = result.ConfidenceScore
    msg.Reasoning = result.Reasoning
    msg.ExamplesUsed = len(examples)
    // Status assignment using thresholds
    ...
}
```

No post-processing, no adjustment arithmetic. The LLM produces the final score.

### Scorer Interface Extension

```go
type Scorer interface {
    Score(ctx context.Context, msg *repository.Message) (*ScorerResult, error)
    ScoreWithContext(ctx context.Context, msg *repository.Message, examples []FewShotExample) (*ScorerResult, error)
}
```

`ScoreWithContext` builds the few-shot prompt and sends it to Ollama. `Score` remains for backward compatibility (equivalent to `ScoreWithContext` with nil examples). The `OllamaClient` implements both.

## Configuration

```toml
[orchestrator.router]
# Existing fields
importance_threshold = 7
confidence_threshold = 0.8
buffer_size_per_source = 100

# Calibration (replaces old vector_* fields)
vector_enabled = false                  # Enable few-shot calibration
vector_similarity_threshold = 0.75      # Minimum cosine similarity for examples
vector_max_examples = 5                 # Maximum few-shot examples per scoring call
```

Removed fields: `vector_top_n` (renamed to `vector_max_examples` for clarity), `vector_damping_factor` (not applicable).

## Model Transition Resilience

Few-shot prompting is inherently model-agnostic. The examples contain raw content and user ratings — no model-specific scores or embeddings in the prompt text. When the inference model changes:

- The LLM reads the same examples and produces scores in its own distribution
- No "score distribution shift" problem — each model interprets examples fresh
- The embedding model is separate from the inference model, so stored vectors remain valid across inference model changes

If the *embedding* model changes, stored vectors become incomparable (different vector spaces). This is an existing limitation of the vector store. The `vector_similarity_threshold` provides natural protection — incomparable vectors produce low similarity scores and are filtered out.

## Error Handling

| Scenario | Behavior |
|---|---|
| Vector store empty (no ratings yet) | No examples in prompt, scoring works as normal |
| Fewer than `max_examples` matches | Include however many are available (0-4) |
| All matches below similarity threshold | No examples in prompt |
| Vector store query fails | Log warning, score without examples |
| `FewShotProvider` is nil | Score without examples (backward compatible) |
| Ollama fails with examples in prompt | Same fallback as today: IS=7, CS=0.0, BUFFERED |

## Calibration Enhancements (Model Tagging, Recency, Cold Start)

The following enhancements from the original Feature 094 scope remain relevant and are incorporated into this redesign:

### Model Version Tagging

Store `scoring_model` on each message so we can track which model produced which scores. This is useful for analytics and debugging even though the few-shot approach doesn't depend on it for correctness. The few-shot examples show user ratings, not Ollama scores, so model transitions don't break calibration.

```sql
ALTER TABLE messages ADD COLUMN scoring_model TEXT NOT NULL DEFAULT '';
```

### Recency Weighting

When selecting few-shot examples, prefer recent ratings over old ones. The vector store returns results by similarity — add a secondary sort by recency. If two messages have similar cosine scores, the more recently rated one is preferred.

This is handled in the `FewShotProvider` implementation: after querying the vector store, sort results by `(similarity * recencyWeight)` where recency decays with a configurable half-life.

```toml
[orchestrator.router]
vector_decay_half_life_days = 30  # 0 = no decay, prefer by similarity only
```

### Cold Start

No special handling needed. With 0 examples, the prompt is identical to the base prompt. With 1-4 examples, the LLM has partial context. With 5 examples, full context. The few-shot approach degrades gracefully by nature — unlike the arithmetic approach, there's no "minimum corpus" threshold below which adjustments become unstable.

### Adjustment Tracking

Record whether few-shot examples were available and how many were used:

```go
type Message struct {
    // ... existing fields ...
    ScoringModel    string // Model that scored this message
    ExamplesUsed    int    // Number of few-shot examples in scoring prompt (0 = no calibration)
}
```

```sql
ALTER TABLE messages ADD COLUMN scoring_model TEXT NOT NULL DEFAULT '';
ALTER TABLE messages ADD COLUMN examples_used INTEGER NOT NULL DEFAULT 0;
```

This replaces `AdjustmentApplied` and `PreAdjustmentScore` from the arithmetic approach. We can't meaningfully track "adjustment" since the LLM produces a single score — but knowing "this message was scored with 3 examples" vs "this message was scored with 0 examples" is useful for evaluating calibration effectiveness.

### Embedding Model Guard

Store the embedding model name in vector metadata (as originally proposed). This is still valuable — if the embedding model changes, old vectors produce meaningless similarity scores. Tagging them allows the provider to filter by current embedding model.

## Files

| File | Action |
|---|---|
| `internal/service/decisionengine/vector_advisor.go` | **Rewrite** — replace `VectorScoreAdvisor` with `FewShotProvider` interface + implementation |
| `internal/service/decisionengine/vector_advisor_test.go` | **Rewrite** — test few-shot example retrieval, truncation, filtering |
| `internal/service/decisionengine/ollama_client.go` | **Modify** — add `ScoreWithContext()`, `buildPromptWithExamples()` |
| `internal/service/decisionengine/ollama_client_test.go` | **Modify** — test prompt construction with 0, 1, 5 examples |
| `internal/service/orchestrator/queue_processor.go` | **Modify** — add `FewShotProvider` field, use `ScoreWithContext`, record `ExamplesUsed` |
| `internal/service/orchestrator/queue_processor_test.go` | **Modify** — test scoring with few-shot examples, graceful degradation |
| `internal/service/vector/chromem_store.go` | **Modify** — add `modelName` to constructor + metadata |
| `internal/service/vector/chromem_store_test.go` | **Modify** — test embedding model metadata |
| `internal/service/vector/interfaces.go` | **Modify** — update constructor signature if needed |
| `internal/repository/message.go` | **Modify** — add `ScoringModel`, `ExamplesUsed` fields |
| `internal/repository/implementation/sqlite/message_impl.go` | **Modify** — persist new fields, schema migration |
| `internal/repository/implementation/sqlite/message_impl_test.go` | **Modify** — test new field persistence |
| `internal/config/config.go` | **Modify** — replace `vector_damping_factor` + `vector_top_n` with `vector_max_examples` + `vector_decay_half_life_days` |
| `internal/config/config_test.go` | **Modify** — validate new/changed fields |
| `cmd/cue/main.go` | **Modify** — wire `FewShotProvider` into `QueueProcessor` instead of `VectorScoreAdvisor` into Router |

## Test Coverage

### FewShotProvider
- Empty vector store → returns empty slice
- 3 matches in store, max_examples=5 → returns 3 examples
- 7 matches in store, max_examples=5 → returns 5 examples
- Matches below similarity threshold filtered out
- Content truncated to 200 chars
- Results ordered by similarity (descending), with recency as tiebreaker
- Recency decay applied correctly (half-life weighting)
- Messages without user ratings excluded (should not exist in store, but defensive)

### Prompt Construction
- 0 examples → base prompt identical to Feature 092
- 1 example → examples section with single entry
- 5 examples → examples section with all entries
- Content in examples is truncated, not raw
- User rating formatted as "N/10"
- JSON mode format field present in request

### QueueProcessor Integration
- Nil FewShotProvider → scoring without examples (backward compatible)
- FewShotProvider returns examples → passed to ScoreWithContext
- FewShotProvider errors → scoring without examples (graceful degradation)
- Fallback scoring unchanged when Ollama fails (IS=7, CS=0.0, BUFFERED)
- `ExamplesUsed` recorded on scored message
- Status assignment uses same thresholds as before

### Model/Embedding Tagging
- `ScoringModel` recorded on all Ollama-scored messages
- Embedding model stored in vector metadata
- Vectors with wrong embedding model filtered on query
