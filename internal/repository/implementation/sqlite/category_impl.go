package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"

	_ "modernc.org/sqlite"
)

// NOTE (Feature 109 Loop 1 RED):
// This file is intentionally stubbed pending Loop 2, which will rewrite
// the SQLite category repository against the reshaped name-keyed schema
// and the new repository.CategoryRepository interface. The methods here
// keep the package compiling for legacy call sites that still reference
// the old shape. Their behaviour is undefined and Loop 2 replaces them
// wholesale; do not rely on them.

const createCategoriesTable = `
CREATE TABLE IF NOT EXISTS categories (
    name_key   TEXT PRIMARY KEY,
    colour     TEXT,
    created_at TEXT NOT NULL DEFAULT ''
);
`

// SQLiteCategoryRepository is a stub implementation pending Loop 2.
type SQLiteCategoryRepository struct {
	db *sql.DB
}

// NewSQLiteCategoryRepository opens a SQLite database at dbPath, enables
// WAL mode and foreign keys, and creates the categories table skeleton.
//
// Stub pending Loop 2.
func NewSQLiteCategoryRepository(dbPath string) (*SQLiteCategoryRepository, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open category database: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable WAL mode: %w", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	if _, err := db.Exec(createCategoriesTable); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create categories table: %w", err)
	}

	return &SQLiteCategoryRepository{db: db}, nil
}

// Insert is a stub pending Loop 2.
func (r *SQLiteCategoryRepository) Insert(ctx context.Context, category *repository.Category) error {
	return repository.ErrNotImplemented
}

// Update is a legacy stub kept so existing call sites compile;
// removed entirely in Loop 2 in favour of Rename/UpdateColour.
func (r *SQLiteCategoryRepository) Update(ctx context.Context, category *repository.Category) error {
	return repository.ErrNotImplemented
}

// Delete is a legacy UUID-keyed stub kept so existing call sites compile;
// the new key-based Delete(string) ships in Loop 2.
func (r *SQLiteCategoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return repository.ErrNotImplemented
}

// QueryAll is a legacy stub kept so the planner presenter's
// CategoryQuerier interface still binds; reshaped to (ctx, bool) in Loop 2.
func (r *SQLiteCategoryRepository) QueryAll(ctx context.Context) ([]*repository.Category, error) {
	return nil, repository.ErrNotImplemented
}

// QueryByName is a legacy stub kept so existing call sites compile;
// removed in Loop 2 in favour of GetByKey.
func (r *SQLiteCategoryRepository) QueryByName(ctx context.Context, name string) (*repository.Category, error) {
	return nil, repository.ErrNotImplemented
}
