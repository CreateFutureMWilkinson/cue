# Feature 5: Slack Watcher

**Phase:** Phase-1-Feature-5
**Status:** Done
**Package:** `internal/service/watcher/`

---

## Overview

Polls Slack channels for new messages, detects channel joins (emitted as `channel_join` events for IS=9 routing), includes thread context for replies, and tracks per-channel timestamps to avoid reprocessing. Read-only — never sends messages to Slack.

## Design Decisions

### Per-Channel Timestamp Tracking

Each channel maintains its own high-water mark (`lastTimestamp`). This ensures channels with no new activity don't block others and allows incremental polling — only messages newer than the last seen timestamp are fetched.

### Channel Join Detection via Known-Channel Set

A `knownChannels` map tracks which channel IDs have been seen. When a new channel ID appears in `GetUserChannels()`, a synthetic `channel_join` message is emitted. This is more reliable than watching for Slack join events, which may be missed during downtime.

### Thread Context as Content Prefix

When a message is a thread reply (`ThreadTS` non-empty), the parent message text is fetched and prepended to the reply content with a newline separator. This gives the router/LLM full conversational context for scoring. If the parent fetch fails, the reply is still included without context — graceful degradation.

### Channel Errors Don't Block Other Channels

If `GetChannelMessages()` fails for one channel, that channel is skipped but others continue processing. A new channel that fails message fetch still emits its `channel_join` event.

## API

### Interface

```go
type SlackAPI interface {
    GetUserChannels(ctx context.Context) ([]SlackChannel, error)
    GetChannelMessages(ctx context.Context, channelID string, oldest string) ([]SlackMessage, error)
    GetThreadReplies(ctx context.Context, channelID string, threadTS string) ([]SlackMessage, error)
}
```

### Constructor

```go
func NewSlackWatcher(api SlackAPI, workspaceID string) (*SlackWatcher, error)
```

Requires non-nil API and non-empty workspace ID.

### Poll

```go
func (w *SlackWatcher) Poll(ctx context.Context) ([]*repository.Message, error)
```

Returns all new messages since last poll, including synthetic channel_join events.

## Polling Flow

1. Check context cancellation
2. Fetch all user channels via `GetUserChannels()`
3. For each channel:
   - Check if channel is new (not in `knownChannels`)
   - Fetch messages since `lastTimestamp[channelID]`
   - If new channel with no messages or fetch error, emit `channel_join` event
   - Convert each message to `repository.Message`
   - For thread replies, prepend parent message text
   - Update `lastTimestamp` to highest seen

## Message Conversion

| Field | Value |
|---|---|
| Source | "slack" |
| SourceAccount | workspaceID |
| Channel | channel name |
| Sender | Slack user ID |
| MessageID | Slack message ID |
| MessageType | "channel_join" or "message" |
| RawContent | message text (with thread parent if reply) |
| Status | "Pending" |

## Error Handling

| Scenario | Behavior |
|---|---|
| GetUserChannels fails | Return error, abort entire poll |
| GetChannelMessages fails | Skip channel, emit channel_join if new, continue others |
| GetThreadReplies fails | Include message without parent context |
| Context cancelled | Return error immediately |

## Test Coverage

14 test cases in `slack_test.go` using testify suites:

- Constructor validation (3): valid config, nil API, empty workspace ID
- Message fetching (2): multi-channel fetch, field population
- Channel join detection (3): new channel detection, multiple new channels, incremental detection
- Thread context (1): parent text prepended to reply
- Error handling (3): channel list error, message fetch error continues, thread fetch error continues
- Context cancellation (1)
- Edge cases (1): empty channel list

## TDD Agent Stats

| Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | orchestrator | — | — | f799fc0 |
| GREEN | Implementer | 115s | 33,346 | 2d63c02 |
| REFACTOR | Refactorer | 78s | 31,657 | bf674d8 |
