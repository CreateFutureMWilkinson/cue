# Feature 046: IMAP Email Client

**Phase:** Phase-5-Feature-046
**Status:** Done
**Packages:** `internal/service/watcher/`, `cmd/cue/`
**Depends on:** Feature 038 (Main Wiring — dynamic watcher construction)

---

## Overview

Implement a real IMAP email client that replaces the `placeholderEmailAPI` in `cmd/cue/main.go`. The placeholder returns `nil` for `FetchNewMessages`, making the Email watcher inert. This feature builds a client that connects to an IMAP server, fetches new messages from the inbox, and extracts sender, subject, and body text.

## Motivation

The `EmailWatcher` (Feature 006) defines an `EmailAPI` interface and is fully tested against mock implementations. However, no real implementation exists — `main.go` uses a `placeholderEmailAPI` struct (lines 284–288) that returns empty results. The email polling loop runs but never fetches any messages.

## Design Decisions

### Pure-Go IMAP Library

Use `github.com/emersion/go-imap/v2` — a widely-used pure-Go IMAP client. No CGO dependency. Pair with `github.com/emersion/go-message` for MIME parsing.

### Connection Management

IMAP connections are stateful (unlike HTTP). The client manages a persistent connection with reconnection logic:

- Connect on first `FetchNewMessages` call
- Keep connection alive between poll cycles
- Reconnect on connection loss (with backoff)
- Use IMAP IDLE if supported, otherwise fall back to polling

### UID Tracking

To fetch only new messages, track the last-seen UID:

- On first poll: fetch the N most recent messages (configurable, default 10)
- On subsequent polls: fetch messages with UID > last-seen UID
- UID persists in memory only (resets on restart — acceptable for Phase 5)

### Interface Implementation

The client implements the existing `EmailAPI` interface from `internal/service/watcher/email.go`:

```go
type EmailAPI interface {
    FetchNewMessages(ctx context.Context, since uint32) ([]EmailMessage, error)
}
```

### Password Handling

The password comes from an environment variable specified in config (`password_env`). The client reads `os.Getenv(passwordEnv)` at construction time. If the env var is empty, return a clear error.

### Constructor

```go
// internal/service/watcher/email_api.go

type IMAPClient struct {
    host        string
    port        int
    username    string
    password    string
    conn        *imapclient.Client
}

func NewIMAPClient(host string, port int, username, passwordEnv string) (*IMAPClient, error)
```

## Error Handling

| Scenario | Behavior |
|---|---|
| IMAP connection refused | Return error, watcher logs and retries next cycle |
| Authentication failure | Return error with "check credentials" message |
| Connection lost mid-fetch | Reconnect, retry once, then return error |
| MIME parse failure | Log warning, skip message body, use subject only |
| Password env var empty | Constructor returns error — fatal for this account |
| TLS handshake failure | Return error with host/port details |

## Integration Points

- **Feature 006** (`email.go`): Implements `EmailAPI` interface
- **Feature 038** (Main Wiring): Watcher factory creates `IMAPClient` per account
- **`cmd/cue/main.go`**: Replace `placeholderEmailAPI` with `NewIMAPClient`

## Test Coverage

8 tests in `IMAPClientSuite`:

| Test | What it verifies |
|---|---|
| `TestNewIMAPClient_EmptyHost` | Constructor rejects empty host |
| `TestNewIMAPClient_ZeroPort` | Constructor rejects port ≤ 0 |
| `TestNewIMAPClient_EmptyUsername` | Constructor rejects empty username |
| `TestNewIMAPClient_PasswordEnvNotSet` | Constructor rejects unset env var |
| `TestNewIMAPClient_PasswordEnvEmpty` | Constructor rejects empty env var value |
| `TestNewIMAPClient_Valid` | Constructor succeeds with valid params |
| `TestFetchNewMessages_ConnectionRefused` | Returns error on unreachable server |
| `TestFetchNewMessages_ContextCancelled` | Respects cancelled context |

## TDD Agent Stats

| TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | test-designer (sonnet) | ~110s | ~30,000 | 07e328e |
| GREEN | implementer (sonnet) | ~195s | ~52,000 | 06db320 |
| REFACTOR | refactorer (sonnet) | ~128s | ~34,000 | 8e61a1b |

## Files

| File | Action |
|---|---|
| `go.mod` | **Modify** — add `github.com/emersion/go-imap/v2`, `github.com/emersion/go-message` |
| `internal/service/watcher/email_api.go` | **New** — `IMAPClient` implementing `EmailAPI` |
| `internal/service/watcher/email_api_test.go` | **New** — tests with mock IMAP server |
| `cmd/cue/main.go` | **Modify** — replace `placeholderEmailAPI` with `NewIMAPClient` |
