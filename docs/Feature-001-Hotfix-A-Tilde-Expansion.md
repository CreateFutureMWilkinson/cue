# Feature 001-Hotfix-A: Tilde Expansion on Default Config

## Overview

Fixes a crash on first run where `config.Load()` returned the default config without expanding `~/` in path fields. SQLite received the literal path `~/.cue/messages.db`, could not open it, and returned error 14 (SQLITE_CANTOPEN).

## Root Cause

`config.Load()` has two return paths:

1. **Config file exists:** Parses TOML, calls `expandPaths(cfg)`, returns — works correctly.
2. **Config file does not exist (first run):** Creates default file, returns `defaultConfig()` **without** calling `expandPaths()` — bug.

The `expandPaths()` function replaces leading `~/` with `os.UserHomeDir()` in `Database.Path`, `Logging.LogDir`, `Notification.AudioDir`, and `Planner.TimerSound`.

## Fix

Added `expandPaths(cfg)` call in the first-run branch of `Load()`, before returning the default config. One-line change in `internal/config/config.go`.

## Files Changed

| File | Change |
|---|---|
| `internal/config/config.go` | Added `expandPaths(cfg)` on first-run path |
| `internal/config/config_test.go` | Updated assertion to expect expanded path |

## Error Handling

No new error paths. `expandPaths()` already handles `os.UserHomeDir()` failure gracefully (no-op).

## Test Coverage

| Test | Description |
|---|---|
| `TestLoadCreatesDefaultFile` | Asserts `Database.Path` is expanded (no `~/` prefix) when config is created for the first time |

## TDD Agent Stats

| Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | Test Designer | 361s | 20,431 | bc94eb4 |
| GREEN | Implementer | 20s | 18,190 | 477aae2 |
| REFACTOR | Refactorer | 37s | 26,637 | — (no changes) |
