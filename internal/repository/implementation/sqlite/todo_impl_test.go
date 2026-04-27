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

func (s *TodoRepositorySuite) TestQueryFilteredDefaultReturnsIncompleteByPriorityDesc() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	todoRepo, _ := s.makeTodoRepo(dbPath)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	// Insert 2 incomplete todos with different priorities and created_at for secondary sort.
	lowPri := &repository.Todo{
		ID:        uuid.New(),
		Title:     "Low priority",
		Priority:  1,
		CreatedAt: now,
	}
	highPri := &repository.Todo{
		ID:        uuid.New(),
		Title:     "High priority",
		Priority:  5,
		CreatedAt: now.Add(time.Second),
	}
	// Same priority as highPri but created earlier — should appear first among ties.
	highPriEarlier := &repository.Todo{
		ID:        uuid.New(),
		Title:     "High priority earlier",
		Priority:  5,
		CreatedAt: now.Add(-time.Second),
	}

	// Insert 1 completed todo — should NOT appear in default (incomplete) filter.
	completedAt := now.Add(time.Hour)
	completed := &repository.Todo{
		ID:          uuid.New(),
		Title:       "Completed todo",
		Priority:    10,
		CreatedAt:   now,
		CompletedAt: &completedAt,
	}

	s.Require().NoError(todoRepo.Insert(ctx, lowPri))
	s.Require().NoError(todoRepo.Insert(ctx, highPri))
	s.Require().NoError(todoRepo.Insert(ctx, highPriEarlier))
	s.Require().NoError(todoRepo.Insert(ctx, completed))

	// Default filter: status="" defaults to "incomplete".
	results, total, err := todoRepo.QueryFiltered(ctx, repository.TodoFilter{})
	s.Require().NoError(err)
	s.Equal(3, total, "total should count all matching (incomplete) todos")
	s.Require().Len(results, 3)

	// Priority DESC: 5, 5, 1. Among priority=5, created_at ASC: earlier first.
	s.Equal(highPriEarlier.ID, results[0].ID, "highest priority + earliest created should be first")
	s.Equal(highPri.ID, results[1].ID, "highest priority + later created should be second")
	s.Equal(lowPri.ID, results[2].ID, "lowest priority should be last")
}

func (s *TodoRepositorySuite) TestQueryFilteredStatusComplete() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	todoRepo, _ := s.makeTodoRepo(dbPath)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	incomplete := &repository.Todo{
		ID:        uuid.New(),
		Title:     "Incomplete task",
		Priority:  3,
		CreatedAt: now,
	}

	completedAt := now.Add(time.Hour)
	completed := &repository.Todo{
		ID:          uuid.New(),
		Title:       "Completed task",
		Priority:    2,
		CreatedAt:   now.Add(time.Second),
		CompletedAt: &completedAt,
	}

	s.Require().NoError(todoRepo.Insert(ctx, incomplete))
	s.Require().NoError(todoRepo.Insert(ctx, completed))

	results, total, err := todoRepo.QueryFiltered(ctx, repository.TodoFilter{Status: "complete"})
	s.Require().NoError(err)
	s.Equal(1, total)
	s.Require().Len(results, 1)
	s.Equal(completed.ID, results[0].ID, "should only return completed todos")
}

func (s *TodoRepositorySuite) TestQueryFilteredStatusAll() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	todoRepo, _ := s.makeTodoRepo(dbPath)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	incomplete := &repository.Todo{
		ID:        uuid.New(),
		Title:     "Incomplete",
		Priority:  1,
		CreatedAt: now,
	}

	completedAt := now.Add(time.Hour)
	completed := &repository.Todo{
		ID:          uuid.New(),
		Title:       "Completed",
		Priority:    5,
		CreatedAt:   now.Add(time.Second),
		CompletedAt: &completedAt,
	}

	s.Require().NoError(todoRepo.Insert(ctx, incomplete))
	s.Require().NoError(todoRepo.Insert(ctx, completed))

	results, total, err := todoRepo.QueryFiltered(ctx, repository.TodoFilter{Status: "all"})
	s.Require().NoError(err)
	s.Equal(2, total, "total should include both incomplete and completed")
	s.Require().Len(results, 2)

	// Priority DESC: completed (5) before incomplete (1).
	s.Equal(completed.ID, results[0].ID, "higher priority should be first")
	s.Equal(incomplete.ID, results[1].ID, "lower priority should be second")
}

func (s *TodoRepositorySuite) TestQueryFilteredCategoryFilter() {
	s.T().Skip("rewritten in Feature 109 Loop 4 against the new category_key FK")
}

func (s *TodoRepositorySuite) TestQueryFilteredSearchMatchesTitleOrDescription() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	todoRepo, _ := s.makeTodoRepo(dbPath)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	// Title matches search term.
	titleMatch := &repository.Todo{
		ID:          uuid.New(),
		Title:       "Fix the deployment pipeline",
		Description: "No relevant info here",
		Priority:    3,
		CreatedAt:   now,
	}
	// Description matches search term.
	descMatch := &repository.Todo{
		ID:          uuid.New(),
		Title:       "Server maintenance",
		Description: "Update the deployment scripts",
		Priority:    2,
		CreatedAt:   now.Add(time.Second),
	}
	// Neither matches.
	noMatch := &repository.Todo{
		ID:          uuid.New(),
		Title:       "Buy groceries",
		Description: "Milk, eggs, bread",
		Priority:    1,
		CreatedAt:   now.Add(2 * time.Second),
	}

	s.Require().NoError(todoRepo.Insert(ctx, titleMatch))
	s.Require().NoError(todoRepo.Insert(ctx, descMatch))
	s.Require().NoError(todoRepo.Insert(ctx, noMatch))

	// Case-insensitive search for "DEPLOYMENT".
	results, total, err := todoRepo.QueryFiltered(ctx, repository.TodoFilter{Search: "DEPLOYMENT"})
	s.Require().NoError(err)
	s.Equal(2, total, "should match title and description case-insensitively")
	s.Require().Len(results, 2)

	// Priority DESC: titleMatch (3) before descMatch (2).
	s.Equal(titleMatch.ID, results[0].ID)
	s.Equal(descMatch.ID, results[1].ID)
}

func (s *TodoRepositorySuite) TestQueryFilteredPagination() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	todoRepo, _ := s.makeTodoRepo(dbPath)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	// Insert 5 todos with distinct priorities for deterministic ordering.
	ids := make([]uuid.UUID, 5)
	for i := 0; i < 5; i++ {
		ids[i] = uuid.New()
		todo := &repository.Todo{
			ID:        ids[i],
			Title:     "Task",
			Priority:  5 - i, // priorities: 5, 4, 3, 2, 1
			CreatedAt: now.Add(time.Duration(i) * time.Second),
		}
		s.Require().NoError(todoRepo.Insert(ctx, todo))
	}

	// Page 1: limit=2, offset=0.
	results, total, err := todoRepo.QueryFiltered(ctx, repository.TodoFilter{Limit: 2, Offset: 0})
	s.Require().NoError(err)
	s.Equal(5, total, "total count should reflect all matching todos, not page size")
	s.Require().Len(results, 2, "should return exactly limit items")
	// Priority DESC: 5, 4.
	s.Equal(ids[0], results[0].ID, "first page first item should be priority 5")
	s.Equal(ids[1], results[1].ID, "first page second item should be priority 4")

	// Page 2: limit=2, offset=2.
	results2, total2, err := todoRepo.QueryFiltered(ctx, repository.TodoFilter{Limit: 2, Offset: 2})
	s.Require().NoError(err)
	s.Equal(5, total2, "total count should be same regardless of offset")
	s.Require().Len(results2, 2)
	// Priority DESC: 3, 2.
	s.Equal(ids[2], results2[0].ID)
	s.Equal(ids[3], results2[1].ID)

	// Page 3: limit=2, offset=4 — only 1 remaining.
	results3, total3, err := todoRepo.QueryFiltered(ctx, repository.TodoFilter{Limit: 2, Offset: 4})
	s.Require().NoError(err)
	s.Equal(5, total3)
	s.Require().Len(results3, 1, "last page should have remaining items only")
	s.Equal(ids[4], results3[0].ID)
}

func (s *TodoRepositorySuite) TestQueryFilteredTotalCountUnaffectedByLimitOffset() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	todoRepo, _ := s.makeTodoRepo(dbPath)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	for i := 0; i < 10; i++ {
		todo := &repository.Todo{
			ID:        uuid.New(),
			Title:     "Task",
			Priority:  i,
			CreatedAt: now.Add(time.Duration(i) * time.Second),
		}
		s.Require().NoError(todoRepo.Insert(ctx, todo))
	}

	// Request with small limit and large offset.
	_, total, err := todoRepo.QueryFiltered(ctx, repository.TodoFilter{Limit: 3, Offset: 0})
	s.Require().NoError(err)
	s.Equal(10, total, "total should be 10 regardless of limit")

	_, total2, err := todoRepo.QueryFiltered(ctx, repository.TodoFilter{Limit: 3, Offset: 6})
	s.Require().NoError(err)
	s.Equal(10, total2, "total should be 10 regardless of offset")
}

func (s *TodoRepositorySuite) TestQueryFilteredPrioritySortHigherFirst() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	todoRepo, _ := s.makeTodoRepo(dbPath)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	// Create todos with priorities 1, 5, 3 — should sort as 5, 3, 1.
	pri1 := &repository.Todo{
		ID:        uuid.New(),
		Title:     "Priority 1",
		Priority:  1,
		CreatedAt: now,
	}
	pri5 := &repository.Todo{
		ID:        uuid.New(),
		Title:     "Priority 5",
		Priority:  5,
		CreatedAt: now.Add(time.Second),
	}
	pri3 := &repository.Todo{
		ID:        uuid.New(),
		Title:     "Priority 3",
		Priority:  3,
		CreatedAt: now.Add(2 * time.Second),
	}

	s.Require().NoError(todoRepo.Insert(ctx, pri1))
	s.Require().NoError(todoRepo.Insert(ctx, pri5))
	s.Require().NoError(todoRepo.Insert(ctx, pri3))

	results, _, err := todoRepo.QueryFiltered(ctx, repository.TodoFilter{})
	s.Require().NoError(err)
	s.Require().Len(results, 3)

	// Higher priority value = higher priority, so DESC order.
	s.Equal(pri5.ID, results[0].ID, "priority 5 should be first (highest)")
	s.Equal(pri3.ID, results[1].ID, "priority 3 should be second")
	s.Equal(pri1.ID, results[2].ID, "priority 1 should be last (lowest)")
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

func (s *TodoRepositorySuite) TestEstimateFieldsRoundTrip() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	todoRepo, _ := s.makeTodoRepo(dbPath)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	userEstimate := 30
	llmEstimate := 45

	// Insert a todo with both estimate fields set.
	todoWithEstimates := &repository.Todo{
		ID:                 uuid.New(),
		Title:              "Task with estimates",
		Description:        "Has both user and LLM estimates",
		Priority:           2,
		EstimateMinutes:    &userEstimate,
		LLMEstimateMinutes: &llmEstimate,
		CreatedAt:          now,
	}

	err := todoRepo.Insert(ctx, todoWithEstimates)
	s.Require().NoError(err)

	got, err := todoRepo.QueryByID(ctx, todoWithEstimates.ID)
	s.Require().NoError(err)
	s.Require().NotNil(got)

	s.Require().NotNil(got.EstimateMinutes, "EstimateMinutes should round-trip as non-nil")
	s.Equal(30, *got.EstimateMinutes)
	s.Require().NotNil(got.LLMEstimateMinutes, "LLMEstimateMinutes should round-trip as non-nil")
	s.Equal(45, *got.LLMEstimateMinutes)

	// Insert a todo with nil estimate fields.
	todoNilEstimates := &repository.Todo{
		ID:          uuid.New(),
		Title:       "Task without estimates",
		Description: "Estimates are nil",
		Priority:    3,
		CreatedAt:   now,
	}

	err = todoRepo.Insert(ctx, todoNilEstimates)
	s.Require().NoError(err)

	got2, err := todoRepo.QueryByID(ctx, todoNilEstimates.ID)
	s.Require().NoError(err)
	s.Require().NotNil(got2)

	s.Nil(got2.EstimateMinutes, "EstimateMinutes should round-trip as nil")
	s.Nil(got2.LLMEstimateMinutes, "LLMEstimateMinutes should round-trip as nil")
}

func (s *TodoRepositorySuite) TestCategoriesAssociation() {
	s.T().Skip("rewritten in Feature 109 Loop 4 against the new category_key FK")
}
