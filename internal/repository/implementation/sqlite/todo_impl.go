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

const createTodoTables = `
CREATE TABLE IF NOT EXISTS todos (
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
CREATE TABLE IF NOT EXISTS todo_categories (
    todo_id TEXT NOT NULL REFERENCES todos(id) ON DELETE CASCADE,
    category_id TEXT NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    PRIMARY KEY (todo_id, category_id)
);
CREATE INDEX IF NOT EXISTS idx_todos_completed ON todos(completed_at);
CREATE INDEX IF NOT EXISTS idx_todos_priority ON todos(priority);
CREATE INDEX IF NOT EXISTS idx_todos_due_date ON todos(due_date);
`

const (
	todoColumnsStr            = "id, title, description, priority, due_date, created_at, completed_at, estimate_minutes, llm_estimate_minutes"
	queryInsertTodo           = "INSERT INTO todos (id, title, description, priority, due_date, created_at, completed_at, estimate_minutes, llm_estimate_minutes) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)"
	queryUpdateTodo           = "UPDATE todos SET title = ?, description = ?, priority = ?, due_date = ?, completed_at = ?, estimate_minutes = ?, llm_estimate_minutes = ? WHERE id = ?"
	queryDeleteTodo           = "DELETE FROM todos WHERE id = ?"
	querySelectTodoByID       = "SELECT " + todoColumnsStr + " FROM todos WHERE id = ?"
	queryCompleteTodo         = "UPDATE todos SET completed_at = ? WHERE id = ?"
	queryInsertTodoCategory   = "INSERT INTO todo_categories (todo_id, category_id) VALUES (?, ?)"
	queryDeleteTodoCategories = "DELETE FROM todo_categories WHERE todo_id = ?"
	querySelectTodoCategories = "SELECT c.id, c.name, c.color FROM categories c INNER JOIN todo_categories tc ON c.id = tc.category_id WHERE tc.todo_id = ?"

	defaultTodoQueryLimit = 50
)

// SQLiteTodoRepository implements repository.TodoRepository using SQLite.
type SQLiteTodoRepository struct {
	db *sql.DB
}

// NewSQLiteTodoRepository opens a SQLite database at dbPath, enables WAL mode
// and foreign keys, creates the todos and todo_categories tables, and returns a ready repository.
func NewSQLiteTodoRepository(dbPath string) (*SQLiteTodoRepository, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open todo database: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable WAL mode: %w", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	if _, err := db.Exec(createTodoTables); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create todo tables: %w", err)
	}

	return &SQLiteTodoRepository{db: db}, nil
}

// Insert adds a new todo and its category associations to the database.
func (r *SQLiteTodoRepository) Insert(ctx context.Context, todo *repository.Todo) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, queryInsertTodo,
		todo.ID.String(),
		todo.Title,
		todo.Description,
		todo.Priority,
		nullableTime(todo.DueDate),
		todo.CreatedAt.Format(time.RFC3339),
		nullableTime(todo.CompletedAt),
		todo.EstimateMinutes,
		todo.LLMEstimateMinutes,
	)
	if err != nil {
		return fmt.Errorf("insert todo: %w", err)
	}

	// TODO(feat-109 Loop 4): rewrite category persistence against the new
	// category_key FK column on tasks. The old todo_categories junction
	// table goes away in Loop 4; for now, skip association writes so the
	// package compiles after the Category struct reshape.
	_ = queryInsertTodoCategory
	for range todo.Categories {
	}

	return tx.Commit()
}

// Update modifies an existing todo's fields and replaces its category associations.
func (r *SQLiteTodoRepository) Update(ctx context.Context, todo *repository.Todo) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, queryUpdateTodo,
		todo.Title,
		todo.Description,
		todo.Priority,
		nullableTime(todo.DueDate),
		nullableTime(todo.CompletedAt),
		todo.EstimateMinutes,
		todo.LLMEstimateMinutes,
		todo.ID.String(),
	)
	if err != nil {
		return fmt.Errorf("update todo: %w", err)
	}

	_, err = tx.ExecContext(ctx, queryDeleteTodoCategories, todo.ID.String())
	if err != nil {
		return fmt.Errorf("delete todo categories: %w", err)
	}

	// TODO(feat-109 Loop 4): rewrite category persistence against the new
	// category_key FK column on tasks. See Insert above.
	for range todo.Categories {
	}

	return tx.Commit()
}

// Delete removes a todo by ID. Cascade handles junction table cleanup.
func (r *SQLiteTodoRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, queryDeleteTodo, id.String())
	if err != nil {
		return fmt.Errorf("delete todo: %w", err)
	}
	return nil
}

// QueryByID returns a todo by ID with populated categories, or nil and an error
// wrapping repository.ErrNotFound if no todo with that ID exists.
func (r *SQLiteTodoRepository) QueryByID(ctx context.Context, id uuid.UUID) (*repository.Todo, error) {
	rows, err := r.db.QueryContext(ctx, querySelectTodoByID, id.String())
	if err != nil {
		return nil, fmt.Errorf("query todo by id: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("query todo by id: %w", err)
		}
		return nil, fmt.Errorf("query todo by id: %w", repository.ErrNotFound)
	}

	todo, err := scanTodo(rows)
	if err != nil {
		return nil, fmt.Errorf("scan todo: %w", err)
	}

	cats, err := r.fetchCategories(ctx, todo.ID)
	if err != nil {
		return nil, err
	}
	todo.Categories = cats

	return todo, nil
}

// QueryFiltered returns todos matching the filter criteria plus a total count for pagination.
// Sort order: priority DESC (higher first), then created_at ASC.
func (r *SQLiteTodoRepository) QueryFiltered(ctx context.Context, filter repository.TodoFilter) ([]*repository.Todo, int, error) {
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
	fromClause := "FROM todos t"
	if needsCategoryJoin {
		fromClause += " INNER JOIN todo_categories tc ON t.id = tc.todo_id INNER JOIN categories c ON tc.category_id = c.id"
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
		return nil, 0, fmt.Errorf("count filtered todos: %w", err)
	}

	// Data query with ORDER BY, LIMIT, OFFSET.
	limit := filter.Limit
	if limit == 0 {
		limit = defaultTodoQueryLimit
	}

	dataQuery := "SELECT t.id, t.title, t.description, t.priority, t.due_date, t.created_at, t.completed_at, t.estimate_minutes, t.llm_estimate_minutes " +
		fromClause + whereClause +
		" ORDER BY t.priority DESC, t.created_at ASC LIMIT ? OFFSET ?"

	dataArgs := append(args, limit, filter.Offset)

	rows, err := r.db.QueryContext(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("query filtered todos: %w", err)
	}
	defer rows.Close()

	var todos []*repository.Todo
	for rows.Next() {
		todo, err := scanTodo(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan filtered todo: %w", err)
		}
		todos = append(todos, todo)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate filtered todos: %w", err)
	}

	// Fetch categories for each todo.
	for _, todo := range todos {
		cats, err := r.fetchCategories(ctx, todo.ID)
		if err != nil {
			return nil, 0, err
		}
		todo.Categories = cats
	}

	return todos, total, nil
}

// Complete sets the completed_at timestamp on a todo.
func (r *SQLiteTodoRepository) Complete(ctx context.Context, id uuid.UUID, completedAt time.Time) error {
	_, err := r.db.ExecContext(ctx, queryCompleteTodo, completedAt.Format(time.RFC3339), id.String())
	if err != nil {
		return fmt.Errorf("complete todo: %w", err)
	}
	return nil
}

// fetchCategories loads the categories associated with a todo.
//
// TODO(feat-109 Loop 4): rewrite against the new category_key FK column on
// tasks. The old todo_categories junction is being torn down; this stub
// returns nil so the package compiles after the Category struct reshape.
func (r *SQLiteTodoRepository) fetchCategories(ctx context.Context, todoID uuid.UUID) ([]repository.Category, error) {
	_ = querySelectTodoCategories
	return nil, nil
}

// scanTodo reads a todo from a sql.Rows scanner.
func scanTodo(rows *sql.Rows) (*repository.Todo, error) {
	var (
		todo               repository.Todo
		idStr              string
		createdAtStr       string
		dueDate            sql.NullString
		completedAt        sql.NullString
		estimateMinutes    sql.NullInt64
		llmEstimateMinutes sql.NullInt64
	)

	err := rows.Scan(
		&idStr,
		&todo.Title,
		&todo.Description,
		&todo.Priority,
		&dueDate,
		&createdAtStr,
		&completedAt,
		&estimateMinutes,
		&llmEstimateMinutes,
	)
	if err != nil {
		return nil, fmt.Errorf("scan todo row: %w", err)
	}

	todo.ID, err = uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("parse todo ID: %w", err)
	}

	todo.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}

	if dueDate.Valid {
		t, err := time.Parse(time.RFC3339, dueDate.String)
		if err != nil {
			return nil, fmt.Errorf("parse due_date: %w", err)
		}
		todo.DueDate = &t
	}

	if completedAt.Valid {
		t, err := time.Parse(time.RFC3339, completedAt.String)
		if err != nil {
			return nil, fmt.Errorf("parse completed_at: %w", err)
		}
		todo.CompletedAt = &t
	}

	if estimateMinutes.Valid {
		v := int(estimateMinutes.Int64)
		todo.EstimateMinutes = &v
	}

	if llmEstimateMinutes.Valid {
		v := int(llmEstimateMinutes.Int64)
		todo.LLMEstimateMinutes = &v
	}

	return &todo, nil
}
