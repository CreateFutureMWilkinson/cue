# Feature 092: Structured Output + Prompt Optimization

**Phase:** Phase-8-Feature-092
**Status:** Planned
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

```go
type ollamaRequest struct {
    Model  string `json:"model"`
    Prompt string `json:"prompt"`
    Stream bool   `json:"stream"`
    Format string `json:"format,omitempty"` // NEW
}
```

Set `Format: "json"` in `sendRequest()`. This tells Ollama to constrain generation to valid JSON, eliminating the need for `extractJSON()`.

### 2. Simplified Prompt Template

**Current** (~130 input tokens):
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

**Proposed** (~80 input tokens):
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

### 3. Remove `extractJSON()`

With `format: json` enforced, the `extractJSON()` function and `markdownFence` constant become dead code. Remove them and parse the response directly.

## Calibration Impact

**This change affects the calibration loop (Feature 042).** A simplified prompt may produce a different score distribution than the current prompt.

If Feature 092 is implemented **before** Feature 094 (calibration loop redesign):
- The current `VectorScoreAdvisor` computes `adjustment = avgUserRating - avgImportanceScore` — if the new prompt shifts importance scores systematically, the advisor will over-correct until new ratings accumulate
- **Mitigation:** The ±2.0 clamp and 0.5 damping factor limit the impact. The system self-corrects as new ratings replace old ones.
- **Recommendation:** Temporarily set `vector_enabled = false` after the prompt change, re-enable once ~20 new ratings accumulate.

If Feature 092 is implemented **after** Feature 094:
- The few-shot approach (Feature 094) is inherently resilient to prompt changes — the LLM reads examples fresh each time. No special mitigation needed.

Once available, the benchmark tool (Feature 093, implemented later) can quantify the score distribution shift.

## Error Handling

No change to error handling strategy. JSON parse failures on Ollama response remain possible (e.g., model hallucinates despite JSON mode) and are handled by the existing fallback path (IS=7, CS=0.0, BUFFERED).

## Configuration

No new configuration fields. The prompt change is internal — the config file controls thresholds, not prompt content.

## Files

| File | Action |
|---|---|
| `internal/service/decisionengine/ollama_client.go` | **Modify** — add `Format` to `ollamaRequest`, simplify `promptTemplate`, remove `extractJSON()` + `markdownFence` |
| `internal/service/decisionengine/ollama_client_test.go` | **Modify** — update prompt expectations, remove `extractJSON` tests, add test for `format: json` in request body |

## Test Coverage

- Request body includes `"format": "json"` field
- Simplified prompt produces valid scoring for representative messages (mock Ollama)
- Raw JSON response (no markdown fences) parsed correctly
- Malformed JSON still triggers fallback behavior
- Reasoning field is single-sentence (validated in mock response tests)
