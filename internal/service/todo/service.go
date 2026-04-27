package todo

import (
	"context"
	"fmt"

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
	return nil, fmt.Errorf("not implemented")
}

// Get returns a single todo by ID.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*repository.Todo, error) {
	return nil, fmt.Errorf("not implemented")
}

// List returns filtered/paginated todos + total count.
func (s *Service) List(ctx context.Context, filter repository.TodoFilter) ([]*repository.Todo, int, error) {
	return nil, 0, fmt.Errorf("not implemented")
}

// Update updates a todo. Returns the updated todo (re-fetched).
func (s *Service) Update(ctx context.Context, todo *repository.Todo) (*repository.Todo, error) {
	return nil, fmt.Errorf("not implemented")
}

// Delete removes a todo by ID.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return fmt.Errorf("not implemented")
}

// EffectiveEstimate returns EstimateMinutes if non-nil and > 0, else LLMEstimateMinutes.
func EffectiveEstimate(t *repository.Todo) *int {
	return nil
}
