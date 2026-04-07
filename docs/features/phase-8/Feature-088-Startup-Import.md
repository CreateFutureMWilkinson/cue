# Feature 088: Startup Import

**Phase:** Phase-8-Feature-088
**Status:** Done
**Packages:** `internal/service/decisionengine/`, `internal/repository/`, `internal/repository/implementation/sqlite/`, `internal/service/watcher/`, `internal/service/orchestrator/`
**Depends on:** Feature 087

---

## Overview

On launch, the orchestrator imports all unseen messages as "Imported" status, seeds watcher cursors from stored DB values to minimize re-fetching, then begins normal polling. Imported messages are never scored, routed, or alerted -- they exist for record-keeping and cursor recovery only.

## Design Decisions

### No FetchAll method needed

The original design proposed a `FetchAll()` method on the `Watcher` interface. During implementation it became clear that `Poll()` already returns all messages on first call because cursors start at zero. Using `Poll()` plus cursor seeding from DB is simpler and avoids adding a redundant interface method.

### CursorSeedable interface

Rather than extending the `Watcher` interface (which would force all implementations to add cursor-seeding methods), an optional `CursorSeedable` interface was introduced. The orchestrator checks at runtime whether a watcher satisfies `CursorSeedable` via type assertion. Watchers that do not implement it are simply polled without cursor seeding.

```go
type CursorSeedable interface {
    SourceInfo() (source string, sourceAccount string)
    SeedCursor(ctx context.Context, repo repository.MessageRepository) error
}
```

### SourceCursor field

A new `SourceCursor string` field on `Message` stores the source-native cursor value: Slack message timestamp for Slack watchers, IMAP UID (as string) for email watchers. This value is persisted in SQLite and used to seed cursors at startup, so subsequent polls only fetch genuinely new messages.

### DistinctChannels query

A new `MessageRepository.DistinctChannels(ctx, source, sourceAccount) ([]string, error)` method returns all distinct channel values for a given source and account. `ImportBaseline` uses this to know which channels need cursor seeding for Slack (which tracks per-channel timestamps).

### Start() integration

`ImportBaseline` runs inside `Start()`'s goroutine before the first `PollOnce`, so it does not block the UI. The sequence is: seed cursors, poll all watchers, insert unseen messages as "Imported", then enter the normal polling loop.

## API

### Constants

- `StatusImported = "Imported"` -- new message status constant in `decisionengine` package

### Message struct

- `Message.SourceCursor string` -- source-native cursor value (Slack timestamp or IMAP UID as string)

### MessageRepository

- `MaxSourceCursor(ctx context.Context, source, sourceAccount, channel string) (string, error)` -- returns the maximum `source_cursor` value for a given source/account/channel combination
- `DistinctChannels(ctx context.Context, source, sourceAccount string) ([]string, error)` -- returns all distinct channel values for a given source/account

### Orchestrator

- `CursorSeedable` interface -- optional interface watchers implement for cursor seeding at startup
- `ImportBaseline(ctx context.Context) error` -- imports unseen messages from all watchers as "Imported" status

### SlackWatcher

- `SetLastTimestamp(channelID, timestamp string)` -- sets the last-seen timestamp for a channel
- `SourceInfo() (string, string)` -- returns `("slack", workspaceID)`
- `SeedCursor(ctx context.Context, repo repository.MessageRepository) error` -- seeds per-channel timestamps from DB

### EmailWatcher

- `SetLastUID(uid uint32)` -- sets the last-seen IMAP UID
- `SourceInfo() (string, string)` -- returns `("email", accountID)`
- `SeedCursor(ctx context.Context, repo repository.MessageRepository) error` -- seeds last UID from DB

## Startup Flow

1. `Start()` launches its goroutine
2. For each watcher that implements `CursorSeedable`, call `SeedCursor()` to restore cursor state from DB
3. Call `ImportBaseline()`:
   a. For each watcher, call `Poll()` (which returns all messages since the seeded cursor -- or all messages if no cursor was seeded)
   b. For each message, check `ExistsByMessageID` in the DB
   c. If not found: insert with `Status = "Imported"`, `SourceCursor` populated
   d. If found: skip
   e. Log summary per source
4. Enter normal polling loop (`PollOnce` on interval)

## "Imported" Status

A new message status alongside Pending/Notified/Buffered/Ignored/Resolved:

- **Imported** -- message existed before Cue started (or was fetched during startup import)
- Never appears in notification panel, feedback review, or any user-facing queue
- Never scored by Ollama or evaluated by rules engine
- Available for cursor recovery and future analysis

## Error Handling

- If cursor seeding fails for a watcher, log the error and continue (the watcher will poll from zero, which may re-fetch some messages but dedup prevents duplicates)
- If a single message insert fails during import, log and continue with remaining messages
- If `ImportBaseline` fails entirely, log the error and proceed to normal polling

## Test Coverage

7 behaviors with dedicated RED-GREEN cycles:

1. **StatusImported** -- constant exists and is used for import inserts
2. **SourceCursor** -- field persisted in SQLite schema, round-trips through store/fetch
3. **MaxSourceCursor** -- repository query returns correct max cursor per source/account/channel
4. **SlackCursor** -- `SetLastTimestamp` + `SourceInfo` + `SeedCursor` on SlackWatcher
5. **EmailCursor** -- `SetLastUID` + `SourceInfo` + `SeedCursor` on EmailWatcher
6. **ImportBaseline** -- orchestrator imports unseen messages, skips duplicates, sets Imported status
7. **CursorSeeding + DistinctChannels** -- `CursorSeedable` type assertion, `DistinctChannels` query, end-to-end seeding
8. **Start integration** -- `ImportBaseline` called before first `PollOnce` in `Start()` goroutine

## TDD Agent Stats

| Behavior | TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|---|
| StatusImported | RED | Test Designer | ~374s | ~22,900 | c0677dd |
| StatusImported | GREEN | orchestrator | -- | -- | 4311c97 |
| SourceCursor | RED | Test Designer | ~28s | ~34,200 | bb947e0 |
| SourceCursor | GREEN | Implementer | ~59s | ~29,900 | 4e8e617 |
| MaxSourceCursor | RED | Test Designer | ~83s | ~49,200 | 464442a |
| MaxSourceCursor | GREEN | orchestrator | -- | -- | a38a31f |
| SlackCursor | RED | Test Designer | ~48s | ~31,800 | 7efab6e |
| SlackCursor | GREEN | orchestrator | -- | -- | 9cfc2ad |
| EmailCursor | RED | Test Designer | ~52s | ~30,800 | 31f8026 |
| EmailCursor | GREEN | orchestrator | -- | -- | 61dc9d8 |
| ImportBaseline | RED | Test Designer | ~70s | ~36,200 | e70add7 |
| ImportBaseline | GREEN | orchestrator | -- | -- | b3de2e2 |
| CursorSeeding | RED | Test Designer | ~144s | ~40,900 | f9ae6e2 |
| CursorSeeding | GREEN | orchestrator | -- | -- | 27d17bc |
| Seedable+Channels | RED | Test Designer | ~58s | ~34,000 | 642d67c |
| Seedable+Channels | GREEN | orchestrator | -- | -- | c5b4372 |
| Start integration | RED | Test Designer | ~55s | ~34,700 | 5c4d5e5 |
| Start integration | GREEN | orchestrator | -- | -- | 90f8764 |
