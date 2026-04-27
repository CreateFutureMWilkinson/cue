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

// CategoryRepository defines the persistence operations the service needs
// for category management. All methods take canonical (already-normalized)
// keys; the service is the single boundary that calls
// repository.NormalizeCategoryKey before invoking these methods.
type CategoryRepository interface {
	Insert(ctx context.Context, c *repository.Category) error
	Rename(ctx context.Context, oldKey, newKey string) error
	UpdateColour(ctx context.Context, key string, colour *string) error
	Delete(ctx context.Context, key string) error
	GetByKey(ctx context.Context, key string) (*repository.Category, error)
	QueryAll(ctx context.Context, withCounts bool) ([]*repository.CategoryWithCount, error)
}

// TimeEstimator provides LLM-based time estimates for task items.
type TimeEstimator interface {
	EstimateMinutes(ctx context.Context, title, description string) (int, error)
}

// Service provides CRUD operations for task items and their categories.
type Service struct {
	repo       TaskRepository
	categories CategoryRepository
	estimator  TimeEstimator
}

// NewService creates a new Service. tasks, categories, and estimator must be non-nil.
func NewService(tasks TaskRepository, categories CategoryRepository, estimator TimeEstimator) (*Service, error) {
	if tasks == nil {
		return nil, fmt.Errorf("todo service: repository must not be nil")
	}
	if categories == nil {
		return nil, fmt.Errorf("todo service: category repository must not be nil")
	}
	if estimator == nil {
		return nil, fmt.Errorf("todo service: estimator must not be nil")
	}
	return &Service{repo: tasks, categories: categories, estimator: estimator}, nil
}

// CreateCategory normalizes rawName, validates colour, and inserts a new
// category. The service is the only boundary that calls
// repository.NormalizeCategoryKey before reaching the repo.
//
// Stub: returns ErrNotImplemented; replaced in Loop 5 GREEN.
func (s *Service) CreateCategory(_ context.Context, _ string, _ *string) (*repository.Category, error) {
	return nil, repository.ErrNotImplemented
}

// RenameCategory normalizes newRawName to a key. If the new key equals
// oldKey, returns the existing category unchanged. Otherwise calls
// categories.Rename and returns the renamed category via GetByKey.
//
// Stub: returns ErrNotImplemented; replaced in Loop 5 GREEN.
func (s *Service) RenameCategory(_ context.Context, _ string, _ string) (*repository.Category, error) {
	return nil, repository.ErrNotImplemented
}

// SetCategoryColour validates colour (if non-nil) then updates it via
// categories.UpdateColour.
//
// Stub: returns ErrNotImplemented; replaced in Loop 5 GREEN.
func (s *Service) SetCategoryColour(_ context.Context, _ string, _ *string) error {
	return repository.ErrNotImplemented
}

// DeleteCategory removes a category by key. Tasks with this category get
// their FK SET NULL via the schema-level cascade.
//
// Stub: returns ErrNotImplemented; replaced in Loop 5 GREEN.
func (s *Service) DeleteCategory(_ context.Context, _ string) error {
	return repository.ErrNotImplemented
}

// GetCategory looks up a category by either its canonical key or any raw
// presentation form. It first tries GetByKey on the input as-is; on
// ErrNotFound it normalizes the input and retries.
//
// Stub: returns ErrNotImplemented; replaced in Loop 5 GREEN.
func (s *Service) GetCategory(_ context.Context, _ string) (*repository.Category, error) {
	return nil, repository.ErrNotImplemented
}

// ListCategories returns all categories, optionally enriched with task counts.
//
// Stub: returns ErrNotImplemented; replaced in Loop 5 GREEN.
func (s *Service) ListCategories(_ context.Context, _ bool) ([]*repository.CategoryWithCount, error) {
	return nil, repository.ErrNotImplemented
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
