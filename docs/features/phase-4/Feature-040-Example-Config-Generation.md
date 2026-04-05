# Feature 040: Example Config Generation

**Phase:** Phase-4-Feature-040
**Status:** Done
**Package:** `internal/config/`, `cmd/cue/`
**Depends on:** Feature 035 (TOML Config Slimming — so the generated example reflects the final config shape)

---

## Overview

Add a CLI command (`cue config example`) that writes an annotated example `config.toml` to stdout or a specified path. The example contains all default values with TOML comments explaining each field, including the default Ollama models required. This gives users a documented starting point without having to read source code.

## Design Decisions

### CLI Subcommand, Not Auto-Generation

The current `config.Load()` auto-creates a bare default config if the file is missing. That behavior stays for first-run simplicity. The new `cue config example` command produces a **richly commented** version that serves as documentation. These are complementary:

- Auto-created config: minimal, gets the app running
- Example config: annotated reference for customization

### Handwritten Template, Not Struct Reflection

The annotated example uses a handwritten TOML template string rather than encoding the struct with `toml.Encode`. Reasons:
- `toml.Encode` doesn't support comments
- Comments are the primary value of the example file
- Field ordering can be intentional (most important first)

The template references the same default values from `defaultConfig()` to stay in sync, but the comments and structure are manually authored.

### Output Behavior

```
cue config example           # prints to stdout
cue config example -o path   # writes to file (fails if file exists, use --force to overwrite)
```

Stdout is the default so it can be piped: `cue config example > ~/.cue/config.toml`

### Existing CLI Framework

The project uses `github.com/urfave/cli/v3` (per CLAUDE.md). The `config` command group with an `example` subcommand fits naturally.

## Example Output

```toml
# Cue Configuration
# ==================
# This file controls local application settings.
# Service accounts (Slack, Email) are configured in the Settings UI
# and stored in the application database.

# ─── Database ───────────────────────────────────────────────
[database]
# Path to the SQLite database file.
# Supports ~ for home directory.
path = "~/.cue/messages.db"

# ─── Ollama (Local LLM) ────────────────────────────────────
[ollama]
# Ollama server connection details.
# Models must be pulled locally before use:
#   ollama pull neural-chat
#   ollama pull nomic-embed-text
host = "localhost"
port = 11434

# Model used for message importance scoring.
inference_model = "neural-chat"

# Model used for generating vector embeddings.
embedding_model = "nomic-embed-text"

# Timeout in seconds for Ollama API calls.
# Messages exceeding this are scored IS=7, CS=0.0 (BUFFERED).
timeout_seconds = 10

# ─── Routing Thresholds ────────────────────────────────────
[orchestrator]
# How often (seconds) to poll all configured watchers.
poll_interval_seconds = 600

[orchestrator.router]
# Messages scoring >= importance AND >= confidence → NOTIFIED.
# Messages scoring >= importance AND < confidence → BUFFERED.
# All others → IGNORED.
importance_threshold = 7
confidence_threshold = 0.8

# Maximum messages retained per source (FIFO eviction).
buffer_size_per_source = 100

# ─── Notifications & Audio ─────────────────────────────────
[notification]
# Master toggle for audio alerts.
audio_enabled = true

# Process messages in batches (recommended).
batch_process = true

# Directory containing custom notification sounds.
# Leave empty to use system fallback beep.
audio_dir = ""

# Minimum seconds between consecutive audio alerts.
audio_cooldown_seconds = 2

# Volume level (0-100).
audio_volume = 100

# Fallback beep frequency in Hz (used when no audio files available).
fallback_frequency = 1000

# Fallback beep duration in milliseconds.
fallback_duration_ms = 200

# ─── GUI ────────────────────────────────────────────────────
[gui]
# Initial window dimensions.
window_width = 1200
window_height = 800

# Character displayed in the center area.
# Options: "none", "fairy"
character = "none"

# ─── Logging ────────────────────────────────────────────────
[logging]
# Log level: "debug", "info", "warn", "error"
log_level = "info"

# Directory for log files. Empty = stderr only.
log_dir = ""

# ─── Day Planner ───────────────────────────────────────────
[planner]
# Working hours (HH:MM format).
workday_start = "09:00"
workday_end = "17:00"

# Stop generating new pomodoro blocks after this time.
planning_cutoff = "16:00"

# Pomodoro cycle durations (minutes).
pomodoro_minutes = 25
short_break_minutes = 5
long_break_minutes = 20
long_break_after_cycles = 4

# Merge meetings closer than this gap (minutes).
meeting_merge_gap_minutes = 5

# Lunch window — planner avoids scheduling here.
lunch_window_start = "12:00"
lunch_window_end = "14:00"

# Timer completion sound file path. Empty = system beep.
timer_sound = ""

# Timer alert volume (0-100).
timer_volume = 75
```

## API

### New CLI Command

```go
// cmd/cue/main.go (or cmd/cue/commands.go)
&cli.Command{
    Name:  "config",
    Usage: "Configuration management",
    Commands: []*cli.Command{
        {
            Name:  "example",
            Usage: "Print an annotated example config.toml",
            Flags: []cli.Flag{
                &cli.StringFlag{
                    Name:    "output",
                    Aliases: []string{"o"},
                    Usage:   "Write to file instead of stdout",
                },
                &cli.BoolFlag{
                    Name:  "force",
                    Usage: "Overwrite existing file",
                },
            },
            Action: configExampleAction,
        },
    },
}
```

### Config Package Function

```go
// internal/config/example.go

// ExampleTOML returns an annotated example configuration as a string.
// All values match defaultConfig() defaults.
func ExampleTOML() string
```

This function lives in the config package so it can be tested independently of the CLI.

## Error Handling

| Scenario | Behavior |
|---|---|
| Output to stdout | Print and exit 0 |
| Output to file, file doesn't exist | Write and exit 0 |
| Output to file, file exists, no --force | Error: "file already exists, use --force to overwrite" |
| Output to file, file exists, --force | Overwrite and exit 0 |
| Write permission denied | Error with OS message |

## Integration Points

- **Feature 035** (TOML Slimming): The example must reflect the slimmed config (no `[slack]`/`[email]` sections)
- **Feature 039** (Ollama Validation): Example comments include `ollama pull` instructions for default models
- **`cmd/cue/main.go`**: CLI command registration

## Test Coverage

- `ExampleTOML()` returns valid TOML (parse with `BurntSushi/toml`)
- Parsed example matches `defaultConfig()` field values
- Example contains expected comment strings (spot-check key comments)
- Example does NOT contain `[slack]` or `[email]` sections
- Example contains Ollama model pull instructions in comments
- CLI: `--output` writes to file
- CLI: `--output` with existing file fails without `--force`
- CLI: `--output --force` overwrites existing file

## Files

| File | Action |
|---|---|
| `internal/config/example.go` | **New** — ExampleTOML() function with annotated template |
| `internal/config/example_test.go` | **New** — validity and content tests |
| `cmd/cue/main.go` | Modify — register `config example` CLI command |

## TDD Agent Stats

| Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | Test Designer | ~47s | ~27,100 | ea6de7f |
| GREEN | Implementer | ~42s | ~30,700 | d741622 |
| REFACTOR | Refactorer | ~35s | ~24,500 | (merged into GREEN) |
