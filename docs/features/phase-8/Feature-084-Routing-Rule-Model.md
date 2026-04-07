# Feature 084: Routing Rule Model + DB Table

**Phase:** Phase-8-Feature-084
**Status:** Done
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
    Source    string    // "email" or "slack"
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

~~`source = "all"` was removed — sender formats differ between email and Slack, making cross-source matching impractical.~~ Source must be `"email"` or `"slack"`.

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
    source TEXT NOT NULL,          -- "email", "slack"
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

Validation lives on the `RoutingRule.Validate()` method and is called by `UpsertRule` before persisting:

- `Pattern` must be a valid Go regexp (compile check on upsert)
- `Source` must be one of: "email", "slack"
- `Field` must be valid for the given source
- `Action` must be "notified" or "ignored" (lowercase — the orchestrator maps these to capitalized status values "Notified"/"Ignored" when applying, see Feature 087)
- `Priority` must be >= 0

All validation errors wrap `ErrInvalidRoutingRule`.

## Design Decisions

- **Validation on model, not repository** — `Validate()` lives on `RoutingRule` so callers (UI, API) can validate before attempting persistence. `UpsertRule` calls it as a safety net.
- **No "all" source** — Originally planned, removed because sender formats differ between email (@user) and Slack (U12345/display name), making cross-source regex matching impractical.
- **No encryption** — Routing rules contain no secrets; plain SQLite storage with no encryptor dependency.
- **Separate repository interface** — `RoutingRuleRepository` is independent of `ServiceConfigRepository`, following the pattern of Category/Todo/Schedule repos.

## Test Coverage

- 9 unit tests for `RoutingRule.Validate()` covering valid rules, invalid source/field/action/priority/regex, and valid field combinations per source
- 12 integration tests for `SQLiteRoutingRuleRepository` covering table creation, insert, update, validation error, get found/not found, list empty/sorted, list by source filtered/empty, delete, and idempotent delete

## TDD Agent Stats

| TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED (Validate) | Test Designer | ~42s | ~26,500 | b181f51 |
| GREEN (Validate) | Implementer | ~30s | ~24,700 | 21d3e2f |
| REFACTOR (Validate) | Refactorer | ~72s | ~28,400 | 2dfa301 |
| RED (SQLite constructor+upsert) | Test Designer | ~63s | ~44,800 | 98218ea |
| GREEN (SQLite constructor+upsert) | Implementer | ~42s | ~27,900 | bea52d5 |
| RED (GetRule) | Test Designer | ~29s | ~26,300 | dc0f047 |
| GREEN (GetRule) | Implementer | ~37s | ~31,000 | eb5cf8b |
| RED (List+Delete) | Test Designer | ~61s | ~32,800 | 79dbe47 |
| GREEN (List+Delete) | Implementer | ~34s | ~28,500 | ba8d789 |
| REFACTOR (all SQLite) | Refactorer | ~79s | ~39,100 | f2b252e |
