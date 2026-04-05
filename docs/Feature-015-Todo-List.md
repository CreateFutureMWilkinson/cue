# Feature 015: Todo List

**Phase:** Phase-2-Feature-015
**Status:** Planned
**Packages:** `internal/repository/`, `internal/repository/implementation/sqlite/`

---

## Overview

Standalone task manager providing CRUD operations for todos with user-defined categories. Todos are the canonical task source for the day planner (Feature 017) and can also be created directly during the planning flow. Categories support name and color, with autocomplete from previously used values. All data persisted in SQLite using the established repository pattern.

## Design Decisions

- **Separate tables for todos, categories, and a junction table** (`todo_categories`) — allows many-to-many relationships and independent category management without denormalization.
- **Categories are user-defined, not config-driven** — stored in SQLite with name + hex color. The UI provides autocomplete from existing categories but accepts freeform input.
- **TodoRepository and CategoryRepository as separate interfaces** — follows existing codebase convention of narrow, consumer-focused interfaces. The todo repo handles todos; category repo handles category CRUD and lookup.
- **Priority is an integer, not an enum** — allows flexible ordering. Lower number = higher priority. The planner uses priority to order tasks in the schedule.
- **DueDate is optional** — not all tasks have deadlines. When present, the planner can use it to influence scheduling urgency.
- **Description is markdown** — stored as raw text, rendered by the UI layer. No processing at the repository level.
- **CompletedAt doubles as status** — nil means incomplete, non-nil means done. No separate status field needed for MVP.
- **SQLite implementation reuses existing patterns** — WAL mode, `CREATE TABLE IF NOT EXISTS`, nullable helpers, error wrapping with context.

## Data Model

### Todo

```go
type Todo struct {
    ID          uuid.UUID
    Title       string
    Description string       // markdown
    Priority    int          // lower = higher priority
    DueDate     *time.Time   // optional
    Categories  []Category
    CreatedAt   time.Time
    CompletedAt *time.Time   // nil = incomplete
}
```

### Category

```go
type Category struct {
    ID    uuid.UUID
    Name  string   // unique, user-defined
    Color string   // hex color, e.g. "#FF5733"
}
```

### SQLite Schema

```sql
CREATE TABLE IF NOT EXISTS todos (
    id          TEXT PRIMARY KEY,
    title       TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    priority    INTEGER NOT NULL DEFAULT 0,
    due_date    TIMESTAMP,
    created_at  TIMESTAMP NOT NULL,
    completed_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS categories (
    id    TEXT PRIMARY KEY,
    name  TEXT NOT NULL UNIQUE,
    color TEXT NOT NULL DEFAULT '#808080'
);

CREATE TABLE IF NOT EXISTS todo_categories (
    todo_id     TEXT NOT NULL REFERENCES todos(id) ON DELETE CASCADE,
    category_id TEXT NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    PRIMARY KEY (todo_id, category_id)
);

CREATE INDEX IF NOT EXISTS idx_todos_completed ON todos(completed_at);
CREATE INDEX IF NOT EXISTS idx_todos_priority ON todos(priority);
CREATE INDEX IF NOT EXISTS idx_todos_due_date ON todos(due_date);
```

## API

### TodoRepository Interface

```go
type TodoRepository interface {
    Insert(ctx context.Context, todo *Todo) error
    Update(ctx context.Context, todo *Todo) error
    Delete(ctx context.Context, id uuid.UUID) error
    QueryByID(ctx context.Context, id uuid.UUID) (*Todo, error)
    QueryIncomplete(ctx context.Context) ([]*Todo, error)
    QueryAll(ctx context.Context) ([]*Todo, error)
    Complete(ctx context.Context, id uuid.UUID, completedAt time.Time) error
}
```

### CategoryRepository Interface

```go
type CategoryRepository interface {
    Insert(ctx context.Context, category *Category) error
    Update(ctx context.Context, category *Category) error
    Delete(ctx context.Context, id uuid.UUID) error
    QueryAll(ctx context.Context) ([]*Category, error)
    QueryByName(ctx context.Context, name string) (*Category, error)
}
```

### Constructors

```go
func NewSQLiteTodoRepository(dbPath string) (*SQLiteTodoRepository, error)
func NewSQLiteCategoryRepository(dbPath string) (*SQLiteCategoryRepository, error)
```

## Error Handling

| Scenario | Behavior |
|---|---|
| Todo not found by ID | Return wrapped `ErrNotFound` |
| Duplicate category name | Return wrapped unique constraint error |
| Category deletion with active references | Junction table rows cascade-deleted |
| Todo deletion | Junction table rows cascade-deleted |
| Invalid UUID format | Parse error from `uuid.Parse` |
| Database locked (WAL) | Retry with backoff, log |
| Nil required fields (Title) | Validation error before insert |

## Integration Points

- **Day Planner (Feature 017):** Reads incomplete todos as task candidates for schedule generation. Writes new todos created during planning flow.
- **Planner UI (Feature 018):** TodoPresenter queries todos for display, supports create/edit/complete/delete actions.
- **Config:** No config section needed for MVP — todo storage uses the same database path as messages (`database.path`).

## Test Coverage Plan

| Package | Suite | Expected Tests |
|---|---|---|
| `sqlite` | `TodoRepositorySuite` | Insert, update, delete, query by ID, query incomplete, query all, complete, not found, categories association |
| `sqlite` | `CategoryRepositorySuite` | Insert, update, delete, query all, query by name, duplicate name, cascade delete |

## TDD Agent Stats

| TDD Cycle | Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|---|
| Todo Repo | RED | Test Designer | — | — | — |
| Todo Repo | GREEN | Implementer | — | — | — |
| Todo Repo | REFACTOR | Refactorer | — | — | — |
| Category Repo | RED | Test Designer | — | — | — |
| Category Repo | GREEN | Implementer | — | — | — |
| Category Repo | REFACTOR | Refactorer | — | — | — |
