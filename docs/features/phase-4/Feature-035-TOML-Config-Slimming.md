# Feature 035: TOML Config Slimming

**Phase:** Phase-4-Feature-035
**Status:** Planned
**Package:** `internal/config/`
**Depends on:** Feature 033 (watchers no longer import config types)

---

## Overview

Remove Slack and Email configuration from the TOML config struct. After Feature 033 decouples watchers from `config.SlackConfig`/`config.EmailConfig`, those types are no longer used by any package. This feature removes them along with their defaults, validation rules, and TOML generation.

The TOML file becomes focused on local machine concerns only: database path, Ollama connection, audio notifications, GUI, logging, and planner.

## Design Decisions

### Clean Removal, No Deprecation

Since there are no existing users, there's no need for backward-compatible deprecated fields or migration stubs. The `Slack` and `Email` fields are removed entirely from the `Config` struct.

### What Stays in config.toml

```toml
[database]
path = "~/.cue/messages.db"

[orchestrator.router]
importance_threshold = 7
confidence_threshold = 0.8
buffer_size_per_source = 100

[ollama]
host = "localhost"
port = 11434
inference_model = "neural-chat"
embedding_model = "nomic-embed-text"
timeout_seconds = 10

[notification]
audio_enabled = true
batch_process = true
audio_dir = ""
audio_cooldown_seconds = 2
audio_volume = 100
fallback_frequency = 1000
fallback_duration_ms = 200

[gui]
window_width = 1200
window_height = 800
character = "none"

[logging]
log_level = "info"
log_dir = ""

[planner]
workday_start = "09:00"
workday_end = "17:00"
# ... all planner fields remain
```

### What Gets Removed

| Removed Struct | Removed Fields |
|---|---|
| `SlackConfig` | `enabled`, `bot_token`, `workspace_id`, `poll_interval_seconds` |
| `EmailConfig` | `enabled`, `imap_host`, `imap_port`, `username`, `password_env`, `poll_interval_seconds` |

| Removed Validation Rules |
|---|
| `slack.poll_interval_seconds must not be negative` |
| `email.poll_interval_seconds must not be negative` |

| Removed Defaults |
|---|
| `Slack.Enabled: true`, `Slack.PollIntervalSeconds: 600` |
| `Email.Enabled: true`, `Email.IMAPHost: "imap.gmail.com"`, `Email.IMAPPort: 993`, etc. |

## API

### Config Struct (After)

```go
type Config struct {
    Database     DatabaseConfig     `toml:"database"`
    Orchestrator OrchestratorConfig `toml:"orchestrator"`
    Ollama       OllamaConfig       `toml:"ollama"`
    Notification NotificationConfig `toml:"notification"`
    GUI          GUIConfig          `toml:"gui"`
    Logging      LoggingConfig      `toml:"logging"`
    Planner      PlannerConfig      `toml:"planner"`
}
```

`SlackConfig` and `EmailConfig` structs are deleted entirely.

### Validate() Changes

Remove these two rules:
```go
{func(cfg *Config) bool { return cfg.Slack.PollIntervalSeconds >= 0 }, "..."},
{func(cfg *Config) bool { return cfg.Email.PollIntervalSeconds >= 0 }, "..."},
```

All other validation rules remain unchanged.

## Error Handling

No new error handling. Existing TOML parsing with `BurntSushi/toml` silently ignores unknown keys, so any old config files with `[slack]`/`[email]` sections will parse without error — those sections are simply ignored.

## Integration Points

- **Feature 033** (Watcher Decoupling): Must complete first — watchers must no longer import `config.SlackConfig`/`config.EmailConfig`
- **Feature 038** (Main Wiring): `main.go` no longer passes `cfg.Slack` or `cfg.Email` to watcher constructors

## Test Coverage

Updates to existing `config_test.go` suite:

- `TestLoadValidConfig`: Remove Slack/Email field assertions, update TOML fixture
- `TestCreateDefaultConfigIfMissing`: Verify generated TOML has no `[slack]`/`[email]` sections
- `TestValidateRequiredFields`: Remove Slack/Email validation test cases
- New: `TestIgnoresUnknownSections`: Load TOML with `[slack]` section, verify no parse error (backward compat for stale files)

## Files

| File | Action |
|---|---|
| `internal/config/config.go` | Modify — remove SlackConfig, EmailConfig structs and fields; update defaultConfig(), Validate(), generateDefaultTOML() |
| `internal/config/config_test.go` | Modify — update test fixtures and assertions |
