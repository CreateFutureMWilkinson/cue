# Feature 109: Todo Domain Restructure

**Phase:** Phase-9-Feature-109
**Status:** Planning
**Depends on:** Feature 101A (Todo CRUD API), Feature 102 (Service Configuration API), Feature 106 (API Client SDK)
**Blocks:** Feature 107 (Fyne Client Re-wire)
**Packages:** `internal/repository/`, `internal/repository/implementation/sqlite/`, `internal/server/handler/`, `internal/server/`, `pkg/client/`

---

## Overview

Promote categories from string-tags-on-tasks to first-class entities with their own CRUD, and namespace todo-related routes under `/api/v1/todo/`. The change exists primarily to give 107's todo list view a stable, query-able category surface (filter dropdown, colour badges, rename without touching every task) and to clean up the route hierarchy before any external consumer locks onto the current paths.

After 109 ships:
- `POST/GET/PUT/DELETE /api/v1/tasks` → `/api/v1/todo/tasks` (paths renamed, no compat shim).
- New `GET/POST /api/v1/todo/categories`, `GET/PUT/DELETE /api/v1/todo/categories/{id}`.
- Tasks reference categories by UUID on writes; responses embed both ids and names for convenience.
- Categories carry `id`, `name`, `colour` (nullable hex), `created_at`, plus a derived `task_count` on list responses.
- `pkg/client/` reflects the path move and adds a sibling `CategoryClient`.

---

## Locked Decisions

### 1. No migration, no compat shim

There are no live deployments and no external SDK consumers. All current code paths are internal to this repo and unreleased. The DB schema is rebuilt cleanly with the new shape; old paths are removed in the same commit they're replaced. No `v1.1`, no transitional period, no rename-detection migration.

### 2. Reference scheme — ID on write, ID + name embed on read

Tasks reference categories by UUID. Wire format:

**Write side** (POST/PUT body) — only ids:
```json
{
  "title": "Write the report",
  "category_ids": ["1f2c…", "9a0d…"]
}
```

**Read side** (GET response) — both ids and a flat embed:
```json
{
  "id": "…",
  "title": "Write the report",
  "category_ids": ["1f2c…", "9a0d…"],
  "categories": [
    {"id": "1f2c…", "name": "work",   "colour": "#3aa"},
    {"id": "9a0d…", "name": "urgent", "colour": "#c44"}
  ]
}
```

Unknown `category_ids` on write → 400. Renames affect zero rows in the tasks table; the embed picks up the new name on the next GET.

### 3. Category schema

| Field | Type | Notes |
|---|---|---|
| `id` | UUID | server-generated |
| `name` | string | unique case-insensitive; max 64 chars; non-empty |
| `colour` | `*string` (UK spelling) | nullable hex `#RRGGBB`; validation rejects malformed values |
| `created_at` | RFC3339 | server-set |
| `task_count` | int | derived; **response-only**, never accepted on writes |

Rejected fields: `description` (overkill), `updated_at` (renames are rare; not worth the column).

### 4. SDK package layout — sibling clients

`pkg/client/categories.go` adds `CategoryClient` as a sibling to `TaskClient`, matching the existing flat shape (`MessagesClient`, `RulesClient`, etc.). No umbrella `TodoClient` — the URL grouping is a server concern, not a client one.

### 5. Server handler layout

- Existing `internal/server/handler/todo.go` renamed → `internal/server/handler/todo_tasks.go`.
- New `internal/server/handler/todo_categories.go`.
- Route mounting in `internal/server/server.go` groups under a `/api/v1/todo/` prefix block.

### 6. DELETE category cascade

Deleting a category removes its rows from the `todo_categories` join table. Tasks remain unchanged otherwise — they simply lose that category tag. No cascade-to-tasks, no soft-delete.

### 7. Name uniqueness

Case-insensitive uniqueness enforced at the DB level (UNIQUE INDEX on `LOWER(name)`). Create/rename returning a duplicate yields `409 Conflict` with `{"error": "category name already exists"}`.

---

## Schema

### `categories` table

```sql
CREATE TABLE categories (
    id         TEXT PRIMARY KEY,           -- UUID
    name       TEXT NOT NULL,
    colour     TEXT,                        -- nullable hex string '#RRGGBB'
    created_at TEXT NOT NULL                -- RFC3339
);
CREATE UNIQUE INDEX categories_name_lower ON categories (LOWER(name));
```

### `todo_categories` join table

```sql
CREATE TABLE todo_categories (
    todo_id     TEXT NOT NULL REFERENCES todos(id)      ON DELETE CASCADE,
    category_id TEXT NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    PRIMARY KEY (todo_id, category_id)
);
CREATE INDEX todo_categories_category_id ON todo_categories (category_id);
```

The previous string-typed `todo.categories` column is removed entirely — clean break, per Decision 1.

---

## Wire Format

### Category endpoints

| Method | Path | Body | Response |
|---|---|---|---|
| GET | `/api/v1/todo/categories` | — | `[{id, name, colour, created_at, task_count}]` |
| POST | `/api/v1/todo/categories` | `{name, colour?}` | `{id, name, colour, created_at, task_count: 0}` |
| GET | `/api/v1/todo/categories/{id}` | — | `{id, name, colour, created_at, task_count}` |
| PUT | `/api/v1/todo/categories/{id}` | `{name, colour?}` | `{id, name, colour, created_at, task_count}` |
| DELETE | `/api/v1/todo/categories/{id}` | — | `204 No Content` |

### Task endpoints

Paths renamed:

| Old | New |
|---|---|
| `GET /api/v1/tasks` | `GET /api/v1/todo/tasks` |
| `POST /api/v1/tasks` | `POST /api/v1/todo/tasks` |
| `GET /api/v1/tasks/{id}` | `GET /api/v1/todo/tasks/{id}` |
| `PUT /api/v1/tasks/{id}` | `PUT /api/v1/todo/tasks/{id}` |
| `DELETE /api/v1/tasks/{id}` | `DELETE /api/v1/todo/tasks/{id}` |

DTO changes:

- Write requests: `categories: []string` removed; replaced by `category_ids: []uuid`.
- Read responses: gain `category_ids: []uuid` (canonical) and `categories: [{id, name}]` (convenience embed; **no** colour to keep payloads lean — fetch the categories endpoint for full detail).
- Filter query: `?category=<name>` becomes `?category_id=<uuid>`. Multiple ids supported via repeated parameters.

---

## TDD Sequence

RED → GREEN → REFACTOR per loop, three commits each, `just fmt` last step before each commit.

| # | Loop | Scope |
|---|---|---|
| 1 | Repository: `Category` model + repo | `repository.Category` gains `id`, `colour`, `created_at`; `categoryRepo` exposes `Insert`, `Update`, `Delete`, `GetByID`, `QueryAll(WithCounts bool)`. SQLite schema rewritten per §"Schema". |
| 2 | Repository: `todo_categories` join + Todo refactor | `repository.Todo.Categories` field type changes from `[]Category` (name-keyed) to `[]uuid.UUID` (canonical) plus a `CategoriesEmbed []Category` (read-only, populated on Get/Query). Todo repo updated to read/write the join table. |
| 3 | Server: `GET /api/v1/todo/categories` | List handler with `task_count` aggregation. |
| 4 | Server: `POST /api/v1/todo/categories` | Create with case-insensitive uniqueness; 409 on duplicate. |
| 5 | Server: `GET/PUT/DELETE /api/v1/todo/categories/{id}` | Single CRUD; PUT updates name + colour; DELETE cascades through join table only. |
| 6 | Server: route move `/api/v1/tasks` → `/api/v1/todo/tasks` | Path constants + route table updates; old paths fully removed. |
| 7 | Server: Task DTO change | Drop `categories: []string`; add `category_ids: []uuid` on write, embed `categories: [{id, name}]` on read. Filter switches to `?category_id=<uuid>`. Reject unknown ids with 400. |
| 8 | Client SDK: `pkg/client/categories.go` | `CategoryClient` interface + concrete adapter; httptest-driven tests mirroring the existing SDK style. |
| 9 | Client SDK: `pkg/client/tasks.go` path + DTO update | Path constants → `/api/v1/todo/tasks`; `Task` DTO replaces `Categories []string` with `CategoryIDs []uuid` + `Categories []CategoryEmbed`; filter uses `CategoryID uuid.UUID` instead of `Category string`. Test fixtures updated. |
| 10 | OpenAPI / docs regen | Swagger annotations on new handlers; regenerate `docs/api/` per Feature 106A's pipeline; update README API examples. |

10 loops, ~5–6 working days.

---

## Wiring Verification

After loop 10:

1. `grep -rn "/api/v1/tasks" internal/ pkg/ cmd/ docs/` — only matches in CHANGELOG/migration notes; no live code or routes.
2. `grep -rn "Categories \[\]string" pkg/client internal/server/handler` — empty.
3. `grep -rn ErrNotImplemented internal/server pkg/client` (non-test) — empty.
4. `cmd/cue-server` boots cleanly against a fresh SQLite file; `GET /api/v1/todo/categories` returns `[]`; creating a category and a task tagged with its id round-trips correctly.
5. `just test`, `just test-ui`, `just security`, `just vulncheck` all green.

---

## Acceptance Criteria

- Categories are CRUD-able as standalone resources with the schema in Decision 3.
- Tasks reference categories by UUID; renames don't touch the tasks table.
- Task list responses include enough category data (id + name) to render UI without a follow-up call.
- Old `/api/v1/tasks` paths return 404 (no shim).
- `pkg/client/` exposes `CategoryClient` and an updated `TaskClient`.
- OpenAPI documentation regenerated and accurate.

---

## Knock-on Effect on Feature 107

107's adapter layer benefits directly:

- Loops 10 + 11 in 107's plan (server categories endpoint + client SDK) **disappear** — 109 delivers them. 107 shrinks from 16 to 14 loops.
- 107's `cmd/cue/adapters/tasks.go` (loop 12 in current 107 plan) targets the new DTO: maps `Task.CategoryIDs` ↔ `repository.Todo.Categories []uuid.UUID`, and consumes the `categories` embed for UI rendering.
- 107's `CategoryQuerier` adapter is straightforward: wraps `CategoryClient.ListCategories`, translates `client.Category` → `repository.Category`.

107's design doc must be updated to depend on 109. Roadmap row updated to reflect the new dependency.

---

## Risk Areas

1. **Filter parameter rename.** `?category=<name>` → `?category_id=<uuid>` is a breaking query change. Any UAT tests, scripts, or notebooks hitting the old query parameter must be updated. Caught by the existing test suite during loop 7.

2. **Embed payload size.** Tasks with many categories grow proportionally. Realistic ceiling is 5–10 categories per task; payload growth is bounded and acceptable. If a future use case pushes this further, the embed becomes opt-in via a query flag — out of scope for 109.

3. **Colour validation.** Hex format only (`#RRGGBB`). Rejecting malformed values at create/update time is cheap; doing it in the repo would catch drift but is overkill for now. Validation lives in the handler layer.

4. **Case-insensitive uniqueness on rename.** Renaming "Work" → "WORK" must succeed (same row). `LOWER(name)` index treats them as the same key; the UPDATE statement must scope the uniqueness check to "any row other than this one." Standard pattern but worth a regression test in loop 5.

5. **Nothing else.** With no migration and no live consumers, the failure modes are bounded to the test suite.

---

## Estimate

- New code: ~600 LOC (handler + repo + tests for categories) + ~150 LOC SDK + ~80 LOC docs annotations.
- Removed/changed: ~200 LOC of string-categories handling across handler, repo, and tests.
- Net: ~+650 LOC, mostly tests and DTO translation.
- Loops: 10, ~5–6 working days.
