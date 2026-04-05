package sqlite_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	sqlite "github.com/CreateFutureMWilkinson/cue/internal/repository/implementation/sqlite"
)

// TodoRepositorySuite tests the SQLite implementation of TodoRepository.
type TodoRepositorySuite struct {
	suite.Suite
}

func TestTodo(t *testing.T) {
	suite.Run(t, new(TodoRepositorySuite))
}

// makeTodoRepo creates a category repo and a todo repo sharing the same DB file.
// The category repo is created first so the categories table exists for foreign keys.
func (s *TodoRepositorySuite) makeTodoRepo(dbPath string) (*sqlite.SQLiteTodoRepository, *sqlite.SQLiteCategoryRepository) {
	catRepo, err := sqlite.NewSQLiteCategoryRepository(dbPath)
	s.Require().NoError(err)
	s.Require().NotNil(catRepo)

	todoRepo, err := sqlite.NewSQLiteTodoRepository(dbPath)
	s.Require().NoError(err)
	s.Require().NotNil(todoRepo)

	return todoRepo, catRepo
}

func (s *TodoRepositorySuite) TestInsert() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	todoRepo, _ := s.makeTodoRepo(dbPath)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)
	dueDate := now.Add(24 * time.Hour)

	todo := &repository.Todo{
		ID:          uuid.New(),
		Title:       "Fix the server",
		Description: "The production server needs attention",
		Priority:    2,
		DueDate:     &dueDate,
		CreatedAt:   now,
	}

	err := todoRepo.Insert(ctx, todo)
	s.Require().NoError(err)

	got, err := todoRepo.QueryByID(ctx, todo.ID)
	s.Require().NoError(err)
	s.Require().NotNil(got)

	s.Equal(todo.ID, got.ID)
	s.Equal(todo.Title, got.Title)
	s.Equal(todo.Description, got.Description)
	s.Equal(todo.Priority, got.Priority)
	s.Require().NotNil(got.DueDate)
	s.WithinDuration(*todo.DueDate, *got.DueDate, time.Second)
	s.WithinDuration(todo.CreatedAt, got.CreatedAt, time.Second)
	s.Nil(got.CompletedAt, "new todo should not be completed")
}

func (s *TodoRepositorySuite) TestUpdate() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	todoRepo, _ := s.makeTodoRepo(dbPath)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	todo := &repository.Todo{
		ID:          uuid.New(),
		Title:       "Original title",
		Description: "Original description",
		Priority:    3,
		CreatedAt:   now,
	}

	err := todoRepo.Insert(ctx, todo)
	s.Require().NoError(err)

	// Update fields.
	newDueDate := now.Add(48 * time.Hour)
	todo.Title = "Updated title"
	todo.Description = "Updated description"
	todo.Priority = 1
	todo.DueDate = &newDueDate

	err = todoRepo.Update(ctx, todo)
	s.Require().NoError(err)

	got, err := todoRepo.QueryByID(ctx, todo.ID)
	s.Require().NoError(err)
	s.Require().NotNil(got)

	s.Equal("Updated title", got.Title)
	s.Equal("Updated description", got.Description)
	s.Equal(1, got.Priority)
	s.Require().NotNil(got.DueDate)
	s.WithinDuration(newDueDate, *got.DueDate, time.Second)
}

func (s *TodoRepositorySuite) TestDelete() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	todoRepo, _ := s.makeTodoRepo(dbPath)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	todo := &repository.Todo{
		ID:          uuid.New(),
		Title:       "To be deleted",
		Description: "This will be removed",
		Priority:    5,
		CreatedAt:   now,
	}

	err := todoRepo.Insert(ctx, todo)
	s.Require().NoError(err)

	err = todoRepo.Delete(ctx, todo.ID)
	s.Require().NoError(err)

	_, err = todoRepo.QueryByID(ctx, todo.ID)
	s.Require().Error(err)
	s.True(errors.Is(err, repository.ErrNotFound),
		"expected error wrapping repository.ErrNotFound, got: %v", err)
}

func (s *TodoRepositorySuite) TestQueryByID() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	todoRepo, _ := s.makeTodoRepo(dbPath)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	todo := &repository.Todo{
		ID:          uuid.New(),
		Title:       "Query me",
		Description: "Find this todo by ID",
		Priority:    4,
		CreatedAt:   now,
	}

	err := todoRepo.Insert(ctx, todo)
	s.Require().NoError(err)

	got, err := todoRepo.QueryByID(ctx, todo.ID)
	s.Require().NoError(err)
	s.Require().NotNil(got)

	s.Equal(todo.ID, got.ID)
	s.Equal(todo.Title, got.Title)
	s.Equal(todo.Description, got.Description)
	s.Equal(todo.Priority, got.Priority)
}

func (s *TodoRepositorySuite) TestQueryByIDNotFound() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	todoRepo, _ := s.makeTodoRepo(dbPath)

	ctx := context.Background()

	randomID := uuid.New()
	got, err := todoRepo.QueryByID(ctx, randomID)
	s.Nil(got, "result should be nil when todo not found")
	s.Require().Error(err)
	s.True(errors.Is(err, repository.ErrNotFound),
		"error should wrap repository.ErrNotFound, got: %v", err)
}

func (s *TodoRepositorySuite) TestQueryIncomplete() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	todoRepo, _ := s.makeTodoRepo(dbPath)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	// Insert 2 incomplete todos with different priorities.
	todo1 := &repository.Todo{
		ID:          uuid.New(),
		Title:       "High priority incomplete",
		Description: "Priority 1",
		Priority:    1,
		CreatedAt:   now,
	}
	todo2 := &repository.Todo{
		ID:          uuid.New(),
		Title:       "Low priority incomplete",
		Description: "Priority 5",
		Priority:    5,
		CreatedAt:   now.Add(time.Second),
	}

	// Insert 1 completed todo.
	completedAt := now.Add(time.Hour)
	todo3 := &repository.Todo{
		ID:          uuid.New(),
		Title:       "Completed todo",
		Description: "Already done",
		Priority:    2,
		CreatedAt:   now.Add(2 * time.Second),
		CompletedAt: &completedAt,
	}

	s.Require().NoError(todoRepo.Insert(ctx, todo1))
	s.Require().NoError(todoRepo.Insert(ctx, todo2))
	s.Require().NoError(todoRepo.Insert(ctx, todo3))

	results, err := todoRepo.QueryIncomplete(ctx)
	s.Require().NoError(err)
	s.Require().Len(results, 2, "should only return incomplete todos")

	// Ordered by priority ASC (lower number = higher priority first).
	s.Equal(todo1.ID, results[0].ID, "highest priority (1) should be first")
	s.Equal(todo2.ID, results[1].ID, "lowest priority (5) should be second")
}

func (s *TodoRepositorySuite) TestQueryAll() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	todoRepo, _ := s.makeTodoRepo(dbPath)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	// Insert 3 todos: 2 incomplete, 1 completed. Use distinct created_at for ordering.
	todo1 := &repository.Todo{
		ID:          uuid.New(),
		Title:       "First created",
		Description: "Oldest",
		Priority:    3,
		CreatedAt:   now,
	}
	todo2 := &repository.Todo{
		ID:          uuid.New(),
		Title:       "Second created",
		Description: "Middle",
		Priority:    1,
		CreatedAt:   now.Add(time.Second),
	}
	completedAt := now.Add(time.Hour)
	todo3 := &repository.Todo{
		ID:          uuid.New(),
		Title:       "Third created (completed)",
		Description: "Newest",
		Priority:    2,
		CreatedAt:   now.Add(2 * time.Second),
		CompletedAt: &completedAt,
	}

	s.Require().NoError(todoRepo.Insert(ctx, todo1))
	s.Require().NoError(todoRepo.Insert(ctx, todo2))
	s.Require().NoError(todoRepo.Insert(ctx, todo3))

	results, err := todoRepo.QueryAll(ctx)
	s.Require().NoError(err)
	s.Require().Len(results, 3, "should return all todos including completed")

	// Ordered by created_at DESC (newest first).
	s.Equal(todo3.ID, results[0].ID, "newest should be first")
	s.Equal(todo2.ID, results[1].ID, "middle should be second")
	s.Equal(todo1.ID, results[2].ID, "oldest should be third")
}

func (s *TodoRepositorySuite) TestComplete() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	todoRepo, _ := s.makeTodoRepo(dbPath)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	todo := &repository.Todo{
		ID:          uuid.New(),
		Title:       "Complete me",
		Description: "Should be marked done",
		Priority:    3,
		CreatedAt:   now,
	}

	err := todoRepo.Insert(ctx, todo)
	s.Require().NoError(err)

	// Verify not completed initially.
	got, err := todoRepo.QueryByID(ctx, todo.ID)
	s.Require().NoError(err)
	s.Nil(got.CompletedAt, "should not be completed initially")

	// Complete the todo.
	completedAt := now.Add(2 * time.Hour)
	err = todoRepo.Complete(ctx, todo.ID, completedAt)
	s.Require().NoError(err)

	// Verify completed.
	got, err = todoRepo.QueryByID(ctx, todo.ID)
	s.Require().NoError(err)
	s.Require().NotNil(got.CompletedAt, "should be completed after calling Complete")
	s.WithinDuration(completedAt, *got.CompletedAt, time.Second)
}

func (s *TodoRepositorySuite) TestCategoriesAssociation() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create category repo first (creates categories table), then todo repo.
	todoRepo, catRepo := s.makeTodoRepo(dbPath)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	// Insert 2 categories via the CategoryRepository.
	cat1 := &repository.Category{
		ID:    uuid.New(),
		Name:  "Work",
		Color: "#FF5733",
	}
	cat2 := &repository.Category{
		ID:    uuid.New(),
		Name:  "Urgent",
		Color: "#FF0000",
	}

	s.Require().NoError(catRepo.Insert(ctx, cat1))
	s.Require().NoError(catRepo.Insert(ctx, cat2))

	// Create a todo with both categories.
	todo := &repository.Todo{
		ID:          uuid.New(),
		Title:       "Categorized task",
		Description: "Has two categories",
		Priority:    1,
		Categories:  []repository.Category{*cat1, *cat2},
		CreatedAt:   now,
	}

	err := todoRepo.Insert(ctx, todo)
	s.Require().NoError(err)

	// Query back and verify categories are populated.
	got, err := todoRepo.QueryByID(ctx, todo.ID)
	s.Require().NoError(err)
	s.Require().NotNil(got)
	s.Require().Len(got.Categories, 2, "todo should have 2 categories")

	// Build a map of category IDs for order-independent comparison.
	catIDs := make(map[uuid.UUID]bool)
	for _, c := range got.Categories {
		catIDs[c.ID] = true
	}
	s.True(catIDs[cat1.ID], "should contain category 'Work'")
	s.True(catIDs[cat2.ID], "should contain category 'Urgent'")

	// Verify category fields are fully populated.
	for _, c := range got.Categories {
		if c.ID == cat1.ID {
			s.Equal("Work", c.Name)
			s.Equal("#FF5733", c.Color)
		} else if c.ID == cat2.ID {
			s.Equal("Urgent", c.Name)
			s.Equal("#FF0000", c.Color)
		}
	}
}
