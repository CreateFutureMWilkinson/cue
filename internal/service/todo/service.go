package todo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
)

// TodoRepository defines the persistence operations the service needs.
type TodoRepository interface {
	Insert(ctx context.Context, todo *repository.Todo) error
	Update(ctx context.Context, todo *repository.Todo) error
	Delete(ctx context.Context, id uuid.UUID) error
	QueryByID(ctx context.Context, id uuid.UUID) (*repository.Todo, error)
	QueryFiltered(ctx context.Context, filter repository.TodoFilter) ([]*repository.Todo, int, error)
}

// TimeEstimator provides LLM-based time estimates for todo items.
type TimeEstimator interface {
	EstimateMinutes(ctx context.Context, title, description string) (int, error)
}

// Service provides CRUD operations for todo items.
type Service struct {
	repo      TodoRepository
	estimator TimeEstimator
}

// NewService creates a new Service. Both repo and estimator must be non-nil.
func NewService(repo TodoRepository, estimator TimeEstimator) (*Service, error) {
	if repo == nil {
		return nil, fmt.Errorf("todo service: repository must not be nil")
	}
	if estimator == nil {
		return nil, fmt.Errorf("todo service: estimator must not be nil")
	}
	return &Service{repo: repo, estimator: estimator}, nil
}

// Create inserts a new todo. Sets ID and CreatedAt if zero. Returns the created todo (re-fetched from DB).
func (s *Service) Create(ctx context.Context, todo *repository.Todo) (*repository.Todo, error) {
	if todo.ID == uuid.Nil {
		todo.ID = uuid.New()
	}
	if todo.CreatedAt.IsZero() {
		todo.CreatedAt = time.Now()
	}
	if err := s.repo.Insert(ctx, todo); err != nil {
		return nil, fmt.Errorf("todo service: create: %w", err)
	}
	return s.repo.QueryByID(ctx, todo.ID)
}

// Get returns a single todo by ID.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*repository.Todo, error) {
	return s.repo.QueryByID(ctx, id)
}

// List returns filtered/paginated todos + total count.
func (s *Service) List(ctx context.Context, filter repository.TodoFilter) ([]*repository.Todo, int, error) {
	return s.repo.QueryFiltered(ctx, filter)
}

// Update updates a todo. Returns the updated todo (re-fetched).
func (s *Service) Update(ctx context.Context, todo *repository.Todo) (*repository.Todo, error) {
	if err := s.repo.Update(ctx, todo); err != nil {
		return nil, fmt.Errorf("todo service: update: %w", err)
	}
	return s.repo.QueryByID(ctx, todo.ID)
}

// Delete removes a todo by ID.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// EffectiveEstimate returns EstimateMinutes if non-nil and > 0, else LLMEstimateMinutes.
func EffectiveEstimate(t *repository.Todo) *int {
	if t.EstimateMinutes != nil && *t.EstimateMinutes > 0 {
		return t.EstimateMinutes
	}
	if t.LLMEstimateMinutes != nil {
		return t.LLMEstimateMinutes
	}
	return nil
}
