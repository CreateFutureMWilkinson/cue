# Feature 098: Message & Notification API

**Phase:** Phase-9-Feature-098
**Status:** Complete
**Package:** `internal/server/handler/`, `internal/repository/`

---

## Overview

Exposes message querying and notification management over REST. This is the highest-value API surface — it's what any alternative UI needs first to show the user what requires their attention.

## Design Decisions

### Preview vs Full Content

**Decision: Return full content.** The API returns `content` (the full `RawContent` field) and lets clients decide on truncation and presentation. The GUI truncates to 15 characters, but a TUI or mobile client may want different truncation or none at all.

### Pagination Strategy

**Decision: Offset-based for v1.** Both `/notifications` and `/messages` use `?limit=N&offset=N`. The notification list is typically small (<100 items) and refreshed frequently; the messages list could grow but offset is simpler to implement. Revisit cursor-based if clients report pagination issues.

Limit is clamped: default 50, max 200 (enforced in `QueryFiltered`).

### Content Sanitization

**Decision: Return raw.** The server returns `content` as the raw stored value from Slack/Email. No server-side sanitization — unknown future clients need to make their own rendering decisions.

### Batch Operations

**Decision: Deferred.** Resolve/dismiss operate on single notifications. Batch support (`POST /api/v1/notifications/batch`) can be added as a non-breaking new endpoint later.

### Handler Architecture

**Decision: Consumer-focused interface.** Handlers depend on `MessageQuerier` (a 3-method interface: `QueryFiltered`, `QueryByID`, `Update`) rather than the full `MessageRepository`. This keeps the handler package testable with minimal mocks and decoupled from repository internals.

The server accepts an optional `Deps` struct with a `Messages` field. When nil, notification/message routes are not registered — the server still functions for health checks.

## Endpoints

### GET /api/v1/notifications

Lists messages with status "Notified", newest first. Pagination via `?limit=50&offset=0`.

### GET /api/v1/notifications/{id}

Full message detail including reasoning, timestamps, and resolved_at.

### POST /api/v1/notifications/{id}/resolve

Sets status to "Resolved" and records `resolved_at`. Returns 404 if not found, 409 if already resolved.

### POST /api/v1/notifications/{id}/dismiss

Sets status to "Ignored". Returns 404 if not found.

### GET /api/v1/messages

General query with filters: `status`, `source`, `channel`, `since` (RFC 3339), `limit`, `offset`.

### GET /api/v1/messages/{id}

Full message detail for any message regardless of status.

## Repository Changes

Added `MessageFilter` struct and `QueryFiltered(ctx, filter) ([]*Message, int, error)` to `MessageRepository` interface. The SQLite implementation builds a dynamic WHERE clause from non-empty filter fields, runs a COUNT(*) for the total, and applies LIMIT/OFFSET with clamping. Existing `QueryByStatus` is unchanged (has a TODO for the user to consolidate later).

## Error Handling

| Condition | HTTP Status | Response |
|---|---|---|
| Message not found | 404 | `{"error": "not found"}` |
| Invalid UUID in path | 404 | `{"error": "not found"}` |
| Already resolved | 409 | `{"error": "already resolved"}` |
| Database error | 500 | `{"error": "..."}` |

## Test Coverage

| Suite | Tests | Coverage |
|---|---|---|
| NotificationHandlerSuite | 8 tests | List, get detail, resolve (success/not-found/conflict), dismiss (success/not-found) |
| MessageHandlerSuite | 3 tests | List with filters, get detail, not found |
| MessageRepoSuite (QueryFiltered) | 1 test | Filter by status with pagination |

## TDD Agent Stats

| Behavior | RED | GREEN | REFACTOR |
|---|---|---|---|
| QueryFiltered | b54e45d | a0bddf1 | 7d2ee1d |
| ListNotifications | 2b5d7bd | c8d8f19 | — |
| GetNotification | ba6f12a | e8c9006 | — |
| ResolveNotification | e41b4b8 | 49285a0 | — |
| DismissNotification | ea91780 | 513d893 | — |
| ListMessages | c5a7d34 | 02d99eb | — |
| GetMessage | 45e53fe | a72e487 | — |
| Wiring | — | 49e6d37 | — |
