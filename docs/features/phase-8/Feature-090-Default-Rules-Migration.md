# Feature 090: Default Rules Migration

**Phase:** Phase-8-Feature-090
**Status:** Done
**Packages:** `internal/repository/implementation/sqlite/`
**Depends on:** Feature 084

---

## Overview

On first run (or when the routing_rules table is empty), seed it with default rules that replace the current hardcoded routing logic. These defaults are fully editable and deletable by the user.

## Default Rules

| Priority | Source | Field | Pattern | Negate | Action | Rationale |
|----------|--------|-------|---------|--------|--------|-----------|
| 0 | slack | message_type | `^channel_join$` | false | notified | User added to a new channel — always important |
| 1 | slack | content | `@username` | false | notified | Direct @mention — always important |

The `@username` pattern is inserted as the literal string `@username` as a placeholder. The SQLite migration runs during repository initialization and does not have access to Slack account configuration. The user is expected to edit this rule in the Settings UI (Feature 089) to replace `@username` with their actual Slack handle (e.g., `@alice`).

## Migration Strategy

- Check if `routing_rules` table is empty after DDL creation
- If empty: insert default rules
- If not empty: do nothing (user has already customized)

This runs as part of the SQLite repository initialization in `NewSQLiteRoutingRuleRepository()`, immediately after the `CREATE TABLE IF NOT EXISTS` statement.

## Design Decisions

1. **Seeding in constructor** — Default rules are inserted in `NewSQLiteRoutingRuleRepository` rather than a separate migration function, keeping initialization atomic and simple.
2. **Raw SQL inserts** — Uses the existing `upsertRoutingRuleSQL` constant directly with `db.Exec` rather than constructing `RoutingRule` objects, since the repo instance isn't returned yet at that point.
3. **Count-based check** — `SELECT COUNT(*) FROM routing_rules` determines emptiness. Simple and reliable.
4. **No empty-state UI change** — The seeding runs before the UI loads, so users never see an empty rules table. The existing "No routing rules configured" message remains as a fallback for edge cases (e.g., user deletes all rules).

## Replaces Hardcoded Logic

After this migration and the orchestrator refactor (Feature 087), the hardcoded routing in `Router.applyDeterministicRules()` is no longer called from the poll path. The Router is removed (see Feature 087 "Router Deprecation"), and deterministic rules are fully replaced by DB-backed rules evaluated by the `RulesEngine` (Feature 085) within the orchestrator.

## Test Coverage

| Test | Verifies |
|------|----------|
| `TestNewSeedsDefaultRulesWhenEmpty` | Fresh DB gets 2 default rules with correct fields |
| `TestNewDoesNotSeedWhenRulesExist` | Pre-existing rules prevent seeding |

## TDD Agent Stats

| TDD Phase | Agent | Duration | Tokens | Commit |
|-----------|-------|----------|--------|--------|
| RED | Test Designer | ~57s | ~31,800 | b74c5c3 |
| GREEN | Implementer | ~58s | ~32,600 | 5c15f87 |
