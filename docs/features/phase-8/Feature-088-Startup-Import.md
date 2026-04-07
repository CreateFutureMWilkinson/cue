# Feature 088: Startup Import

**Phase:** Phase-8-Feature-088
**Status:** Planned
**Packages:** `internal/service/orchestrator/`, `internal/service/watcher/`
**Depends on:** Feature 087

---

## Overview

On launch, import all unseen messages from configured accounts as "Imported" status. This establishes a baseline of known messages so subsequent polls only process genuinely new arrivals. Imported messages are never scored or routed — they exist for record-keeping and future analysis only.

## Startup Flow

Before the orchestrator begins its polling loop:

1. For each configured watcher, fetch all available messages
2. For each message, check `ExistsByMessageID` in the DB
3. If not found: insert with `Status = "Imported"`, no scoring fields set
4. If found: skip (already known)
5. Log summary: `"startup import: N new records from source"` 

## "Imported" Status

A new message status alongside the existing Pending/Notified/Buffered/Ignored/Resolved:

- **Imported** — message existed before Cue started, stored for record-keeping only
- Never appears in notification panel, feedback review, or any user-facing queue
- Available for future analysis (e.g., training routing rules, identifying patterns)

## Email: INBOX Only

The email watcher's `FetchNewMessages` must only query the INBOX folder, not all folders. This is an IMAP client constraint — the `IMAPClient` should `SELECT INBOX` before fetching.

Verify the current `IMAPClient` implementation respects this. If it queries multiple folders, restrict it.

## Watcher Interface Extension

The current `Watcher` interface (defined in `internal/service/orchestrator/orchestrator.go`) has only:

```go
type Watcher interface {
    Poll(ctx context.Context) ([]*repository.Message, error)
}
```

`Poll()` returns messages new since the last poll. For import, we need all available messages. Add a new method:

```go
type Watcher interface {
    Poll(ctx context.Context) ([]*repository.Message, error)
    FetchAll(ctx context.Context) ([]*repository.Message, error)  // NEW
}
```

`FetchAll` returns all messages currently available in the source (INBOX for email, all joined channels for Slack). It does not track "last seen" state — it returns everything and lets the orchestrator dedup via `ExistsByMessageID`.

Both `SlackWatcher` and `EmailWatcher` must implement `FetchAll`. The implementation is similar to `Poll` but without the "since last poll" filter.

## Import vs Poll Distinction

The orchestrator has a dedicated import method, separate from `PollOnce`:

```go
func (o *Orchestrator) ImportBaseline(ctx context.Context) error
```

Called once at startup before `Start()`. It calls `watcher.FetchAll()` for each watcher, deduplicates via `ExistsByMessageID`, and inserts new messages as `"Imported"`. It does NOT call `watcher.Poll()` (which would advance the "last seen" cursor).

## Performance

Startup import may be slow for accounts with large inboxes. It should:

- Not block the UI (run in a background goroutine)
- Emit progress events: `"importing email: 42 messages..."`, `"import complete"`
- Not interfere with the UI becoming responsive
