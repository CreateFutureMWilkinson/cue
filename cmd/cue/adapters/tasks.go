package adapters

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/pkg/client"
)

// TasksAdapter satisfies presenter.TodoQuerier on top of
// client.TaskClient. It maps between repository.Task (uuid.UUID,
// time.Time, single CategoryKey pointer) and the SDK's Task DTO
// (string timestamps, embedded Category{key,name}).
type TasksAdapter struct {
	client client.TaskClient
}

// NewTasksAdapter wraps the given SDK task client.
func NewTasksAdapter(c client.TaskClient) *TasksAdapter {
	return &TasksAdapter{client: c}
}

// QueryFiltered fetches tasks via /api/v1/todo/tasks with the given
// filter encoded as query parameters.
func (a *TasksAdapter) QueryFiltered(ctx context.Context, filter repository.TaskFilter) ([]*repository.Task, int, error) {
	dtos, total, err := a.client.ListTasks(ctx, client.TaskFilter{
		Status:   filter.Status,
		Category: filter.CategoryKey,
		Search:   filter.Search,
		Limit:    filter.Limit,
		Offset:   filter.Offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list tasks: %w", err)
	}
	out := make([]*repository.Task, 0, len(dtos))
	for i := range dtos {
		out = append(out, taskDTOToRepo(dtos[i]))
	}
	return out, total, nil
}

// Insert creates a task via POST /api/v1/todo/tasks. The server
// stamps an ID, CreatedAt, and any normalised category key; the
// adapter copies those back onto the supplied task pointer.
func (a *TasksAdapter) Insert(ctx context.Context, task *repository.Task) error {
	if task == nil {
		return fmt.Errorf("tasks adapter: cannot insert nil task")
	}
	req := client.CreateTaskRequest{
		Title:           task.Title,
		Description:     task.Description,
		Priority:        task.Priority,
		DueDate:         timePtrToRFC3339Ptr(task.DueDate),
		Category:        task.CategoryKey,
		EstimateMinutes: task.EstimateMinutes,
	}
	dto, err := a.client.CreateTask(ctx, req)
	if err != nil {
		return fmt.Errorf("create task: %w", err)
	}
	*task = *taskDTOToRepo(*dto)
	return nil
}

// Update replaces a task via PUT /api/v1/todo/tasks/{id}. All
// non-nil fields on the supplied repository.Task are sent; nil
// pointers leave the corresponding field untouched on the server,
// except for CategoryKey, which is tri-state: nil leaves the
// existing category alone, an empty string clears it, and any other
// value sets it.
func (a *TasksAdapter) Update(ctx context.Context, task *repository.Task) error {
	if task == nil {
		return fmt.Errorf("tasks adapter: cannot update nil task")
	}
	req := client.UpdateTaskRequest{
		Title:           &task.Title,
		Description:     &task.Description,
		Priority:        &task.Priority,
		DueDate:         timePtrToRFC3339Ptr(task.DueDate),
		EstimateMinutes: task.EstimateMinutes,
		CompletedAt:     timePtrToRFC3339Ptr(task.CompletedAt),
	}
	switch {
	case task.CategoryKey != nil && *task.CategoryKey == "":
		req.ClearCategory = true
	case task.CategoryKey != nil:
		req.Category = task.CategoryKey
	}
	dto, err := a.client.UpdateTask(ctx, task.ID, req)
	if err != nil {
		return fmt.Errorf("update task %s: %w", task.ID, err)
	}
	*task = *taskDTOToRepo(*dto)
	return nil
}

// Complete marks a task complete by setting completed_at via the
// update endpoint. The server treats a non-nil completed_at as the
// "completed" transition; the repository contract's separate
// Complete method is preserved for ergonomics.
func (a *TasksAdapter) Complete(ctx context.Context, id uuid.UUID, completedAt time.Time) error {
	formatted := completedAt.UTC().Format(time.RFC3339)
	if _, err := a.client.UpdateTask(ctx, id, client.UpdateTaskRequest{
		CompletedAt: &formatted,
	}); err != nil {
		return fmt.Errorf("complete task %s: %w", id, err)
	}
	return nil
}

// CategoriesAdapter satisfies presenter.CategoryQuerier on top of
// client.CategoryClient. The category list is implicitly sorted by
// the server; the adapter stamps zero TaskCount when the SDK
// envelope omits it (all reads include it today).
type CategoriesAdapter struct {
	client client.CategoryClient
}

// NewCategoriesAdapter wraps the given SDK category client.
func NewCategoriesAdapter(c client.CategoryClient) *CategoriesAdapter {
	return &CategoriesAdapter{client: c}
}

// QueryAll returns every user-defined category. The withCounts
// parameter is honoured by the server unconditionally — the adapter
// passes it through for API parity with the local repository, but
// counts are always populated on the wire.
func (a *CategoriesAdapter) QueryAll(ctx context.Context, _ bool) ([]*repository.CategoryWithCount, error) {
	dtos, err := a.client.ListCategories(ctx)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	out := make([]*repository.CategoryWithCount, 0, len(dtos))
	for i := range dtos {
		out = append(out, categoryDTOToRepo(dtos[i]))
	}
	return out, nil
}

func taskDTOToRepo(t client.Task) *repository.Task {
	r := &repository.Task{
		ID:                 t.ID,
		Title:              t.Title,
		Description:        t.Description,
		Priority:           t.Priority,
		DueDate:            parseRFC3339Ptr(t.DueDate),
		EstimateMinutes:    t.EstimateMinutes,
		LLMEstimateMinutes: t.LLMEstimateMinutes,
		CreatedAt:          parseRFC3339OrZero(t.CreatedAt),
		CompletedAt:        parseRFC3339Ptr(t.CompletedAt),
	}
	if t.Category != nil {
		key := t.Category.Key
		r.CategoryKey = &key
	}
	return r
}

func categoryDTOToRepo(c client.Category) *repository.CategoryWithCount {
	return &repository.CategoryWithCount{
		Category: repository.Category{
			NameKey:   c.Key,
			Colour:    c.Colour,
			CreatedAt: parseRFC3339OrZero(c.CreatedAt),
		},
		TaskCount: c.TaskCount,
	}
}

func parseRFC3339Ptr(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return nil
	}
	return &t
}

func timePtrToRFC3339Ptr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format(time.RFC3339)
	return &s
}
