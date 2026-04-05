# Feature 031 Hotfix A: Encrypted Credential Storage

**Phase:** Phase-4-Feature-031-Hotfix-A (security)
**Status:** In Progress
**Packages:** `internal/secret/`, `internal/repository/`, `internal/repository/implementation/sqlite/`, `internal/service/watcher/`, `internal/ui/presenter/`, `cmd/cue/`

---

## Overview

Service account credentials (Slack tokens, email passwords) are stored insecurely: Slack tokens as plaintext TEXT in SQLite, email passwords only as environment variable names (never the actual password). This hotfix introduces AES-256-GCM encryption at rest for all service account credentials and adds a new `CalendarAccount` model with encrypted ICS URL storage.

## Findings

### Plaintext Slack Token Storage

**Location:** `internal/repository/implementation/sqlite/service_config_impl.go:19`
**Issue:** `slack_accounts.token` column stores API tokens as plaintext TEXT. Any process with read access to `~/.cue/messages.db` can extract tokens.

### Email Password Not Stored

**Location:** `internal/repository/service_config.go:28`
**Issue:** `EmailAccount.PasswordEnv` stores only the environment variable name, not the actual password. The real password is read from the environment at runtime (`email_api.go:34`), meaning credentials cannot be managed through the Settings UI and require manual environment setup.

### No Calendar Account Persistence

**Issue:** Calendar ICS URLs exist only in memory via `ICSProvider` (`internal/service/calendar/calendar.go:39`). No database model exists for calendar account configuration.

### No Encryption Layer

**Issue:** No encryption utilities exist in the codebase. All data written to SQLite is plaintext.

## Design

### Encryption: AES-256-GCM with Key File

New package `internal/secret/` provides an `Encryptor` interface and a `KeyFileEncryptor` implementation:

- **Key:** 32 random bytes auto-generated at `~/.cue/secret.key` (mode 0600) on first use
- **Algorithm:** AES-256-GCM (authenticated encryption)
- **Format:** `nonce(12 bytes) || ciphertext` — random nonce per encryption call
- **Dependencies:** Go stdlib only (`crypto/aes`, `crypto/cipher`, `crypto/rand`)

Key file loss makes stored credentials unrecoverable. This is acceptable for a local-first tool — users re-enter credentials via Settings UI.

### Transparent Encryption in Repository Layer

Encryption sits in `SQLiteServiceConfigRepository`, which gains an `Encryptor` dependency. Domain objects (`SlackAccount`, `EmailAccount`, `CalendarAccount`) carry plaintext credentials. Encrypt-on-write and decrypt-on-read are invisible to callers.

### Model Changes

| Model | Before | After |
|---|---|---|
| `SlackAccount.Token` | Plaintext `string` | Plaintext `string` (encrypted at storage layer) |
| `EmailAccount.PasswordEnv` | Env var name `string` | Renamed to `Password` — actual password `string` (encrypted at storage layer) |
| `CalendarAccount` | Does not exist | New model: `ID`, `Enabled`, `Name`, `ICSURL`, `PollIntervalSeconds`, timestamps |

### Schema Changes (No Migration)

Old plaintext columns are replaced. Existing data is not migrated — users re-enter credentials.

```sql
-- slack_accounts: token TEXT → token_encrypted BLOB
-- email_accounts: password_env TEXT → password_encrypted BLOB
-- calendar_accounts: NEW table with ics_url_encrypted BLOB
```

### Consumer Changes

- `NewIMAPClient` accepts password directly instead of env var name; `os.Getenv` call removed
- `cmd/cue/main.go` creates `KeyFileEncryptor` at startup, passes to service config repository
- Presenter validation updated for `Password` field

## Error Handling

| Error | Action |
|---|---|
| Key file directory not writable | Constructor returns error; app fails to start with clear message |
| Key file corrupted (not 32 bytes) | Constructor returns error |
| Decryption failure (tampered data) | Return wrapped error; account unusable until re-saved |
| Key file lost | Stored credentials unrecoverable; user re-enters via Settings UI |

## API

### Encryptor Interface

```go
type Encryptor interface {
    Encrypt(plaintext []byte) ([]byte, error)
    Decrypt(ciphertext []byte) ([]byte, error)
}
```

### CalendarAccount Repository Methods

```go
ListCalendarAccounts(ctx context.Context) ([]*CalendarAccount, error)
GetCalendarAccount(ctx context.Context, id uuid.UUID) (*CalendarAccount, error)
UpsertCalendarAccount(ctx context.Context, acct *CalendarAccount) error
DeleteCalendarAccount(ctx context.Context, id uuid.UUID) error
```

### Constructor Change

```go
// Before
NewSQLiteServiceConfigRepository(db *sql.DB) (*SQLiteServiceConfigRepository, error)

// After
NewSQLiteServiceConfigRepository(db *sql.DB, enc secret.Encryptor) (*SQLiteServiceConfigRepository, error)
```

## Integration Points

- **Settings UI** (`service_settings_presenter.go`) — already handles Slack/Email CRUD; needs `Password` field rename and calendar account support (future UI work)
- **Watcher factory** (`cmd/cue/main.go`) — reads decrypted credentials from repository to construct API clients
- **Calendar service** (`internal/service/calendar/`) — future: read `CalendarAccount.ICSURL` from repository instead of constructor parameter

## Breaking Changes

- `EmailAccount.PasswordEnv` renamed to `EmailAccount.Password`
- `NewSQLiteServiceConfigRepository` requires `secret.Encryptor` parameter
- `NewIMAPClient` accepts password directly, not env var name
- Existing Slack tokens and email env var names in database are not migrated — must be re-entered

## Test Coverage

| Package | Tests |
|---|---|
| `internal/secret/` | Round-trip, tampered ciphertext, too-short input, key file creation/reuse, invalid key length, nonce uniqueness |
| `sqlite/service_config_impl` | Encrypted CRUD for all three account types; raw-query verification that BLOBs ≠ plaintext |
| `watcher/email_api` | Direct password acceptance; empty password rejection |
| `presenter/service_settings` | `Password` field validation |

## Files Modified/Created

| Action | Path |
|---|---|
| Create | `internal/secret/secret.go` |
| Create | `internal/secret/secret_test.go` |
| Modify | `internal/repository/service_config.go` |
| Modify | `internal/repository/service_config_test.go` |
| Modify | `internal/repository/implementation/sqlite/service_config_impl.go` |
| Modify | `internal/repository/implementation/sqlite/service_config_impl_test.go` |
| Modify | `internal/service/watcher/email_api.go` |
| Modify | `internal/service/watcher/email_api_test.go` |
| Modify | `internal/ui/presenter/service_settings_presenter.go` |
| Modify | `internal/ui/presenter/service_settings_presenter_test.go` |
| Modify | `cmd/cue/main.go` |
