# Feature 101A: Todo CRUD API

**Phase:** Phase-9-Feature-101A
**Status:** Done
**Package:** `internal/server/handler/`, `internal/repository/`
**Depends on:** 097

---

## Overview

Expose full CRUD operations for todos (tasks) over REST. Adds time estimation to the todo model — stored as integer minutes with two fields: a user-provided estimate and an LLM-generated estimate. The effective estimate is `user_estimate ?? llm_estimate`. LLM estimation runs asynchronously on task creation and when a user clears their estimate to zero/nil.

This feature is a prerequisite for Feature 101 (Day Planner API), which will consume these endpoints for task selection.

## Model Changes

### New Fields on `Todo`

| Field | Type | Description |
|-------|------|-------------|
| `EstimateMinutes` | `*int` | User-provided time estimate in minutes. Supersedes LLM estimate. |
| `LLMEstimateMinutes` | `*int` | LLM-generated time estimate in minutes. Populated asynchronously. |

**Effective estimate:** `EstimateMinutes` if non-nil and > 0, otherwise `LLMEstimateMinutes`.

### Priority Semantics Change

The existing model uses `Priority int // lower = higher priority`. This API changes the convention:

- **Higher value = higher priority** (e.g., priority 10 is more important than priority 1)
- **Default: 0** (lowest priority)
- Primary sort field — higher priority tasks returned first

This is a breaking change to the existing priority convention. The SQLite migration and any existing UI code referencing priority ordering must be updated.

### SQLite Migration

Add columns to `todos` table:

```sql
ALTER TABLE todos ADD COLUMN estimate_minutes INTEGER;
ALTER TABLE todos ADD COLUMN llm_estimate_minutes INTEGER;
```

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/tasks` | List tasks (filtered, paginated) |
| POST | `/api/v1/tasks` | Create task |
| GET | `/api/v1/tasks/{id}` | Get single task |
| PUT | `/api/v1/tasks/{id}` | Update task |
| DELETE | `/api/v1/tasks/{id}` | Delete task (hard delete) |

### List Tasks

```
GET /api/v1/tasks?status=incomplete&category=code-review&search=deploy&limit=50&offset=0
```

**Query Parameters:**

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `status` | string | `incomplete` | `incomplete`, `complete`, or `all` |
| `category` | string | — | Filter by category name |
| `search` | string | — | Fuzzy match against title AND description |
| `limit` | int | 50 | Page size |
| `offset` | int | 0 | Pagination offset |

**Sort order:** Primary sort by `priority` descending (higher first), secondary by `created_at` ascending.

**Response:**
```json
{
  "tasks": [
    {
      "id": "uuid",
      "title": "Review PR #42",
      "description": "Check the new auth middleware",
      "priority": 5,
      "due_date": "2026-04-11T00:00:00Z",
      "categories": ["code-review", "team"],
      "estimate_minutes": null,
      "llm_estimate_minutes": 30,
      "effective_estimate_minutes": 30,
      "created_at": "2026-04-10T09:00:00Z",
      "completed_at": null
    }
  ],
  "total": 42,
  "count": 1
}
```

### Create Task

```
POST /api/v1/tasks
Content-Type: application/json

{
  "title": "Review PR #42",
  "description": "Check the new auth middleware",
  "priority": 5,
  "due_date": "2026-04-11",
  "categories": ["code-review", "team"],
  "estimate_minutes": null
}
```

**Required fields:** `title`
**Optional fields:** `description`, `priority` (default 0), `due_date`, `categories`, `estimate_minutes`

**Response:** 201 Created with the full task object.

**LLM estimation:** If `estimate_minutes` is nil or 0, triggers an async LLM estimation. The response returns immediately with `llm_estimate_minutes: null`. A subsequent GET will include the LLM estimate once it completes.

### Get Task

```
GET /api/v1/tasks/{id}
```

Returns the full task object. 404 if not found.

### Update Task

```
PUT /api/v1/tasks/{id}
Content-Type: application/json

{
  "title": "Review PR #42 (updated)",
  "priority": 8,
  "estimate_minutes": 45,
  "completed_at": "2026-04-10T15:00:00Z"
}
```

Full replacement of provided fields. Omitted fields retain current values.

**Completion:** Set `completed_at` to a timestamp to mark complete, set to `null` to reopen.

**LLM re-estimation trigger:** If `estimate_minutes` is explicitly set to 0 or null AND the previous value was non-zero, triggers async LLM re-estimation (clears existing `llm_estimate_minutes` and re-runs).

**Response:** 200 OK with the full updated task object. 404 if not found.

### Delete Task

```
DELETE /api/v1/tasks/{id}
```

Hard delete. Returns 204 No Content. 404 if not found.

## Async LLM Estimation

### Trigger Conditions

1. **On create:** Always, unless user provides `estimate_minutes` > 0
2. **On update:** Only when `estimate_minutes` changes to 0/nil from a non-zero value

### Implementation

A goroutine-based estimator that:
1. Receives estimation requests via a channel
2. Calls Ollama with the task's title and description
3. Parses the response as minutes (integer)
4. Updates the task's `llm_estimate_minutes` in the DB
5. On Ollama failure: sets a default estimate (e.g., 30 minutes) and logs the error

The estimator uses the existing `OllamaGenerator` interface but with a time-focused prompt (not pomodoro-focused).

### Interface

```go
type TimeEstimator interface {
    EstimateMinutes(ctx context.Context, title, description string) (int, error)
}
```

## Design Decisions

1. **Hard delete** — no soft delete for tasks. Completed tasks are retained via `completed_at`.
2. **Integer minutes** — presentation formatting (hours, pomodoros) is a UI concern.
3. **Two estimate fields** — user estimate always supersedes LLM. Keeps provenance clear.
4. **Async estimation** — create/update returns immediately. Client polls or uses WebSocket events to detect estimate arrival.
5. **Search** — single `search` param matches both title and description. SQL LIKE-based for v1 (no full-text search).
6. **Priority reversal** — higher value = higher priority (99 > 0), default 0. This deliberately reverses the existing convention (`lower = higher priority`) used in the Todo model and planner presenter. Existing priority references to update: model comment and sort order in repository queries. UI code does not need updating. This is a known breaking change.

## Behaviors to Implement

1. **List tasks handler** — filtered, paginated, sorted by priority desc
2. **Create task handler** — validate, insert, trigger async estimation
3. **Get task handler** — by ID, 404 handling
4. **Update task handler** — partial update, completion, re-estimation trigger
5. **Delete task handler** — hard delete, 404 handling
6. **Async time estimator** — goroutine worker, Ollama integration, DB update
7. **Repository changes** — new fields, filtered query method, priority sort reversal
8. **SQLite migration** — add estimate columns

## Testing Considerations

- CRUD happy paths for all 5 endpoints
- List filtering: by status, category, search, pagination
- Sort order: priority descending, then created_at ascending
- LLM estimation: mock estimator, verify async trigger on create
- LLM re-estimation: verify trigger when estimate cleared to 0
- No re-estimation when user provides estimate > 0
- Effective estimate calculation: user > LLM > nil
- Validation: missing title on create, invalid UUID on get/update/delete
- Priority default: task created without priority gets 0

## Implementation Summary

### Packages Changed

| Package | Changes |
|---------|---------|
| `internal/repository/todo.go` | Added `EstimateMinutes *int` and `LLMEstimateMinutes *int` to `Todo` model. Replaced `QueryIncomplete`/`QueryAll` with `QueryFiltered(filter TodoFilter)` on `TodoRepository` interface. |
| `internal/repository/implementation/sqlite/todo_impl.go` | Schema migration adds `estimate_minutes` and `llm_estimate_minutes` columns. `QueryFiltered` builds dynamic WHERE clause for status/category/search filtering with pagination and priority-descending sort. |
| `internal/service/planner/estimator.go` | Renamed `EstimatePomodoros` to `EstimateMinutes` on `OllamaTaskEstimator`. Returns integer minutes with 30-minute fallback on Ollama failure. |
| `internal/service/todo/` | New `TodoService` owning `TodoRepository` + async `TimeEstimator` worker. CRUD methods delegate to repository; create/update trigger async LLM estimation via channel when no user estimate provided. |
| `internal/server/handler/todo.go` | Five REST handlers following the buffer handler pattern: list (filtered/paginated), create, get, update, delete. JSON request/response with validation. |
| `internal/server/server.go` | `Deps` struct gained `TodoService` field. Routes registered under `/api/v1/tasks`. |
| `internal/server/composition.go` | `TodoService` wired with todo repository and planner estimator as `TimeEstimator`. |
| `internal/ui/presenter/planner_presenter.go` | Updated interface call from `QueryIncomplete`/`QueryAll` to `QueryFiltered` (minimal change, UI defunct). |

### Breaking Changes

- **Priority semantics reversed:** Higher value = higher priority (was lower = higher). Default 0.
- **`EstimatePomodoros` removed:** Replaced by `EstimateMinutes` on `OllamaTaskEstimator` interface.
- **`QueryIncomplete`/`QueryAll` removed:** Replaced by `QueryFiltered` on `TodoRepository` interface.

## Test Coverage Summary

- Todo model estimate fields and effective estimate logic
- `QueryFiltered` with status, category, search, pagination, and sort order
- `EstimateMinutes` with Ollama success and 30-minute fallback
- `TodoService` CRUD operations (create, get, update, delete, list)
- Async LLM estimation trigger on create (no user estimate) and update (estimate cleared)
- All five REST handlers: success paths, validation errors, not-found, pagination
- Server wiring: `TodoService` in `Deps`, route registration

## TDD Agent Stats

| Behavior | TDD Phase | Agent | Duration | Tokens |
|----------|-----------|-------|----------|--------|
| Estimate fields | RED | Test Designer | ~55s | ~33,000 |
| Estimate fields | GREEN | Implementer | ~53s | ~34,000 |
| QueryFiltered | RED | Test Designer | ~271s | ~62,000 |
| QueryFiltered | GREEN | Implementer | ~66s | ~38,000 |
| QueryFiltered | REFACTOR | Refactorer | ~84s | ~41,000 |
| EstimateMinutes | RED | Test Designer | ~92s | ~32,000 |
| EstimateMinutes | GREEN | Implementer | ~30s | ~26,000 |
| TodoService CRUD | RED | Test Designer | ~92s | ~32,000 |
| TodoService CRUD | GREEN | Implementer | ~53s | ~29,000 |
| Async estimation | RED | Test Designer | ~76s | ~33,000 |
| Async estimation | GREEN | Implementer | ~42s | ~32,000 |
| Handlers | RED | Test Designer | ~103s | ~48,000 |
| Handlers | GREEN | Implementer | ~79s | ~41,000 |
| Server wiring | RED | Test Designer | ~94s | ~39,000 |
| Server wiring | GREEN | Implementer | ~32s | ~28,000 |
