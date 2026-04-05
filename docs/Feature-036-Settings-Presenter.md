# Feature 036: Settings Presenter Expansion

**Phase:** Phase-4-Feature-036
**Status:** Planned
**Package:** `internal/ui/presenter/`
**Depends on:** Features 031, 032, 034

---

## Overview

Expand the settings presenter to manage service account CRUD (Slack and Email) and watcher lifecycle. Currently the `SettingsPresenter` only handles audio volume. After this feature, it orchestrates adding, editing, deleting, and toggling service accounts through the `ServiceConfigRepository` (Feature 031/032) and manages watcher start/stop through the `WatcherManager` interface (Feature 034).

## Design Decisions

### Split or Extend Presenter

Two options considered:
1. **Extend `SettingsPresenter`** with new methods for account management
2. **Create separate `ServiceSettingsPresenter`**

Decision: **Create a new `ServiceSettingsPresenter`**. The audio settings presenter remains focused on audio concerns. The new presenter handles service account CRUD and watcher lifecycle. This keeps each presenter focused and testable with clear mock boundaries.

### Watcher Factory Function

The presenter needs to create watchers when accounts are added/enabled, but it shouldn't import the watcher package directly (that would create a UI → service dependency). Solution: accept a factory function:

```go
type WatcherFactory func(accountType string, accountID uuid.UUID) error
```

The factory is implemented in `main.go` where it has access to both the watcher constructors and the orchestrator. The presenter calls the factory; the factory builds the watcher and calls `AddWatcher`.

Alternative: The presenter calls `WatcherManager.AddWatcher` directly, receiving the constructed watcher. But the presenter doesn't know how to construct watchers (it would need API clients, etc.). The factory pattern keeps construction knowledge in `main.go`.

### Account Lifecycle

| User Action | Presenter Behavior |
|---|---|
| Add account | Validate fields → `repo.UpsertSlackAccount()` → `factory("slack", id)` |
| Edit account | Validate fields → `repo.UpsertSlackAccount()` → `manager.RemoveWatcher(name)` → `factory("slack", id)` |
| Delete account | `manager.RemoveWatcher(name)` → `repo.DeleteSlackAccount(id)` |
| Toggle enabled | `repo.UpsertSlackAccount()` → if enabled: `factory()`, if disabled: `manager.RemoveWatcher()` |

Watcher names follow the convention: `"slack:<workspace_id>"` or `"email:<username>"`.

## API

### Constructor

```go
func NewServiceSettingsPresenter(
    repo repository.ServiceConfigRepository,
    manager orchestrator.WatcherManager,
    factory WatcherFactory,
) *ServiceSettingsPresenter
```

### Methods

```go
// Slack account management
func (p *ServiceSettingsPresenter) ListSlackAccounts(ctx context.Context) ([]*repository.SlackAccount, error)
func (p *ServiceSettingsPresenter) SaveSlackAccount(ctx context.Context, acct *repository.SlackAccount) error
func (p *ServiceSettingsPresenter) DeleteSlackAccount(ctx context.Context, id uuid.UUID) error
func (p *ServiceSettingsPresenter) ToggleSlackAccount(ctx context.Context, id uuid.UUID, enabled bool) error

// Email account management
func (p *ServiceSettingsPresenter) ListEmailAccounts(ctx context.Context) ([]*repository.EmailAccount, error)
func (p *ServiceSettingsPresenter) SaveEmailAccount(ctx context.Context, acct *repository.EmailAccount) error
func (p *ServiceSettingsPresenter) DeleteEmailAccount(ctx context.Context, id uuid.UUID) error
func (p *ServiceSettingsPresenter) ToggleEmailAccount(ctx context.Context, id uuid.UUID, enabled bool) error
```

### Validation

The presenter validates account fields before persisting:

**Slack:**
- `BotToken` must not be empty
- `WorkspaceID` must not be empty
- `PollIntervalSeconds` must be > 0

**Email:**
- `IMAPHost` must not be empty
- `IMAPPort` must be > 0
- `Username` must not be empty
- `PasswordEnv` must not be empty
- `PollIntervalSeconds` must be > 0

## Error Handling

| Scenario | Behavior |
|---|---|
| Validation failure | Return descriptive error, do not persist |
| Repository error on save | Return wrapped error, do not start watcher |
| Watcher factory error | Account is saved but watcher not running; return error to UI for display |
| Repository error on delete | Return wrapped error, watcher may still be running (caller retries) |

## Integration Points

- **Feature 031/032** (Repository): Consumes `ServiceConfigRepository` for persistence
- **Feature 034** (Orchestrator): Consumes `WatcherManager` for watcher lifecycle
- **Feature 037** (Settings UI): UI calls presenter methods in response to user actions
- **Feature 038** (Main Wiring): Instantiates presenter with real dependencies

## Test Coverage

Full mock-based test suite (mock repository, mock WatcherManager, mock factory):

- List accounts (empty, populated)
- Save new Slack account — repo called, factory called
- Save new Email account — repo called, factory called
- Edit existing account — repo updated, old watcher removed, new watcher started
- Delete Slack account — watcher removed, repo delete called
- Delete Email account — watcher removed, repo delete called
- Toggle enable — watcher started via factory
- Toggle disable — watcher removed
- Validation: empty BotToken rejected
- Validation: empty WorkspaceID rejected
- Validation: empty Username rejected
- Validation: invalid poll interval rejected
- Repository error propagation
- Factory error: account saved, error returned to caller

## Files

| File | Action |
|---|---|
| `internal/ui/presenter/service_settings_presenter.go` | **New** — service account CRUD + watcher lifecycle |
| `internal/ui/presenter/service_settings_presenter_test.go` | **New** — full mock-based test suite |
