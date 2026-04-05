# Feature 048: Unused Config Field Wiring

**Phase:** Phase-5-Feature-048
**Status:** Planned
**Packages:** `internal/config/`, `internal/repository/implementation/sqlite/`, `internal/service/orchestrator/`, `cmd/cue/`
**Depends on:** —

---

## Overview

Wire four config fields that are loaded and validated but never used at runtime: `BufferSizePerSource`, `BatchProcess`, `Slack.Enabled`, and `Email.Enabled`. Each field has a clear intended purpose but was never connected to the code that should consume it.

## Inventory

### 1. `RouterConfig.BufferSizePerSource` (config.go:74)

**Current state:** Loaded from `[orchestrator.router] buffer_size_per_source`, validated, but ignored. The SQLite repository hardcodes `const maxMessagesPerSource = 100` (message_impl.go:42) and uses it for FIFO eviction.

**Fix:** Pass `BufferSizePerSource` from config through to the SQLite repository constructor, replacing the hardcoded constant.

```go
// Before:
const maxMessagesPerSource = 100

// After:
type SQLiteMessageRepository struct {
    db                  *sql.DB
    maxMessagesPerSource int
}

func NewSQLiteMessageRepository(db *sql.DB, maxMessagesPerSource int) (*SQLiteMessageRepository, error)
```

`main.go` passes `cfg.Orchestrator.Router.BufferSizePerSource` to the constructor.

### 2. `NotificationConfig.BatchProcess` (config.go:87)

**Current state:** Loaded from `[notification] batch_process`, set to `true` in defaults, never checked. The system always processes messages in batches regardless.

**Fix:** This field has no meaningful behavioral toggle — the system is inherently batch-based (10-minute poll cycles). **Remove the field** from config, defaults, and validation. It's dead config that misleads users into thinking they can switch to real-time processing.

### 3. `SlackConfig.Enabled` (config.go:52)

**Current state:** Loaded from `[slack] enabled`, never checked. Slack watchers are always created.

**Fix:** Check `cfg.Slack.Enabled` before creating the Slack watcher in `main.go`. When `false`, skip watcher creation and log at info level. In the Phase 4 world (Feature 038 — dynamic accounts from DB), the `Enabled` flag lives on each `ServiceConfig` row and is already checked by the settings presenter. For Phase 5, ensure the TOML-based `Enabled` flag is respected in the pre-038 startup path, and that per-account `Enabled` is respected in the post-038 path.

### 4. `EmailConfig.Enabled` (config.go:59)

**Current state:** Same as Slack — loaded, never checked.

**Fix:** Same pattern as Slack. Check `cfg.Email.Enabled` before creating the Email watcher.

## Design Decisions

### BufferSizePerSource as Constructor Param

Making it a constructor parameter rather than a global/config lookup keeps the repository testable and explicit. Tests can pass any value without touching config.

### Remove BatchProcess, Don't Wire It

Wiring a no-op config field creates user confusion. The batch processing model is architectural, not configurable. Removing the field is cleaner than adding a check that only validates `true`.

### Enabled Flags — Simple Guard

```go
if cfg.Slack.Enabled {
    slackWatcher, err := watcher.NewSlackWatcher(...)
    watchers["slack"] = slackWatcher
}
```

When disabled, the watcher map simply omits that source. The orchestrator handles zero or partial watchers gracefully (tested in Feature 034).

## Error Handling

| Scenario | Behavior |
|---|---|
| `BufferSizePerSource` ≤ 0 in config | Validation rejects at startup |
| Both Slack and Email disabled | App starts with zero watchers, logs info |
| User has `batch_process` in old config | TOML parser ignores unknown keys — no error |

## Integration Points

- **Feature 002** (SQLite Repository): Constructor gains `maxMessagesPerSource` parameter
- **Feature 007** (Orchestrator): Handles zero watchers (already tested)
- **Feature 038** (Main Wiring): Per-account `Enabled` field already modeled in `ServiceConfig`
- **`cmd/cue/main.go`**: Passes buffer size to repo, checks enabled flags

## Test Coverage

### BufferSizePerSource
- Repository with `maxMessagesPerSource=5` evicts at 6th message per source
- Repository with `maxMessagesPerSource=1` keeps only latest per source
- Constructor rejects ≤ 0

### Enabled Flags
- Slack disabled → no Slack watcher in map
- Email disabled → no Email watcher in map
- Both disabled → empty watcher map, orchestrator starts without error

### BatchProcess Removal
- Config without `batch_process` loads successfully
- Config with `batch_process` loads successfully (TOML ignores unknown)
- Default config generation (Feature 040) omits `batch_process`

## Files

| File | Action |
|---|---|
| `internal/repository/implementation/sqlite/message_impl.go` | **Modify** — replace `maxMessagesPerSource` constant with constructor parameter |
| `internal/repository/implementation/sqlite/message_impl_test.go` | **Modify** — update constructor calls, add eviction threshold tests |
| `internal/config/config.go` | **Modify** — remove `BatchProcess` field, remove from defaults and validation |
| `internal/config/config_test.go` | **Modify** — remove `BatchProcess` tests |
| `cmd/cue/main.go` | **Modify** — pass `BufferSizePerSource` to repo, add enabled-flag guards |
