package todo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
)

// TaskRepository defines the persistence operations the service needs.
type TaskRepository interface {
	Insert(ctx context.Context, task *repository.Task) error
	Update(ctx context.Context, task *repository.Task) error
	Delete(ctx context.Context, id uuid.UUID) error
	QueryByID(ctx context.Context, id uuid.UUID) (*repository.Task, error)
	QueryFiltered(ctx context.Context, filter repository.TaskFilter) ([]*repository.Task, int, error)
}

// TimeEstimator provides LLM-based time estimates for task items.
type TimeEstimator interface {
	EstimateMinutes(ctx context.Context, title, description string) (int, error)
}

// Service provides CRUD operations for task items.
type Service struct {
	repo      TaskRepository
	estimator TimeEstimator
}

// NewService creates a new Service. Both repo and estimator must be non-nil.
func NewService(repo TaskRepository, estimator TimeEstimator) (*Service, error) {
	if repo == nil {
		return nil, fmt.Errorf("todo service: repository must not be nil")
	}
	if estimator == nil {
		return nil, fmt.Errorf("todo service: estimator must not be nil")
	}
	return &Service{repo: repo, estimator: estimator}, nil
}

// Create inserts a new task. Sets ID and CreatedAt if zero. Returns the created task (re-fetched from DB).
// If EstimateMinutes is nil or zero, triggers async LLM estimation.
func (s *Service) Create(ctx context.Context, task *repository.Task) (*repository.Task, error) {
	if task.ID == uuid.Nil {
		task.ID = uuid.New()
	}
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}
	if err := s.repo.Insert(ctx, task); err != nil {
		return nil, fmt.Errorf("todo service: create: %w", err)
	}

	if task.EstimateMinutes == nil || *task.EstimateMinutes == 0 {
		taskID := task.ID
		title := task.Title
		description := task.Description
		go s.asyncEstimate(taskID, title, description) // #nosec G118 — intentionally outlives request
	}

	return s.repo.QueryByID(ctx, task.ID)
}

// Get returns a single task by ID.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*repository.Task, error) {
	return s.repo.QueryByID(ctx, id)
}

// List returns filtered/paginated tasks + total count.
func (s *Service) List(ctx context.Context, filter repository.TaskFilter) ([]*repository.Task, int, error) {
	return s.repo.QueryFiltered(ctx, filter)
}

// Update updates a task. Returns the updated task (re-fetched).
// If EstimateMinutes was previously non-nil and > 0 but is now nil or zero,
// clears LLMEstimateMinutes and triggers async re-estimation.
func (s *Service) Update(ctx context.Context, task *repository.Task) (*repository.Task, error) {
	existing, err := s.repo.QueryByID(ctx, task.ID)
	if err != nil {
		return nil, fmt.Errorf("todo service: update: %w", err)
	}

	needsEstimation := existing.EstimateMinutes != nil && *existing.EstimateMinutes > 0 &&
		(task.EstimateMinutes == nil || *task.EstimateMinutes == 0)

	if needsEstimation {
		task.LLMEstimateMinutes = nil
	}

	if err := s.repo.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("todo service: update: %w", err)
	}

	if needsEstimation {
		taskID := task.ID
		title := task.Title
		description := task.Description
		go s.asyncEstimate(taskID, title, description) // #nosec G118 — intentionally outlives request
	}

	return s.repo.QueryByID(ctx, task.ID)
}

// Delete removes a task by ID.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// asyncEstimate calls the LLM estimator and persists the result on the task.
func (s *Service) asyncEstimate(taskID uuid.UUID, title, description string) {
	ctx := context.Background()
	estimate, err := s.estimator.EstimateMinutes(ctx, title, description)
	if err != nil {
		return
	}
	t, err := s.repo.QueryByID(ctx, taskID)
	if err != nil {
		return
	}
	t.LLMEstimateMinutes = &estimate
	_ = s.repo.Update(ctx, t)
}

// EffectiveEstimate returns EstimateMinutes if non-nil and > 0, else LLMEstimateMinutes.
func EffectiveEstimate(t *repository.Task) *int {
	if t.EstimateMinutes != nil && *t.EstimateMinutes > 0 {
		return t.EstimateMinutes
	}
	if t.LLMEstimateMinutes != nil {
		return t.LLMEstimateMinutes
	}
	return nil
}
