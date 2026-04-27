# Feature 103: Routing Rules API

**Phase:** Phase-9-Feature-103
**Status:** Done
**Package:** `internal/server/handler/`, `internal/repository/`, `internal/service/decisionengine/`, `internal/ui/presenter/`

---

## Overview

Expose CRUD operations for routing rules via REST API and migrate the data model from single-field matching to multi-pattern matching. Rules are deterministic filters evaluated before Ollama scoring, with priority ordering and regex-based pattern matching.

### Model Migration

The RoutingRule model was redesigned from single-field matching (`Source`/`Field`/`Negate`/`Pattern`) to multi-pattern matching:

| Old Field | New Field(s) | Notes |
|---|---|---|
| `Source` | `SourceType` | Renamed for clarity |
| `Field` + `Pattern` | `ChannelPattern`, `ContentPattern`, `MessageType` | Split into dedicated pattern fields |
| `Negate` | (removed) | No longer needed with multi-pattern AND logic |
| (none) | `Name` | User-friendly label |
| (none) | `SourceAccount` | Optional FK to service config account |

All set fields use AND logic — every non-empty field must match for the rule to trigger.

## Endpoints

### List Rules

```
GET /api/v1/rules
GET /api/v1/rules?source_type=slack
GET /api/v1/rules?source_account=<uuid>
```

Returns rules ordered by priority ascending. Supports filtering by source type or source account.

**Response:**
```json
{
  "rules": [
    {
      "id": "uuid",
      "name": "Channel Join",
      "priority": 0,
      "source_type": "slack",
      "channel_pattern": "",
      "content_pattern": "",
      "message_type": "channel_join",
      "action": "notified",
      "enabled": true,
      "created_at": "2026-04-01T00:00:00Z",
      "updated_at": "2026-04-01T00:00:00Z"
    }
  ],
  "count": 1
}
```

### Get Rule

```
GET /api/v1/rules/{id}
```

### Create Rule

```
POST /api/v1/rules
```

**Request:**
```json
{
  "name": "Deployment Alerts",
  "source_type": "slack",
  "channel_pattern": "deploy-.*",
  "content_pattern": "(failed|error|rollback)",
  "message_type": "",
  "action": "notified"
}
```

Returns 201 with the created rule. Validation errors return 400.

### Update Rule

```
PUT /api/v1/rules/{id}
```

Full replacement. Same body as create. Returns 200.

### Patch Rule (Reorder / Toggle)

```
PATCH /api/v1/rules/{id}
```

**Request (reorder):**
```json
{"priority": 5}
```

**Request (toggle):**
```json
{"enabled": false}
```

Returns 204 No Content.

### Delete Rule

```
DELETE /api/v1/rules/{id}
```

Idempotent. Returns 204 No Content.

## Design Decisions

1. **Priority conflicts**: Duplicate priorities allowed. PATCH with priority triggers `ReorderRule` which shifts adjacent rules.
2. **Built-in rule deletion**: Allowed. Users have full control; defaults can be recreated.
3. **Rule change propagation**: Immediate reload via `RulesPresenter.WithReloader` callback that calls `Orchestrator.ReloadRules`.
4. **Regex safety**: Compile-time validation only. Go's RE2 guarantees linear-time matching.

## Architecture

```
HTTP Handler (rules.go)
  → RulesManager interface
    → RulesPresenter (implements interface)
      → RoutingRuleRepository (persistence)
      → reloader callback → Orchestrator.ReloadRules
```

The `RulesPresenter` is wired in `composition.go` with a reload callback that reloads rules from the database and passes them to the orchestrator's rules engine.

## Error Handling

| Error | HTTP Status |
|---|---|
| Invalid UUID path param | 400 |
| Invalid JSON body | 400 |
| Validation error (bad regex, invalid source type) | 400 |
| Rule not found | 404 |
| Internal error | 500 |

## Test Coverage

- **Repository model tests**: 8 tests (validation, source types, regex patterns)
- **SQLite implementation tests**: 14 tests (CRUD, listing, filtering, seeding)
- **Rules engine tests**: 12 tests (matching, AND logic, source account scoping, priority)
- **Handler tests**: 20 tests (all endpoints, error cases, filtering)
- **Presenter tests**: 12 tests (delegation, reorder, toggle, reload callback)
- **Server wiring tests**: 2 tests (routes registered/not registered)
- **UI acceptance tests**: 3 skipped (old form elements removed), remainder updated

## Files Changed

| File | Change |
|---|---|
| `internal/repository/routing_rule.go` | Model migration: new struct, Validate(), interface |
| `internal/repository/implementation/sqlite/routing_rule_impl.go` | New schema, scan, upsert, seeds, list methods |
| `internal/service/decisionengine/rules_engine.go` | Multi-pattern AND matching, compiled channel/content patterns |
| `internal/server/handler/rules.go` | **New** — all rule handlers + RulesManager interface |
| `internal/ui/presenter/rules_presenter.go` | Reload callback, new list methods |
| `internal/server/server.go` | Rules field in Deps, route registration |
| `internal/server/composition.go` | Wire RulesPresenter with reload callback |
| `internal/service/orchestrator/orchestrator.go` | Updated reasoning format |
| `internal/ui/settings_view.go` | Updated form for new model fields |
