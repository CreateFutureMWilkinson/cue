package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"

	_ "modernc.org/sqlite"
)

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

// isUniqueViolation reports whether err looks like a SQLite UNIQUE
// constraint failure. modernc.org/sqlite surfaces these as a string
// containing "UNIQUE constraint failed".
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// nullableColour converts a *string to a value suitable for ?-binding
// where nil is persisted as SQL NULL.
func nullableColour(c *string) any {
	if c == nil {
		return nil
	}
	return *c
}

// Insert persists a new category. Returns repository.ErrDuplicate on a
// UNIQUE-key collision against an existing name_key.
func (r *SQLiteCategoryRepository) Insert(ctx context.Context, c *repository.Category) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO categories (name_key, colour, created_at) VALUES (?, ?, ?)`,
		c.NameKey,
		nullableColour(c.Colour),
		c.CreatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("insert category %q: %w", c.NameKey, repository.ErrDuplicate)
		}
		return fmt.Errorf("insert category %q: %w", c.NameKey, err)
	}
	return nil
}

// Rename moves a category from oldKey to newKey. Returns ErrNotFound if
// no row matches oldKey, or ErrDuplicate if newKey already exists.
func (r *SQLiteCategoryRepository) Rename(ctx context.Context, oldKey, newKey string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE categories SET name_key = ? WHERE name_key = ?`,
		newKey, oldKey,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("rename category %q -> %q: %w", oldKey, newKey, repository.ErrDuplicate)
		}
		return fmt.Errorf("rename category %q -> %q: %w", oldKey, newKey, err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rename category %q -> %q: rows affected: %w", oldKey, newKey, err)
	}
	if rows == 0 {
		return fmt.Errorf("rename category %q: %w", oldKey, repository.ErrNotFound)
	}
	return nil
}

// UpdateColour sets the colour of an existing category. nil colour
// stores SQL NULL. Returns ErrNotFound if the key does not exist.
func (r *SQLiteCategoryRepository) UpdateColour(ctx context.Context, key string, colour *string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE categories SET colour = ? WHERE name_key = ?`,
		nullableColour(colour), key,
	)
	if err != nil {
		return fmt.Errorf("update colour for %q: %w", key, err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update colour for %q: rows affected: %w", key, err)
	}
	if rows == 0 {
		return fmt.Errorf("update colour for %q: %w", key, repository.ErrNotFound)
	}
	return nil
}

// Delete removes a category by key. Returns ErrNotFound if no row
// matches.
func (r *SQLiteCategoryRepository) Delete(ctx context.Context, key string) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM categories WHERE name_key = ?`, key,
	)
	if err != nil {
		return fmt.Errorf("delete category %q: %w", key, err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete category %q: rows affected: %w", key, err)
	}
	if rows == 0 {
		return fmt.Errorf("delete category %q: %w", key, repository.ErrNotFound)
	}
	return nil
}

// parseCategoryFields constructs a Category from the common scanned fields.
func parseCategoryFields(nameKey string, colour sql.NullString, createdAtStr string) (*repository.Category, error) {
	cat := &repository.Category{NameKey: nameKey}
	if colour.Valid {
		v := colour.String
		cat.Colour = &v
	}

	createdAt, err := time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	cat.CreatedAt = createdAt
	return cat, nil
}

// scanCategoryRow scans (name_key, colour, created_at) into a Category.
func scanCategoryRow(scanner interface {
	Scan(dest ...any) error
}) (*repository.Category, error) {
	var (
		nameKey      string
		colour       sql.NullString
		createdAtStr string
	)
	if err := scanner.Scan(&nameKey, &colour, &createdAtStr); err != nil {
		return nil, err
	}

	return parseCategoryFields(nameKey, colour, createdAtStr)
}

// GetByKey looks up a single category by its canonical key. Returns
// ErrNotFound if no row matches.
func (r *SQLiteCategoryRepository) GetByKey(ctx context.Context, key string) (*repository.Category, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT name_key, colour, created_at FROM categories WHERE name_key = ?`, key,
	)
	cat, err := scanCategoryRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("category %q: %w", key, repository.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("get category %q: %w", key, err)
	}
	return cat, nil
}

// QueryAll returns every category ordered by name_key. When withCounts is
// true the result includes the number of tasks referencing each category
// via tasks.category_key; otherwise TaskCount is zero.
func (r *SQLiteCategoryRepository) QueryAll(ctx context.Context, withCounts bool) ([]*repository.CategoryWithCount, error) {
	if withCounts {
		return r.queryAllWithCounts(ctx)
	}
	return r.queryAllWithoutCounts(ctx)
}

func (r *SQLiteCategoryRepository) queryAllWithoutCounts(ctx context.Context) ([]*repository.CategoryWithCount, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT name_key, colour, created_at FROM categories ORDER BY name_key`,
	)
	if err != nil {
		return nil, fmt.Errorf("query categories: %w", err)
	}
	defer rows.Close()

	out := make([]*repository.CategoryWithCount, 0)
	for rows.Next() {
		cat, err := scanCategoryRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		out = append(out, &repository.CategoryWithCount{Category: *cat})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate categories: %w", err)
	}
	return out, nil
}

func (r *SQLiteCategoryRepository) queryAllWithCounts(ctx context.Context) ([]*repository.CategoryWithCount, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT c.name_key, c.colour, c.created_at, COALESCE(COUNT(t.id), 0) AS task_count
		FROM categories c
		LEFT JOIN tasks t ON t.category_key = c.name_key
		GROUP BY c.name_key, c.colour, c.created_at
		ORDER BY c.name_key
	`)
	if err != nil {
		return nil, fmt.Errorf("query categories with counts: %w", err)
	}
	defer rows.Close()

	out := make([]*repository.CategoryWithCount, 0)
	for rows.Next() {
		var (
			nameKey      string
			colour       sql.NullString
			createdAtStr string
			taskCount    int
		)
		if err := rows.Scan(&nameKey, &colour, &createdAtStr, &taskCount); err != nil {
			return nil, fmt.Errorf("scan category with count: %w", err)
		}

		cat, err := parseCategoryFields(nameKey, colour, createdAtStr)
		if err != nil {
			return nil, fmt.Errorf("parse category fields: %w", err)
		}

		out = append(out, &repository.CategoryWithCount{Category: *cat, TaskCount: taskCount})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate categories with counts: %w", err)
	}
	return out, nil
}
