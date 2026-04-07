# Feature 067A: Email Encryption Setting

**Phase:** Phase-6-Feature-067A
**Type:** Bugfix (Hotfix)
**Severity:** High
**Status:** Done
**Packages:** `internal/ui/`, `internal/repository/`, `internal/repository/implementation/sqlite/`, `internal/service/watcher/`
**Related:** Feature 067 (Email Add Account), Feature 053 (Email Mention Detection)

---

## Problem

The email account settings form collects IMAP Host, Port, Username, Password, and Poll Interval — but has no way to configure connection encryption. The underlying `IMAPClient` (`email_api.go:52-53`) connects via **plain TCP** using `net.Dialer`, transmitting credentials in cleartext. This is a security issue for any real IMAP server and makes the email integration unusable for standard providers (Gmail, Outlook, Fastmail, etc.) which require SSL/TLS or STARTTLS.

## Root Cause

1. **Data model gap:** `EmailAccount` (`repository/service_config.go:21-32`) has no encryption field.
2. **Schema gap:** `email_accounts` table (`service_config_impl.go:28-40`) has no encryption column.
3. **UI gap:** `createEmailAccountForm()` (`settings_view.go:37-104`) has no encryption selector.
4. **Client gap:** `IMAPClient` (`email_api.go:22-41`) accepts no encryption parameter and always uses plain TCP.

## Proposed Fix

### 1. Data Model — Add `Encryption` field to `EmailAccount`

```go
type EmailAccount struct {
    ID                  uuid.UUID
    Enabled             bool
    IMAPHost            string
    IMAPPort            int
    Username            string
    Password            string
    Encryption          string  // "none", "ssl_tls", "starttls"
    PollIntervalSeconds int
    CreatedAt           time.Time
    UpdatedAt           time.Time
}
```

Valid values:
| Value | Meaning | Typical Port |
|---|---|---|
| `"ssl_tls"` | Implicit TLS — TLS handshake on connect (recommended) | 993 |
| `"starttls"` | Plain connect, then upgrade via STARTTLS command | 143 |
| `"none"` | No encryption (development/testing only) | 143 |

Default: `"ssl_tls"` (safest for new accounts).

### 2. Database Schema — Add `encryption` column

```sql
ALTER TABLE email_accounts ADD COLUMN encryption TEXT NOT NULL DEFAULT 'ssl_tls';
```

Run as a migration in `NewSQLiteServiceConfigRepository`. Existing rows get `ssl_tls` as default (safe assumption since most configured IMAP servers use port 993).

Update column constants:
```go
emailAccountColumns = "id, enabled, imap_host, imap_port, username, password_encrypted, encryption, poll_interval_seconds, created_at, updated_at"
```

### 3. UI Form — Add encryption dropdown

Add a `widget.NewSelect` between Password and Poll Interval fields:

```go
encryptionSelect := widget.NewSelect(
    []string{"SSL/TLS (Recommended)", "STARTTLS", "None"},
    nil,
)
encryptionSelect.SetSelected("SSL/TLS (Recommended)")
```

Map display values to stored values:
| Display | Stored |
|---|---|
| "SSL/TLS (Recommended)" | `"ssl_tls"` |
| "STARTTLS" | `"starttls"` |
| "None" | `"none"` |

### 4. IMAPClient — Add TLS support

Update constructor to accept encryption mode:

```go
func NewIMAPClient(host string, port int, username, password, encryption string) (*IMAPClient, error)
```

Update `FetchNewMessages` connection logic:

```go
switch c.encryption {
case "ssl_tls":
    tlsConf := &tls.Config{ServerName: c.host}
    conn, err = tls.DialWithDialer(&net.Dialer{}, "tcp", addr, tlsConf)
case "starttls":
    conn, err = (&net.Dialer{}).DialContext(ctx, "tcp", addr)
    // after greeting: imapClient.StartTLS(tlsConf)
default: // "none"
    conn, err = (&net.Dialer{}).DialContext(ctx, "tcp", addr)
}
```

The `go-imap/v2` library's `imapclient` supports STARTTLS via the `StartTLS` method.

### 5. Presenter/Watcher Plumbing

The `ServiceSettingsPresenter.SaveEmailAccount()` already passes through the full `EmailAccount` struct — no presenter changes needed beyond ensuring the new field flows through. The email watcher that constructs the `IMAPClient` must read the `Encryption` field from the account config and pass it to `NewIMAPClient`.

## Test Strategy

### Behaviors

1. **Model & schema** — `EmailAccount` has `Encryption` field; DB migration adds column with default.
2. **UI dropdown** — Email form shows encryption selector with 3 options; default is SSL/TLS.
3. **Form save** — Selected encryption value persists to `EmailAccount.Encryption`.
4. **IMAPClient SSL/TLS** — With `encryption="ssl_tls"`, client uses `tls.Dial`.
5. **IMAPClient STARTTLS** — With `encryption="starttls"`, client connects plain then upgrades.
6. **IMAPClient None** — With `encryption="none"`, client uses plain TCP (existing behavior).
7. **Migration** — Existing email accounts without encryption column get `"ssl_tls"` default.

### TDD Micro-Loops

| # | Behavior | Scope |
|---|---|---|
| 1 | EmailAccount model has Encryption field | `repository/` |
| 2 | SQLite schema migration + upsert/scan with encryption | `repository/implementation/sqlite/` |
| 3 | UI form has encryption dropdown, default SSL/TLS | `internal/ui/` |
| 4 | Form save includes encryption value | `internal/ui/` |
| 5 | IMAPClient constructor accepts encryption param | `internal/service/watcher/` |
| 6 | IMAPClient connects with TLS when ssl_tls | `internal/service/watcher/` |
| 7 | IMAPClient issues STARTTLS when starttls | `internal/service/watcher/` |

## Files to Change

| File | Change |
|---|---|
| `internal/repository/service_config.go` | Add `Encryption string` to `EmailAccount` |
| `internal/repository/implementation/sqlite/service_config_impl.go` | Migration, update columns, upsert, scan |
| `internal/ui/settings_view.go` | Add encryption dropdown to email form |
| `internal/service/watcher/email_api.go` | Add `encryption` param, TLS/STARTTLS logic |
| `internal/service/watcher/email_api_test.go` | New tests for each encryption mode |
| `internal/ui/settings_view_test.go` | Verify dropdown exists and default value |
| `internal/ui/settings_interaction_test.go` | Verify encryption value saved |
| `tests/ui/settings_acceptance_test.go` | Update email form field count (5 -> 6 entries + 1 select) |

## Acceptance Criteria

- [x] `EmailAccount` struct has `Encryption` field with valid values `"ssl_tls"`, `"starttls"`, `"none"`
- [x] SQLite migration adds `encryption` column; existing rows default to `"ssl_tls"`
- [x] Email settings form shows encryption dropdown between Password and Poll Interval
- [x] Default selection is "SSL/TLS (Recommended)"
- [x] Saved accounts persist the selected encryption mode
- [x] `IMAPClient` with `ssl_tls` connects via `tls.Dial` (implicit TLS)
- [x] `IMAPClient` with `starttls` connects plain then upgrades via STARTTLS
- [x] `IMAPClient` with `none` connects via plain TCP (existing behavior)
- [x] All existing email tests remain green
- [x] All existing settings tests remain green

## Implementation Notes

All 10 acceptance criteria are satisfied. Key implementation details:

- **STARTTLS approach:** Used `imapclient.NewStartTLS()` instead of manual STARTTLS negotiation, because `StartTLS` is unexported in `go-imap/v2`. The `NewStartTLS` constructor connects via plain TCP and immediately upgrades to TLS before the IMAP greeting.
- **Migration:** SQLite `ALTER TABLE` adds `encryption TEXT NOT NULL DEFAULT 'ssl_tls'` — existing accounts automatically default to the safest option.
- **UI dropdown:** Three options ("SSL/TLS (Recommended)", "STARTTLS", "None") mapped to stored values `ssl_tls`, `starttls`, `none`.

### TDD Agent Stats

| # | Behavior | TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|---|---|
| 1 | Model + schema | RED | Test Designer | ~37s | ~26,000 | 0b87e3e |
| 1 | Model + schema | GREEN | Implementer | ~51s | ~29,000 | 0b87e3e |
| 3 | UI dropdown | RED | Test Designer | ~40s | ~32,000 | 84dcf88 |
| 3 | UI dropdown | GREEN | Implementer | ~28s | ~22,000 | 84dcf88 |
| 4 | Form save | RED | Test Designer | ~67s | ~36,000 | c0d90a6 |
| 4 | Form save | GREEN | orchestrator | manual | — | c0d90a6 |
| 5 | IMAPClient constructor | RED+GREEN | Implementer | ~54s | ~30,000 | f3072d8 |
| 6 | IMAPClient TLS | RED+GREEN | Implementer | ~101s | ~32,000 | 3c4754a |
