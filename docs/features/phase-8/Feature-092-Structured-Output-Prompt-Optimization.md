# Feature 092: Structured Output + Prompt Optimization

**Phase:** Phase-8-Feature-092
**Status:** Done
**Packages:** `internal/service/decisionengine/`
**Depends on:** None (can be implemented independently)

---

## Overview

Add Ollama's JSON mode (`"format": "json"`) to the scoring request and simplify the prompt template to reduce token count and generation time. These changes improve reliability (guaranteed valid JSON output), reduce latency per inference call, and lay the groundwork for running smaller models that benefit more from constrained output formats.

## Motivation

The current `ollama_client.go` implementation has two inefficiencies:

1. **No structured output constraint** — the request uses `/api/generate` with no `format` field. The model is free to wrap JSON in markdown fences, add preamble text, or produce malformed output. The `extractJSON()` function exists solely to work around this.
2. **Verbose prompt** — the prompt template (~130 input tokens) includes redundant instruction ("Respond ONLY with valid JSON. Do not include any other text") that becomes unnecessary with JSON mode enforced at the API level. The `reasoning` field is unconstrained, often producing 2-3 sentences that account for 50-70% of output tokens.

With the trickle queue (Feature 086) processing one message at a time, total throughput depends on per-message latency. Every token saved compounds across hundreds of messages per day.

## Changes

### 1. Add `format` Field to Request

The `ollamaRequest` struct now includes a `Format` field with `omitempty` tag:

```go
type ollamaRequest struct {
    Model  string `json:"model"`
    Prompt string `json:"prompt"`
    Stream bool   `json:"stream"`
    Format string `json:"format,omitempty"`
}
```

Score requests include `Format: "json"`. Generate requests omit it (empty string + omitempty = absent from JSON).

### 2. Refactored Request/Response Pipeline

The monolithic `sendRequest` method was decomposed into focused methods:

| Method | Responsibility |
|---|---|
| `createRequest(ctx, prompt)` | Builds HTTP request without format field (for Generate) |
| `createJSONRequest(ctx, prompt)` | Builds HTTP request with `format: "json"` (for Score) |
| `sendRequest(req)` | Sends HTTP request, checks status, returns body bytes |
| `processResponse(body)` | Parses outer Ollama JSON envelope, returns response string |
| `processJSONResponse(body)` | Calls processResponse then parses inner scorer JSON |

Call chains:
- **Score:** `createJSONRequest` → `sendRequest` → `processJSONResponse`
- **Generate:** `createRequest` → `sendRequest` → `processResponse`

### 3. Simplified Prompt Template

**Before** (~130 input tokens):
```
You are an ADHD-friendly message importance scorer. Evaluate the following message and return a JSON object with these fields:
- importance_score: a float from 0 to 10 indicating how important/urgent this message is
- confidence_score: a float from 0.0 to 1.0 indicating your confidence in the rating
- reasoning: a brief explanation of your rating

Message details:
- Source: %s
- Sender: %s
- Channel: %s
- Content: %s

Respond ONLY with valid JSON. Do not include any other text.
```

**After** (~80 input tokens):
```
Score this message's importance for an ADHD user who needs to catch critical items (deadlines, outages, @mentions) without noise.

Source: %s | Sender: %s | Channel: %s
Content: %s

{"importance_score": 0-10, "confidence_score": 0.0-1.0, "reasoning": "one sentence"}
```

Key changes:
- Removed "Respond ONLY with valid JSON" — enforced by `format: json`
- Compressed message details to single-line pipe-delimited format
- Added user context ("ADHD user who needs to catch critical items") to improve relevance
- Constrained reasoning to "one sentence" via the example JSON structure
- The example JSON at the end acts as a schema hint for smaller models

### 4. Removed Dead Code

- `extractJSON()` function — no longer needed with JSON mode
- `markdownFence` constant — only used by extractJSON
- `TestScore_ResponseWithMarkdownWrapping_ExtractsJSON` test — tests dead code path

## Calibration Impact

**This change affects the calibration loop (Feature 042).** A simplified prompt may produce a different score distribution than the current prompt.

Since Feature 092 is implemented **before** Feature 094 (calibration loop redesign):
- The current `VectorScoreAdvisor` computes `adjustment = avgUserRating - avgImportanceScore` — if the new prompt shifts importance scores systematically, the advisor will over-correct until new ratings accumulate
- **Mitigation:** The ±2.0 clamp and 0.5 damping factor limit the impact. The system self-corrects as new ratings replace old ones.
- **Recommendation:** Temporarily set `vector_enabled = false` after the prompt change, re-enable once ~20 new ratings accumulate.

## Error Handling

No change to error handling strategy. JSON parse failures on Ollama response remain possible (e.g., model hallucinates despite JSON mode) and are handled by the existing fallback path (IS=7, CS=0.0, BUFFERED).

## Configuration

No new configuration fields. The prompt change is internal — the config file controls thresholds, not prompt content.

## Files Changed

| File | Action |
|---|---|
| `internal/service/decisionengine/ollama_client.go` | **Modified** — added `Format` to `ollamaRequest`, split `sendRequest` into 5 focused methods, simplified `promptTemplate`, removed `extractJSON()` + `markdownFence` |
| `internal/service/decisionengine/ollama_client_test.go` | **Modified** — added format:json tests, added simplified prompt test, removed markdown wrapping test |

## Test Coverage

- Request body includes `"format": "json"` field for Score calls
- Request body omits `"format"` field for Generate calls
- Simplified prompt contains ADHD context, pipe-delimited metadata, JSON schema hint
- Simplified prompt does not contain removed verbose instructions
- Raw JSON response parsed correctly (existing tests)
- Malformed JSON still triggers fallback behavior (existing test)
- All existing Score/Generate tests pass with refactored pipeline

## TDD Agent Stats

| Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED (format:json) | Test Designer | ~44s | ~30,500 | 8a2ccd0 |
| GREEN (format:json) | Implementer | ~38s | ~25,100 | 4c6d28c |
| REFACTOR (split sendRequest) | Refactorer | ~105s | ~38,100 | 8461806 |
| RED (simplified prompt) | Test Designer | ~36s | ~32,500 | 599cd19 |
| GREEN (simplified prompt) | Implementer | ~24s | ~22,400 | b10bb2e |
| REFACTOR (unused ctx, SplitSeq) | orchestrator | manual | — | eb07cf5 |
| CHORE (nosec G704) | orchestrator | manual | — | e727859 |
