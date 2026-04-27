package sqlite_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	sqlite "github.com/CreateFutureMWilkinson/cue/internal/repository/implementation/sqlite"

	_ "modernc.org/sqlite"
)

// CategoryRepositorySuite exercises the SQLite implementation of
// repository.CategoryRepository against the Feature 109 name-keyed
// schema.
type CategoryRepositorySuite struct {
	suite.Suite
	repo   *sqlite.SQLiteCategoryRepository
	dbPath string
}

func TestCategory(t *testing.T) {
	suite.Run(t, new(CategoryRepositorySuite))
}

func (s *CategoryRepositorySuite) SetupTest() {
	tmpDir := s.T().TempDir()
	s.dbPath = filepath.Join(tmpDir, "categories.db")

	repo, err := sqlite.NewSQLiteCategoryRepository(s.dbPath)
	s.Require().NoError(err)
	s.Require().NotNil(repo)
	s.repo = repo
}

// strPtr is a small convenience for nullable colour values.
func strPtr(v string) *string { return &v }

// makeCategory returns a Category with the given key and optional colour.
func (s *CategoryRepositorySuite) makeCategory(key string, colour *string) *repository.Category {
	return &repository.Category{
		NameKey:   key,
		Colour:    colour,
		CreatedAt: time.Now().UTC().Truncate(time.Second),
	}
}

// --- Behaviour 1: Insert + GetByKey round-trip ---

func (s *CategoryRepositorySuite) TestInsertAndGetByKeyRoundTripWithoutColour() {
	ctx := context.Background()
	cat := s.makeCategory("work", nil)

	err := s.repo.Insert(ctx, cat)
	s.Require().NoError(err)

	got, err := s.repo.GetByKey(ctx, "work")
	s.Require().NoError(err)
	s.Require().NotNil(got)
	s.Equal("work", got.NameKey)
	s.Nil(got.Colour)
	s.WithinDuration(cat.CreatedAt, got.CreatedAt, time.Second)
}

func (s *CategoryRepositorySuite) TestInsertAndGetByKeyRoundTripWithColour() {
	ctx := context.Background()
	cat := s.makeCategory("foo_bar", strPtr("#3aa3aa"))

	err := s.repo.Insert(ctx, cat)
	s.Require().NoError(err)

	got, err := s.repo.GetByKey(ctx, "foo_bar")
	s.Require().NoError(err)
	s.Require().NotNil(got)
	s.Equal("foo_bar", got.NameKey)
	s.Require().NotNil(got.Colour)
	s.Equal("#3aa3aa", *got.Colour)
}

// --- Behaviour 2: Insert duplicate name_key ---

func (s *CategoryRepositorySuite) TestInsertDuplicateKeyReturnsErrDuplicate() {
	ctx := context.Background()

	s.Require().NoError(s.repo.Insert(ctx, s.makeCategory("work", nil)))

	err := s.repo.Insert(ctx, s.makeCategory("work", strPtr("#ffffff")))
	s.ErrorIs(err, repository.ErrDuplicate)
}

// --- Behaviour 3: Rename to a fresh key ---

func (s *CategoryRepositorySuite) TestRenameToFreshKey() {
	ctx := context.Background()

	s.Require().NoError(s.repo.Insert(ctx, s.makeCategory("old_key", strPtr("#112233"))))

	err := s.repo.Rename(ctx, "old_key", "new_key")
	s.Require().NoError(err)

	got, err := s.repo.GetByKey(ctx, "old_key")
	s.ErrorIs(err, repository.ErrNotFound)
	s.Nil(got)

	got, err = s.repo.GetByKey(ctx, "new_key")
	s.Require().NoError(err)
	s.Require().NotNil(got)
	s.Equal("new_key", got.NameKey)
	s.Require().NotNil(got.Colour)
	s.Equal("#112233", *got.Colour)
}

// --- Behaviour 4: Rename onto an existing key ---

func (s *CategoryRepositorySuite) TestRenameToExistingKeyReturnsErrDuplicate() {
	ctx := context.Background()

	s.Require().NoError(s.repo.Insert(ctx, s.makeCategory("alpha", nil)))
	s.Require().NoError(s.repo.Insert(ctx, s.makeCategory("beta", nil)))

	err := s.repo.Rename(ctx, "alpha", "beta")
	s.ErrorIs(err, repository.ErrDuplicate)
}

// --- Behaviour 5: Rename a missing key ---

func (s *CategoryRepositorySuite) TestRenameMissingKeyReturnsErrNotFound() {
	ctx := context.Background()

	err := s.repo.Rename(ctx, "ghost", "fresh")
	s.ErrorIs(err, repository.ErrNotFound)
}

// --- Behaviour 6: UpdateColour to a value, then nil ---

func (s *CategoryRepositorySuite) TestUpdateColourSetThenClear() {
	ctx := context.Background()

	s.Require().NoError(s.repo.Insert(ctx, s.makeCategory("design", nil)))

	err := s.repo.UpdateColour(ctx, "design", strPtr("#abcdef"))
	s.Require().NoError(err)

	got, err := s.repo.GetByKey(ctx, "design")
	s.Require().NoError(err)
	s.Require().NotNil(got.Colour)
	s.Equal("#abcdef", *got.Colour)

	err = s.repo.UpdateColour(ctx, "design", nil)
	s.Require().NoError(err)

	got, err = s.repo.GetByKey(ctx, "design")
	s.Require().NoError(err)
	s.Nil(got.Colour, "colour should be cleared to NULL")
}

// --- Behaviour 7: UpdateColour on missing key ---

func (s *CategoryRepositorySuite) TestUpdateColourMissingKeyReturnsErrNotFound() {
	ctx := context.Background()

	err := s.repo.UpdateColour(ctx, "ghost", strPtr("#000000"))
	s.ErrorIs(err, repository.ErrNotFound)
}

// --- Behaviour 8: Delete then second Delete ---

func (s *CategoryRepositorySuite) TestDeleteRemovesRowAndIsNotIdempotent() {
	ctx := context.Background()

	s.Require().NoError(s.repo.Insert(ctx, s.makeCategory("temp", nil)))

	err := s.repo.Delete(ctx, "temp")
	s.Require().NoError(err)

	got, err := s.repo.GetByKey(ctx, "temp")
	s.ErrorIs(err, repository.ErrNotFound)
	s.Nil(got)

	err = s.repo.Delete(ctx, "temp")
	s.ErrorIs(err, repository.ErrNotFound)
}

// --- Behaviour 9: GetByKey unknown ---

func (s *CategoryRepositorySuite) TestGetByKeyUnknownReturnsErrNotFound() {
	ctx := context.Background()

	got, err := s.repo.GetByKey(ctx, "nope")
	s.ErrorIs(err, repository.ErrNotFound)
	s.Nil(got)
}

// --- Behaviour 10: QueryAll without counts is sorted by name_key ---

func (s *CategoryRepositorySuite) TestQueryAllWithoutCountsOrdered() {
	ctx := context.Background()

	s.Require().NoError(s.repo.Insert(ctx, s.makeCategory("charlie", nil)))
	s.Require().NoError(s.repo.Insert(ctx, s.makeCategory("alpha", nil)))
	s.Require().NoError(s.repo.Insert(ctx, s.makeCategory("bravo", nil)))

	got, err := s.repo.QueryAll(ctx, false)
	s.Require().NoError(err)
	s.Require().Len(got, 3)
	s.Equal("alpha", got[0].NameKey)
	s.Equal("bravo", got[1].NameKey)
	s.Equal("charlie", got[2].NameKey)
	for _, c := range got {
		s.Equal(0, c.TaskCount, "TaskCount should be zero when withCounts=false")
	}
}

// --- Behaviour 11: QueryAll with counts joins against tasks.category_key ---

func (s *CategoryRepositorySuite) TestQueryAllWithCountsJoinsTasks() {
	ctx := context.Background()

	// Loop 4 will add the tasks table with a category_key column. For
	// this loop we create a minimal stand-in by hand so that the JOIN
	// has somewhere to land.
	db, err := sql.Open("sqlite", s.dbPath)
	s.Require().NoError(err)
	s.T().Cleanup(func() { _ = db.Close() })

	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS tasks (
			id           TEXT PRIMARY KEY,
			category_key TEXT NULL
		)
	`)
	s.Require().NoError(err)

	s.Require().NoError(s.repo.Insert(ctx, s.makeCategory("work", nil)))
	s.Require().NoError(s.repo.Insert(ctx, s.makeCategory("home", nil)))
	s.Require().NoError(s.repo.Insert(ctx, s.makeCategory("idle", nil)))

	for _, row := range []struct {
		id, cat string
	}{
		{"t1", "work"},
		{"t2", "work"},
		{"t3", "work"},
		{"t4", "home"},
	} {
		_, err = db.ExecContext(ctx,
			"INSERT INTO tasks (id, category_key) VALUES (?, ?)", row.id, row.cat)
		s.Require().NoError(err)
	}

	got, err := s.repo.QueryAll(ctx, true)
	s.Require().NoError(err)
	s.Require().Len(got, 3, "expected one row per category, including idle with zero tasks")

	counts := map[string]int{}
	for _, c := range got {
		counts[c.NameKey] = c.TaskCount
	}
	s.Equal(3, counts["work"])
	s.Equal(1, counts["home"])
	s.Equal(0, counts["idle"], "categories with no referencing tasks should still appear with TaskCount=0")
}
