package todo_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/service/todo"
)

// --- Mock repository ---

type mockTodoRepo struct {
	insertFn        func(ctx context.Context, t *repository.Todo) error
	updateFn        func(ctx context.Context, t *repository.Todo) error
	deleteFn        func(ctx context.Context, id uuid.UUID) error
	queryByIDFn     func(ctx context.Context, id uuid.UUID) (*repository.Todo, error)
	queryFilteredFn func(ctx context.Context, f repository.TodoFilter) ([]*repository.Todo, int, error)
}

func (m *mockTodoRepo) Insert(ctx context.Context, t *repository.Todo) error {
	if m.insertFn != nil {
		return m.insertFn(ctx, t)
	}
	return nil
}

func (m *mockTodoRepo) Update(ctx context.Context, t *repository.Todo) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, t)
	}
	return nil
}

func (m *mockTodoRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockTodoRepo) QueryByID(ctx context.Context, id uuid.UUID) (*repository.Todo, error) {
	if m.queryByIDFn != nil {
		return m.queryByIDFn(ctx, id)
	}
	return nil, fmt.Errorf("unexpected QueryByID call")
}

func (m *mockTodoRepo) QueryFiltered(ctx context.Context, f repository.TodoFilter) ([]*repository.Todo, int, error) {
	if m.queryFilteredFn != nil {
		return m.queryFilteredFn(ctx, f)
	}
	return nil, 0, fmt.Errorf("unexpected QueryFiltered call")
}

// --- Mock estimator ---

type mockEstimator struct{}

func (m *mockEstimator) EstimateMinutes(ctx context.Context, title, description string) (int, error) {
	return 0, nil
}

// --- Suite ---

type TodoServiceSuite struct {
	suite.Suite
}

func TestTodoService(t *testing.T) {
	suite.Run(t, new(TodoServiceSuite))
}

// --- Constructor tests ---

func (s *TodoServiceSuite) TestNewServiceNilRepo() {
	svc, err := todo.NewService(nil, &mockEstimator{})
	s.Error(err)
	s.Nil(svc)
}

func (s *TodoServiceSuite) TestNewServiceNilEstimator() {
	svc, err := todo.NewService(&mockTodoRepo{}, nil)
	s.Error(err)
	s.Nil(svc)
}

// --- Create tests ---

func (s *TodoServiceSuite) TestCreateSetIDAndCreatedAt() {
	ctx := context.Background()

	var insertedTodo *repository.Todo
	repo := &mockTodoRepo{
		insertFn: func(_ context.Context, t *repository.Todo) error {
			insertedTodo = t
			return nil
		},
		queryByIDFn: func(_ context.Context, id uuid.UUID) (*repository.Todo, error) {
			return &repository.Todo{
				ID:        id,
				Title:     "test task",
				CreatedAt: insertedTodo.CreatedAt,
			}, nil
		},
	}

	svc, err := todo.NewService(repo, &mockEstimator{})
	s.Require().NoError(err)

	input := &repository.Todo{Title: "test task"}
	result, err := svc.Create(ctx, input)
	s.Require().NoError(err)

	// ID should have been set (non-zero)
	s.NotEqual(uuid.Nil, insertedTodo.ID, "Insert should receive a non-zero ID")
	// CreatedAt should have been set
	s.False(insertedTodo.CreatedAt.IsZero(), "Insert should receive a non-zero CreatedAt")
	// Result should be the re-fetched todo
	s.NotNil(result)
	s.Equal("test task", result.Title)
}

func (s *TodoServiceSuite) TestCreatePreservesUserFields() {
	ctx := context.Background()
	dueDate := time.Now().Add(24 * time.Hour)
	estimate := 45

	var insertedTodo *repository.Todo
	repo := &mockTodoRepo{
		insertFn: func(_ context.Context, t *repository.Todo) error {
			insertedTodo = t
			return nil
		},
		queryByIDFn: func(_ context.Context, id uuid.UUID) (*repository.Todo, error) {
			return insertedTodo, nil
		},
	}

	svc, err := todo.NewService(repo, &mockEstimator{})
	s.Require().NoError(err)

	input := &repository.Todo{
		Title:           "important task",
		Description:     "do the thing",
		Priority:        5,
		DueDate:         &dueDate,
		EstimateMinutes: &estimate,
	}
	result, err := svc.Create(ctx, input)
	s.Require().NoError(err)

	s.Equal("important task", result.Title)
	s.Equal("do the thing", result.Description)
	s.Equal(5, result.Priority)
	s.NotNil(result.DueDate)
	s.Equal(dueDate.Unix(), result.DueDate.Unix())
	s.NotNil(result.EstimateMinutes)
	s.Equal(45, *result.EstimateMinutes)
}

// --- Get tests ---

func (s *TodoServiceSuite) TestGet() {
	ctx := context.Background()
	id := uuid.New()
	expected := &repository.Todo{ID: id, Title: "found"}

	repo := &mockTodoRepo{
		queryByIDFn: func(_ context.Context, qid uuid.UUID) (*repository.Todo, error) {
			s.Equal(id, qid)
			return expected, nil
		},
	}

	svc, err := todo.NewService(repo, &mockEstimator{})
	s.Require().NoError(err)

	result, err := svc.Get(ctx, id)
	s.Require().NoError(err)
	s.Equal(expected, result)
}

func (s *TodoServiceSuite) TestGetNotFound() {
	ctx := context.Background()
	id := uuid.New()

	repo := &mockTodoRepo{
		queryByIDFn: func(_ context.Context, _ uuid.UUID) (*repository.Todo, error) {
			return nil, repository.ErrNotFound
		},
	}

	svc, err := todo.NewService(repo, &mockEstimator{})
	s.Require().NoError(err)

	result, err := svc.Get(ctx, id)
	s.ErrorIs(err, repository.ErrNotFound)
	s.Nil(result)
}

// --- List tests ---

func (s *TodoServiceSuite) TestList() {
	ctx := context.Background()
	expected := []*repository.Todo{
		{ID: uuid.New(), Title: "one"},
		{ID: uuid.New(), Title: "two"},
	}
	filter := repository.TodoFilter{Status: "incomplete", Limit: 10}

	repo := &mockTodoRepo{
		queryFilteredFn: func(_ context.Context, f repository.TodoFilter) ([]*repository.Todo, int, error) {
			s.Equal(filter, f)
			return expected, 2, nil
		},
	}

	svc, err := todo.NewService(repo, &mockEstimator{})
	s.Require().NoError(err)

	results, total, err := svc.List(ctx, filter)
	s.Require().NoError(err)
	s.Equal(expected, results)
	s.Equal(2, total)
}

// --- Update tests ---

func (s *TodoServiceSuite) TestUpdate() {
	ctx := context.Background()
	id := uuid.New()
	input := &repository.Todo{ID: id, Title: "updated"}

	var updateCalled bool
	repo := &mockTodoRepo{
		updateFn: func(_ context.Context, t *repository.Todo) error {
			updateCalled = true
			s.Equal(id, t.ID)
			return nil
		},
		queryByIDFn: func(_ context.Context, qid uuid.UUID) (*repository.Todo, error) {
			s.Equal(id, qid)
			return &repository.Todo{ID: id, Title: "updated"}, nil
		},
	}

	svc, err := todo.NewService(repo, &mockEstimator{})
	s.Require().NoError(err)

	result, err := svc.Update(ctx, input)
	s.Require().NoError(err)
	s.True(updateCalled, "repo.Update should have been called")
	s.Equal("updated", result.Title)
}

// --- Delete tests ---

func (s *TodoServiceSuite) TestDelete() {
	ctx := context.Background()
	id := uuid.New()

	var deleteCalled bool
	repo := &mockTodoRepo{
		deleteFn: func(_ context.Context, did uuid.UUID) error {
			deleteCalled = true
			s.Equal(id, did)
			return nil
		},
	}

	svc, err := todo.NewService(repo, &mockEstimator{})
	s.Require().NoError(err)

	err = svc.Delete(ctx, id)
	s.Require().NoError(err)
	s.True(deleteCalled, "repo.Delete should have been called")
}

// --- EffectiveEstimate tests ---

func (s *TodoServiceSuite) TestEffectiveEstimateUserOverridesLLM() {
	user := 45
	llm := 30
	t := &repository.Todo{EstimateMinutes: &user, LLMEstimateMinutes: &llm}
	result := todo.EffectiveEstimate(t)
	s.Require().NotNil(result)
	s.Equal(45, *result)
}

func (s *TodoServiceSuite) TestEffectiveEstimateFallsBackToLLM() {
	llm := 30
	t := &repository.Todo{EstimateMinutes: nil, LLMEstimateMinutes: &llm}
	result := todo.EffectiveEstimate(t)
	s.Require().NotNil(result)
	s.Equal(30, *result)
}

func (s *TodoServiceSuite) TestEffectiveEstimateBothNil() {
	t := &repository.Todo{EstimateMinutes: nil, LLMEstimateMinutes: nil}
	result := todo.EffectiveEstimate(t)
	s.Nil(result)
}
