# Feature 090: Default Rules Migration

**Phase:** Phase-8-Feature-090
**Status:** Planned
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

The `@username` pattern is inserted as the literal string `@username` as a placeholder. The SQLite migration runs during repository initialization and does not have access to Slack account configuration. The user is expected to edit this rule in the Settings UI (Feature 089) to replace `@username` with their actual Slack handle (e.g., `@alice`). The empty-state message in the Rules tab should mention this.

## Migration Strategy

- Check if `routing_rules` table is empty after DDL creation
- If empty: insert default rules
- If not empty: do nothing (user has already customized)

This runs as part of the SQLite repository initialization, alongside existing table creation and ALTER TABLE migrations.

## Replaces Hardcoded Logic

After this migration and the orchestrator refactor (Feature 087), the hardcoded routing in `Router.applyDeterministicRules()` is no longer called from the poll path. The Router is removed (see Feature 087 "Router Deprecation"), and deterministic rules are fully replaced by DB-backed rules evaluated by the `RulesEngine` (Feature 085) within the orchestrator.
