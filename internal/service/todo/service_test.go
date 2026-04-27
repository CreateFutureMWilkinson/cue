package todo_test

import (
	"context"
	"fmt"
	"sync"
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

// trackingEstimator records calls and signals when estimation completes.
type trackingEstimator struct {
	mu         sync.Mutex
	callCount  int
	lastTitle  string
	lastDesc   string
	result     int
	err        error
	calledCh   chan struct{} // closed on first call
	closedOnce sync.Once
}

func newTrackingEstimator(result int, err error) *trackingEstimator {
	return &trackingEstimator{
		result:   result,
		err:      err,
		calledCh: make(chan struct{}),
	}
}

func (e *trackingEstimator) EstimateMinutes(ctx context.Context, title, description string) (int, error) {
	e.mu.Lock()
	e.callCount++
	e.lastTitle = title
	e.lastDesc = description
	e.mu.Unlock()
	e.closedOnce.Do(func() { close(e.calledCh) })
	return e.result, e.err
}

func (e *trackingEstimator) called() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.callCount > 0
}

// inMemoryTodoRepo stores todos in memory so async updates are visible.
type inMemoryTodoRepo struct {
	mu    sync.Mutex
	store map[uuid.UUID]*repository.Todo
}

func newInMemoryTodoRepo() *inMemoryTodoRepo {
	return &inMemoryTodoRepo{store: make(map[uuid.UUID]*repository.Todo)}
}

func (r *inMemoryTodoRepo) Insert(_ context.Context, t *repository.Todo) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *t
	r.store[t.ID] = &cp
	return nil
}

func (r *inMemoryTodoRepo) Update(_ context.Context, t *repository.Todo) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.store[t.ID]; !ok {
		return repository.ErrNotFound
	}
	cp := *t
	r.store[t.ID] = &cp
	return nil
}

func (r *inMemoryTodoRepo) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.store, id)
	return nil
}

func (r *inMemoryTodoRepo) QueryByID(_ context.Context, id uuid.UUID) (*repository.Todo, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.store[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *t
	return &cp, nil
}

func (r *inMemoryTodoRepo) QueryFiltered(_ context.Context, _ repository.TodoFilter) ([]*repository.Todo, int, error) {
	return nil, 0, nil
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

// --- Async estimation tests ---

func (s *TodoServiceSuite) TestCreateTriggersAsyncEstimation() {
	ctx := context.Background()
	repo := newInMemoryTodoRepo()
	estimator := newTrackingEstimator(25, nil)

	svc, err := todo.NewService(repo, estimator)
	s.Require().NoError(err)

	// Create a todo with no user estimate — should trigger async LLM estimation.
	input := &repository.Todo{
		Title:       "write quarterly report",
		Description: "compile metrics and narrative for Q2",
	}
	created, err := svc.Create(ctx, input)
	s.Require().NoError(err)
	s.NotEqual(uuid.Nil, created.ID)

	// Wait for the async estimator to be called (short timeout).
	select {
	case <-estimator.calledCh:
		// good — estimator was invoked
	case <-time.After(500 * time.Millisecond):
		s.Fail("estimator was not called within timeout — Create should trigger async estimation when EstimateMinutes is nil")
	}

	// After estimation, the repo should have LLMEstimateMinutes set.
	// Give a small window for the async goroutine to persist the update.
	time.Sleep(50 * time.Millisecond)

	updated, err := repo.QueryByID(ctx, created.ID)
	s.Require().NoError(err)
	s.Require().NotNil(updated.LLMEstimateMinutes, "LLMEstimateMinutes should be set after async estimation")
	s.Equal(25, *updated.LLMEstimateMinutes)
}

func (s *TodoServiceSuite) TestCreateSkipsEstimationWhenUserEstimateProvided() {
	ctx := context.Background()
	repo := newInMemoryTodoRepo()
	estimator := newTrackingEstimator(25, nil)

	svc, err := todo.NewService(repo, estimator)
	s.Require().NoError(err)

	// Create a todo WITH a user estimate — should NOT trigger LLM estimation.
	userEst := 45
	input := &repository.Todo{
		Title:           "plan team offsite",
		EstimateMinutes: &userEst,
	}
	_, err = svc.Create(ctx, input)
	s.Require().NoError(err)

	// Wait a bit to ensure estimator is NOT called.
	time.Sleep(200 * time.Millisecond)

	s.False(estimator.called(), "estimator should NOT be called when user provides EstimateMinutes > 0")
}

func (s *TodoServiceSuite) TestUpdateTriggersReEstimationWhenEstimateCleared() {
	ctx := context.Background()
	repo := newInMemoryTodoRepo()
	estimator := newTrackingEstimator(30, nil)

	svc, err := todo.NewService(repo, estimator)
	s.Require().NoError(err)

	// Pre-seed a todo with a user estimate and an existing LLM estimate.
	userEst := 60
	llmEst := 55
	id := uuid.New()
	existing := &repository.Todo{
		ID:                 id,
		Title:              "refactor auth module",
		Description:        "extract middleware, add tests",
		EstimateMinutes:    &userEst,
		LLMEstimateMinutes: &llmEst,
		CreatedAt:          time.Now(),
	}
	err = repo.Insert(ctx, existing)
	s.Require().NoError(err)

	// Update: clear the user estimate (set to nil).
	updateInput := &repository.Todo{
		ID:              id,
		Title:           "refactor auth module",
		Description:     "extract middleware, add tests",
		EstimateMinutes: nil, // cleared
		CreatedAt:       existing.CreatedAt,
	}
	_, err = svc.Update(ctx, updateInput)
	s.Require().NoError(err)

	// Wait for the async re-estimator to be called.
	select {
	case <-estimator.calledCh:
		// good
	case <-time.After(500 * time.Millisecond):
		s.Fail("estimator was not called within timeout — Update should trigger re-estimation when EstimateMinutes is cleared")
	}

	// After re-estimation, LLMEstimateMinutes should be updated to the new value.
	time.Sleep(50 * time.Millisecond)

	refreshed, err := repo.QueryByID(ctx, id)
	s.Require().NoError(err)
	s.Require().NotNil(refreshed.LLMEstimateMinutes, "LLMEstimateMinutes should be set after re-estimation")
	s.Equal(30, *refreshed.LLMEstimateMinutes)
}

func (s *TodoServiceSuite) TestUpdateSkipsEstimationWhenEstimateStaysNonZero() {
	ctx := context.Background()
	repo := newInMemoryTodoRepo()
	estimator := newTrackingEstimator(30, nil)

	svc, err := todo.NewService(repo, estimator)
	s.Require().NoError(err)

	// Pre-seed a todo with a user estimate.
	userEst := 45
	id := uuid.New()
	existing := &repository.Todo{
		ID:              id,
		Title:           "deploy staging env",
		EstimateMinutes: &userEst,
		CreatedAt:       time.Now(),
	}
	err = repo.Insert(ctx, existing)
	s.Require().NoError(err)

	// Update with estimate staying non-zero — no estimation should trigger.
	newEst := 45
	updateInput := &repository.Todo{
		ID:              id,
		Title:           "deploy staging env",
		EstimateMinutes: &newEst,
		CreatedAt:       existing.CreatedAt,
	}
	_, err = svc.Update(ctx, updateInput)
	s.Require().NoError(err)

	// Wait a bit to ensure estimator is NOT called.
	time.Sleep(200 * time.Millisecond)

	s.False(estimator.called(), "estimator should NOT be called when EstimateMinutes stays non-zero")
}
