package client

import (
	"context"

	"github.com/google/uuid"
)

// Task mirrors the server's taskItem DTO returned by /api/v1/tasks routes.
//
// Nullable fields are represented as pointers: DueDate and CompletedAt use
// *string (RFC3339) with omitempty so the server's nullable timestamps round-
// trip correctly. EstimateMinutes, LLMEstimateMinutes, and
// EffectiveEstimateMinutes are *int to distinguish unset (nil) from zero.
type Task struct {
	ID                       uuid.UUID `json:"id"`
	Title                    string    `json:"title"`
	Description              string    `json:"description"`
	Priority                 int       `json:"priority"`
	DueDate                  *string   `json:"due_date,omitempty"`
	Categories               []string  `json:"categories"`
	EstimateMinutes          *int      `json:"estimate_minutes"`
	LLMEstimateMinutes       *int      `json:"llm_estimate_minutes"`
	EffectiveEstimateMinutes *int      `json:"effective_estimate_minutes"`
	CreatedAt                string    `json:"created_at"`
	CompletedAt              *string   `json:"completed_at,omitempty"`
}

// CreateTaskRequest is the POST body for creating a task via
// POST /api/v1/tasks. Only Title is required server-side; other fields
// use omitempty so zero-valued defaults are not transmitted.
type CreateTaskRequest struct {
	Title           string   `json:"title"`
	Description     string   `json:"description,omitempty"`
	Priority        int      `json:"priority,omitempty"`
	DueDate         *string  `json:"due_date,omitempty"`
	Categories      []string `json:"categories,omitempty"`
	EstimateMinutes *int     `json:"estimate_minutes,omitempty"`
}

// UpdateTaskRequest is the PUT body for partial updates via
// PUT /api/v1/tasks/{id}. All fields are optional; nil pointers and nil
// Categories slices are omitted from the outgoing JSON via omitempty so
// the server only applies fields the caller explicitly set.
type UpdateTaskRequest struct {
	Title           *string  `json:"title,omitempty"`
	Description     *string  `json:"description,omitempty"`
	Priority        *int     `json:"priority,omitempty"`
	DueDate         *string  `json:"due_date,omitempty"`
	Categories      []string `json:"categories,omitempty"`
	EstimateMinutes *int     `json:"estimate_minutes,omitempty"`
	CompletedAt     *string  `json:"completed_at,omitempty"`
}

// TaskFilter captures the optional query parameters accepted by
// GET /api/v1/tasks. Empty strings and zero ints are omitted from the
// outgoing query string.
type TaskFilter struct {
	Status   string
	Category string
	Search   string
	Limit    int
	Offset   int
}

// TaskClient wraps /api/v1/tasks routes: listing, creating, fetching,
// updating (partial), and deleting tasks.
type TaskClient interface {
	ListTasks(ctx context.Context, filter TaskFilter) ([]Task, int, error)
	CreateTask(ctx context.Context, req CreateTaskRequest) (*Task, error)
	GetTask(ctx context.Context, id uuid.UUID) (*Task, error)
	UpdateTask(ctx context.Context, id uuid.UUID, req UpdateTaskRequest) (*Task, error)
	DeleteTask(ctx context.Context, id uuid.UUID) error
}

// taskAdapter is the concrete TaskClient backed by an *APIClient.
type taskAdapter struct {
	client *APIClient
}

// NewTaskClient returns a TaskClient backed by the given APIClient.
func NewTaskClient(c *APIClient) TaskClient {
	return &taskAdapter{client: c}
}

// ListTasks is a stub. Replaced during the GREEN phase.
func (a *taskAdapter) ListTasks(_ context.Context, _ TaskFilter) ([]Task, int, error) {
	return nil, 0, ErrNotImplemented
}

// CreateTask is a stub. Replaced during the GREEN phase.
func (a *taskAdapter) CreateTask(_ context.Context, _ CreateTaskRequest) (*Task, error) {
	return nil, ErrNotImplemented
}

// GetTask is a stub. Replaced during the GREEN phase.
func (a *taskAdapter) GetTask(_ context.Context, _ uuid.UUID) (*Task, error) {
	return nil, ErrNotImplemented
}

// UpdateTask is a stub. Replaced during the GREEN phase.
func (a *taskAdapter) UpdateTask(_ context.Context, _ uuid.UUID, _ UpdateTaskRequest) (*Task, error) {
	return nil, ErrNotImplemented
}

// DeleteTask is a stub. Replaced during the GREEN phase.
func (a *taskAdapter) DeleteTask(_ context.Context, _ uuid.UUID) error {
	return ErrNotImplemented
}
