# Feature 085: Rules Engine

**Phase:** Phase-8-Feature-085
**Status:** Planned
**Packages:** `internal/service/decisionengine/`
**Depends on:** Feature 084

---

## Overview

Implement the rules evaluation engine that takes a message and a sorted list of routing rules, evaluates each rule against the message's fields, and returns the action from the first matching rule or "queue" if no rule matches.

## API

```go
type RulesEngine struct {
    rules []*repository.RoutingRule // pre-sorted by priority
}

func NewRulesEngine(rules []*repository.RoutingRule) *RulesEngine

// Evaluate tests the message against all rules in priority order.
// Returns ("notified", rule) or ("ignored", rule) on first match.
// Returns ("queue", nil) if no rule matches.
func (e *RulesEngine) Evaluate(msg *repository.Message) (action string, matched *repository.RoutingRule)
```

## Matching Logic

For each rule (sorted by priority ascending):

1. Check source scope: if `rule.Source != "all"` and `rule.Source != msg.Source`, skip
2. Extract field value from message based on `rule.Field`
3. Compile and match regex: `regexp.MustCompile(rule.Pattern).MatchString(fieldValue)`
4. If `rule.Negate`, invert the match result
5. If matched and `rule.Enabled`, return `rule.Action`
6. If no rule matches, return `"queue"`

## Field Extraction

```go
func extractField(msg *repository.Message, field string) string {
    switch field {
    case "sender":
        return msg.Sender
    case "subject":
        // Email subject is first line of RawContent (Subject\nBody format)
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

- Regexps should be pre-compiled when the engine is constructed (not on every evaluation)
- Rules list should be refreshed when rules are added/edited/deleted, not on every message
- Invalid regexps should be skipped with a warning (not crash the engine)

## Unmatched Default

Messages not matched by any rule always return `"queue"`. This is not configurable.
