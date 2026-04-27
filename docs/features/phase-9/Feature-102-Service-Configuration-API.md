# Feature 102: Service Configuration API

**Phase:** Phase-9-Feature-102
**Status:** Done
**Package:** `internal/service/servicemanager/`, `internal/server/handler/`
**Depends on:** Feature 097, Feature 099A (Server Orchestrator Wiring)

---

## Overview

Exposes CRUD operations for Slack, Email, and Calendar account configurations via REST API. Adding/removing/toggling an account registers/deregisters the corresponding watcher with the orchestrator at runtime. A status summary endpoint reports all accounts and their watcher registration state.

## Design Decisions

1. **ServiceManager service layer** — A new `ServiceManager` in `internal/service/servicemanager/` wraps `ServiceConfigRepository` + orchestrator `WatcherLifecycle` + `WatcherFactory` + `MessageDeleter`. Both HTTP handlers and (hypothetically) other consumers share this single service. The defunct UI presenter was not updated.

2. **Credentials never exposed** — GET responses mask tokens/passwords with `"***"` (`CredentialMask`). Update requests with empty or masked credential fields preserve the existing stored values (partial update semantics).

3. **Validate on save** — No separate `/validate` endpoint. Credential validation runs synchronously during create/update with a 30s timeout. Validators are injected via functional options (`WithSlackValidator`, `WithEmailValidator`, `WithCalendarValidator`).

4. **Cascade message deletion** — Deleting an account also deletes all associated messages from the messages table via `DeleteBySourceAccount(ctx, source, sourceAccount)`. New method added to `MessageRepository` interface.

5. **Graceful watcher teardown** — Disabling an account calls `RemoveWatcher` which removes it from the orchestrator's watcher map. In-flight polls complete naturally since the orchestrator snapshots watchers at poll start.

6. **Status endpoint** — `GET /api/v1/services/status` returns account ID, type, name, enabled status, and watcher registration state. Poll stats (last/next poll, message count) deferred to a future feature.

7. **Calendar accounts have no watchers** — Calendar account CRUD persists config but does not interact with the orchestrator's watcher lifecycle (no watcher factory call, no watcher removal).

## Endpoints

### Slack Accounts

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/services/slack` | List all Slack accounts |
| GET | `/api/v1/services/slack/{id}` | Get account (masked token) |
| POST | `/api/v1/services/slack` | Create account (validates) |
| PUT | `/api/v1/services/slack/{id}` | Update account (partial) |
| DELETE | `/api/v1/services/slack/{id}` | Delete + cascade messages |
| POST | `/api/v1/services/slack/{id}/toggle` | Enable/disable |

### Email Accounts

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/services/email` | List all Email accounts |
| GET | `/api/v1/services/email/{id}` | Get account (masked password) |
| POST | `/api/v1/services/email` | Create account (validates) |
| PUT | `/api/v1/services/email/{id}` | Update account (partial) |
| DELETE | `/api/v1/services/email/{id}` | Delete + cascade messages |
| POST | `/api/v1/services/email/{id}/toggle` | Enable/disable |

### Calendar Accounts

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/services/calendar` | List all Calendar accounts |
| GET | `/api/v1/services/calendar/{id}` | Get account |
| POST | `/api/v1/services/calendar` | Create account (validates) |
| PUT | `/api/v1/services/calendar/{id}` | Update account (partial) |
| DELETE | `/api/v1/services/calendar/{id}` | Delete |
| POST | `/api/v1/services/calendar/{id}/toggle` | Enable/disable |

### Status

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/services/status` | All accounts with watcher state |

## Architecture

```
handler/service.go          HTTP ↔ JSON translation (19 handlers)
        ↓
servicemanager/manager.go   Business logic (CRUD + watcher lifecycle)
        ↓                   ↓                    ↓
ServiceConfigRepository   WatcherLifecycle    MessageDeleter
(SQLite, encrypted)       (Orchestrator)      (SQLite)
```

## Error Handling

| Status | Condition |
|--------|-----------|
| 200 | Success (list, get, update, status) |
| 201 | Created |
| 204 | Deleted, toggled |
| 400 | Invalid UUID, malformed JSON |
| 404 | Account not found |
| 500 | Internal error |

## Test Coverage

- **ServiceManager:** 55+ tests covering constructor validation, list/get/create/update/delete/toggle for all 3 account types, credential masking, credential preservation on update, cascade deletion, watcher lifecycle, status summary with watcher registration checks.
- **HTTP Handlers:** 16 tests covering success paths, error paths (400, 404), and all account types.
- **Repository:** 2 tests for `DeleteBySourceAccount`.

## TDD Agent Stats

See `docs/agent-log.md` for per-phase details.
