# Feature 033: Watcher Config Decoupling

**Phase:** Phase-4-Feature-033
**Status:** Done
**Package:** `internal/service/watcher/`
**Depends on:** None (parallel with Features 031/032)

---

## Overview

Replace the watcher package's dependency on `internal/config` types (`config.SlackConfig`, `config.EmailConfig`) with locally-defined config structs. This decouples watchers from the TOML config system, enabling construction from any config source (database, tests, etc.).

## Design Decisions

### Local Config Structs

Each watcher defines its own minimal config struct containing only the fields it actually uses:

- `SlackWatcher` uses `WorkspaceID` from config — nothing else (BotToken and PollInterval are used elsewhere)
- `EmailWatcher` uses `Username` from config — nothing else

This is the minimal surface. The watcher doesn't need to know about `Enabled`, `BotToken`, `IMAPHost`, etc. — those are consumed by the API client layer or the orchestrator.

### Remove `internal/config` Import

After this change, `internal/service/watcher/` no longer imports `internal/config`. The conversion from `repository.SlackAccount` → `watcher.SlackWatcherConfig` happens at the call site (`main.go`).

## API

### Before

```go
// slack.go
func NewSlackWatcher(api SlackAPI, cfg config.SlackConfig) (*SlackWatcher, error)

// email.go
func NewEmailWatcher(api EmailAPI, cfg config.EmailConfig) (*EmailWatcher, error)
```

### After

```go
// slack.go
type SlackWatcherConfig struct {
    WorkspaceID string
}
func NewSlackWatcher(api SlackAPI, cfg SlackWatcherConfig) (*SlackWatcher, error)

// email.go
type EmailWatcherConfig struct {
    Username string
}
func NewEmailWatcher(api EmailAPI, cfg EmailWatcherConfig) (*EmailWatcher, error)
```

### Validation

Constructor validation remains identical:
- `NewSlackWatcher`: rejects nil api, empty `WorkspaceID`
- `NewEmailWatcher`: rejects nil api, empty `Username`

## Error Handling

No changes to error handling. Same constructor validation errors.

## Integration Points

- **Feature 035** (TOML Slimming): Depends on this — once watchers don't import `config.SlackConfig`/`config.EmailConfig`, those types can be removed from the config package
- **Feature 038** (Main Wiring): Constructs watchers using new config types, converting from DB domain types

## Test Coverage

All existing watcher tests updated to use new config types. No new test cases needed — this is a signature change, not a behavior change.

- `slack_test.go`: Replace `config.SlackConfig{WorkspaceID: "W123"}` with `SlackWatcherConfig{WorkspaceID: "W123"}`
- `email_test.go`: Replace `config.EmailConfig{Username: "user@example.com"}` with `EmailWatcherConfig{Username: "user@example.com"}`
- All existing assertions remain identical

## Files

| File | Action |
|---|---|
| `internal/service/watcher/slack.go` | Modify — add `SlackWatcherConfig`, update constructor signature, remove `config` import |
| `internal/service/watcher/slack_test.go` | Modify — update config type in test setup |
| `internal/service/watcher/email.go` | Modify — add `EmailWatcherConfig`, update constructor signature, remove `config` import |
| `internal/service/watcher/email_test.go` | Modify — update config type in test setup |

## TDD Agent Stats

| TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | Test Designer | 45s | 22,310 | 91b7c78 |
| GREEN | Implementer | 38s | 24,875 | 3c75e1a |
| REFACTOR | Refactorer | 25s | 18,420 | 4026572 |
