package sqlite

import (
	"context"
	"database/sql"
	"fmt"
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
    completed_at TIMESTAMP
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
	todoColumnsStr             = "id, title, description, priority, due_date, created_at, completed_at"
	queryInsertTodo            = "INSERT INTO todos (id, title, description, priority, due_date, created_at, completed_at) VALUES (?, ?, ?, ?, ?, ?, ?)"
	queryUpdateTodo            = "UPDATE todos SET title = ?, description = ?, priority = ?, due_date = ?, completed_at = ? WHERE id = ?"
	queryDeleteTodo            = "DELETE FROM todos WHERE id = ?"
	querySelectTodoByID        = "SELECT " + todoColumnsStr + " FROM todos WHERE id = ?"
	querySelectIncompleteTodos = "SELECT " + todoColumnsStr + " FROM todos WHERE completed_at IS NULL ORDER BY priority ASC"
	querySelectAllTodos        = "SELECT " + todoColumnsStr + " FROM todos ORDER BY created_at DESC"
	queryCompleteTodo          = "UPDATE todos SET completed_at = ? WHERE id = ?"
	queryInsertTodoCategory    = "INSERT INTO todo_categories (todo_id, category_id) VALUES (?, ?)"
	queryDeleteTodoCategories  = "DELETE FROM todo_categories WHERE todo_id = ?"
	querySelectTodoCategories  = "SELECT c.id, c.name, c.color FROM categories c INNER JOIN todo_categories tc ON c.id = tc.category_id WHERE tc.todo_id = ?"
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
		db.Close()
		return nil, fmt.Errorf("enable WAL mode: %w", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	if _, err := db.Exec(createTodoTables); err != nil {
		db.Close()
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
	)
	if err != nil {
		return fmt.Errorf("insert todo: %w", err)
	}

	for _, cat := range todo.Categories {
		_, err = tx.ExecContext(ctx, queryInsertTodoCategory, todo.ID.String(), cat.ID.String())
		if err != nil {
			return fmt.Errorf("insert todo category: %w", err)
		}
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
		todo.ID.String(),
	)
	if err != nil {
		return fmt.Errorf("update todo: %w", err)
	}

	_, err = tx.ExecContext(ctx, queryDeleteTodoCategories, todo.ID.String())
	if err != nil {
		return fmt.Errorf("delete todo categories: %w", err)
	}

	for _, cat := range todo.Categories {
		_, err = tx.ExecContext(ctx, queryInsertTodoCategory, todo.ID.String(), cat.ID.String())
		if err != nil {
			return fmt.Errorf("insert todo category: %w", err)
		}
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
	row := r.db.QueryRowContext(ctx, querySelectTodoByID, id.String())

	todo, err := scanTodo(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("query todo by id: %w", repository.ErrNotFound)
		}
		return nil, fmt.Errorf("query todo by id: %w", err)
	}

	cats, err := r.fetchCategories(ctx, todo.ID)
	if err != nil {
		return nil, err
	}
	todo.Categories = cats

	return todo, nil
}

// QueryIncomplete returns all incomplete todos ordered by priority ASC.
func (r *SQLiteTodoRepository) QueryIncomplete(ctx context.Context) ([]*repository.Todo, error) {
	rows, err := r.db.QueryContext(ctx, querySelectIncompleteTodos)
	if err != nil {
		return nil, fmt.Errorf("query incomplete todos: %w", err)
	}
	defer rows.Close()

	todos, err := scanTodos(rows)
	if err != nil {
		return nil, err
	}

	for _, todo := range todos {
		cats, err := r.fetchCategories(ctx, todo.ID)
		if err != nil {
			return nil, err
		}
		todo.Categories = cats
	}

	return todos, nil
}

// QueryAll returns all todos ordered by created_at DESC.
func (r *SQLiteTodoRepository) QueryAll(ctx context.Context) ([]*repository.Todo, error) {
	rows, err := r.db.QueryContext(ctx, querySelectAllTodos)
	if err != nil {
		return nil, fmt.Errorf("query all todos: %w", err)
	}
	defer rows.Close()

	todos, err := scanTodos(rows)
	if err != nil {
		return nil, err
	}

	for _, todo := range todos {
		cats, err := r.fetchCategories(ctx, todo.ID)
		if err != nil {
			return nil, err
		}
		todo.Categories = cats
	}

	return todos, nil
}

// Complete sets the completed_at timestamp on a todo.
func (r *SQLiteTodoRepository) Complete(ctx context.Context, id uuid.UUID, completedAt time.Time) error {
	_, err := r.db.ExecContext(ctx, queryCompleteTodo, completedAt.Format(time.RFC3339), id.String())
	if err != nil {
		return fmt.Errorf("complete todo: %w", err)
	}
	return nil
}

// fetchCategories loads the categories associated with a todo via the junction table.
func (r *SQLiteTodoRepository) fetchCategories(ctx context.Context, todoID uuid.UUID) ([]repository.Category, error) {
	rows, err := r.db.QueryContext(ctx, querySelectTodoCategories, todoID.String())
	if err != nil {
		return nil, fmt.Errorf("fetch todo categories: %w", err)
	}
	defer rows.Close()

	var cats []repository.Category
	for rows.Next() {
		var (
			cat   repository.Category
			idStr string
		)
		if err := rows.Scan(&idStr, &cat.Name, &cat.Color); err != nil {
			return nil, fmt.Errorf("scan todo category: %w", err)
		}
		cat.ID, err = uuid.Parse(idStr)
		if err != nil {
			return nil, fmt.Errorf("parse category ID: %w", err)
		}
		cats = append(cats, cat)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate todo categories: %w", err)
	}

	return cats, nil
}

// scanTodo scans a single row into a Todo.
func scanTodo(row *sql.Row) (*repository.Todo, error) {
	var (
		todo         repository.Todo
		idStr        string
		createdAtStr string
		dueDate      sql.NullString
		completedAt  sql.NullString
	)

	err := row.Scan(
		&idStr,
		&todo.Title,
		&todo.Description,
		&todo.Priority,
		&dueDate,
		&createdAtStr,
		&completedAt,
	)
	if err != nil {
		return nil, err
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

	return &todo, nil
}

// scanTodos scans rows into a slice of Todo pointers.
func scanTodos(rows *sql.Rows) ([]*repository.Todo, error) {
	var todos []*repository.Todo
	for rows.Next() {
		var (
			todo         repository.Todo
			idStr        string
			createdAtStr string
			dueDate      sql.NullString
			completedAt  sql.NullString
		)

		err := rows.Scan(
			&idStr,
			&todo.Title,
			&todo.Description,
			&todo.Priority,
			&dueDate,
			&createdAtStr,
			&completedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan todo: %w", err)
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

		todos = append(todos, &todo)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate todos: %w", err)
	}
	return todos, nil
}
