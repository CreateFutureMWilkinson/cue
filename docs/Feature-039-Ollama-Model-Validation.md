# Feature 039: Ollama Model Validation on Startup

**Phase:** Phase-4-Feature-039
**Status:** Planned
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
      "model": "neural-chat:latest",
      ...
    }
  ]
}
```

Model names in Ollama include an implicit `:latest` tag when no tag is specified. The validation must handle both `"neural-chat"` (from config) matching `"neural-chat:latest"` (from API).

### Validation Function

A standalone function rather than a method on `OllamaClient`, since this runs once at startup before the client is used for scoring. It takes the Ollama base URL and the model names to check:

```go
func ValidateOllamaModels(ctx context.Context, baseURL string, models []string) error
```

This returns an error listing all missing models (not just the first one found missing), so the user can pull everything needed in one go.

### Startup Behavior

| Scenario | Behavior |
|---|---|
| Ollama reachable, both models present | Continue startup normally |
| Ollama reachable, one or both models missing | Log error listing missing models with `ollama pull <model>` instructions, exit with non-zero status |
| Ollama unreachable (connection refused) | Log warning: "Ollama not reachable at <host>:<port>, skipping model validation. Scoring will use fallback." Continue startup — Ollama may start later |
| Ollama returns unexpected response | Log warning, continue startup |

The distinction: missing models is a **configuration error** (user needs to fix it), while Ollama being down is a **transient condition** (it may come up later and the app has fallback scoring).

### Tag Matching

Config value `"neural-chat"` should match Ollama model `"neural-chat:latest"`. The matching logic:
- If the config value contains `:`, match exactly
- If not, match against `<name>:latest`
- Also match the bare name from the `name` field (Ollama sometimes returns both forms)

## API

### New Function

```go
// internal/service/decisionengine/ollama_models.go

// OllamaModel represents a model returned by the Ollama API.
type OllamaModel struct {
    Name string `json:"name"`
}

// OllamaTagsResponse is the response from GET /api/tags.
type OllamaTagsResponse struct {
    Models []OllamaModel `json:"models"`
}

// ValidateOllamaModels checks that all specified models are available in the
// local Ollama instance. Returns an error listing any missing models.
func ValidateOllamaModels(ctx context.Context, baseURL string, models []string) error
```

### Usage in main.go

```go
ollamaURL := fmt.Sprintf("http://%s:%d", cfg.Ollama.Host, cfg.Ollama.Port)
models := []string{cfg.Ollama.InferenceModel, cfg.Ollama.EmbeddingModel}

if err := decisionengine.ValidateOllamaModels(ctx, ollamaURL, models); err != nil {
    log.Fatalf("Ollama model validation failed: %v", err)
}
```

## Error Handling

| Scenario | Error Message |
|---|---|
| Single model missing | `missing Ollama model "neural-chat"; run: ollama pull neural-chat` |
| Multiple models missing | `missing Ollama models: "neural-chat", "nomic-embed-text"; run: ollama pull neural-chat && ollama pull nomic-embed-text` |
| Ollama unreachable | No error returned — log a warning, return nil |
| Unexpected API response | No error returned — log a warning, return nil |

## Integration Points

- **Feature 038** (Main Wiring): Called early in startup, after config load, before orchestrator creation
- **`internal/config/config.go`**: Reads `cfg.Ollama.InferenceModel` and `cfg.Ollama.EmbeddingModel`

## Test Coverage

Using a test HTTP server (httptest.NewServer) to mock Ollama API:

- Both models present — no error
- One model missing — error names the missing model
- Both models missing — error names both models
- Tag matching: config `"neural-chat"` matches API `"neural-chat:latest"`
- Tag matching: config `"neural-chat:v2"` matches exactly
- Ollama unreachable — no error (warning only)
- Ollama returns invalid JSON — no error (warning only)
- Ollama returns empty model list — error for all requested models
- Context cancellation — returns context error
- Empty model list input — no error (nothing to validate)

## Files

| File | Action |
|---|---|
| `internal/service/decisionengine/ollama_models.go` | **New** — ValidateOllamaModels function + types |
| `internal/service/decisionengine/ollama_models_test.go` | **New** — full test suite with httptest server |
| `cmd/cue/main.go` | Modify — call ValidateOllamaModels at startup |
