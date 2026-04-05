# Feature 3: Deterministic Routing Rules

**Phase:** Phase-1-Feature-3
**Status:** Done
**Package:** `internal/service/decisionengine/`

---

## Overview

Decision engine router that applies deterministic rules first (channel join, @mention), then falls back to scorer-based evaluation via the Scorer interface. Messages are routed to one of three statuses — Notified, Buffered, or Ignored — based on configurable importance and confidence thresholds.

## Design Decisions

### Deterministic Rules Short-Circuit Scorer

Channel join (IS=9) and @mention (IS=8) are evaluated before the scorer is called. This ensures critical events are never misclassified by LLM inference and avoids unnecessary Ollama round-trips.

### No Sender-Based Importance

Per CLAUDE.md: importance is NEVER determined by sender identity. A message from the CEO is routed identically to one from an intern. Only content and event type matter.

### Inclusive Thresholds

Status assignment uses `>=` (not `>`). A message with exactly IS=7.0 and CS=0.8 is Notified, not Buffered. This matches the spec in CLAUDE.md Section 4.

### Fallback on Scorer Error

If the scorer fails (timeout, invalid response), the message gets IS=7, CS=0.0, Status=Buffered. This is a safe default — the user reviews it manually rather than missing it or being falsely alerted.

## API

### Interfaces

```go
// Scorer evaluates message content via LLM inference.
type Scorer interface {
    Score(ctx context.Context, msg *repository.Message) (*ScorerResult, error)
}

type ScorerResult struct {
    ImportanceScore float64 // 0-10
    ConfidenceScore float64 // 0.0-1.0
    Reasoning       string
}
```

### Constructor

```go
func NewRouter(scorer Scorer, usernames []string, cfg RouterConfig) (*Router, error)
```

Requires non-nil scorer and non-empty usernames list. RouterConfig provides thresholds.

### Methods

```go
// Route a single message through deterministic rules, then scorer.
func (r *Router) Route(ctx context.Context, msg *repository.Message) (*repository.Message, error)

// RouteBatch routes multiple messages. Individual scorer errors don't abort the batch.
func (r *Router) RouteBatch(ctx context.Context, msgs []*repository.Message) ([]*repository.Message, error)
```

## Routing Logic

### Deterministic Rules (Applied First)

| Rule | Condition | IS | CS | Status |
|---|---|---|---|---|
| Channel join | `MessageType == "channel_join"` | 9.0 | 1.0 | Notified |
| @mention | Content contains `@username` (case-insensitive) | 8.0 | 1.0 | Notified |

Channel join takes precedence over @mention.

### Threshold-Based Routing (Scorer Results)

```
IS >= threshold AND CS >= threshold → Notified
IS >= threshold AND CS < threshold  → Buffered
IS < threshold                      → Ignored
```

Default thresholds: importance=7, confidence=0.8.

### Fallback

```
Scorer error → IS=7.0, CS=0.0, Status=Buffered
Reasoning includes the error message for debugging.
```

## Error Handling

| Scenario | Behavior |
|---|---|
| Nil scorer in constructor | Error returned |
| Empty usernames in constructor | Error returned |
| Scorer returns error during Route | Fallback applied, message still routed |
| Scorer error during RouteBatch | Fallback for that message, batch continues |

## Test Coverage

23 test cases in `router_test.go` using testify suites:

- Constructor validation (3): nil scorer, empty usernames, valid inputs
- Deterministic rules (6): channel join, @mention, case insensitivity, multiple usernames, precedence, scorer short-circuit (panic scorer)
- Scorer-based routing (7): high/high→Notified, high/low→Buffered, low→Ignored, exact threshold, below threshold, reasoning preservation, custom thresholds
- Fallback (2): error→Buffered, error message in reasoning
- Batch processing (3): mixed messages, empty slice, scorer failure doesn't abort batch
- Sender independence (1): CEO vs intern identical routing

## TDD Agent Stats

| Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | orchestrator | — | — | 226cc71 |
| GREEN | Implementer | 118s | 24,324 | 3eee015 |
| REFACTOR | Refactorer | 93s | 33,577 | 21b9f14 |
