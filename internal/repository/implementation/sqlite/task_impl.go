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

const createTaskTables = `
CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    priority INTEGER NOT NULL DEFAULT 0,
    due_date TIMESTAMP,
    created_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    estimate_minutes INTEGER,
    llm_estimate_minutes INTEGER
);
CREATE TABLE IF NOT EXISTS task_categories (
    task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    category_id TEXT NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    PRIMARY KEY (task_id, category_id)
);
CREATE INDEX IF NOT EXISTS idx_tasks_completed ON tasks(completed_at);
CREATE INDEX IF NOT EXISTS idx_tasks_priority ON tasks(priority);
CREATE INDEX IF NOT EXISTS idx_tasks_due_date ON tasks(due_date);
`

const (
	taskColumnsStr            = "id, title, description, priority, due_date, created_at, completed_at, estimate_minutes, llm_estimate_minutes"
	queryInsertTask           = "INSERT INTO tasks (id, title, description, priority, due_date, created_at, completed_at, estimate_minutes, llm_estimate_minutes) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)"
	queryUpdateTask           = "UPDATE tasks SET title = ?, description = ?, priority = ?, due_date = ?, completed_at = ?, estimate_minutes = ?, llm_estimate_minutes = ? WHERE id = ?"
	queryDeleteTask           = "DELETE FROM tasks WHERE id = ?"
	querySelectTaskByID       = "SELECT " + taskColumnsStr + " FROM tasks WHERE id = ?"
	queryCompleteTask         = "UPDATE tasks SET completed_at = ? WHERE id = ?"
	queryInsertTaskCategory   = "INSERT INTO task_categories (task_id, category_id) VALUES (?, ?)"
	queryDeleteTaskCategories = "DELETE FROM task_categories WHERE task_id = ?"
	querySelectTaskCategories = "SELECT c.id, c.name, c.color FROM categories c INNER JOIN task_categories tc ON c.id = tc.category_id WHERE tc.task_id = ?"

	defaultTaskQueryLimit = 50
)

// SQLiteTaskRepository implements repository.TaskRepository using SQLite.
type SQLiteTaskRepository struct {
	db *sql.DB
}

// NewSQLiteTaskRepository opens a SQLite database at dbPath, enables WAL mode
// and foreign keys, creates the tasks and task_categories tables, and returns a ready repository.
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

// Insert adds a new task and its category associations to the database.
func (r *SQLiteTaskRepository) Insert(ctx context.Context, task *repository.Task) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, queryInsertTask,
		task.ID.String(),
		task.Title,
		task.Description,
		task.Priority,
		nullableTime(task.DueDate),
		task.CreatedAt.Format(time.RFC3339),
		nullableTime(task.CompletedAt),
		task.EstimateMinutes,
		task.LLMEstimateMinutes,
	)
	if err != nil {
		return fmt.Errorf("insert task: %w", err)
	}

	// TODO(feat-109 Loop 4): rewrite category persistence against the new
	// category_key FK column on tasks. The old task_categories junction
	// table goes away in Loop 4; for now, skip association writes so the
	// package compiles after the Category struct reshape.
	_ = queryInsertTaskCategory
	for range task.Categories {
	}

	return tx.Commit()
}

// Update modifies an existing task's fields and replaces its category associations.
func (r *SQLiteTaskRepository) Update(ctx context.Context, task *repository.Task) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, queryUpdateTask,
		task.Title,
		task.Description,
		task.Priority,
		nullableTime(task.DueDate),
		nullableTime(task.CompletedAt),
		task.EstimateMinutes,
		task.LLMEstimateMinutes,
		task.ID.String(),
	)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	_, err = tx.ExecContext(ctx, queryDeleteTaskCategories, task.ID.String())
	if err != nil {
		return fmt.Errorf("delete task categories: %w", err)
	}

	// TODO(feat-109 Loop 4): rewrite category persistence against the new
	// category_key FK column on tasks. See Insert above.
	for range task.Categories {
	}

	return tx.Commit()
}

// Delete removes a task by ID. Cascade handles junction table cleanup.
func (r *SQLiteTaskRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, queryDeleteTask, id.String())
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	return nil
}

// QueryByID returns a task by ID with populated categories, or nil and an error
// wrapping repository.ErrNotFound if no task with that ID exists.
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

	cats, err := r.fetchCategories(ctx, task.ID)
	if err != nil {
		return nil, err
	}
	task.Categories = cats

	return task, nil
}

// QueryFiltered returns tasks matching the filter criteria plus a total count for pagination.
// Sort order: priority DESC (higher first), then created_at ASC.
func (r *SQLiteTaskRepository) QueryFiltered(ctx context.Context, filter repository.TaskFilter) ([]*repository.Task, int, error) {
	// Build WHERE clause dynamically.
	var whereClauses []string
	var args []any
	needsCategoryJoin := filter.Category != ""

	// Status filter: default to "incomplete".
	status := filter.Status
	if status == "" {
		status = "incomplete"
	}
	switch status {
	case "incomplete":
		whereClauses = append(whereClauses, "t.completed_at IS NULL")
	case "complete":
		whereClauses = append(whereClauses, "t.completed_at IS NOT NULL")
	case "all":
		// No status filter.
	}

	// Category filter via junction table.
	if needsCategoryJoin {
		whereClauses = append(whereClauses, "c.name = ?")
		args = append(args, filter.Category)
	}

	// Search filter: case-insensitive LIKE on title OR description.
	if filter.Search != "" {
		whereClauses = append(whereClauses, "(t.title LIKE ? OR t.description LIKE ?)")
		searchPattern := "%" + filter.Search + "%"
		args = append(args, searchPattern, searchPattern)
	}

	// Assemble FROM clause.
	fromClause := "FROM tasks t"
	if needsCategoryJoin {
		fromClause += " INNER JOIN task_categories tc ON t.id = tc.task_id INNER JOIN categories c ON tc.category_id = c.id"
	}

	// Assemble WHERE clause.
	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Count query (no LIMIT/OFFSET).
	countQuery := "SELECT COUNT(*) " + fromClause + whereClause
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count filtered tasks: %w", err)
	}

	// Data query with ORDER BY, LIMIT, OFFSET.
	limit := filter.Limit
	if limit == 0 {
		limit = defaultTaskQueryLimit
	}

	dataQuery := "SELECT t.id, t.title, t.description, t.priority, t.due_date, t.created_at, t.completed_at, t.estimate_minutes, t.llm_estimate_minutes " +
		fromClause + whereClause +
		" ORDER BY t.priority DESC, t.created_at ASC LIMIT ? OFFSET ?"

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

	// Fetch categories for each task.
	for _, task := range tasks {
		cats, err := r.fetchCategories(ctx, task.ID)
		if err != nil {
			return nil, 0, err
		}
		task.Categories = cats
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

// fetchCategories loads the categories associated with a task.
//
// TODO(feat-109 Loop 4): rewrite against the new category_key FK column on
// tasks. The old task_categories junction is being torn down; this stub
// returns nil so the package compiles after the Category struct reshape.
func (r *SQLiteTaskRepository) fetchCategories(ctx context.Context, taskID uuid.UUID) ([]repository.Category, error) {
	_ = querySelectTaskCategories
	return nil, nil
}

// scanTask reads a task from a sql.Rows scanner.
func scanTask(rows *sql.Rows) (*repository.Task, error) {
	var (
		task               repository.Task
		idStr              string
		createdAtStr       string
		dueDate            sql.NullString
		completedAt        sql.NullString
		estimateMinutes    sql.NullInt64
		llmEstimateMinutes sql.NullInt64
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

	return &task, nil
}
