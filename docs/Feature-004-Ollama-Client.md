# Feature 4: Ollama Client + Scoring

**Phase:** Phase-1-Feature-4
**Status:** Done
**Package:** `internal/service/decisionengine/`

---

## Overview

HTTP client implementing the Scorer interface for local Ollama LLM inference. Constructs ADHD-context-aware prompts, sends them to the Ollama `/api/generate` endpoint, and parses the response through two levels of JSON extraction (outer Ollama envelope, inner scorer result). Handles markdown code block wrapping that LLMs sometimes produce.

## Design Decisions

### Two-Level JSON Parsing

Ollama's `/api/generate` returns a JSON envelope with a `response` string field. The actual scorer JSON is inside that string. This two-level parse is necessary because Ollama doesn't natively return structured output — the LLM's text response is embedded as a string.

### Markdown Code Block Extraction

LLMs frequently wrap JSON in markdown fences (` ```json ... ``` `). The `extractJSON()` function strips these before parsing. This handles both fenced and unfenced responses without requiring the LLM to be perfectly compliant.

### Context Propagation via http.NewRequestWithContext

Full timeout and cancellation support — the HTTP request respects both the `http.Client.Timeout` and any context deadline. This enables the router's fallback behavior when Ollama is slow.

### No Retry Logic

The client makes a single attempt. Retry and fallback are the caller's responsibility (the Router applies fallback scoring on error). This keeps the client simple and the retry policy centralized.

## API

### Constructor

```go
func NewOllamaClient(baseURL, model string, timeout time.Duration) (*OllamaClient, error)
```

Validates all three arguments: baseURL not empty, model not empty, timeout > 0.

### Score Method

```go
func (c *OllamaClient) Score(ctx context.Context, msg *repository.Message) (*ScorerResult, error)
```

Implements the `Scorer` interface. Builds a prompt from the message fields, sends to Ollama, parses the response.

## Prompt Template

```
You are an ADHD-friendly message importance scorer. Evaluate the following message
and return a JSON object with these fields:
- importance_score: a float from 0 to 10
- confidence_score: a float from 0.0 to 1.0
- reasoning: a brief explanation of your rating

Message details:
- Source: {source}
- Sender: {sender}
- Channel: {channel}
- Content: {rawContent}

Respond ONLY with valid JSON. Do not include any other text.
```

## Request/Response Flow

1. Build prompt from message fields (Source, Sender, Channel, RawContent)
2. Marshal `ollamaRequest{Model, Prompt, Stream: false}` to JSON
3. POST to `{baseURL}/api/generate` with context
4. Verify HTTP 200
5. Parse outer `ollamaResponse{Response string}`
6. Extract JSON from markdown fences if present
7. Parse inner `scorerResponse{ImportanceScore, ConfidenceScore, Reasoning}`
8. Return as `*ScorerResult`

## Error Handling

| Scenario | Error Message |
|---|---|
| Empty baseURL | "baseURL must not be empty" |
| Empty model | "model must not be empty" |
| Timeout <= 0 | "timeout must be greater than zero" |
| JSON marshal fails | "marshalling Ollama request: ..." |
| HTTP request fails | "sending HTTP request to Ollama: ..." |
| Non-200 status | "Ollama API returned non-200 status: {code}" |
| Response body read fails | "reading Ollama response body: ..." |
| Outer JSON parse fails | "parsing Ollama response JSON: ..." |
| Inner JSON parse fails | "parsing scorer response JSON: ..." |

## Integration Points

- **Router** (Feature 3): Consumes OllamaClient via the Scorer interface for non-deterministic messages
- **Config** (Feature 1): baseURL constructed from `ollama.host` + `ollama.port`, model from `ollama.inference_model`, timeout from `ollama.timeout_seconds`

## Test Coverage

15 test cases in `ollama_client_test.go` using testify suites:

- Constructor validation (4): valid inputs, empty baseURL, empty model, zero timeout
- Interface compliance (1): OllamaClient implements Scorer
- Happy path (1): valid response parsed to correct scores and reasoning
- Request format (2): correct HTTP method/path/model/stream, prompt includes source
- Timeout handling (1): 50ms context timeout against 500ms slow server
- JSON parsing errors (2): invalid inner JSON, invalid outer JSON
- HTTP errors (2): non-200 status code, connection refused
- Markdown extraction (1): JSON wrapped in code fences
- Prompt validation (1): prompt requests JSON format with all required fields

## TDD Agent Stats

| Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | orchestrator | — | — | ad31a75 |
| GREEN | Implementer | 496s | 26,271 | 9c7113a |
| REFACTOR | Refactorer | 150s | 38,042 | ce1de93 |
