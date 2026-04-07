# Feature 081 — Slack Token Setup Instructions in Settings

| Field | Value |
|---|---|
| Phase | 6 |
| Type | Enhancement |
| Severity | Low |
| Status | Planned |
| Depends on | 068 |
| UI Tests | Yes |

## Problem

The Slack "Add Account" form asks for a token but provides no guidance on how to obtain one. Slack's token setup process is non-trivial (create app, set scopes, install to workspace, copy token). Users need in-app instructions or a clear link to documentation.

## Required Changes

Add an instructional section to the Slack account form (above or below the token field) that explains:

1. **Go to** Slack API apps page (https://api.slack.com/apps)
2. **Create a new app** (or select existing) for your workspace
3. **Required OAuth scopes** (User Token Scopes):
   - `channels:history` — read messages in public channels
   - `channels:read` — list public channels
   - `groups:history` — read messages in private channels
   - `groups:read` — list private channels
   - `im:history` — read direct messages
   - `im:read` — list direct message channels
   - `mpim:history` — read group direct messages
   - `mpim:read` — list group DM channels
   - `users:read` — resolve user display names
4. **Install the app** to your workspace
5. **Copy the User OAuth Token** (starts with `xoxp-`)

### UI Presentation

- Collapsible/expandable instruction panel (default collapsed to keep the form clean)
- Or a "How to get a token" link/button that expands the instructions
- Scopes listed in a copyable format

## Acceptance Criteria

- Slack "Add Account" form includes visible instructions or a link to expand them
- Instructions list all required scopes
- Instructions are accurate for User OAuth Tokens (not Bot tokens)

## UI Test Coverage

- UI acceptance test: verify instructions element exists on the Slack account form
