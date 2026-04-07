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

Note: the `@username` pattern should be populated from the Slack account's configured username at seed time, or left as a placeholder the user edits.

## Migration Strategy

- Check if `routing_rules` table is empty after DDL creation
- If empty: insert default rules
- If not empty: do nothing (user has already customized)

This runs as part of the SQLite repository initialization, alongside existing table creation and ALTER TABLE migrations.

## Replaces Hardcoded Logic

After this migration and the orchestrator refactor (Feature 087), the hardcoded routing in `Router.routeDeterministic()` is no longer called from the poll path. The router's deterministic rules are fully replaced by DB-backed rules evaluated by the rules engine.
