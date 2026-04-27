# Feature 100: Feedback Buffer API

**Phase:** Phase-9-Feature-100
**Status:** Planning
**Package:** `internal/server/handler/`

---

## Overview

Expose the feedback buffer review workflow over REST. Users review messages that were scored as important but with low confidence (BUFFERED status), rate them 0-10, and optionally provide text feedback. This data improves future scoring via vector embeddings.

## Endpoints

### List Buffered Messages

```
GET /api/v1/buffer?limit=50&offset=0
```

Returns messages with status "Buffered", ordered by `created_at` descending.

**Response:**
```json
{
  "messages": [
    {
      "id": "uuid",
      "source": "slack",
      "sender": "bob",
      "channel": "#deployments",
      "content": "Deploying v2.3.1 to staging, should be done in 20 min",
      "importance_score": 7.5,
      "confidence_score": 0.6,
      "reasoning": "Deployment notification, moderate urgency",
      "created_at": "2026-04-10T10:00:00Z"
    }
  ],
  "total": 8,
  "count": 8
}
```

### Get Buffered Message

```
GET /api/v1/buffer/{id}
```

Returns full details for a single buffered message. 404 if not found or not in Buffered status.

### Rate Buffered Message

```
POST /api/v1/buffer/{id}/rate
```

**Request:**
```json
{
  "rating": 7,
  "feedback": "This was actually important, deployment blocked my PR"
}
```

- `rating`: Required, integer 0-10.
- `feedback`: Optional, free-text string.

**Behavior:**
1. Validate rating is 0-10
2. Save rating and feedback to message
3. Store vector embedding of content + feedback (if vector store configured)
4. Set message status to "Resolved"
5. Return 200 with updated message

**Error cases:**
- 400: Rating out of range or missing
- 404: Message not found
- 409: Message already resolved/rated

### Delete Buffered Message

```
DELETE /api/v1/buffer/{id}
```

Removes a buffered message from the review queue (sets status to "Ignored" without rating). For messages the user deems not worth reviewing.

### Buffer Stats

```
GET /api/v1/buffer/stats
```

**Response:**
```json
{
  "total_buffered": 8,
  "by_source": {
    "slack": 5,
    "email": 3
  }
}
```

Useful for badges/indicators in alternative UIs ("3 messages to review").

## Design Decisions to Make

### Review Workflow: Sequential vs Random Access

The GUI presents buffered messages one at a time (sequential review). The `FeedbackPresenter` maintains a cursor position and advances on rate/skip.

**Question: Should the API enforce sequential review or allow random access?**

- **Sequential** (mirror GUI): Client calls `GET /buffer/next` to get the current review item, then rates or skips to advance. Server maintains cursor state.
- **Random access** (recommended): Client fetches the list, picks any item to rate. No server-side cursor. Simpler, more flexible, works for any UI pattern (list view, card view, single-item view).

**Recommendation:** Random access. The API should be stateless. The sequential one-at-a-time pattern is a GUI UX choice, not a business requirement. A TUI might want to show all 8 buffered messages at once and let the user pick.

### Vector Embedding on Rate

When a user rates a message, the `BufferService.SaveRating()` optionally stores a vector embedding. This is transparent to the API caller — the client sends a rating, the server handles embedding internally.

**Question: Should the API expose whether embedding succeeded?** Options:
- Silent: Always return 200 if the rating saved, regardless of embedding success. (Current GUI behavior — embedding failure is logged, not shown.)
- Informational: Include `"embedding_stored": true/false` in the response.

**Recommendation:** Silent for v1. Embedding is an internal optimization. The client doesn't need to know or act on it.

## Behaviors to Implement

1. **List buffered messages handler** — Query status "Buffered", paginated.
2. **Get buffered message handler** — By ID, 404 if not found or not buffered.
3. **Rate message handler** — Validate rating 0-10, delegate to `BufferService.SaveRating()`, return updated message.
4. **Delete buffered message handler** — Delegate to `BufferService.DeleteMessage()`.
5. **Buffer stats handler** — Count buffered by source.

## Dependencies

- `BufferService` (implements `BufferReviewer` interface)
- `MessageRepository` (for stats queries — may need `CountByStatusAndSource`)

## Questions Summary

1. Sequential review workflow or random access?
2. Expose embedding success/failure to API clients?
3. Should rating allow decimal values (e.g., 7.5) or strictly integers 0-10?
4. Should there be a `PATCH /buffer/{id}/rate` to update a previously submitted rating, or is rating final?
5. Should `DELETE /buffer/{id}` actually delete the message from the database or just change its status?
