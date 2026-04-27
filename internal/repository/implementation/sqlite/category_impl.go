package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"

	_ "modernc.org/sqlite"
)

// Loop 2 RED: stub implementation of repository.CategoryRepository against
// the new name-keyed schema (Feature 109 Decision 4). All methods return
// repository.ErrNotImplemented so that the test suite in
// category_impl_test.go compiles and fails with assertion errors rather
// than build errors. Loop 2 GREEN replaces these stubs with real SQL.

const createCategoriesTable = `
CREATE TABLE IF NOT EXISTS categories (
    name_key   TEXT PRIMARY KEY,
    colour     TEXT,
    created_at TEXT NOT NULL
);
`

// Compile-time check that SQLiteCategoryRepository satisfies the new
// repository.CategoryRepository contract.
var _ repository.CategoryRepository = (*SQLiteCategoryRepository)(nil)

// SQLiteCategoryRepository persists categories in a SQLite database
// using the name-keyed schema (no UUIDs, no display-name column).
type SQLiteCategoryRepository struct {
	db *sql.DB
}

// NewSQLiteCategoryRepository opens a SQLite database at dbPath, enables
// WAL mode and foreign keys, and creates the categories table if it does
// not already exist.
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

// Insert persists a new category. Returns repository.ErrDuplicate on a
// UNIQUE-key collision against an existing name_key.
//
// Stub: returns ErrNotImplemented; replaced in Loop 2 GREEN.
func (r *SQLiteCategoryRepository) Insert(ctx context.Context, c *repository.Category) error {
	return repository.ErrNotImplemented
}

// Rename moves a category from oldKey to newKey. Returns ErrNotFound if
// no row matches oldKey, or ErrDuplicate if newKey already exists.
//
// Stub: returns ErrNotImplemented; replaced in Loop 2 GREEN.
func (r *SQLiteCategoryRepository) Rename(ctx context.Context, oldKey, newKey string) error {
	return repository.ErrNotImplemented
}

// UpdateColour sets the colour of an existing category. nil colour
// stores SQL NULL. Returns ErrNotFound if the key does not exist.
//
// Stub: returns ErrNotImplemented; replaced in Loop 2 GREEN.
func (r *SQLiteCategoryRepository) UpdateColour(ctx context.Context, key string, colour *string) error {
	return repository.ErrNotImplemented
}

// Delete removes a category by key. Returns ErrNotFound if no row
// matches.
//
// Stub: returns ErrNotImplemented; replaced in Loop 2 GREEN.
func (r *SQLiteCategoryRepository) Delete(ctx context.Context, key string) error {
	return repository.ErrNotImplemented
}

// GetByKey looks up a single category by its canonical key. Returns
// ErrNotFound if no row matches.
//
// Stub: returns ErrNotImplemented; replaced in Loop 2 GREEN.
func (r *SQLiteCategoryRepository) GetByKey(ctx context.Context, key string) (*repository.Category, error) {
	return nil, repository.ErrNotImplemented
}

// QueryAll returns every category ordered by name_key. When withCounts is
// true the result includes the number of tasks referencing each category
// via tasks.category_key; otherwise TaskCount is zero.
//
// Stub: returns ErrNotImplemented; replaced in Loop 2 GREEN.
func (r *SQLiteCategoryRepository) QueryAll(ctx context.Context, withCounts bool) ([]*repository.CategoryWithCount, error) {
	return nil, repository.ErrNotImplemented
}
