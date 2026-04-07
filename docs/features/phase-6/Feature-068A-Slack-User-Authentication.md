# Feature 068A: Slack User Authentication

**Phase:** Phase-6-Feature-068A
**Type:** Bugfix (Hotfix)
**Severity:** High
**Status:** Done
**Packages:** `internal/ui/`, `internal/repository/`, `internal/repository/implementation/sqlite/`
**Related:** Feature 068 (Slack Add Account), Feature 060 (Settings View)

---

## Problem

The Slack settings form was configured for **bot** authentication (label said "Bot Token", stored a single opaque token + workspace ID). Cue is a personal productivity tool that monitors channels *the user* is in — it should authenticate as the **user**, not as a bot application. Bot tokens (`xoxb-`) can only see channels the bot has been explicitly invited to. User tokens (`xoxp-`) see everything the user sees.

Additionally, the token field was not masked, exposing full-access credentials in the UI. The form also lacked a username field needed for @mention detection.

## Solution

### 1. Data Model

Added `Username string` field to `SlackAccount` struct in `internal/repository/service_config.go`.

### 2. SQLite Schema Migration

Added `ALTER TABLE slack_accounts ADD COLUMN username TEXT NOT NULL DEFAULT ''` migration. Updated column list, upsert, and scan to include the username field. Existing rows get empty string default.

### 3. UI Form Changes

- **Relabeled** placeholder from "Bot Token" to "User OAuth Token (xoxp-...)"
- **Masked** token entry with `Password = true`
- **Added** username entry field with placeholder "Your Slack Username (@handle)"
- **Updated** validation to require all 4 fields
- **Wired** username into `SlackAccount.Username` on save

## Test Coverage

- SQLite round-trip test for Username field persistence
- UI acceptance tests updated for 4-field form
- UI interaction tests updated for 4-field form with username
- All existing Slack and settings tests remain green

## TDD Agent Stats

| Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| UI tests | orchestrator | manual | — | 4728dc6 |
| RED (model + DB test) | Test Designer | ~30s | ~31,579 | 3653cc3 |
| GREEN (SQLite impl) | Implementer | ~36s | ~29,720 | 873f32f |
| GREEN (UI form) | orchestrator | manual | — | 9c5e747 |
