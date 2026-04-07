# Feature 068A: Slack User Authentication

**Phase:** Phase-6-Feature-068A
**Type:** Bugfix (Hotfix)
**Severity:** High
**Status:** Planned
**Packages:** `internal/ui/`, `internal/repository/`, `internal/repository/implementation/sqlite/`
**Related:** Feature 068 (Slack Add Account), Feature 060 (Settings View)

---

## Problem

The Slack settings form is configured for **bot** authentication (label says "Bot Token", stores a single opaque token + workspace ID). Cue is a personal productivity tool that monitors channels *the user* is in — it should authenticate as the **user**, not as a bot application. Bot tokens (`xoxb-`) can only see channels the bot has been explicitly invited to. User tokens (`xoxp-`) see everything the user sees.

## Investigation: Slack User Authentication

### Token Types

| Prefix | Type | Use Case |
|---|---|---|
| `xoxp-` | **User OAuth token** | Acts as the user. Sees all channels the user is in. Recommended for Cue. |
| `xoxb-` | Bot token | Acts as a bot. Only sees channels it's been invited to. Current (wrong) approach. |
| `xoxc-`/`xoxd-` | Browser session tokens | Extracted from browser DevTools. Undocumented, fragile, against Slack TOS. Not viable. |
| `xapp-` | App-level token | For Socket Mode WebSocket. Requires bot token. Not needed for polling. |

### Recommended Approach: OAuth User Token (`xoxp-`)

For a single-user local-only desktop app, the simplest reliable flow is:

1. **User creates a Slack App** at `api.slack.com/apps` (one-time, ~2 minutes).
2. Under **OAuth & Permissions**, adds **User Token Scopes** (not Bot Token Scopes).
3. Clicks **"Install to Workspace"** — this is a simplified flow that does not require implementing an OAuth redirect server. It generates an `xoxp-` token immediately.
4. Copies the **User OAuth Token** into the Cue settings form.

The token does **not** expire by default. No refresh logic needed unless token rotation is explicitly enabled (not recommended for single-user local apps).

### Required OAuth Scopes (User Token)

For Cue's use case (read-only, monitor channels the user is in, detect @mentions):

| Scope | Purpose |
|---|---|
| `channels:read` | List public channels, get channel info |
| `channels:history` | Read message history in public channels |
| `groups:read` | List private channels the user is in |
| `groups:history` | Read message history in private channels |
| `users:read` | Look up user info (resolve sender names, detect @mentions) |

Optional (if monitoring DMs in future):
- `im:read`, `im:history` — direct messages
- `mpim:read`, `mpim:history` — group DMs

### Why Not Other Approaches

| Approach | Why Not |
|---|---|
| **Socket Mode** | Requires `xapp-` + `xoxb-` (bot only). Adds WebSocket complexity for a polling app. |
| **Browser session tokens** (`xoxc-`/`xoxd-`) | Undocumented, against Slack TOS, expires on logout/password change. Unreliable. |
| **Token rotation** | Adds refresh logic for minimal benefit in a single-user local app. |
| **Full OAuth redirect flow** | Overkill — requires running a local HTTP server. "Install to Workspace" is simpler. |

### Library Compatibility

The `slack-go/slack` library (if adopted later for real API calls) is fully token-agnostic. `slack.New("xoxp-...")` works identically to `slack.New("xoxb-...")`. The `SlackAPI` interface in `watcher/slack.go` abstracts over the token type — no watcher changes needed.

## Proposed Fix

### 1. UI Form — Relabel and add Username field

Replace the current bot-oriented form with user-oriented fields:

| Field | Widget | Placeholder | Validation |
|---|---|---|---|
| User OAuth Token | `widget.NewEntry()` (Password=true) | "User OAuth Token (xoxp-...)" | Required, non-empty |
| Workspace ID | `widget.NewEntry()` | "Workspace ID (T...)" | Required, non-empty |
| Username | `widget.NewEntry()` | "Your Slack Username (@handle)" | Required, non-empty |
| Poll Interval | `widget.NewEntry()` | "Poll Interval (seconds)" | Required, positive integer |

Key changes:
- **"Bot Token" -> "User OAuth Token"** — relabeled, placeholder shows `xoxp-` prefix, masked (Password=true since it's a credential)
- **New "Username" field** — needed for @mention detection (the router checks for `@username` in message content)
- Token entry gets `Password = true` to mask the credential in the UI

### 2. Data Model — Add `Username` field to `SlackAccount`

```go
type SlackAccount struct {
    ID                  uuid.UUID
    Enabled             bool
    Token               string
    WorkspaceID         string
    Username            string  // NEW: user's Slack handle for @mention detection
    PollIntervalSeconds int
    CreatedAt           time.Time
    UpdatedAt           time.Time
}
```

### 3. Database Schema — Add `username` column

```sql
ALTER TABLE slack_accounts ADD COLUMN username TEXT NOT NULL DEFAULT '';
```

Run as a migration in `NewSQLiteServiceConfigRepository`. Existing rows get empty string default (user can edit the account to add their username).

Update column constants:
```go
slackAccountColumns = "id, enabled, token_encrypted, workspace_id, username, poll_interval_seconds, created_at, updated_at"
```

### 4. Token Masking

The token entry should use `Password = true` to mask the credential in the UI, matching how the email password field works. Currently the Slack token field is unmasked — a security oversight since tokens are full-access credentials.

## Test Strategy

### Behaviors

1. **UI relabel** — Slack form shows "User OAuth Token" (not "Bot Token"), token field is masked, username field exists.
2. **Username field** — Form has 4 entry fields (was 3); username is required in validation.
3. **Model update** — `SlackAccount` has `Username` field.
4. **Schema migration** — DB migration adds `username` column with empty default.
5. **Save flow** — Form save persists username to `SlackAccount.Username`.
6. **Token masking** — Token entry widget has `Password = true`.

### TDD Micro-Loops

| # | Behavior | Scope |
|---|---|---|
| 1 | SlackAccount model has Username field | `repository/` |
| 2 | SQLite schema migration + upsert/scan with username | `repository/implementation/sqlite/` |
| 3 | UI form relabeled, token masked, 4 fields | `internal/ui/` |
| 4 | Username required in form validation | `internal/ui/` |
| 5 | Form save includes username | `internal/ui/` |

## Files to Change

| File | Change |
|---|---|
| `internal/repository/service_config.go` | Add `Username string` to `SlackAccount` |
| `internal/repository/implementation/sqlite/service_config_impl.go` | Migration, update columns, upsert, scan |
| `internal/ui/settings_view.go` | Relabel form, add username field, mask token, update validation |
| `internal/ui/settings_view_test.go` | Update Slack form field count, check labels |
| `internal/ui/settings_interaction_test.go` | Update Slack form interaction tests |
| `tests/ui/settings_acceptance_test.go` | Update Slack tab acceptance tests (3 -> 4 entries) |

## Setup Guide (for docs)

Include in feature documentation for end users:

### Setting Up Slack User Authentication

1. Go to [api.slack.com/apps](https://api.slack.com/apps) and click **Create New App** > **From scratch**.
2. Name it anything (e.g., "Cue Monitor") and select your workspace.
3. Go to **OAuth & Permissions** in the left sidebar.
4. Under **User Token Scopes** (NOT Bot Token Scopes), add:
   - `channels:read`
   - `channels:history`
   - `groups:read`
   - `groups:history`
   - `users:read`
5. Click **Install to Workspace** at the top of the page and authorize.
6. Copy the **User OAuth Token** (starts with `xoxp-`).
7. In Cue Settings > Slack > Add Account, paste the token, enter your workspace ID and @username.

## Acceptance Criteria

- [ ] Slack form label says "User OAuth Token", not "Bot Token"
- [ ] Token field placeholder shows "User OAuth Token (xoxp-...)"
- [ ] Token field is masked (Password = true)
- [ ] Form has 4 entry fields: Token, Workspace ID, Username, Poll Interval
- [ ] Username is required in validation
- [ ] `SlackAccount` struct has `Username` field
- [ ] SQLite migration adds `username` column; existing rows default to empty string
- [ ] Saved accounts persist the username
- [ ] All existing Slack tests remain green
- [ ] All existing settings tests remain green
