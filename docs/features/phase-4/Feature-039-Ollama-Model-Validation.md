# Feature 039: Ollama Model Validation on Startup

**Phase:** Phase-4-Feature-039
**Status:** Done
**Package:** `internal/service/decisionengine/`
**Depends on:** None (independent of Features 031-038)

---

## Overview

On application startup, verify that the Ollama models configured in `config.toml` (`inference_model` and `embedding_model`) are actually available in the local Ollama instance. If a model is missing, the app logs a clear error and exits gracefully rather than failing later with a cryptic Ollama API error during message scoring.

## Design Decisions

### Ollama `/api/tags` Endpoint

Ollama exposes `GET /api/tags` which returns a JSON list of all locally available models. This is the standard way to enumerate installed models without pulling or running anything.

Response shape:
```json
{
  "models": [
    {
      "name": "neural-chat:latest",
      "model": "neural-chat:latest"
    }
  ]
}
```

Model names in Ollama include an implicit `:latest` tag when no tag is specified. The validation handles both `"neural-chat"` (from config) matching `"neural-chat:latest"` (from API).

### Standalone Function

A standalone function rather than a method on `OllamaClient`, since this runs once at startup before the client is used for scoring:

```go
func ValidateOllamaModels(ctx context.Context, baseURL string, models []string) error
```

Returns an error listing all missing models (not just the first one found missing), so the user can pull everything needed in one go.

### Startup Behavior

| Scenario | Behavior |
|---|---|
| Ollama reachable, both models present | Continue startup normally |
| Ollama reachable, one or both models missing | Log error listing missing models with `ollama pull` instructions, exit with non-zero status |
| Ollama unreachable (connection refused) | Return nil — Ollama may start later, app has fallback scoring |
| Ollama returns unexpected response | Return nil — log warning, continue startup |

The distinction: missing models is a **configuration error** (user needs to fix it), while Ollama being down is a **transient condition**.

### Tag Matching

Config value `"neural-chat"` matches Ollama model `"neural-chat:latest"`. The matching logic:
- If the config value contains `:`, match exactly
- If not, append `:latest` and match against API model names

### Helper Extraction (Refactor)

Extracted two helpers from the main function:
- `isModelAvailable()` — encapsulates tag-aware lookup logic
- `formatMissingModelsError()` — builds user-friendly error with `ollama pull` instructions

## API

### New Function

```go
// internal/service/decisionengine/ollama_models.go

func ValidateOllamaModels(ctx context.Context, baseURL string, models []string) error
```

### Usage in main.go

```go
ollamaURL := fmt.Sprintf("http://%s:%d", cfg.Ollama.Host, cfg.Ollama.Port)
if err := decisionengine.ValidateOllamaModels(ctx, ollamaURL, []string{
    cfg.Ollama.InferenceModel,
    cfg.Ollama.EmbeddingModel,
}); err != nil {
    return fmt.Errorf("ollama model validation: %w", err)
}
```

## Error Handling

| Scenario | Error Message |
|---|---|
| Single model missing | `missing Ollama model "neural-chat"; run: ollama pull neural-chat` |
| Multiple models missing | `missing Ollama models: "neural-chat", "nomic-embed-text"; run: ollama pull neural-chat && ollama pull nomic-embed-text` |
| Ollama unreachable | No error returned — return nil |
| Unexpected API response | No error returned — return nil |

## Integration Points

- **Feature 038** (Main Wiring): Called early in startup, after config load, before orchestrator creation
- **`internal/config/config.go`**: Reads `cfg.Ollama.InferenceModel` and `cfg.Ollama.EmbeddingModel`

## Test Coverage

12 tests using `httptest.NewServer` to mock Ollama API:

| Test | Validates |
|---|---|
| `TestBothModelsPresent` | Both models found — no error |
| `TestOneModelMissing` | Error names the missing model, not the present one |
| `TestBothModelsMissing` | Error names both missing models with `ollama pull` hint |
| `TestImplicitLatestTag` | Config `"neural-chat"` matches API `"neural-chat:latest"` |
| `TestExactTagMatch` | Config `"neural-chat:v2"` matches `"neural-chat:v2"` exactly |
| `TestExactTagMismatch` | Config `"neural-chat:v2"` does NOT match `"neural-chat:latest"` |
| `TestOllamaUnreachable` | Connection refused returns nil (warning only) |
| `TestInvalidJSON` | Malformed JSON returns nil (warning only) |
| `TestEmptyModelList` | Empty Ollama response errors for all requested models |
| `TestContextCancellation` | Cancelled context returns `context.Canceled` |
| `TestEmptyModelListInput` | Empty input slice returns nil immediately |
| `TestNilModelListInput` | Nil input slice returns nil immediately |

## Files

| File | Action |
|---|---|
| `internal/service/decisionengine/ollama_models.go` | **New** — ValidateOllamaModels function + helpers |
| `internal/service/decisionengine/ollama_models_test.go` | **New** — 12-test suite with httptest server |
| `cmd/cue/main.go` | Modified — call ValidateOllamaModels at startup |

## TDD Agent Stats

| Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | Test Designer | ~45s | ~22,000 | b6f7834 |
| GREEN | Implementer | ~37s | ~23,000 | 670fe2a |
| REFACTOR | Refactorer | ~68s | ~26,000 | f3c7770 |
