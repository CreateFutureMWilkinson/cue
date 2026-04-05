# Feature 045: Slack API Client

**Phase:** Phase-5-Feature-045
**Status:** Planned
**Packages:** `internal/service/watcher/`, `cmd/cue/`
**Depends on:** Feature 038 (Main Wiring — dynamic watcher construction)

---

## Overview

Implement a real Slack API client that replaces the `placeholderSlackAPI` in `cmd/cue/main.go`. The placeholder returns `nil` for all API calls (`GetUserChannels`, `GetChannelMessages`, `GetThreadReplies`), making the Slack watcher inert. This feature builds a client that calls the Slack Web API using a bot token to fetch channels, messages, and thread replies.

## Motivation

The `SlackWatcher` (Feature 005) defines a `SlackAPI` interface and is fully tested against mock implementations. However, no real implementation of `SlackAPI` exists — `main.go` uses a `placeholderSlackAPI` struct (lines 263–275) that returns empty results. The Slack polling loop runs but never fetches any messages.

## Design Decisions

### Slack Web API (HTTP, No SDK)

Use direct HTTP calls to the Slack Web API rather than a third-party SDK. This keeps the dependency tree minimal and pure-Go:

- `conversations.list` — list channels the bot is in
- `conversations.history` — fetch messages from a channel
- `conversations.replies` — fetch thread replies

All calls use the bot token in the `Authorization: Bearer` header.

### Rate Limiting

Slack's Web API has tier-based rate limits. The client implements:

- Respect `Retry-After` headers on 429 responses
- Default backoff of 60 seconds on rate limit (matches error handling spec §12)
- No concurrent API calls per workspace — sequential within a poll cycle

### Interface Implementation

The client implements the existing `SlackAPI` interface from `internal/service/watcher/slack.go`:

```go
type SlackAPI interface {
    GetUserChannels(ctx context.Context) ([]SlackChannel, error)
    GetChannelMessages(ctx context.Context, channelID string, oldest string) ([]SlackMessage, error)
    GetThreadReplies(ctx context.Context, channelID string, threadTS string) ([]SlackMessage, error)
}
```

### Constructor

```go
// internal/service/watcher/slack_api.go

type SlackWebClient struct {
    httpClient  *http.Client
    botToken    string
    baseURL     string  // default: "https://slack.com/api"
}

func NewSlackWebClient(botToken string, opts ...SlackClientOption) (*SlackWebClient, error)
```

Options allow overriding `baseURL` for testing with `httptest.NewServer`.

## Error Handling

| Scenario | Behavior |
|---|---|
| Invalid bot token (401) | Return error, watcher logs and retries next cycle |
| Rate limited (429) | Respect `Retry-After`, return error for this cycle |
| Network error | Return error, watcher logs and retries next cycle |
| Malformed JSON response | Return error, log, skip this cycle |
| Channel not found | Skip channel, continue with others |

## Integration Points

- **Feature 005** (`slack.go`): Implements `SlackAPI` interface
- **Feature 038** (Main Wiring): Watcher factory creates `SlackWebClient` per account
- **`cmd/cue/main.go`**: Replace `placeholderSlackAPI` with `NewSlackWebClient(botToken)`

## Test Coverage

- `GetUserChannels` returns parsed channels from mock server
- `GetChannelMessages` returns messages with sender, content, timestamp
- `GetChannelMessages` with `oldest` parameter filters correctly
- `GetThreadReplies` returns thread messages
- Rate limit (429) handled with retry-after
- Invalid token (401) returns descriptive error
- Network timeout returns error
- Malformed JSON returns error
- Empty response (no channels/messages) returns empty slice, not error

## Files

| File | Action |
|---|---|
| `internal/service/watcher/slack_api.go` | **New** — `SlackWebClient` implementing `SlackAPI` |
| `internal/service/watcher/slack_api_test.go` | **New** — tests with `httptest.NewServer` |
| `cmd/cue/main.go` | **Modify** — replace `placeholderSlackAPI` with `NewSlackWebClient` |
