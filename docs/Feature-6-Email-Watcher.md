# Feature 6: Email Watcher

**Phase:** Phase-1-Feature-6
**Status:** Done
**Package:** `internal/service/watcher/`

---

## Overview

Polls IMAP inbox for new messages using UID-based incremental fetching, detects @mentions by checking if the user's email address appears in To/CC/BCC (case-insensitive), and combines Subject + Body as raw content for LLM scoring. Read-only — never sends emails.

## Design Decisions

### UID-Based Incremental Polling

IMAP UIDs are immutable and ever-increasing per mailbox. The watcher tracks the highest UID seen and only fetches messages with UID > lastUID. This is simpler and more reliable than timestamp-based polling and naturally handles mailbox compaction.

### Atomic Polling (All or Nothing)

Unlike Slack's multi-channel approach where one channel failure doesn't block others, email polling is single-source. If `FetchNewMessages()` fails, the entire poll fails and the UID doesn't advance. This ensures no messages are silently skipped.

### Mention Detection via Address Matching

The user's email address is checked against To, CC, and BCC fields (case-insensitive). If found, `MessageType` is set to "mention" which the router uses for IS=8 deterministic scoring. This mirrors Slack's @mention detection but adapted for email semantics.

### Subject + Body as Content

`RawContent` combines `Subject + "\n" + Body` to give the LLM both the topic and detail for scoring. No HTML parsing — body is text-only per CLAUDE.md spec.

### No Thread Context

Email threading is implicit (Subject line, In-Reply-To header) and IMAP doesn't natively expose thread structure. Each email is scored independently. The subject line provides sufficient thread context for the LLM.

## API

### Interface

```go
type EmailAPI interface {
    FetchNewMessages(ctx context.Context, lastUID uint32) ([]EmailMessage, error)
}
```

### Types

```go
type EmailMessage struct {
    UID       uint32
    MessageID string   // RFC 5322 Message-ID header
    From      string
    Subject   string
    Folder    string
    Body      string   // Text only
    To        []string
    CC        []string
    BCC       []string
}
```

### Constructor

```go
func NewEmailWatcher(api EmailAPI, username string) (*EmailWatcher, error)
```

Requires non-nil API and non-empty username (user's email address for mention detection).

### Poll

```go
func (w *EmailWatcher) Poll(ctx context.Context) ([]*repository.Message, error)
```

## Polling Flow

1. Check context cancellation
2. Call `FetchNewMessages(ctx, lastUID)`
3. For each email:
   - Convert to `repository.Message`
   - Check if user is mentioned in To/CC/BCC (case-insensitive)
   - Set MessageType to "mention" or "message"
   - Combine Subject + Body as RawContent
4. Update lastUID to highest UID in batch

## Message Conversion

| Field | Value |
|---|---|
| Source | "email" |
| SourceAccount | username |
| Channel | folder name (e.g., "INBOX") |
| Sender | From address |
| MessageID | RFC 5322 Message-ID |
| MessageType | "mention" or "message" |
| RawContent | Subject + "\n" + Body |
| Status | "Pending" |

## Error Handling

| Scenario | Behavior |
|---|---|
| FetchNewMessages fails | Return error, entire poll aborted, UID unchanged |
| Context cancelled | Return error immediately |
| Invalid email data | Passed through as-is (repository/router validates) |

## Test Coverage

16 test cases in `email_test.go` using testify suites:

- Constructor validation (3): valid config, nil API, empty username
- Message fetching (2): multi-message fetch, field population
- Mention detection (5): user in To, CC, BCC, not in recipients, case-insensitive matching
- Content composition (1): RawContent contains Subject + Body
- UID tracking (2): avoids reprocessing, tracks highest UID in batch
- Error handling (1): API error propagates
- Context cancellation (1)
- Edge cases (1): empty inbox returns empty

## TDD Agent Stats

| Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | orchestrator | — | — | b5216cd |
| GREEN | orchestrator | — | — | e729f70 |
| REFACTOR | orchestrator | — | — | adfe21f |
