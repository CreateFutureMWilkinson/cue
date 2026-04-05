# Feature 038: Main Wiring Update

**Phase:** Phase-4-Feature-038
**Status:** Planned
**Package:** `cmd/cue/`
**Depends on:** All previous Phase 4 features (031-037)

---

## Overview

Update the composition root (`cmd/cue/main.go`) to wire together all Phase 4 changes. The startup flow changes from "load all config from TOML, build watchers, start" to "load local config from TOML, open DB, query service accounts, build watchers dynamically, start".

## Design Decisions

### Startup Order

The key constraint is that the database path comes from TOML (needed before DB opens), while service accounts come from the DB (needed to build watchers). This creates a two-phase config load:

1. **TOML phase**: Load `config.toml` → get database path, Ollama config, audio config, GUI config, router thresholds, planner config
2. **DB phase**: Open SQLite → create `ServiceConfigRepository` → query enabled accounts → build watchers

### Watcher Construction

For each enabled account from the DB, `main.go` constructs the appropriate watcher and registers it with the orchestrator:

```go
// For each enabled Slack account:
slackCfg := watcher.SlackWatcherConfig{WorkspaceID: acct.WorkspaceID}
sw, err := watcher.NewSlackWatcher(slackAPI, slackCfg)
orch.AddWatcher("slack:"+acct.WorkspaceID, sw)

// For each enabled Email account:
emailCfg := watcher.EmailWatcherConfig{Username: acct.Username}
ew, err := watcher.NewEmailWatcher(emailAPI, emailCfg)
orch.AddWatcher("email:"+acct.Username, ew)
```

### Watcher Factory for Settings Presenter

The `ServiceSettingsPresenter` needs a factory function to create watchers when users add accounts at runtime. This factory is defined as a closure in `main.go` that captures the API clients and orchestrator:

```go
factory := func(accountType string, accountID uuid.UUID) error {
    // Query account from repo
    // Build watcher with appropriate API client
    // Call orch.AddWatcher(name, watcher)
}
```

### Orchestrator Construction

The orchestrator is now created with zero watchers. The `PollIntervalSeconds` comes from... where? Currently it comes from `cfg.Slack.PollIntervalSeconds`. After slimming, there are two options:

1. Use a fixed default (600 seconds) in the orchestrator config
2. Keep a `[orchestrator]` section in TOML with `poll_interval_seconds`

Decision: The orchestrator's poll interval is the tick rate for checking all watchers. It should be a TOML config value under `[orchestrator]`:

```toml
[orchestrator]
poll_interval_seconds = 600

[orchestrator.router]
importance_threshold = 7
confidence_threshold = 0.8
buffer_size_per_source = 100
```

This is already partially supported — `OrchestratorConfig` exists with `PollIntervalSeconds`. It just needs to be populated from TOML rather than derived from `cfg.Slack.PollIntervalSeconds`.

## Startup Flow

```
1.  Load TOML config
2.  Validate TOML config
3.  Open SQLite database (cfg.Database.Path)
4.  Create SQLiteMessageRepository(db)
5.  Create SQLiteServiceConfigRepository(db)
6.  Create Router (cfg.Orchestrator.Router thresholds)
7.  Create AlertService (cfg.Notification)
8.  Create Orchestrator (cfg.Orchestrator, router, repo, nil watchers, eventCh, alerter)
9.  Query ServiceConfigRepository for enabled Slack accounts
10. For each: create SlackWatcher → orch.AddWatcher("slack:<workspace_id>", sw)
11. Query ServiceConfigRepository for enabled Email accounts
12. For each: create EmailWatcher → orch.AddWatcher("email:<username>", ew)
13. Create WatcherFactory closure
14. Create ServiceSettingsPresenter(serviceConfigRepo, orch, factory)
15. Create SettingsPresenter (audio volume)
16. Create MainWindow (cfg.GUI, both presenters, ...)
17. Start Orchestrator
18. Run UI event loop
```

## Removed Code

- `cfg.Slack` references (struct no longer exists)
- `cfg.Email` references (struct no longer exists)
- Direct watcher construction from config types
- Static watcher map passed to `NewOrchestrator`

## Error Handling

| Scenario | Behavior |
|---|---|
| DB open fails | Fatal error, exit |
| ServiceConfigRepository creation fails | Fatal error, exit |
| Account query fails at startup | Log error, start with zero watchers (degraded mode) |
| Individual watcher creation fails | Log error, skip that watcher, continue with others |
| All watchers fail to create | Start anyway — user can fix via Settings UI |

The key principle: the app should always start, even if service accounts are misconfigured. The Settings UI provides the recovery path.

## Integration Points

- **Feature 031/032** (Repository): Creates and queries `ServiceConfigRepository`
- **Feature 033** (Watcher Decoupling): Uses new `SlackWatcherConfig`/`EmailWatcherConfig`
- **Feature 034** (Dynamic Watchers): Creates orchestrator with zero watchers, uses `AddWatcher`
- **Feature 035** (TOML Slimming): Uses slimmed `Config` struct (no Slack/Email)
- **Feature 036** (Settings Presenter): Instantiates with real deps
- **Feature 037** (Settings UI): Passes presenters to settings window

## Test Coverage

`main.go` is the composition root — it's verified through integration testing and manual verification rather than unit tests.

Verification checklist:
1. App starts with empty DB (no accounts) — no crash
2. App starts with pre-configured accounts in DB — watchers created and polling
3. Settings UI accessible from menu bar
4. Add account via Settings → watcher starts immediately
5. Delete account via Settings → watcher stops immediately
6. Restart app → accounts persist, watchers recreate from DB
7. `just test` passes across entire project

## Files

| File | Action |
|---|---|
| `cmd/cue/main.go` | Modify — rewire startup flow, remove Slack/Email config references, add ServiceConfigRepository, factory, presenter wiring |
