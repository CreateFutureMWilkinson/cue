# Feature 044: Ollama Scorer Wiring

**Phase:** Phase-5-Feature-044
**Status:** Planned
**Packages:** `internal/service/decisionengine/`, `cmd/cue/`
**Depends on:** Feature 039 (Ollama Model Validation)

---

## Overview

Replace the `placeholderScorer` in `cmd/cue/main.go` with the real `OllamaClient` that calls the local Ollama inference API. The `OllamaClient` already exists and is fully tested (`internal/service/decisionengine/ollama_client.go`), but `main.go` bypasses it with a placeholder that returns hardcoded `ImportanceScore: 5.0, ConfidenceScore: 0.5` for every message.

## Motivation

The `OllamaClient` was built and tested in Feature 004. It implements the `Scorer` interface and calls Ollama's `/api/generate` endpoint with a structured prompt. However, `main.go` never instantiates it — instead using a `placeholderScorer` struct (lines 253–261) that makes the entire routing engine inert. Every non-deterministic message gets IS=5.0/CS=0.5, which means it's always IGNORED (below the default threshold of IS≥7).

## Design Decisions

### Direct Wiring

The `OllamaClient` constructor already exists:

```go
func NewOllamaClient(cfg OllamaClientConfig) (*OllamaClient, error)
```

The wiring in `main.go` replaces:

```go
// Before:
router, err := decisionengine.NewRouter(&placeholderScorer{}, usernames, routerCfg)

// After:
ollamaClient, err := decisionengine.NewOllamaClient(decisionengine.OllamaClientConfig{
    Host:           cfg.Ollama.Host,
    Port:           cfg.Ollama.Port,
    InferenceModel: cfg.Ollama.InferenceModel,
    TimeoutSeconds: cfg.Ollama.TimeoutSeconds,
})
router, err := decisionengine.NewRouter(ollamaClient, usernames, routerCfg)
```

### Startup Validation Ordering

With Feature 039 (Ollama Model Validation) in place, the startup sequence becomes:

1. Load config
2. Validate Ollama models are available (Feature 039)
3. Create `OllamaClient` with validated config
4. Create Router with real scorer

If Ollama is unreachable at startup, Feature 039 logs a warning and continues. The `OllamaClient` will hit the fallback path (IS=7, CS=0.0 → BUFFERED) when it can't reach Ollama at scoring time, which is the designed behavior.

### Remove Placeholder

Delete the `placeholderScorer` struct entirely from `main.go`. It has no tests and serves no purpose once the real client is wired.

## Error Handling

| Scenario | Behavior |
|---|---|
| `OllamaClient` constructor fails | Fatal error, exit (bad config) |
| Ollama unreachable at scoring time | Existing fallback: IS=7, CS=0.0 → BUFFERED |
| Ollama returns invalid JSON | Existing fallback: IS=7, CS=0.0 → BUFFERED |

All error handling already exists in `OllamaClient` and `Router` — this feature is purely wiring.

## Integration Points

- **Feature 004** (`ollama_client.go`): Already implemented, now wired
- **Feature 039** (Ollama Model Validation): Validates models before client creation
- **`cmd/cue/main.go`**: Replace placeholder with real instantiation

## Test Coverage

This is a wiring change in the composition root. Verified through:

- `OllamaClient` unit tests already pass (Feature 004)
- Router integration tests already pass with mock scorer
- Manual verification: start app with Ollama running, send message, observe real scoring
- `just test` passes across entire project (no test changes expected)

## Files

| File | Action |
|---|---|
| `cmd/cue/main.go` | **Modify** — replace `placeholderScorer` with `OllamaClient`, remove placeholder struct |
