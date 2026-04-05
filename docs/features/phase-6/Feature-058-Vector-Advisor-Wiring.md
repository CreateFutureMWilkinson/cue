# Feature 058: Vector Score Advisor Wiring

**Phase:** Phase-6-Feature-058
**Type:** Bugfix
**Severity:** Medium
**Status:** Done
**Packages:** `cmd/cue/`
**Related:** Feature 042 (Vector-Assisted Routing), Feature 043 (chromem-go Vector Database)

---

## Overview

Wires the `VectorScoreAdvisor` into the composition root (`cmd/cue/main.go`) so that vector-assisted routing is active when `vector_enabled = true` in config. Previously the router always received `nil` for the advisor parameter, silently disabling the feature.

## Bug Description

The router was initialized with `nil` for the vector score advisor parameter in `main.go`, even though:
- The `VectorScoreAdvisor` interface and implementation exist (Feature 042)
- The `vectorStore` was already created and passed to the buffer service
- Config fields `vector_enabled`, `vector_similarity_threshold`, `vector_top_n`, `vector_damping_factor` exist and are validated

The vector scoring feature was completely disabled in the production setup despite being fully implemented.

## Root Cause

The composition root in `main.go` was not updated to wire the vector advisor when Feature 042 was completed.

## Design Decisions

### Extracted `buildVectorAdvisor` helper

Rather than inlining the conditional construction in `run()`, a standalone `buildVectorAdvisor` function was extracted. This keeps `run()` readable and makes the wiring logic independently testable without booting the entire application.

### Reordered initialization in `run()`

The vector store creation was moved before the router creation so that `buildVectorAdvisor` can be called with the already-constructed `vectorStore` and `repo` before the router is instantiated.

## API

```go
func buildVectorAdvisor(
    cfg config.RouterConfig,
    querier vector.VectorQuerier,
    msgQuerier decisionengine.MessageQuerier,
) (decisionengine.VectorScoreAdvisor, error)
```

- Returns `nil, nil` when `cfg.VectorEnabled` is false (router operates without vector advice)
- Returns a fully constructed `VectorScoreAdvisor` when enabled, using config thresholds

## Error Handling

- If `buildVectorAdvisor` returns an error (e.g., nil dependencies when enabled), startup fails with a clear message
- When disabled, nil advisor is safe — the router already checks `advisor != nil` before calling `Advise()`

## Integration Points

- **Router** (`internal/service/decisionengine/router.go`) — receives the advisor; applies score adjustments after Ollama scoring
- **Vector store** (`internal/service/vector/`) — provides `VectorQuerier` for similarity search
- **Message repository** (`internal/repository/`) — provides `MessageQuerier` for user rating lookups

## Test Coverage

| Test | Behavior |
|---|---|
| `TestBuildVectorAdvisor_Enabled` | Non-nil advisor returned when `VectorEnabled=true` |
| `TestBuildVectorAdvisor_Disabled` | Nil advisor returned when `VectorEnabled=false` |

## TDD Agent Stats

| TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | Test Designer | ~46s | ~30,000 | d7a53f0 |
| GREEN | Implementer | ~64s | ~41,000 | 32501c2 |
| REFACTOR | Refactorer | ~41s | ~27,000 | a6123b8 |
