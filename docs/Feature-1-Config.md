# Feature 1: Config Loading + Validation

**Phase:** Phase-1-Feature-1
**Status:** Done
**Package:** `internal/config/`

---

## Overview

TOML-based configuration loading with safe defaults, auto-creation on first run, tilde expansion for paths, and table-driven validation. The config file at `~/.cue/config.toml` is the single source of truth for all runtime settings — no hardcoded values or CLI flags for feature toggles.

## Design Decisions

### TOML Over Other Formats

TOML was chosen for human readability and native support for nested sections (database, slack, email, ollama, etc.). The `github.com/BurntSushi/toml` library provides strict type checking at parse time — type mismatches (e.g., string where int expected) are caught during `Load()`.

### Auto-Creation with Defaults

When the config file doesn't exist, `Load()` creates parent directories (mode 0750) and writes a default config (mode 0600) rather than erroring. This ensures first-run experience requires zero manual setup. Sensitive fields (BotToken, WorkspaceID, Username) are left empty in defaults.

### Tilde Expansion

`Database.Path` and `Logging.LogDir` support `~/` prefix expansion to the user's home directory. If `os.UserHomeDir()` fails (e.g., containerized environments), paths remain unexpanded — silent degradation rather than error.

### Validation Separate from Loading

`Load()` does not call `Validate()` — the caller decides when to validate. This allows loading a partial config for inspection before enforcing rules.

## API

### Functions

```go
// Load reads or creates config from the given path.
func Load(path string) (*Config, error)

// Validate checks all required fields and logical bounds.
func (c *Config) Validate() error
```

### Config Structure

```go
type Config struct {
    Database     DatabaseConfig     // path to SQLite database
    Slack        SlackConfig        // bot token, workspace, poll interval
    Email        EmailConfig        // IMAP settings, poll interval
    Orchestrator OrchestratorConfig // router thresholds, buffer size
    Ollama       OllamaConfig       // host, port, models, timeout
    Notification NotificationConfig // audio toggle, batch flag
    GUI          GUIConfig          // host, port
    Logging      LoggingConfig      // level, directory
}
```

## Validation Rules

| Field | Rule | Error Message |
|---|---|---|
| `database.path` | Not empty | "database.path must not be empty" |
| `ollama.host` | Not empty | "ollama.host must not be empty" |
| `ollama.port` | > 0 | "ollama.port must be greater than 0" |
| `ollama.inference_model` | Not empty | "ollama.inference_model must not be empty" |
| `ollama.embedding_model` | Not empty | "ollama.embedding_model must not be empty" |
| `slack.poll_interval_seconds` | >= 0 | "slack.poll_interval_seconds must not be negative" |
| `email.poll_interval_seconds` | >= 0 | "email.poll_interval_seconds must not be negative" |
| `gui.port` | > 0 | "gui.port must be greater than 0" |

## Defaults

Key defaults matching CLAUDE.md Section 6:

- Database: `~/.cue/messages.db`
- Slack/Email poll interval: 600s (10 minutes)
- Ollama: `localhost:11434`, models `neural-chat` / `nomic-embed-text`, 10s timeout
- Router: importance threshold 7, confidence threshold 0.8, buffer 100 per source
- GUI: `localhost:8080`

## Error Handling

| Scenario | Behavior |
|---|---|
| Config file missing | Create with defaults, return default config |
| Directory creation fails | Wrapped error returned |
| TOML type mismatch | Parse error from toml.DecodeFile |
| Tilde expansion fails | Silent no-op (paths unchanged) |
| Validation fails | First failing rule's error returned |

## Test Coverage

6 test cases in `config_test.go` using testify suites:

- Full round-trip parse of all fields
- Auto-creation with default values
- Validation of 8 required field rules (table-driven subtests)
- TOML type mismatch rejection (4 subtests)
- Tilde expansion for home directory paths
- Absolute path preservation (no-op expansion)

## TDD Agent Stats

| Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | orchestrator | — | 4,593 | cd5c731 |
| GREEN | orchestrator | — | 3,587 | ff7c5e0 |
| REFACTOR | orchestrator | — | 2,684 | fee15bc |
