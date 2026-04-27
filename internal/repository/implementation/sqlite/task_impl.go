package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"

	_ "modernc.org/sqlite"
)

// createTaskTables creates the tasks table with a single nullable
// category_key FK referencing categories(name_key). The legacy
// task_categories junction table is gone — Feature 109 Decision 2
// promotes the relationship to a single FK column with cascade rules.
const createTaskTables = `
CREATE TABLE IF NOT EXISTS tasks (
    id                   TEXT PRIMARY KEY,
    title                TEXT NOT NULL,
    description          TEXT NOT NULL DEFAULT '',
    priority             INTEGER NOT NULL DEFAULT 0,
    due_date             TIMESTAMP,
    created_at           TIMESTAMP NOT NULL,
    completed_at         TIMESTAMP,
    estimate_minutes     INTEGER,
    llm_estimate_minutes INTEGER,
    category_key         TEXT REFERENCES categories(name_key)
                              ON UPDATE CASCADE
                              ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_tasks_completed ON tasks(completed_at);
CREATE INDEX IF NOT EXISTS idx_tasks_priority ON tasks(priority);
CREATE INDEX IF NOT EXISTS idx_tasks_due_date ON tasks(due_date);
CREATE INDEX IF NOT EXISTS tasks_category_key ON tasks(category_key);
`

const (
	taskColumnsStr      = "id, title, description, priority, due_date, created_at, completed_at, estimate_minutes, llm_estimate_minutes, category_key"
	queryInsertTask     = "INSERT INTO tasks (id, title, description, priority, due_date, created_at, completed_at, estimate_minutes, llm_estimate_minutes, category_key) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
	queryUpdateTask     = "UPDATE tasks SET title = ?, description = ?, priority = ?, due_date = ?, completed_at = ?, estimate_minutes = ?, llm_estimate_minutes = ?, category_key = ? WHERE id = ?"
	queryDeleteTask     = "DELETE FROM tasks WHERE id = ?"
	querySelectTaskByID = "SELECT " + taskColumnsStr + " FROM tasks WHERE id = ?"
	queryCompleteTask   = "UPDATE tasks SET completed_at = ? WHERE id = ?"

	defaultTaskQueryLimit = 50
)

// SQLiteTaskRepository implements repository.TaskRepository using SQLite.
type SQLiteTaskRepository struct {
	db *sql.DB
}

// NewSQLiteTaskRepository opens a SQLite database at dbPath, enables WAL mode
// and foreign keys, creates the tasks table with its category_key FK, and
// returns a ready repository.
func NewSQLiteTaskRepository(dbPath string) (*SQLiteTaskRepository, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open task database: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable WAL mode: %w", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	if _, err := db.Exec(createTaskTables); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create task tables: %w", err)
	}

	return &SQLiteTaskRepository{db: db}, nil
}

// nullableCategoryKey converts a *string to a value suitable for ?-binding
// where nil is persisted as SQL NULL.
//
// TODO(feat-109 Loop 4 GREEN): wire this through Insert/Update so
// category_key actually round-trips. Currently a placeholder.
func nullableCategoryKey(k *string) any {
	if k == nil {
		return nil
	}
	return *k
}

// Insert adds a new task to the database.
//
// TODO(feat-109 Loop 4 GREEN): persist task.CategoryKey via the new
// category_key column. The current implementation always writes NULL
// regardless of the value of CategoryKey so new tests fail meaningfully.
func (r *SQLiteTaskRepository) Insert(ctx context.Context, task *repository.Task) error {
	_ = nullableCategoryKey // referenced once GREEN wires it.

	_, err := r.db.ExecContext(ctx, queryInsertTask,
		task.ID.String(),
		task.Title,
		task.Description,
		task.Priority,
		nullableTime(task.DueDate),
		task.CreatedAt.Format(time.RFC3339),
		nullableTime(task.CompletedAt),
		task.EstimateMinutes,
		task.LLMEstimateMinutes,
		nil, // STUB: should be nullableCategoryKey(task.CategoryKey)
	)
	if err != nil {
		return fmt.Errorf("insert task: %w", err)
	}
	return nil
}

// Update modifies an existing task's fields.
//
// TODO(feat-109 Loop 4 GREEN): persist task.CategoryKey via the new
// category_key column. Current STUB always writes NULL.
func (r *SQLiteTaskRepository) Update(ctx context.Context, task *repository.Task) error {
	_, err := r.db.ExecContext(ctx, queryUpdateTask,
		task.Title,
		task.Description,
		task.Priority,
		nullableTime(task.DueDate),
		nullableTime(task.CompletedAt),
		task.EstimateMinutes,
		task.LLMEstimateMinutes,
		nil, // STUB: should be nullableCategoryKey(task.CategoryKey)
		task.ID.String(),
	)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	return nil
}

// Delete removes a task by ID.
func (r *SQLiteTaskRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, queryDeleteTask, id.String())
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	return nil
}

// QueryByID returns a task by ID, or nil and an error wrapping
// repository.ErrNotFound if no task with that ID exists.
//
// TODO(feat-109 Loop 4 GREEN): scan the new category_key column into
// task.CategoryKey. Current STUB scans the column but discards it.
func (r *SQLiteTaskRepository) QueryByID(ctx context.Context, id uuid.UUID) (*repository.Task, error) {
	rows, err := r.db.QueryContext(ctx, querySelectTaskByID, id.String())
	if err != nil {
		return nil, fmt.Errorf("query task by id: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("query task by id: %w", err)
		}
		return nil, fmt.Errorf("query task by id: %w", repository.ErrNotFound)
	}

	task, err := scanTask(rows)
	if err != nil {
		return nil, fmt.Errorf("scan task: %w", err)
	}

	return task, nil
}

// QueryFiltered returns tasks matching the filter criteria plus a total count for pagination.
// Sort order: priority DESC (higher first), then created_at ASC.
//
// TODO(feat-109 Loop 4 GREEN): apply filter.CategoryKey via
// `WHERE category_key = ?`. Current STUB ignores CategoryKey.
func (r *SQLiteTaskRepository) QueryFiltered(ctx context.Context, filter repository.TaskFilter) ([]*repository.Task, int, error) {
	// Build WHERE clause dynamically.
	var whereClauses []string
	var args []any

	// Status filter: default to "incomplete".
	status := filter.Status
	if status == "" {
		status = "incomplete"
	}
	switch status {
	case "incomplete":
		whereClauses = append(whereClauses, "completed_at IS NULL")
	case "complete":
		whereClauses = append(whereClauses, "completed_at IS NOT NULL")
	case "all":
		// No status filter.
	}

	// STUB: filter.CategoryKey ignored — Loop 4 GREEN wires this.
	_ = filter.CategoryKey

	// Search filter: case-insensitive LIKE on title OR description.
	if filter.Search != "" {
		whereClauses = append(whereClauses, "(title LIKE ? OR description LIKE ?)")
		searchPattern := "%" + filter.Search + "%"
		args = append(args, searchPattern, searchPattern)
	}

	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	countQuery := "SELECT COUNT(*) FROM tasks" + whereClause
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count filtered tasks: %w", err)
	}

	limit := filter.Limit
	if limit == 0 {
		limit = defaultTaskQueryLimit
	}

	dataQuery := "SELECT " + taskColumnsStr + " FROM tasks" + whereClause +
		" ORDER BY priority DESC, created_at ASC LIMIT ? OFFSET ?"

	dataArgs := append(args, limit, filter.Offset)

	rows, err := r.db.QueryContext(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("query filtered tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*repository.Task
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan filtered task: %w", err)
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate filtered tasks: %w", err)
	}

	return tasks, total, nil
}

// Complete sets the completed_at timestamp on a task.
func (r *SQLiteTaskRepository) Complete(ctx context.Context, id uuid.UUID, completedAt time.Time) error {
	_, err := r.db.ExecContext(ctx, queryCompleteTask, completedAt.Format(time.RFC3339), id.String())
	if err != nil {
		return fmt.Errorf("complete task: %w", err)
	}
	return nil
}

// scanTask reads a task from a sql.Rows scanner.
//
// TODO(feat-109 Loop 4 GREEN): populate task.CategoryKey from the
// scanned category_key column. Current STUB scans into a local
// variable that is discarded.
func scanTask(rows *sql.Rows) (*repository.Task, error) {
	var (
		task               repository.Task
		idStr              string
		createdAtStr       string
		dueDate            sql.NullString
		completedAt        sql.NullString
		estimateMinutes    sql.NullInt64
		llmEstimateMinutes sql.NullInt64
		categoryKey        sql.NullString // STUB: not wired to task.CategoryKey
	)

	err := rows.Scan(
		&idStr,
		&task.Title,
		&task.Description,
		&task.Priority,
		&dueDate,
		&createdAtStr,
		&completedAt,
		&estimateMinutes,
		&llmEstimateMinutes,
		&categoryKey,
	)
	if err != nil {
		return nil, fmt.Errorf("scan task row: %w", err)
	}

	task.ID, err = uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("parse task ID: %w", err)
	}

	task.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}

	if dueDate.Valid {
		t, err := time.Parse(time.RFC3339, dueDate.String)
		if err != nil {
			return nil, fmt.Errorf("parse due_date: %w", err)
		}
		task.DueDate = &t
	}

	if completedAt.Valid {
		t, err := time.Parse(time.RFC3339, completedAt.String)
		if err != nil {
			return nil, fmt.Errorf("parse completed_at: %w", err)
		}
		task.CompletedAt = &t
	}

	if estimateMinutes.Valid {
		v := int(estimateMinutes.Int64)
		task.EstimateMinutes = &v
	}

	if llmEstimateMinutes.Valid {
		v := int(llmEstimateMinutes.Int64)
		task.LLMEstimateMinutes = &v
	}

	// STUB: categoryKey scanned but not assigned to task.CategoryKey.

	return &task, nil
}
