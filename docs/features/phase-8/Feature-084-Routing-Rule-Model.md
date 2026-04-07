# Feature 084: Routing Rule Model + DB Table

**Phase:** Phase-8-Feature-084
**Status:** Planned
**Packages:** `internal/repository/`, `internal/repository/implementation/sqlite/`
**Depends on:** Features 031, 032

---

## Overview

Define the `RoutingRule` data model and SQLite persistence layer for user-configurable deterministic routing rules. Rules are regex-based pattern matchers that evaluate message fields and route to NOTIFIED or IGNORED without LLM involvement.

## Data Model

```go
type RoutingRule struct {
    ID        uuid.UUID
    Priority  int       // 0 = highest, ascending. Controls evaluation order.
    Source    string    // "email", "slack", or "all"
    Field     string    // Field to match against (source-dependent)
    Negate    bool      // true = "not matches", false = "matches"
    Pattern   string    // Go regexp pattern
    Action    string    // "notified" or "ignored"
    Enabled   bool
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

## Available Fields by Source

| Source | Field | Description |
|--------|-------|-------------|
| email | `sender` | From address |
| email | `subject` | Email subject line |
| slack | `sender` | User ID/name |
| slack | `channel` | Channel name |
| slack | `content` | Message text |
| slack | `message_type` | e.g., "channel_join" |

Rules with `source = "all"` can only match fields common to both sources (`sender`). Validation must reject any field other than `sender` when source is `"all"`.

## Actions

Only two actions:

- **NOTIFIED** — the user is certain they want to know about this
- **IGNORED** — the user is certain they don't care

BUFFERED is exclusively an Ollama output. Deterministic rules represent certainty, so BUFFERED doesn't apply.

## Repository Interface

```go
type RoutingRuleRepository interface {
    ListRules(ctx context.Context) ([]*RoutingRule, error)
    ListRulesBySource(ctx context.Context, source string) ([]*RoutingRule, error)
    GetRule(ctx context.Context, id uuid.UUID) (*RoutingRule, error)
    UpsertRule(ctx context.Context, rule *RoutingRule) error
    DeleteRule(ctx context.Context, id uuid.UUID) error
}
```

`ListRules` and `ListRulesBySource` return rules sorted by priority ascending.

## SQLite Schema

```sql
CREATE TABLE routing_rules (
    id TEXT PRIMARY KEY,
    priority INTEGER NOT NULL,
    source TEXT NOT NULL,          -- "email", "slack", "all"
    field TEXT NOT NULL,           -- "sender", "subject", "channel", etc.
    negate INTEGER NOT NULL DEFAULT 0,
    pattern TEXT NOT NULL,         -- Go regexp
    action TEXT NOT NULL,          -- "notified", "ignored"
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX idx_routing_rules_priority ON routing_rules(priority);
CREATE INDEX idx_routing_rules_source ON routing_rules(source);
```

## Validation

- `Pattern` must be a valid Go regexp (compile check on upsert)
- `Source` must be one of: "email", "slack", "all"
- `Field` must be valid for the given source
- `Action` must be "notified" or "ignored" (lowercase — the orchestrator maps these to capitalized status values "Notified"/"Ignored" when applying, see Feature 087)
- `Priority` must be >= 0
- `Field` must be `"sender"` when `Source` is `"all"` (only field common to both email and slack)
