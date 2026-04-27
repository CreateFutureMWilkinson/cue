# Feature 100: Feedback Buffer API

**Phase:** Phase-9-Feature-100
**Status:** Complete
**Package:** `internal/server/handler/`

---

## Overview

Expose the feedback buffer review workflow over REST. Users review messages that were scored as important but with low confidence (BUFFERED status), rate them 0-10, and optionally provide text feedback. This data improves future scoring via vector embeddings.

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/buffer` | List buffered messages (paginated) |
| GET | `/api/v1/buffer/{id}` | Get single buffered message detail |
| POST | `/api/v1/buffer/{id}/rate` | Rate a buffered message 0-10 |
| DELETE | `/api/v1/buffer/{id}` | Dismiss (soft delete) a buffered message |
| GET | `/api/v1/buffer/stats` | Buffer stats with per-source breakdown |

### List Buffered Messages

```
GET /api/v1/buffer?limit=50&offset=0
```

Returns messages with status "Buffered". Response includes `messages` array, `total` (full count), and `count` (page size).

### Get Buffered Message

```
GET /api/v1/buffer/{id}
```

Returns full `messageDetail` for a single buffered message. 404 if not found or not in Buffered status.

### Rate Buffered Message

```
POST /api/v1/buffer/{id}/rate
Content-Type: application/json

{"rating": 7, "feedback": "This was actually important"}
```

- `rating`: Required, integer 0-10
- `feedback`: Optional free-text string
- Returns 400 for invalid rating, 404 if not found, 409 if already resolved

Delegates to `BufferService.SaveRating()` which sets status to "Resolved" and silently stores vector embedding.

### Delete Buffered Message

```
DELETE /api/v1/buffer/{id}
```

Soft delete — sets status to "Resolved" without rating. `UserRating == nil` distinguishes dismissed from rated.

### Buffer Stats

```
GET /api/v1/buffer/stats
```

Returns `total_buffered` count and `by_source` map (e.g., `{"slack": 5, "email": 3}`).

## Design Decisions

1. **Random access** — stateless API, no server-side cursor
2. **Silent embedding** — embedding success/failure not exposed to API
3. **Integer ratings** — 0-10 only, matches data model
4. **Final ratings** — rate once, 409 if already resolved
5. **Soft delete** — `DeleteMessage()` sets status "Resolved" with nil `UserRating`

## Architecture

### Handler Interface

```go
type BufferRater interface {
    SaveRating(ctx context.Context, messageID uuid.UUID, rating int, feedback *string) error
    DeleteMessage(ctx context.Context, messageID uuid.UUID) error
}
```

List, get, and stats handlers use the existing `MessageQuerier` (already in `Deps`). Rate and delete handlers additionally need `BufferRater`.

### Wiring

`BufferService` created in `composition.go` via `constructHTTPServer()` with message repo and vector store, injected into `Deps.Buffer`. Routes registered behind `if s.deps.Buffer != nil` guard for rate/delete; list/get/stats always available when `Messages` is set.

## Files Modified

| File | Changes |
|------|---------|
| `internal/server/handler/buffer.go` | New — 5 handlers, `BufferRater` interface, response types |
| `internal/server/handler/buffer_test.go` | New — `BufferHandlerSuite` with 15 test cases |
| `internal/server/server.go` | Added `Buffer BufferRater` to `Deps`, 5 routes in `registerRoutes()` |
| `internal/server/composition.go` | Create `BufferService` in `constructHTTPServer()` |

## Test Coverage

15 test cases covering all 5 endpoints:
- List: success with pagination
- Get: success, not-buffered rejection, not-found
- Rate: success, invalid rating (high/negative), missing body, not-found, already-resolved
- Delete: success, not-found, already-resolved
- Stats: success, empty, error

## TDD Agent Stats

| Behavior | RED | GREEN | REFACTOR |
|----------|-----|-------|----------|
| B1 List | 46s / 32K tokens | 49s / 30K tokens | 62s / 30K tokens |
| B2 Get | 31s / 31K tokens | 32s / 27K tokens | — |
| B3 Rate | 87s / 35K tokens | 46s / 30K tokens | — |
| B4 Delete | 32s / 30K tokens | 22s / 25K tokens | — |
| B5 Stats | 44s / 33K tokens | 20s / 31K tokens | — |
