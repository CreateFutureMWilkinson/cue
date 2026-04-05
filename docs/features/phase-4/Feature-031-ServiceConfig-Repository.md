# Feature 031: ServiceConfig Repository Interface

**Phase:** Phase-4-Feature-031
**Status:** Planned
**Package:** `internal/repository/`

---

## Overview

Define domain types and a repository interface for storing external service account configuration (Slack workspaces, Email accounts) in the database. This replaces TOML-based service config with a database-backed, multi-account-capable data layer.

## Design Decisions

### Typed Domain Types Over Generic Key-Value

Each service type gets its own struct (`SlackAccount`, `EmailAccount`) rather than a generic `map[string]string` or `ServiceConfig{Key, Value}`. This preserves Go type safety, enables column-level DB constraints, and avoids marshal/unmarshal overhead.

### Multi-Account From the Start

Each struct represents one account. The repository returns slices, supporting multiple Slack workspaces and multiple email accounts. This avoids a future schema migration.

### Interface in `internal/repository/`

Following the existing pattern (`MessageRepository` in `internal/repository/message.go`), the interface lives at the repository package level while implementations live in `internal/repository/implementation/sqlite/`.

## API

### Domain Types

```go
type SlackAccount struct {
    ID                  uuid.UUID
    Enabled             bool
    BotToken            string
    WorkspaceID         string
    PollIntervalSeconds int
    CreatedAt           time.Time
    UpdatedAt           time.Time
}

type EmailAccount struct {
    ID                  uuid.UUID
    Enabled             bool
    IMAPHost            string
    IMAPPort            int
    Username            string
    PasswordEnv         string
    PollIntervalSeconds int
    CreatedAt           time.Time
    UpdatedAt           time.Time
}
```

### Interface

```go
type ServiceConfigRepository interface {
    // Slack accounts
    ListSlackAccounts(ctx context.Context) ([]*SlackAccount, error)
    GetSlackAccount(ctx context.Context, id uuid.UUID) (*SlackAccount, error)
    UpsertSlackAccount(ctx context.Context, acct *SlackAccount) error
    DeleteSlackAccount(ctx context.Context, id uuid.UUID) error

    // Email accounts
    ListEmailAccounts(ctx context.Context) ([]*EmailAccount, error)
    GetEmailAccount(ctx context.Context, id uuid.UUID) (*EmailAccount, error)
    UpsertEmailAccount(ctx context.Context, acct *EmailAccount) error
    DeleteEmailAccount(ctx context.Context, id uuid.UUID) error
}
```

## Error Handling

This feature defines the interface only. Error semantics:
- `Get*` with unknown ID returns a sentinel error (e.g., `ErrNotFound`)
- `Delete*` with unknown ID is a no-op (idempotent)
- `Upsert*` inserts or updates based on primary key

## Integration Points

- **Feature 032** (SQLite ServiceConfig): Implements this interface
- **Feature 036** (Settings Presenter): Consumes this interface for account CRUD
- **Feature 038** (Main Wiring): Queries accounts at startup to build watchers

## Test Coverage

- Compilation test: mock struct satisfies `ServiceConfigRepository` interface
- Type field verification tests

## Files

| File | Action |
|---|---|
| `internal/repository/service_config.go` | **New** — domain types + interface |
