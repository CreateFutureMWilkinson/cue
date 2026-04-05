package sqlite_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	sqlite "github.com/CreateFutureMWilkinson/cue/internal/repository/implementation/sqlite"

	_ "modernc.org/sqlite"
)

// CategoryRepositorySuite tests the SQLite implementation of CategoryRepository.
// NOTE: Tests expect repository.ErrNotFound to be defined as a sentinel error
// in the repository package (internal/repository/errors.go or similar).
type CategoryRepositorySuite struct {
	suite.Suite
}

func TestCategory(t *testing.T) {
	suite.Run(t, new(CategoryRepositorySuite))
}

func (s *CategoryRepositorySuite) TestInsertAndQueryAll() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	repo, err := sqlite.NewSQLiteCategoryRepository(dbPath)
	s.Require().NoError(err)
	s.Require().NotNil(repo)

	ctx := context.Background()

	cat := &repository.Category{
		ID:    uuid.New(),
		Name:  "Work",
		Color: "#FF5733",
	}

	err = repo.Insert(ctx, cat)
	s.Require().NoError(err)

	results, err := repo.QueryAll(ctx)
	s.Require().NoError(err)
	s.Require().Len(results, 1)

	got := results[0]
	s.Equal(cat.ID, got.ID)
	s.Equal(cat.Name, got.Name)
	s.Equal(cat.Color, got.Color)
}

func (s *CategoryRepositorySuite) TestInsertAndQueryByName() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	repo, err := sqlite.NewSQLiteCategoryRepository(dbPath)
	s.Require().NoError(err)

	ctx := context.Background()

	cat := &repository.Category{
		ID:    uuid.New(),
		Name:  "Personal",
		Color: "#00FF00",
	}

	err = repo.Insert(ctx, cat)
	s.Require().NoError(err)

	got, err := repo.QueryByName(ctx, "Personal")
	s.Require().NoError(err)
	s.Require().NotNil(got)

	s.Equal(cat.ID, got.ID)
	s.Equal(cat.Name, got.Name)
	s.Equal(cat.Color, got.Color)
}

func (s *CategoryRepositorySuite) TestUpdate() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	repo, err := sqlite.NewSQLiteCategoryRepository(dbPath)
	s.Require().NoError(err)

	ctx := context.Background()

	cat := &repository.Category{
		ID:    uuid.New(),
		Name:  "OldName",
		Color: "#111111",
	}

	err = repo.Insert(ctx, cat)
	s.Require().NoError(err)

	// Update name and color.
	cat.Name = "NewName"
	cat.Color = "#222222"

	err = repo.Update(ctx, cat)
	s.Require().NoError(err)

	// Query back and verify updated fields.
	results, err := repo.QueryAll(ctx)
	s.Require().NoError(err)
	s.Require().Len(results, 1)

	got := results[0]
	s.Equal(cat.ID, got.ID)
	s.Equal("NewName", got.Name)
	s.Equal("#222222", got.Color)
}

func (s *CategoryRepositorySuite) TestDelete() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	repo, err := sqlite.NewSQLiteCategoryRepository(dbPath)
	s.Require().NoError(err)

	ctx := context.Background()

	cat := &repository.Category{
		ID:    uuid.New(),
		Name:  "ToDelete",
		Color: "#AABBCC",
	}

	err = repo.Insert(ctx, cat)
	s.Require().NoError(err)

	// Verify it exists.
	results, err := repo.QueryAll(ctx)
	s.Require().NoError(err)
	s.Require().Len(results, 1)

	// Delete.
	err = repo.Delete(ctx, cat.ID)
	s.Require().NoError(err)

	// Verify it's gone.
	results, err = repo.QueryAll(ctx)
	s.Require().NoError(err)
	s.Len(results, 0)
}

func (s *CategoryRepositorySuite) TestQueryByNameNotFound() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	repo, err := sqlite.NewSQLiteCategoryRepository(dbPath)
	s.Require().NoError(err)

	ctx := context.Background()

	// Query a name that does not exist.
	got, err := repo.QueryByName(ctx, "NonExistent")
	s.Nil(got, "result should be nil when category not found")
	s.Require().Error(err)
	s.True(errors.Is(err, repository.ErrNotFound),
		"error should wrap repository.ErrNotFound, got: %v", err)
}

func (s *CategoryRepositorySuite) TestDuplicateName() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	repo, err := sqlite.NewSQLiteCategoryRepository(dbPath)
	s.Require().NoError(err)

	ctx := context.Background()

	cat1 := &repository.Category{
		ID:    uuid.New(),
		Name:  "Duplicate",
		Color: "#FF0000",
	}
	cat2 := &repository.Category{
		ID:    uuid.New(),
		Name:  "Duplicate",
		Color: "#00FF00",
	}

	err = repo.Insert(ctx, cat1)
	s.Require().NoError(err)

	// Second insert with the same name should fail.
	err = repo.Insert(ctx, cat2)
	s.Require().Error(err, "inserting a category with a duplicate name should return an error")
}

func (s *CategoryRepositorySuite) TestCascadeDeleteCategories() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	repo, err := sqlite.NewSQLiteCategoryRepository(dbPath)
	s.Require().NoError(err)

	ctx := context.Background()

	// Insert a category via the repository.
	cat := &repository.Category{
		ID:    uuid.New(),
		Name:  "CascadeTest",
		Color: "#ABCDEF",
	}
	err = repo.Insert(ctx, cat)
	s.Require().NoError(err)

	// Manually create a todos table and a todo_categories junction table via raw SQL,
	// since TodoRepository does not exist yet.
	db, err := sql.Open("sqlite", dbPath)
	s.Require().NoError(err)
	defer db.Close()

	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS todos (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL
		)
	`)
	s.Require().NoError(err)

	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS todo_categories (
			todo_id TEXT NOT NULL,
			category_id TEXT NOT NULL,
			PRIMARY KEY (todo_id, category_id),
			FOREIGN KEY (todo_id) REFERENCES todos(id),
			FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE
		)
	`)
	s.Require().NoError(err)

	// Insert a todo and link it to the category.
	todoID := uuid.New()
	_, err = db.ExecContext(ctx, `INSERT INTO todos (id, title) VALUES (?, ?)`,
		todoID.String(), "Test Todo")
	s.Require().NoError(err)

	_, err = db.ExecContext(ctx, `INSERT INTO todo_categories (todo_id, category_id) VALUES (?, ?)`,
		todoID.String(), cat.ID.String())
	s.Require().NoError(err)

	// Verify junction row exists.
	var count int
	err = db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM todo_categories WHERE category_id = ?`, cat.ID.String()).Scan(&count)
	s.Require().NoError(err)
	s.Equal(1, count, "junction table should have 1 row before delete")

	// Delete the category via the repository.
	err = repo.Delete(ctx, cat.ID)
	s.Require().NoError(err)

	// Verify the junction table row is gone (cascade delete).
	err = db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM todo_categories WHERE category_id = ?`, cat.ID.String()).Scan(&count)
	s.Require().NoError(err)
	s.Equal(0, count, "junction table row should be deleted when category is deleted (cascade)")
}
