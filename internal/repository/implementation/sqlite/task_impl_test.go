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

// TaskRepositorySuite tests the SQLite implementation of TaskRepository.
type TaskRepositorySuite struct {
	suite.Suite
}

func TestTask(t *testing.T) {
	suite.Run(t, new(TaskRepositorySuite))
}

// makeTaskRepo creates a category repo and a task repo sharing the same DB file.
// The category repo is created first so the categories table exists for foreign keys.
func (s *TaskRepositorySuite) makeTaskRepo(dbPath string) (*sqlite.SQLiteTaskRepository, *sqlite.SQLiteCategoryRepository) {
	catRepo, err := sqlite.NewSQLiteCategoryRepository(dbPath)
	s.Require().NoError(err)
	s.Require().NotNil(catRepo)

	taskRepo, err := sqlite.NewSQLiteTaskRepository(dbPath)
	s.Require().NoError(err)
	s.Require().NotNil(taskRepo)

	return taskRepo, catRepo
}

func (s *TaskRepositorySuite) TestInsert() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	taskRepo, _ := s.makeTaskRepo(dbPath)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)
	dueDate := now.Add(24 * time.Hour)

	task := &repository.Task{
		ID:          uuid.New(),
		Title:       "Fix the server",
		Description: "The production server needs attention",
		Priority:    2,
		DueDate:     &dueDate,
		CreatedAt:   now,
	}

	err := taskRepo.Insert(ctx, task)
	s.Require().NoError(err)

	got, err := taskRepo.QueryByID(ctx, task.ID)
	s.Require().NoError(err)
	s.Require().NotNil(got)

	s.Equal(task.ID, got.ID)
	s.Equal(task.Title, got.Title)
	s.Equal(task.Description, got.Description)
	s.Equal(task.Priority, got.Priority)
	s.Require().NotNil(got.DueDate)
	s.WithinDuration(*task.DueDate, *got.DueDate, time.Second)
	s.WithinDuration(task.CreatedAt, got.CreatedAt, time.Second)
	s.Nil(got.CompletedAt, "new task should not be completed")
}

func (s *TaskRepositorySuite) TestUpdate() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	taskRepo, _ := s.makeTaskRepo(dbPath)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	task := &repository.Task{
		ID:          uuid.New(),
		Title:       "Original title",
		Description: "Original description",
		Priority:    3,
		CreatedAt:   now,
	}

	err := taskRepo.Insert(ctx, task)
	s.Require().NoError(err)

	// Update fields.
	newDueDate := now.Add(48 * time.Hour)
	task.Title = "Updated title"
	task.Description = "Updated description"
	task.Priority = 1
	task.DueDate = &newDueDate

	err = taskRepo.Update(ctx, task)
	s.Require().NoError(err)

	got, err := taskRepo.QueryByID(ctx, task.ID)
	s.Require().NoError(err)
	s.Require().NotNil(got)

	s.Equal("Updated title", got.Title)
	s.Equal("Updated description", got.Description)
	s.Equal(1, got.Priority)
	s.Require().NotNil(got.DueDate)
	s.WithinDuration(newDueDate, *got.DueDate, time.Second)
}

func (s *TaskRepositorySuite) TestDelete() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	taskRepo, _ := s.makeTaskRepo(dbPath)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	task := &repository.Task{
		ID:          uuid.New(),
		Title:       "To be deleted",
		Description: "This will be removed",
		Priority:    5,
		CreatedAt:   now,
	}

	err := taskRepo.Insert(ctx, task)
	s.Require().NoError(err)

	err = taskRepo.Delete(ctx, task.ID)
	s.Require().NoError(err)

	_, err = taskRepo.QueryByID(ctx, task.ID)
	s.Require().Error(err)
	s.True(errors.Is(err, repository.ErrNotFound),
		"expected error wrapping repository.ErrNotFound, got: %v", err)
}

func (s *TaskRepositorySuite) TestQueryByID() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	taskRepo, _ := s.makeTaskRepo(dbPath)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	task := &repository.Task{
		ID:          uuid.New(),
		Title:       "Query me",
		Description: "Find this task by ID",
		Priority:    4,
		CreatedAt:   now,
	}

	err := taskRepo.Insert(ctx, task)
	s.Require().NoError(err)

	got, err := taskRepo.QueryByID(ctx, task.ID)
	s.Require().NoError(err)
	s.Require().NotNil(got)

	s.Equal(task.ID, got.ID)
	s.Equal(task.Title, got.Title)
	s.Equal(task.Description, got.Description)
	s.Equal(task.Priority, got.Priority)
}

func (s *TaskRepositorySuite) TestQueryByIDNotFound() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	taskRepo, _ := s.makeTaskRepo(dbPath)

	ctx := context.Background()

	randomID := uuid.New()
	got, err := taskRepo.QueryByID(ctx, randomID)
	s.Nil(got, "result should be nil when task not found")
	s.Require().Error(err)
	s.True(errors.Is(err, repository.ErrNotFound),
		"error should wrap repository.ErrNotFound, got: %v", err)
}

func (s *TaskRepositorySuite) TestQueryFilteredDefaultReturnsIncompleteByPriorityDesc() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	taskRepo, _ := s.makeTaskRepo(dbPath)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	// Insert 2 incomplete tasks with different priorities and created_at for secondary sort.
	lowPri := &repository.Task{
		ID:        uuid.New(),
		Title:     "Low priority",
		Priority:  1,
		CreatedAt: now,
	}
	highPri := &repository.Task{
		ID:        uuid.New(),
		Title:     "High priority",
		Priority:  5,
		CreatedAt: now.Add(time.Second),
	}
	// Same priority as highPri but created earlier — should appear first among ties.
	highPriEarlier := &repository.Task{
		ID:        uuid.New(),
		Title:     "High priority earlier",
		Priority:  5,
		CreatedAt: now.Add(-time.Second),
	}

	// Insert 1 completed task — should NOT appear in default (incomplete) filter.
	completedAt := now.Add(time.Hour)
	completed := &repository.Task{
		ID:          uuid.New(),
		Title:       "Completed task",
		Priority:    10,
		CreatedAt:   now,
		CompletedAt: &completedAt,
	}

	s.Require().NoError(taskRepo.Insert(ctx, lowPri))
	s.Require().NoError(taskRepo.Insert(ctx, highPri))
	s.Require().NoError(taskRepo.Insert(ctx, highPriEarlier))
	s.Require().NoError(taskRepo.Insert(ctx, completed))

	// Default filter: status="" defaults to "incomplete".
	results, total, err := taskRepo.QueryFiltered(ctx, repository.TaskFilter{})
	s.Require().NoError(err)
	s.Equal(3, total, "total should count all matching (incomplete) tasks")
	s.Require().Len(results, 3)

	// Priority DESC: 5, 5, 1. Among priority=5, created_at ASC: earlier first.
	s.Equal(highPriEarlier.ID, results[0].ID, "highest priority + earliest created should be first")
	s.Equal(highPri.ID, results[1].ID, "highest priority + later created should be second")
	s.Equal(lowPri.ID, results[2].ID, "lowest priority should be last")
}

func (s *TaskRepositorySuite) TestQueryFilteredStatusComplete() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	taskRepo, _ := s.makeTaskRepo(dbPath)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	incomplete := &repository.Task{
		ID:        uuid.New(),
		Title:     "Incomplete task",
		Priority:  3,
		CreatedAt: now,
	}

	completedAt := now.Add(time.Hour)
	completed := &repository.Task{
		ID:          uuid.New(),
		Title:       "Completed task",
		Priority:    2,
		CreatedAt:   now.Add(time.Second),
		CompletedAt: &completedAt,
	}

	s.Require().NoError(taskRepo.Insert(ctx, incomplete))
	s.Require().NoError(taskRepo.Insert(ctx, completed))

	results, total, err := taskRepo.QueryFiltered(ctx, repository.TaskFilter{Status: "complete"})
	s.Require().NoError(err)
	s.Equal(1, total)
	s.Require().Len(results, 1)
	s.Equal(completed.ID, results[0].ID, "should only return completed tasks")
}

func (s *TaskRepositorySuite) TestQueryFilteredStatusAll() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	taskRepo, _ := s.makeTaskRepo(dbPath)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	incomplete := &repository.Task{
		ID:        uuid.New(),
		Title:     "Incomplete",
		Priority:  1,
		CreatedAt: now,
	}

	completedAt := now.Add(time.Hour)
	completed := &repository.Task{
		ID:          uuid.New(),
		Title:       "Completed",
		Priority:    5,
		CreatedAt:   now.Add(time.Second),
		CompletedAt: &completedAt,
	}

	s.Require().NoError(taskRepo.Insert(ctx, incomplete))
	s.Require().NoError(taskRepo.Insert(ctx, completed))

	results, total, err := taskRepo.QueryFiltered(ctx, repository.TaskFilter{Status: "all"})
	s.Require().NoError(err)
	s.Equal(2, total, "total should include both incomplete and completed")
	s.Require().Len(results, 2)

	// Priority DESC: completed (5) before incomplete (1).
	s.Equal(completed.ID, results[0].ID, "higher priority should be first")
	s.Equal(incomplete.ID, results[1].ID, "lower priority should be second")
}

// insertCategory is a test helper that creates a category row directly
// via the category repo so tasks can FK-reference it.
func (s *TaskRepositorySuite) insertCategory(catRepo *sqlite.SQLiteCategoryRepository, key string) {
	s.T().Helper()
	err := catRepo.Insert(context.Background(), &repository.Category{
		NameKey:   key,
		CreatedAt: time.Now().UTC().Truncate(time.Second),
	})
	s.Require().NoError(err)
}

func (s *TaskRepositorySuite) TestInsertWithCategoryKey() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	taskRepo, catRepo := s.makeTaskRepo(dbPath)
	s.insertCategory(catRepo, "work")

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)
	key := "work"

	task := &repository.Task{
		ID:          uuid.New(),
		Title:       "Categorized task",
		Priority:    1,
		CategoryKey: &key,
		CreatedAt:   now,
	}
	s.Require().NoError(taskRepo.Insert(ctx, task))

	got, err := taskRepo.QueryByID(ctx, task.ID)
	s.Require().NoError(err)
	s.Require().NotNil(got)
	s.Require().NotNil(got.CategoryKey, "CategoryKey should round-trip non-nil")
	s.Equal("work", *got.CategoryKey)
}

func (s *TaskRepositorySuite) TestInsertWithNilCategoryKey() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	taskRepo, _ := s.makeTaskRepo(dbPath)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	task := &repository.Task{
		ID:          uuid.New(),
		Title:       "Uncategorized task",
		Priority:    1,
		CategoryKey: nil,
		CreatedAt:   now,
	}
	s.Require().NoError(taskRepo.Insert(ctx, task))

	got, err := taskRepo.QueryByID(ctx, task.ID)
	s.Require().NoError(err)
	s.Require().NotNil(got)
	s.Nil(got.CategoryKey, "nil CategoryKey should round-trip as nil")
}

func (s *TaskRepositorySuite) TestInsertWithUnknownCategoryKey() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	taskRepo, _ := s.makeTaskRepo(dbPath)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)
	missing := "no_such_category"

	task := &repository.Task{
		ID:          uuid.New(),
		Title:       "Bad FK",
		Priority:    1,
		CategoryKey: &missing,
		CreatedAt:   now,
	}
	err := taskRepo.Insert(ctx, task)
	s.Require().Error(err, "inserting with unknown category_key should fail FK constraint")
}

func (s *TaskRepositorySuite) TestUpdateChangesCategoryKey() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	taskRepo, catRepo := s.makeTaskRepo(dbPath)
	s.insertCategory(catRepo, "work")
	s.insertCategory(catRepo, "home")

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)
	keyA := "work"
	keyB := "home"

	task := &repository.Task{
		ID:          uuid.New(),
		Title:       "Switcheroo",
		Priority:    1,
		CategoryKey: &keyA,
		CreatedAt:   now,
	}
	s.Require().NoError(taskRepo.Insert(ctx, task))

	task.CategoryKey = &keyB
	s.Require().NoError(taskRepo.Update(ctx, task))

	got, err := taskRepo.QueryByID(ctx, task.ID)
	s.Require().NoError(err)
	s.Require().NotNil(got.CategoryKey)
	s.Equal("home", *got.CategoryKey, "Update should change category_key from work -> home")
}

func (s *TaskRepositorySuite) TestUpdateClearsCategoryKey() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	taskRepo, catRepo := s.makeTaskRepo(dbPath)
	s.insertCategory(catRepo, "work")

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)
	key := "work"

	task := &repository.Task{
		ID:          uuid.New(),
		Title:       "To be uncategorized",
		Priority:    1,
		CategoryKey: &key,
		CreatedAt:   now,
	}
	s.Require().NoError(taskRepo.Insert(ctx, task))

	task.CategoryKey = nil
	s.Require().NoError(taskRepo.Update(ctx, task))

	got, err := taskRepo.QueryByID(ctx, task.ID)
	s.Require().NoError(err)
	s.Nil(got.CategoryKey, "Update with nil CategoryKey should clear category_key")
}

func (s *TaskRepositorySuite) TestQueryFilteredByCategoryKey() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	taskRepo, catRepo := s.makeTaskRepo(dbPath)
	s.insertCategory(catRepo, "work")
	s.insertCategory(catRepo, "home")

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)
	work := "work"
	home := "home"

	t1 := &repository.Task{ID: uuid.New(), Title: "w1", Priority: 3, CategoryKey: &work, CreatedAt: now}
	t2 := &repository.Task{ID: uuid.New(), Title: "w2", Priority: 2, CategoryKey: &work, CreatedAt: now.Add(time.Second)}
	t3 := &repository.Task{ID: uuid.New(), Title: "h1", Priority: 1, CategoryKey: &home, CreatedAt: now.Add(2 * time.Second)}

	s.Require().NoError(taskRepo.Insert(ctx, t1))
	s.Require().NoError(taskRepo.Insert(ctx, t2))
	s.Require().NoError(taskRepo.Insert(ctx, t3))

	results, total, err := taskRepo.QueryFiltered(ctx, repository.TaskFilter{CategoryKey: "work"})
	s.Require().NoError(err)
	s.Equal(2, total, "only the 2 work tasks should match")
	s.Require().Len(results, 2)

	for _, r := range results {
		s.Require().NotNil(r.CategoryKey)
		s.Equal("work", *r.CategoryKey)
	}

	allResults, allTotal, err := taskRepo.QueryFiltered(ctx, repository.TaskFilter{})
	s.Require().NoError(err)
	s.Equal(3, allTotal, "empty filter should return all 3 incomplete tasks")
	s.Require().Len(allResults, 3)
}

func (s *TaskRepositorySuite) TestRenameCategoryCascadesToTasks() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	taskRepo, catRepo := s.makeTaskRepo(dbPath)
	s.insertCategory(catRepo, "work")

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)
	key := "work"

	task := &repository.Task{
		ID:          uuid.New(),
		Title:       "Cascading rename",
		Priority:    1,
		CategoryKey: &key,
		CreatedAt:   now,
	}
	s.Require().NoError(taskRepo.Insert(ctx, task))

	// Rename the category — ON UPDATE CASCADE should propagate.
	s.Require().NoError(catRepo.Rename(ctx, "work", "professional"))

	got, err := taskRepo.QueryByID(ctx, task.ID)
	s.Require().NoError(err)
	s.Require().NotNil(got.CategoryKey, "task should still reference a category after rename")
	s.Equal("professional", *got.CategoryKey, "ON UPDATE CASCADE should propagate the new key")
}

func (s *TaskRepositorySuite) TestDeleteCategorySetsTasksCategoryToNull() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	taskRepo, catRepo := s.makeTaskRepo(dbPath)
	s.insertCategory(catRepo, "work")

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)
	key := "work"

	task := &repository.Task{
		ID:          uuid.New(),
		Title:       "Will outlive its category",
		Priority:    1,
		CategoryKey: &key,
		CreatedAt:   now,
	}
	s.Require().NoError(taskRepo.Insert(ctx, task))

	s.Require().NoError(catRepo.Delete(ctx, "work"))

	got, err := taskRepo.QueryByID(ctx, task.ID)
	s.Require().NoError(err, "task should still exist after its category is deleted")
	s.Require().NotNil(got)
	s.Nil(got.CategoryKey, "ON DELETE SET NULL should clear category_key")
}

func (s *TaskRepositorySuite) TestQueryFilteredSearchMatchesTitleOrDescription() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	taskRepo, _ := s.makeTaskRepo(dbPath)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	// Title matches search term.
	titleMatch := &repository.Task{
		ID:          uuid.New(),
		Title:       "Fix the deployment pipeline",
		Description: "No relevant info here",
		Priority:    3,
		CreatedAt:   now,
	}
	// Description matches search term.
	descMatch := &repository.Task{
		ID:          uuid.New(),
		Title:       "Server maintenance",
		Description: "Update the deployment scripts",
		Priority:    2,
		CreatedAt:   now.Add(time.Second),
	}
	// Neither matches.
	noMatch := &repository.Task{
		ID:          uuid.New(),
		Title:       "Buy groceries",
		Description: "Milk, eggs, bread",
		Priority:    1,
		CreatedAt:   now.Add(2 * time.Second),
	}

	s.Require().NoError(taskRepo.Insert(ctx, titleMatch))
	s.Require().NoError(taskRepo.Insert(ctx, descMatch))
	s.Require().NoError(taskRepo.Insert(ctx, noMatch))

	// Case-insensitive search for "DEPLOYMENT".
	results, total, err := taskRepo.QueryFiltered(ctx, repository.TaskFilter{Search: "DEPLOYMENT"})
	s.Require().NoError(err)
	s.Equal(2, total, "should match title and description case-insensitively")
	s.Require().Len(results, 2)

	// Priority DESC: titleMatch (3) before descMatch (2).
	s.Equal(titleMatch.ID, results[0].ID)
	s.Equal(descMatch.ID, results[1].ID)
}

func (s *TaskRepositorySuite) TestQueryFilteredPagination() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	taskRepo, _ := s.makeTaskRepo(dbPath)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	// Insert 5 tasks with distinct priorities for deterministic ordering.
	ids := make([]uuid.UUID, 5)
	for i := 0; i < 5; i++ {
		ids[i] = uuid.New()
		task := &repository.Task{
			ID:        ids[i],
			Title:     "Task",
			Priority:  5 - i, // priorities: 5, 4, 3, 2, 1
			CreatedAt: now.Add(time.Duration(i) * time.Second),
		}
		s.Require().NoError(taskRepo.Insert(ctx, task))
	}

	// Page 1: limit=2, offset=0.
	results, total, err := taskRepo.QueryFiltered(ctx, repository.TaskFilter{Limit: 2, Offset: 0})
	s.Require().NoError(err)
	s.Equal(5, total, "total count should reflect all matching tasks, not page size")
	s.Require().Len(results, 2, "should return exactly limit items")
	// Priority DESC: 5, 4.
	s.Equal(ids[0], results[0].ID, "first page first item should be priority 5")
	s.Equal(ids[1], results[1].ID, "first page second item should be priority 4")

	// Page 2: limit=2, offset=2.
	results2, total2, err := taskRepo.QueryFiltered(ctx, repository.TaskFilter{Limit: 2, Offset: 2})
	s.Require().NoError(err)
	s.Equal(5, total2, "total count should be same regardless of offset")
	s.Require().Len(results2, 2)
	// Priority DESC: 3, 2.
	s.Equal(ids[2], results2[0].ID)
	s.Equal(ids[3], results2[1].ID)

	// Page 3: limit=2, offset=4 — only 1 remaining.
	results3, total3, err := taskRepo.QueryFiltered(ctx, repository.TaskFilter{Limit: 2, Offset: 4})
	s.Require().NoError(err)
	s.Equal(5, total3)
	s.Require().Len(results3, 1, "last page should have remaining items only")
	s.Equal(ids[4], results3[0].ID)
}

func (s *TaskRepositorySuite) TestQueryFilteredTotalCountUnaffectedByLimitOffset() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	taskRepo, _ := s.makeTaskRepo(dbPath)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	for i := 0; i < 10; i++ {
		task := &repository.Task{
			ID:        uuid.New(),
			Title:     "Task",
			Priority:  i,
			CreatedAt: now.Add(time.Duration(i) * time.Second),
		}
		s.Require().NoError(taskRepo.Insert(ctx, task))
	}

	// Request with small limit and large offset.
	_, total, err := taskRepo.QueryFiltered(ctx, repository.TaskFilter{Limit: 3, Offset: 0})
	s.Require().NoError(err)
	s.Equal(10, total, "total should be 10 regardless of limit")

	_, total2, err := taskRepo.QueryFiltered(ctx, repository.TaskFilter{Limit: 3, Offset: 6})
	s.Require().NoError(err)
	s.Equal(10, total2, "total should be 10 regardless of offset")
}

func (s *TaskRepositorySuite) TestQueryFilteredPrioritySortHigherFirst() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	taskRepo, _ := s.makeTaskRepo(dbPath)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	// Create tasks with priorities 1, 5, 3 — should sort as 5, 3, 1.
	pri1 := &repository.Task{
		ID:        uuid.New(),
		Title:     "Priority 1",
		Priority:  1,
		CreatedAt: now,
	}
	pri5 := &repository.Task{
		ID:        uuid.New(),
		Title:     "Priority 5",
		Priority:  5,
		CreatedAt: now.Add(time.Second),
	}
	pri3 := &repository.Task{
		ID:        uuid.New(),
		Title:     "Priority 3",
		Priority:  3,
		CreatedAt: now.Add(2 * time.Second),
	}

	s.Require().NoError(taskRepo.Insert(ctx, pri1))
	s.Require().NoError(taskRepo.Insert(ctx, pri5))
	s.Require().NoError(taskRepo.Insert(ctx, pri3))

	results, _, err := taskRepo.QueryFiltered(ctx, repository.TaskFilter{})
	s.Require().NoError(err)
	s.Require().Len(results, 3)

	// Higher priority value = higher priority, so DESC order.
	s.Equal(pri5.ID, results[0].ID, "priority 5 should be first (highest)")
	s.Equal(pri3.ID, results[1].ID, "priority 3 should be second")
	s.Equal(pri1.ID, results[2].ID, "priority 1 should be last (lowest)")
}

func (s *TaskRepositorySuite) TestComplete() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	taskRepo, _ := s.makeTaskRepo(dbPath)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	task := &repository.Task{
		ID:          uuid.New(),
		Title:       "Complete me",
		Description: "Should be marked done",
		Priority:    3,
		CreatedAt:   now,
	}

	err := taskRepo.Insert(ctx, task)
	s.Require().NoError(err)

	// Verify not completed initially.
	got, err := taskRepo.QueryByID(ctx, task.ID)
	s.Require().NoError(err)
	s.Nil(got.CompletedAt, "should not be completed initially")

	// Complete the task.
	completedAt := now.Add(2 * time.Hour)
	err = taskRepo.Complete(ctx, task.ID, completedAt)
	s.Require().NoError(err)

	// Verify completed.
	got, err = taskRepo.QueryByID(ctx, task.ID)
	s.Require().NoError(err)
	s.Require().NotNil(got.CompletedAt, "should be completed after calling Complete")
	s.WithinDuration(completedAt, *got.CompletedAt, time.Second)
}

func (s *TaskRepositorySuite) TestEstimateFieldsRoundTrip() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	taskRepo, _ := s.makeTaskRepo(dbPath)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	userEstimate := 30
	llmEstimate := 45

	// Insert a task with both estimate fields set.
	taskWithEstimates := &repository.Task{
		ID:                 uuid.New(),
		Title:              "Task with estimates",
		Description:        "Has both user and LLM estimates",
		Priority:           2,
		EstimateMinutes:    &userEstimate,
		LLMEstimateMinutes: &llmEstimate,
		CreatedAt:          now,
	}

	err := taskRepo.Insert(ctx, taskWithEstimates)
	s.Require().NoError(err)

	got, err := taskRepo.QueryByID(ctx, taskWithEstimates.ID)
	s.Require().NoError(err)
	s.Require().NotNil(got)

	s.Require().NotNil(got.EstimateMinutes, "EstimateMinutes should round-trip as non-nil")
	s.Equal(30, *got.EstimateMinutes)
	s.Require().NotNil(got.LLMEstimateMinutes, "LLMEstimateMinutes should round-trip as non-nil")
	s.Equal(45, *got.LLMEstimateMinutes)

	// Insert a task with nil estimate fields.
	taskNilEstimates := &repository.Task{
		ID:          uuid.New(),
		Title:       "Task without estimates",
		Description: "Estimates are nil",
		Priority:    3,
		CreatedAt:   now,
	}

	err = taskRepo.Insert(ctx, taskNilEstimates)
	s.Require().NoError(err)

	got2, err := taskRepo.QueryByID(ctx, taskNilEstimates.ID)
	s.Require().NoError(err)
	s.Require().NotNil(got2)

	s.Nil(got2.EstimateMinutes, "EstimateMinutes should round-trip as nil")
	s.Nil(got2.LLMEstimateMinutes, "LLMEstimateMinutes should round-trip as nil")
}
