package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"

	_ "modernc.org/sqlite"
)

const createCategoriesTable = `
CREATE TABLE IF NOT EXISTS categories (
    id    TEXT PRIMARY KEY,
    name  TEXT NOT NULL UNIQUE,
    color TEXT NOT NULL DEFAULT '#808080'
);
`

// SQLiteCategoryRepository implements repository.CategoryRepository using SQLite.
type SQLiteCategoryRepository struct {
	db *sql.DB
}

// NewSQLiteCategoryRepository opens a SQLite database at dbPath, enables WAL mode
// and foreign keys, creates the categories table, and returns a ready repository.
func NewSQLiteCategoryRepository(dbPath string) (*SQLiteCategoryRepository, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open category database: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable WAL mode: %w", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	if _, err := db.Exec(createCategoriesTable); err != nil {
		db.Close()
		return nil, fmt.Errorf("create categories table: %w", err)
	}

	return &SQLiteCategoryRepository{db: db}, nil
}

// Insert adds a new category to the database.
func (r *SQLiteCategoryRepository) Insert(ctx context.Context, category *repository.Category) error {
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO categories (id, name, color) VALUES (?, ?, ?)",
		category.ID.String(), category.Name, category.Color,
	)
	if err != nil {
		return fmt.Errorf("insert category: %w", err)
	}
	return nil
}

// Update modifies an existing category's name and color.
func (r *SQLiteCategoryRepository) Update(ctx context.Context, category *repository.Category) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE categories SET name = ?, color = ? WHERE id = ?",
		category.Name, category.Color, category.ID.String(),
	)
	if err != nil {
		return fmt.Errorf("update category: %w", err)
	}
	return nil
}

// Delete removes a category by ID.
func (r *SQLiteCategoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		"DELETE FROM categories WHERE id = ?",
		id.String(),
	)
	if err != nil {
		return fmt.Errorf("delete category: %w", err)
	}
	return nil
}

// QueryAll returns all categories.
func (r *SQLiteCategoryRepository) QueryAll(ctx context.Context) ([]*repository.Category, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, name, color FROM categories")
	if err != nil {
		return nil, fmt.Errorf("query all categories: %w", err)
	}
	defer rows.Close()

	var categories []*repository.Category
	for rows.Next() {
		cat, err := scanCategory(rows)
		if err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		categories = append(categories, cat)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate categories: %w", err)
	}
	return categories, nil
}

// QueryByName returns a single category by name, or an error wrapping
// repository.ErrNotFound if no category with that name exists.
func (r *SQLiteCategoryRepository) QueryByName(ctx context.Context, name string) (*repository.Category, error) {
	row := r.db.QueryRowContext(ctx, "SELECT id, name, color FROM categories WHERE name = ?", name)

	var idStr string
	var cat repository.Category
	err := row.Scan(&idStr, &cat.Name, &cat.Color)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("query category by name: %w", repository.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("query category by name: %w", err)
	}

	cat.ID, err = uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("parse category id: %w", err)
	}
	return &cat, nil
}

// scanCategory reads a category from a sql.Rows scanner.
func scanCategory(rows *sql.Rows) (*repository.Category, error) {
	var idStr string
	var cat repository.Category
	if err := rows.Scan(&idStr, &cat.Name, &cat.Color); err != nil {
		return nil, err
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, err
	}
	cat.ID = id
	return &cat, nil
}
