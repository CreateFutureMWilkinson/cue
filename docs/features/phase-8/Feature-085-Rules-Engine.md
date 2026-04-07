# Feature 085: Rules Engine

**Phase:** Phase-8-Feature-085
**Status:** Done
**Packages:** `internal/service/decisionengine/`
**Depends on:** Feature 084

---

## Overview

Implement the rules evaluation engine that takes a message and a sorted list of routing rules, evaluates each rule against the message's fields, and returns the action from the first matching rule or "queue" if no rule matches.

## API

```go
type compiledRule struct {
    rule    *repository.RoutingRule
    pattern *regexp.Regexp
}

type RulesEngine struct {
    rules []compiledRule // pre-sorted by priority, regexps pre-compiled
}

func NewRulesEngine(rules []*repository.RoutingRule) *RulesEngine

// Evaluate tests the message against all rules in priority order.
// Returns ("notified", rule) or ("ignored", rule) on first match.
// Returns ("queue", nil) if no rule matches.
func (e *RulesEngine) Evaluate(msg *repository.Message) (action string, matched *repository.RoutingRule)
```

## Matching Logic

For each rule (sorted by priority ascending):

1. Check source scope: if `rule.Source != msg.Source`, skip
2. Extract field value from message based on `rule.Field`
3. Match against pre-compiled regex: `compiledRule.pattern.MatchString(fieldValue)`
4. If `rule.Negate`, invert the match result
5. If matched and `rule.Enabled`, return `rule.Action`
6. If no rule matches, return `"queue"`

## Field Extraction

The email watcher stores messages with `RawContent = subject + "\n" + body` (see `internal/service/watcher/email.go`, `convertEmailMessage()`). The `"subject"` field extraction relies on this format.

```go
func extractField(msg *repository.Message, field string) string {
    switch field {
    case "sender":
        return msg.Sender
    case "subject":
        // Email subject is first line of RawContent (Subject\nBody format,
        // set by EmailWatcher.convertEmailMessage)
        if idx := strings.Index(msg.RawContent, "\n"); idx >= 0 {
            return msg.RawContent[:idx]
        }
        return msg.RawContent
    case "channel":
        return msg.Channel
    case "content":
        return msg.RawContent
    case "message_type":
        return msg.MessageType
    default:
        return ""
    }
}
```

## Performance

- Regexps are pre-compiled in `NewRulesEngine()` using `regexp.Compile()` (not `MustCompile`)
- Invalid patterns should never reach the engine — Feature 084 validates patterns at upsert time, and Feature 089 validates in the UI before save. `NewRulesEngine()` uses `regexp.Compile()` as a defensive measure; if a pattern fails to compile, it is skipped with a logged warning rather than panicking
- Rules list should be refreshed when rules are added/edited/deleted, not on every message (the orchestrator calls `NewRulesEngine()` with fresh rules from the DB)

## Unmatched Default

Messages not matched by any rule always return `"queue"`. This is not configurable.

## Test Coverage

14 tests (9 top-level + 5 subtests in table-driven field extraction):

| Test | Behavior |
|---|---|
| TestNewRulesEngineFiltersDisabledRules | Disabled rules excluded at construction |
| TestNewRulesEngineSkipsInvalidPatterns | Invalid regex logged via slog.Warn, rule skipped |
| TestNewRulesEngineSortsByPriority | Rules sorted by priority ascending regardless of input order |
| TestNewRulesEngineEmptyRules | Nil/empty input returns engine that evaluates to "queue" |
| TestEvaluateSourceScopeMismatch | Rule source != message source → skip |
| TestEvaluateFieldExtraction (table) | sender, channel, content, message_type, subject (first line) |
| TestEvaluateSubjectExtractionNoNewline | Subject = full RawContent when no newline |
| TestEvaluateNegation | Negate inverts match: non-match → true, match → false |
| TestEvaluateFirstMatchWins | Lowest priority number wins after sorting |

## TDD Agent Stats

| TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED (construction) | Test Designer | ~59s | ~31,500 | 37f6f98 |
| GREEN (construction + evaluate) | Implementer | ~35s | ~26,300 | 687575c |
| REFACTOR (extractField, sort) | Refactorer | ~69s | ~30,500 | 3056de0 |
| RED (evaluate behaviors) | Test Designer | ~54s | ~31,400 | 28bfaf2 |
| REFACTOR (test helpers) | Refactorer | ~148s | ~37,200 | 596a433 |
