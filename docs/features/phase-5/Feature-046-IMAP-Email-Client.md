# Feature 046: IMAP Email Client

**Phase:** Phase-5-Feature-046
**Status:** Planned
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

Testing real IMAP requires a mock server. Use `github.com/emersion/go-imap/v2/imap` test utilities or a lightweight in-process IMAP server for tests:

- `FetchNewMessages` returns parsed messages from mock server
- Message extraction: sender, subject, body text, folder
- @mention detection: user's email in To/CC/BCC fields
- UID tracking: second poll returns only new messages
- Connection failure returns descriptive error
- Authentication failure returns descriptive error
- Reconnection after connection drop
- Empty inbox returns empty slice, not error
- MIME multipart message: text/plain body extracted correctly
- HTML-only message: basic text extraction or skip

## Files

| File | Action |
|---|---|
| `go.mod` | **Modify** — add `github.com/emersion/go-imap/v2`, `github.com/emersion/go-message` |
| `internal/service/watcher/email_api.go` | **New** — `IMAPClient` implementing `EmailAPI` |
| `internal/service/watcher/email_api_test.go` | **New** — tests with mock IMAP server |
| `cmd/cue/main.go` | **Modify** — replace `placeholderEmailAPI` with `NewIMAPClient` |
