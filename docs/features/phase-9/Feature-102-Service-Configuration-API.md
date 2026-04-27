# Feature 102: Service Configuration API

**Phase:** Phase-9-Feature-102
**Status:** Planning
**Package:** `internal/server/handler/`

---

## Overview

Expose CRUD operations for Slack, Email, and Calendar account configurations. This enables alternative UIs to manage which data sources Cue monitors. Adding/removing an account must also register/deregister the corresponding watcher with the orchestrator at runtime.

## Endpoints

### Slack Accounts

```
GET    /api/v1/services/slack              → list all Slack accounts
GET    /api/v1/services/slack/{id}         → get account details
POST   /api/v1/services/slack              → create new account
PUT    /api/v1/services/slack/{id}         → update account
DELETE /api/v1/services/slack/{id}         → delete account
POST   /api/v1/services/slack/{id}/toggle  → enable/disable
POST   /api/v1/services/slack/{id}/validate → test credentials
```

### Email Accounts

```
GET    /api/v1/services/email              → list all Email accounts
GET    /api/v1/services/email/{id}         → get account details
POST   /api/v1/services/email              → create new account
PUT    /api/v1/services/email/{id}         → update account
DELETE /api/v1/services/email/{id}         → delete account
POST   /api/v1/services/email/{id}/toggle  → enable/disable
POST   /api/v1/services/email/{id}/validate → test IMAP credentials
```

### Calendar Accounts

```
GET    /api/v1/services/calendar              → list all Calendar accounts
GET    /api/v1/services/calendar/{id}         → get account details
POST   /api/v1/services/calendar              → create new account
PUT    /api/v1/services/calendar/{id}         → update account
DELETE /api/v1/services/calendar/{id}         → delete account
POST   /api/v1/services/calendar/{id}/toggle  → enable/disable
POST   /api/v1/services/calendar/{id}/validate → test ICS URL
```

### Example: Create Slack Account

**Request:**
```json
{
  "name": "Work Slack",
  "bot_token": "xoxp-...",
  "workspace_id": "T...",
  "enabled": true
}
```

**Response (201):**
```json
{
  "id": "uuid",
  "name": "Work Slack",
  "workspace_id": "T...",
  "enabled": true,
  "created_at": "2026-04-10T14:30:00Z"
}
```

**Note:** Sensitive fields (tokens, passwords) are NEVER returned in GET responses. They are write-only.

### Validate Endpoint

```
POST /api/v1/services/slack/{id}/validate
```

**Response (200):**
```json
{
  "valid": true,
  "message": "Successfully connected to workspace"
}
```

**Response (200, invalid):**
```json
{
  "valid": false,
  "message": "Authentication failed: invalid token"
}
```

Validation is always 200 (the request succeeded) — the `valid` field indicates whether credentials work.

**Question: Should validation be synchronous or fire-and-forget?** IMAP validation can be slow (5-10s for connection + auth). Recommend synchronous with a 15s timeout — the client needs the result before proceeding.

## Design Decisions to Make

### Credential Storage

Credentials are currently stored encrypted in SQLite via `ServiceConfigRepository` using a `secret.key` file.

**Question: How should the API handle credential input?**

- Tokens/passwords are provided in POST/PUT request bodies
- They're stored encrypted server-side
- They're NEVER returned in GET responses (replaced with `"***"` or omitted)
- Update requests with empty/null credential fields preserve existing credentials (partial update)

**Question: Should the API support reading credentials?** A "show token" button in a UI is convenient but exposes secrets over HTTP. Options:
- Never expose credentials via API (must check DB directly)
- Expose via a separate privileged endpoint (`POST /services/slack/{id}/reveal-token`)
- Always mask in responses

**Recommendation:** Never expose. This is a local-only app, but good security hygiene costs nothing. If a user needs to see their token, they can check their config or the DB.

### Watcher Lifecycle

When an account is created/enabled, a watcher must be registered with the orchestrator. When deleted/disabled, it must be deregistered. This is currently handled by `ServiceSettingsPresenter` calling `WatcherFactory` and `WatcherRemover`.

**Question: Should the API handler directly manage watcher lifecycle, or go through an intermediary service?**

- **Direct**: Handler calls `watcherFactory(account)` and `orchestrator.AddWatcher()` / `orchestrator.RemoveWatcher()`. Simple but couples HTTP handler to orchestrator internals.
- **Service layer**: New `ServiceManager` that wraps repo + watcher lifecycle. Handler calls `serviceManager.CreateSlackAccount()` which handles both persistence and watcher registration.

**Recommendation:** Service layer. The GUI presenter already does this dance (save to repo, create watcher, add to orchestrator). Extracting it into a reusable service benefits both GUI and server. This is a case where the refactor pays for itself.

### Validation Before Save

**Question: Should account creation automatically validate credentials, or should validation be a separate explicit step?**

- **Auto-validate on create**: Safer — prevents saving broken configs. But blocks the create call.
- **Separate validation**: Client creates the account, then calls validate. Account exists in DB even if invalid. Simpler for UIs that want to save-then-test.

**Recommendation:** Separate validation. The GUI currently does save-then-test. Keep the same pattern for the API.

## Behaviors to Implement

1. **List accounts handlers** (Slack, Email, Calendar) — Query from `ServiceConfigRepository`.
2. **Get account handler** — By ID, mask credentials in response.
3. **Create account handler** — Validate input, save to repo, optionally create and register watcher.
4. **Update account handler** — Partial update support (preserve credentials if not provided).
5. **Delete account handler** — Remove from repo, deregister watcher.
6. **Toggle handler** — Enable/disable account, register/deregister watcher accordingly.
7. **Validate handler** — Run appropriate validator (Slack/IMAP/ICS), return result.
8. **ServiceManager extraction** — Refactor watcher lifecycle out of presenter into shared service.

## Testing Considerations

- Create account → verify watcher registered with orchestrator.
- Delete account → verify watcher deregistered.
- Toggle off → verify watcher removed. Toggle on → verify watcher re-created.
- Update credentials → verify new watcher created with new credentials.
- Validate with bad credentials → verify clean error, account not modified.
- Credential masking: GET response must never include raw tokens.

## Questions Summary

1. Should the API ever expose stored credentials?
2. Auto-validate on create or separate explicit validation?
3. Extract `ServiceManager` service layer or keep logic in handler?
4. Should account deletion cascade-delete associated messages?
5. What happens to in-flight polls when a watcher is removed mid-cycle?
6. Should there be a `GET /api/v1/services/status` summary endpoint showing all account health?
