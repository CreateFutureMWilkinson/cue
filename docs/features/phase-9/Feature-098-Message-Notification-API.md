# Feature 098: Message & Notification API

**Phase:** Phase-9-Feature-098
**Status:** Planning
**Package:** `internal/server/handler/`

---

## Overview

Expose message querying and notification management over REST. This is the highest-value API surface — it's what any alternative UI needs first to show the user what requires their attention.

## Endpoints

### List Notifications

```
GET /api/v1/notifications?limit=50&offset=0
```

Returns messages with status "Notified" that haven't been resolved. Ordered by `created_at` descending (newest first).

**Response:**
```json
{
  "notifications": [
    {
      "id": "uuid",
      "source": "slack",
      "source_account": "T...",
      "sender": "alice",
      "channel": "#incidents",
      "preview": "Server CPU at 9...",
      "importance_score": 9.0,
      "confidence_score": 0.95,
      "created_at": "2026-04-10T14:30:00Z"
    }
  ],
  "total": 12
}
```

**Question: Preview truncation.** The GUI truncates fields to 15 characters for the notification queue. Should the API return full content and let the client truncate, or include both `preview` (truncated) and `content` (full)? Recommend: return full content, let clients decide presentation.

### Get Notification Detail

```
GET /api/v1/notifications/{id}
```

Returns full message details including raw content, reasoning, and timestamps.

**Response:**
```json
{
  "id": "uuid",
  "source": "slack",
  "source_account": "T...",
  "channel": "#incidents",
  "sender": "alice",
  "message_id": "slack-native-id",
  "content": "Server CPU at 98% — need someone to look at prod-web-03 immediately",
  "importance_score": 9.0,
  "confidence_score": 0.95,
  "reasoning": "Server incident requiring immediate attention, high urgency language",
  "status": "Notified",
  "created_at": "2026-04-10T14:30:00Z",
  "updated_at": "2026-04-10T14:30:00Z"
}
```

### Resolve Notification

```
POST /api/v1/notifications/{id}/resolve
```

Sets status to "Resolved" and records `resolved_at`. Returns 200 on success, 404 if not found, 409 if already resolved.

### Dismiss Notification

```
POST /api/v1/notifications/{id}/dismiss
```

Sets status to "Ignored". Returns 200 on success, 404 if not found.

### List Messages (General)

```
GET /api/v1/messages?status=Notified&source=slack&limit=50&offset=0
```

More general query endpoint. Supports filtering by status, source, date range.

**Query parameters:**
- `status` — Filter by status (Notified, Buffered, Ignored, Resolved, Pending)
- `source` — Filter by source (slack, email)
- `channel` — Filter by channel name
- `since` — Only messages created after this RFC 3339 timestamp
- `limit` — Page size (default 50, max 200)
- `offset` — Pagination offset

**Question: Should this endpoint exist alongside the notifications endpoint?** The notifications endpoint is a convenience view (status=Notified, unresolved). The messages endpoint is the general-purpose query. Having both reduces confusion — clients that only care about "what needs attention" use `/notifications`, clients that need full history use `/messages`.

### Get Message by ID

```
GET /api/v1/messages/{id}
```

Same shape as notification detail but works for any message regardless of status.

## Design Decisions to Make

### Pagination Strategy

**Question: Offset-based or cursor-based pagination?**

- **Offset-based** (`?offset=50&limit=50`): Simple, familiar. But if new messages arrive between page fetches, items can shift — client might see duplicates or miss items.
- **Cursor-based** (`?after=uuid&limit=50`): Stable under concurrent writes. More complex to implement (need indexed cursor column). Better for real-time systems.

The notification list is small (typically <100 items) and refreshed frequently. Offset is probably fine. The general messages endpoint could grow large over time — cursor might be worth it there.

**Recommendation:** Offset for v1 on both endpoints. Revisit if clients report pagination issues with large message histories.

### Content Sanitization

**Question: Should `raw_content` be returned as-is, or sanitized?**

Message content comes from Slack and Email — it could contain HTML, markdown, emoji shortcodes, or arbitrary formatting. Options:
- Return raw, let client handle rendering
- Strip to plain text server-side
- Return both `raw_content` and `plain_text`

**Recommendation:** Return `content` as the raw stored value. The server shouldn't make presentation decisions for unknown future clients.

### Batch Operations

**Question: Should resolve/dismiss support batch operations?**

```
POST /api/v1/notifications/batch
{"action": "resolve", "ids": ["uuid1", "uuid2", ...]}
```

A TUI or mobile UI might want "resolve all" or "resolve selected". Adding batch support later is non-breaking (new endpoint), so deferring is safe.

## Behaviors to Implement

1. **List notifications handler** — Query messages with status "Notified", return paginated JSON.
2. **Get notification detail handler** — Query by ID, return full message, 404 if missing.
3. **Resolve notification handler** — Update status to "Resolved", set `resolved_at`, 404/409 error cases.
4. **Dismiss notification handler** — Update status to "Ignored", 404 if missing.
5. **List messages handler** — General query with status/source/channel/date filters, pagination.
6. **Get message handler** — By ID, any status.

## Dependencies

- `MessageRepository` interface (already exists: `QueryByStatus`, `QueryByID`, `Update`)
- May need a new `QueryFiltered(ctx, filters) ([]*Message, int, error)` method on the repository for the general messages endpoint with combined filters, or compose from existing methods.

## Questions Summary

1. Return full content or truncated previews?
2. Offset or cursor pagination?
3. Sanitize content or return raw?
4. Batch resolve/dismiss in v1 or defer?
5. Does `MessageRepository` need a new filtered query method, or can existing methods compose?
6. Should the response include a `self` link or other HATEOAS-style navigation?
