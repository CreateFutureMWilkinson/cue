# Feature 109: Todo Domain Restructure

**Phase:** Phase-9-Feature-109
**Status:** Done
**Depends on:** Feature 101A (Todo CRUD API), Feature 102 (Service Configuration API), Feature 106 (API Client SDK)
**Blocks:** Feature 107 (Fyne Client Re-wire)
**Packages:** `internal/repository/`, `internal/repository/implementation/sqlite/`, `internal/service/todo/`, `internal/server/handler/`, `internal/server/`, `pkg/client/`

---

## Overview

Restructure the todo domain so the Fyne client (Feature 107) lands on a stable, query-able shape:

- Promote categories to first-class resources with their own CRUD.
- Each task carries **at most one** category (single FK column, no join table).
- Categories are **name-keyed** with a normalized lowercase form — no UUIDs.
- Rename the internal `Todo` type → `Task` everywhere; the bounded-context package stays `internal/service/todo/` and becomes the single owner of both tasks and categories.
- Group tasks + categories under a `/api/v1/todo/` URL prefix.

There are no live deployments and no external SDK consumers. The schema is rebuilt cleanly with the new shape; old paths are removed in the same commit they're replaced. No migration shim, no v1.1 transition.

---

## Locked Decisions

### 1. No migration, no compat shim

All changes ship in a clean schema. Old `/api/v1/tasks` paths return 404 once 109 is merged. The `categories` column on the legacy `todos` table is removed entirely; the table is renamed to `tasks` with a new `category_key` FK column.

### 2. One category per task

Tasks reference at most one category via a nullable FK. No join table.

```sql
ALTER TABLE tasks ADD COLUMN category_key TEXT NULL
    REFERENCES categories(name_key)
    ON UPDATE CASCADE
    ON DELETE SET NULL;
```

`ON DELETE SET NULL`: deleting a category leaves the task in place, uncategorized.
`ON UPDATE CASCADE`: renaming a category propagates the new key to all tagged tasks in one statement.

### 3. Categories are name-keyed (no UUID)

Primary key is the **normalized name**. Lookups, FKs, and routes all use the key. Presentation form is **derived programmatically** from the key — no `display_name` column.

#### Normalization (`NormalizeCategoryKey`)

Input → key:

1. Trim leading/trailing whitespace.
2. Reject if input contains an underscore (`_`). Error: `"underscores not allowed — use spaces"`.
3. Reject if empty after trim, longer than 64 chars, or contains anything other than ASCII letters, digits, or whitespace.
4. Lowercase.
5. Collapse runs of whitespace, replace each space run with a single `_`.

| Input | Key |
|---|---|
| `FOOBAR` | `foobar` |
| `foo bar` | `foo_bar` |
| `foo BAR` | `foo_bar` |
| `  Foo   Bar  ` | `foo_bar` |
| `foo_bar` | rejected (contains `_`) |
| `foo!` | rejected (non-alphanumeric, non-space) |
| `""` | rejected (empty) |
| 65-char string | rejected (too long) |

#### Presentation (`PresentCategoryName`)

Key → display: replace `_` with space, title-case each word.

| Key | Display |
|---|---|
| `foobar` | `Foobar` |
| `foo_bar` | `Foo Bar` |
| `api_docs` | `Api Docs` |

Title-casing is mechanical — acronym info is lost, accepted.

Both functions live in `internal/repository/category.go` and are pure / table-tested.

### 4. Schema

```sql
CREATE TABLE categories (
    name_key   TEXT PRIMARY KEY,   -- lowercase, _ for spaces
    colour     TEXT,                -- nullable hex '#RRGGBB'
    created_at TEXT NOT NULL        -- RFC3339
);

CREATE TABLE tasks (
    -- existing todo columns, renamed table only
    id                   TEXT PRIMARY KEY,
    title                TEXT NOT NULL,
    description          TEXT NOT NULL DEFAULT '',
    priority             INTEGER NOT NULL DEFAULT 0,
    due_date             TEXT,
    estimate_minutes     INTEGER,
    llm_estimate_minutes INTEGER,
    created_at           TEXT NOT NULL,
    completed_at         TEXT,
    -- new
    category_key         TEXT REFERENCES categories(name_key)
                              ON UPDATE CASCADE
                              ON DELETE SET NULL
);
CREATE INDEX tasks_category_key ON tasks (category_key);
```

The legacy string-typed `todos.categories` column is removed entirely.

### 5. URL grouping — `/api/v1/todo/`

Tasks and categories share the `/api/v1/todo/` prefix. The grouping reflects the bounded context (`internal/service/todo/`), even though resources internally type as `Task` and `Category`.

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/todo/categories` | List with `task_count` |
| `POST` | `/api/v1/todo/categories` | Create from raw input |
| `GET` | `/api/v1/todo/categories/{name}` | Lookup by any form |
| `PUT` | `/api/v1/todo/categories/{name}` | Update display name and/or colour |
| `DELETE` | `/api/v1/todo/categories/{name}` | Remove (cascades SET NULL on tasks) |
| `GET` | `/api/v1/todo/tasks` | List, optional `?category=` filter |
| `POST` | `/api/v1/todo/tasks` | Create |
| `GET` | `/api/v1/todo/tasks/{id}` | Lookup |
| `PUT` | `/api/v1/todo/tasks/{id}` | Update |
| `DELETE` | `/api/v1/todo/tasks/{id}` | Remove |

`{name}` accepts any case/spacing form; the handler normalizes before lookup.

### 6. Rename `Todo` → `Task`

The bounded-context package keeps the name `internal/service/todo/`. Resources within it name themselves: `Task`, `Category`. So the call sites read `todo.Service.CreateTask(...)`, `todo.Service.RenameCategory(...)`.

| Old | New |
|---|---|
| `repository.Todo` | `repository.Task` |
| `repository.TodoRepository` | `repository.TaskRepository` |
| `repository.TodoFilter` | `repository.TaskFilter` |
| `internal/repository/todo.go` | `internal/repository/task.go` |
| `internal/repository/implementation/sqlite/todo_impl.go` | `.../sqlite/task_impl.go` |
| `internal/server/handler/todo.go` | `internal/server/handler/tasks.go` |
| `todos` table | `tasks` table |

`internal/service/todo/` package name is preserved.

### 7. `todo.Service` owns categories

The existing `Service` is extended to inject a `CategoryRepository` alongside the task repo, and to expose category operations. Handlers stay thin; the service is the **single place** that turns raw input through `NormalizeCategoryKey` before hitting the repo. Repos deal only in canonical keys.

```go
type Service struct {
    tasks      TaskRepository
    categories CategoryRepository
    estimator  TimeEstimator
}

func NewService(tasks TaskRepository, categories CategoryRepository, estimator TimeEstimator) (*Service, error)

// Categories
func (s *Service) CreateCategory(ctx context.Context, rawName string, colour *string) (*repository.Category, error)
func (s *Service) RenameCategory(ctx context.Context, oldKey, newRawName string) (*repository.Category, error)
func (s *Service) SetCategoryColour(ctx context.Context, key string, colour *string) error
func (s *Service) DeleteCategory(ctx context.Context, key string) error
func (s *Service) GetCategory(ctx context.Context, rawNameOrKey string) (*repository.Category, error)
func (s *Service) ListCategories(ctx context.Context, withCounts bool) ([]*repository.CategoryWithCount, error)

// Tasks (existing CRUD; signatures retitled Todo→Task)
```

`GetCategory` accepts any form — it normalizes before lookup.

### 8. Wire format

#### Categories

| Direction | Field | Example |
|---|---|---|
| Write (POST/PUT body) | `name`, `colour?` | `{"name": "foo BAR", "colour": "#3aa"}` |
| Read (single) | `{key, name, colour, created_at, task_count}` | `{"key":"foo_bar","name":"Foo Bar","colour":"#3aa","created_at":"...","task_count":4}` |
| Read (list) | array of single | — |
| Path param | accepts any form | `GET /api/v1/todo/categories/Foo%20Bar` ≡ `.../foo_bar` |

#### Tasks

| Direction | Field | Example |
|---|---|---|
| Write | `category` (string or null) | `{"title":"x","category":"foo BAR"}` |
| Read | `category` (object or null) | `{"id":"…","category":{"key":"foo_bar","name":"Foo Bar"}}` |
| Filter | `?category=` (any form) | `?category=Foo%20Bar` ≡ `?category=foo_bar` |

`category` on read embeds `{key, name}` only — no colour, no `task_count`. Clients fetch `/api/v1/todo/categories/{key}` for full detail. Unknown category on write → `400`.

### 9. Validation

- **Category name** — per `NormalizeCategoryKey` rules (Decision 3).
- **Colour** — nullable; if present, must match `^#[0-9A-Fa-f]{6}$`. Validation in handler before service.
- **Duplicate key on create** — `409 Conflict` (`"category key already exists"`). Translated from sqlite `UNIQUE` violation in the repo layer.
- **Rename to existing key** — `409 Conflict`.
- **Rename with same key** (case-only diff is impossible by design — keys are already lowercase) — no-op when input normalizes to the same existing key; PUT returns the existing row unchanged.

### 10. Repository interfaces

```go
// internal/repository/category.go
type Category struct {
    NameKey   string  // PK; lowercase, _ for spaces
    Colour    *string // nullable hex '#RRGGBB'
    CreatedAt time.Time
}

type CategoryWithCount struct {
    Category
    TaskCount int
}

type CategoryRepository interface {
    Insert(ctx context.Context, c *Category) error
    Rename(ctx context.Context, oldKey, newKey string) error
    UpdateColour(ctx context.Context, key string, colour *string) error
    Delete(ctx context.Context, key string) error
    GetByKey(ctx context.Context, key string) (*Category, error)
    QueryAll(ctx context.Context, withCounts bool) ([]*CategoryWithCount, error)
}

// Pure normalization helpers
func NormalizeCategoryKey(input string) (string, error)
func PresentCategoryName(key string) string
```

```go
// internal/repository/task.go (renamed from todo.go)
type Task struct {
    ID                 uuid.UUID
    Title              string
    Description        string
    Priority           int
    DueDate            *time.Time
    CategoryKey        *string  // FK to categories.name_key; nullable
    EstimateMinutes    *int
    LLMEstimateMinutes *int
    CreatedAt          time.Time
    CompletedAt        *time.Time
}

type TaskFilter struct {
    Status      string
    CategoryKey string  // empty = no filter
    Search      string
    Limit       int
    Offset      int
}

type TaskRepository interface {
    Insert(ctx context.Context, t *Task) error
    Update(ctx context.Context, t *Task) error
    Delete(ctx context.Context, id uuid.UUID) error
    QueryByID(ctx context.Context, id uuid.UUID) (*Task, error)
    QueryFiltered(ctx context.Context, filter TaskFilter) ([]*Task, int, error)
    Complete(ctx context.Context, id uuid.UUID, completedAt time.Time) error
}
```

---

## TDD Loop Plan

Per `CLAUDE.md` §13: each loop = RED (test-designer) → GREEN (implementer) → REFACTOR (refactorer), three commits, `just fmt` last before each commit. Agent teams used throughout.

| # | Loop | Scope |
|---|---|---|
| 1 | Category model + normalization | `repository.Category` reshape; `NormalizeCategoryKey` + `PresentCategoryName` pure functions with table tests covering all rules from Decision 3. |
| 2 | SQLite categories repo | New `categories` table; `CategoryRepository` impl with `Insert/Rename/UpdateColour/Delete/GetByKey/QueryAll`. UNIQUE-violation translation to `repository.ErrDuplicate`. Tests use `s.T().TempDir()`. |
| 3 | Rename Todo → Task | Sweep across repo, sqlite, service, handler, tests. Table `todos` → `tasks`. No behavioural change in this loop. |
| 4 | Task `category_key` FK | Add `Task.CategoryKey *string`; SQLite column with FK + cascade rules; repo Insert/Update read/write the column; `TaskFilter.CategoryKey` filters via `WHERE category_key = ?`. |
| 5 | `todo.Service` category methods | Inject `CategoryRepository`; implement `CreateCategory/RenameCategory/SetCategoryColour/DeleteCategory/GetCategory/ListCategories`. Service is the only place that calls `NormalizeCategoryKey`. |
| 6 | Categories HTTP handler | New `internal/server/handler/categories.go` mounted at `/api/v1/todo/categories`. Validates colour, surfaces 409 on duplicate, 404 on unknown key, 400 on bad input. |
| 7 | Tasks routes + DTO update | Move `/api/v1/tasks` → `/api/v1/todo/tasks`. DTO: write accepts `category: string\|null`, server normalizes; read returns `category: {key,name}\|null`. Filter `?category=` accepts any form. Reject unknown category on write with 400. Old paths fully removed. |
| 8 | Client SDK | New `pkg/client/categories.go` with `CategoryClient`. Update `pkg/client/tasks.go`: paths → `/api/v1/todo/tasks`, `Task` DTO uses `Category *CategoryEmbed` and a `CategoryInput string` for writes; filter uses `CategoryKey string`. httptest-driven tests. |
| 9 | OpenAPI regen + docs | Swagger annotations on new handlers; regen `docs/api/` per Feature 106A pipeline; update CHANGELOG (Breaking), README, agent-log, Roadmap row → Done; update 107 doc to reflect dependency satisfaction. |

9 loops total, ~5 working days.

---

## Wiring Verification

After loop 9, before security checks:

1. `grep -rn "/api/v1/tasks" internal/ pkg/ cmd/` — only matches in CHANGELOG/migration notes.
2. `grep -rn "Categories \[\]string\|Categories \[\]Category" pkg/client internal/server/handler internal/repository` — empty.
3. `grep -rn "ErrNotImplemented" internal/server pkg/client internal/repository` (non-test) — empty.
4. `grep -rn "QueryByName" internal/` — empty.
5. `grep -rn "TodoRepository\|TodoFilter\|repository\.Todo\b" internal/ pkg/ cmd/` — empty (only the package name `service/todo/` remains).
6. `cmd/cue-server` boots against fresh SQLite: `GET /api/v1/todo/categories` → `[]`; `POST /api/v1/todo/categories` with `{"name":"foo BAR"}` → `{key:"foo_bar",name:"Foo Bar",...}`; `POST /api/v1/todo/tasks` with `{"category":"foo bar"}` round-trips correctly.
7. `just test && just test-ui && just security && just vulncheck` all green.

---

## Acceptance Criteria

- Categories CRUD-able as standalone resources with the schema in Decision 4.
- Each task references at most one category via a nullable FK; deleting a category sets dependent task FKs to NULL.
- Category renames cascade through `tasks.category_key` via `ON UPDATE CASCADE`.
- Wire format follows Decision 8: write accepts raw input, read returns the derived display name.
- All routes live under `/api/v1/todo/`; old `/api/v1/tasks` paths return 404.
- `pkg/client/` exposes `CategoryClient` and an updated `TaskClient`.
- OpenAPI documentation regenerated and accurate.
- All existing tests pass (after rename adjustments) and new tests achieve ≥80% coverage on new code.

---

## Knock-on Effect on Feature 107

107's adapter layer benefits directly:

- 107's loops adding server categories endpoint + client SDK **disappear** — 109 delivers them.
- 107's `cmd/cue/adapters/tasks.go` consumes `client.Task.Category` (single `*CategoryEmbed`) for UI rendering — simpler than the planned multi-category embed.
- 107's `CategoryQuerier` adapter wraps `CategoryClient.ListCategories`.

107's design doc must be updated to depend on the shipped 109 contract during loop 9.

---

## Risk Areas

1. **Rename sweep (Loop 3).** Must touch many files in lockstep. Mitigation: dedicated loop, no behavioural change, full `just test` between RED and GREEN.
2. **Path-param normalization.** `GET /api/v1/todo/categories/Foo%20Bar` must decode + normalize. Tested in Loop 6 RED with mixed-case + URL-encoded variants.
3. **Cascade semantics.** `ON DELETE SET NULL` and `ON UPDATE CASCADE` rely on `PRAGMA foreign_keys = ON`. The sqlite implementation already enables foreign keys per existing repo init; verified in Loop 2 RED.
4. **Underscore rejection.** Users typing `foo_bar` see 400. UI clarifies via the error message; `pkg/client` surfaces it as a typed validation error.

---

## Estimate

- New code: ~500 LOC (handler + repo + service for categories) + ~150 LOC SDK + ~80 LOC docs annotations.
- Removed/changed: ~250 LOC of string-categories handling + Todo→Task rename diff.
- Net: ~+500 LOC, mostly tests and DTO translation.
- Loops: 9, ~5 working days.

---

## TDD Agent Stats

| Loop | Phase    | Agent           | Commit    |
|------|----------|-----------------|-----------|
| 1    | Red      | test-designer   | `d816710` |
| 1    | Green    | implementer     | `8fb477a` |
| 1    | Refactor | refactorer      | `317d64a` |
| 2    | Red      | test-designer   | `3adc648` |
| 2    | Green    | implementer     | `8b4f3a1` |
| 2    | Refactor | refactorer      | `0774a80` |
| 3    | Combined | implementer     | `0e671ec` |
| 4    | Red      | test-designer   | `1f515e5` |
| 4    | Green    | implementer     | `3b9317e` |
| 4    | Refactor | refactorer      | (no-op)   |
| 5    | Red      | test-designer   | `60df1b5` |
| 5    | Green    | implementer     | `8611caa` |
| 5    | Refactor | refactorer      | `bafd79c` |
| 6    | Red      | test-designer   | `01ba32c` |
| 6    | Green    | implementer     | `f3210cc` |
| 6    | Refactor | refactorer      | `22a92f9` |
| 7    | Red      | test-designer   | `bd593b1` |
| 7    | Green    | implementer     | `42a1b8e` |
| 7    | Refactor | refactorer      | (no-op)   |
| 8    | Red      | test-designer   | `b279f79` |
| 8    | Green    | implementer     | `6af61c1` |
| 8    | Fix      | direct          | `5cda984` |
| 8    | Refactor | refactorer      | (no-op)   |
| —    | Wiring   | direct          | `96ddab1` |
