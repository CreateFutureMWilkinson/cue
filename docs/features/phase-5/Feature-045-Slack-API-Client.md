# Feature 045: Slack API Client

**Phase:** Phase-5-Feature-045
**Status:** Done
**Packages:** `internal/service/watcher/`, `cmd/cue/`
**Depends on:** Feature 038 (Main Wiring — dynamic watcher construction)

---

## Overview

Implements a real Slack API client that replaces the `placeholderSlackAPI` in `cmd/cue/main.go`. The `SlackWebClient` calls the Slack Web API using a user token to fetch channels, messages, and thread replies. The placeholder (which returned empty results, making the Slack watcher inert) has been removed.

## Motivation

The `SlackWatcher` (Feature 005) defines a `SlackAPI` interface and is fully tested against mock implementations. However, no real implementation existed — `main.go` used a `placeholderSlackAPI` struct that returned empty results. This feature provides the real HTTP client.

## Design Decisions

### Slack Web API (HTTP, No SDK)

Direct HTTP calls to the Slack Web API rather than a third-party SDK, keeping the dependency tree minimal and pure-Go:

- `conversations.list` — list channels the user is in
- `conversations.history` — fetch messages from a channel
- `conversations.replies` — fetch thread replies

All calls use the user token in the `Authorization: Bearer` header.

### User Token (Not Bot Token)

The client uses a Slack user token (`xoxp-...`) rather than a bot token (`xoxb-...`). This gives Cue visibility into exactly the channels and messages the user can see, which is the correct model for an ADHD productivity assistant monitoring *your* messages. The `SlackAccount.Token` field (renamed from `BotToken` in this feature) is generic to support either token type.

### Rate Limiting

Slack's Web API has tier-based rate limits. The client:

- Returns an error containing "rate limit" on 429 responses (the watcher retries next cycle)
- No concurrent API calls per workspace — sequential within a poll cycle

### Shared HTTP Helper

A `doRequest` method centralizes HTTP request creation, auth header injection, query parameter encoding, status checking, and JSON decoding. All three API methods delegate to it, eliminating duplication.

### Constructor

```go
// internal/service/watcher/slack_api.go

type SlackWebClient struct {
    token      string
    baseURL    string          // default: "https://slack.com/api"
    httpClient *http.Client    // default: 30s timeout
}

func NewSlackWebClient(token string, opts ...SlackClientOption) (*SlackWebClient, error)
```

Functional options:
- `WithBaseURL(url)` — override base URL (for testing with `httptest.NewServer`)
- `WithTimeout(d)` — override HTTP client timeout

## Error Handling

| Scenario | Behavior |
|---|---|
| Empty token | Constructor returns error |
| Invalid token (401) | Return error containing "auth", watcher retries next cycle |
| Rate limited (429) | Return error containing "rate limit", watcher retries next cycle |
| Forbidden (403) | Return error with "forbidden: insufficient permissions" |
| Network error/timeout | Return error, watcher retries next cycle |
| Malformed JSON response | Return error, skip this cycle |

## Integration Points

- **Feature 005** (`slack.go`): Implements `SlackAPI` interface
- **Feature 038** (Main Wiring): Watcher factory creates `SlackWebClient` per account
- **`cmd/cue/main.go`**: `placeholderSlackAPI` removed, replaced with `NewSlackWebClient(acct.Token)`

## Test Coverage

| Test | What it verifies |
|---|---|
| `TestConstructorRejectsEmptyToken` | Empty token returns error |
| `TestGetUserChannels` | Parses `conversations.list` response, verifies auth header |
| `TestGetChannelMessages` | Parses `conversations.history` with sender, text, timestamp, thread_ts |
| `TestGetChannelMessagesWithOldest` | `oldest` query param passed correctly |
| `TestGetThreadReplies` | Parses `conversations.replies` with parent and reply messages |
| `TestRateLimitReturnsError` | 429 → error containing "rate limit" |
| `TestInvalidTokenReturnsError` | 401 → error containing "auth" |
| `TestNetworkTimeout` | Blocked server → timeout error |
| `TestMalformedJSON` | Invalid JSON → error |
| `TestEmptyResponseReturnsEmptySlice` | Empty channels array → non-nil empty slice |

## TDD Agent Stats

| Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | test-designer | ~51s | ~25,000 | 1bc9f91 |
| GREEN | implementer | ~57s | ~28,000 | 16eb9c0 |
| REFACTOR | refactorer | ~88s | ~30,000 | 29b00d1 |

## Files

| File | Action |
|---|---|
| `internal/service/watcher/slack_api.go` | **New** — `SlackWebClient` implementing `SlackAPI` |
| `internal/service/watcher/slack_api_test.go` | **New** — 10 tests with `httptest.NewServer` |
| `cmd/cue/main.go` | **Modified** — replaced `placeholderSlackAPI` with `NewSlackWebClient` |
| `internal/repository/service_config.go` | **Modified** — `BotToken` → `Token` |
| `internal/repository/implementation/sqlite/service_config_impl.go` | **Modified** — SQL column `bot_token` → `token` + migration |
| `.claude/CLAUDE.md` | **Modified** — config example updated |
