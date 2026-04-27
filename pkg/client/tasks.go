package client

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/google/uuid"
)

const tasksPath = "/api/v1/tasks"

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

// ListTasks issues GET /api/v1/tasks with the provided filter encoded as
// query parameters. Returns the tasks slice, total count, or an error.
func (a *taskAdapter) ListTasks(ctx context.Context, filter TaskFilter) ([]Task, int, error) {
	q := url.Values{}
	if filter.Status != "" {
		q.Set("status", filter.Status)
	}
	if filter.Category != "" {
		q.Set("category", filter.Category)
	}
	if filter.Search != "" {
		q.Set("search", filter.Search)
	}
	if filter.Limit > 0 {
		q.Set("limit", strconv.Itoa(filter.Limit))
	}
	if filter.Offset > 0 {
		q.Set("offset", strconv.Itoa(filter.Offset))
	}

	path := buildPath(tasksPath, q)

	var out struct {
		Tasks []Task `json:"tasks"`
		Total int    `json:"total"`
		Count int    `json:"count"`
	}
	if err := a.client.doJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, 0, err
	}
	return out.Tasks, out.Total, nil
}

// CreateTask issues POST /api/v1/tasks with the provided request body and
// returns the created Task decoded from the 201 response.
func (a *taskAdapter) CreateTask(ctx context.Context, req CreateTaskRequest) (*Task, error) {
	var task Task
	if err := a.client.doJSON(ctx, http.MethodPost, tasksPath, req, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

// GetTask issues GET /api/v1/tasks/{id} and returns the decoded Task.
func (a *taskAdapter) GetTask(ctx context.Context, id uuid.UUID) (*Task, error) {
	var task Task
	if err := a.client.doJSON(ctx, http.MethodGet, tasksPath+"/"+id.String(), nil, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

// UpdateTask issues PUT /api/v1/tasks/{id} with the provided partial update
// body. Nil pointer and nil slice fields are omitted from the outgoing JSON
// via omitempty so the server only applies fields explicitly set.
func (a *taskAdapter) UpdateTask(ctx context.Context, id uuid.UUID, req UpdateTaskRequest) (*Task, error) {
	var task Task
	if err := a.client.doJSON(ctx, http.MethodPut, tasksPath+"/"+id.String(), req, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

// DeleteTask issues DELETE /api/v1/tasks/{id}. The server responds with
// 204 No Content on success; doJSON's nil-out path skips body decoding.
func (a *taskAdapter) DeleteTask(ctx context.Context, id uuid.UUID) error {
	return a.client.doJSON(ctx, http.MethodDelete, tasksPath+"/"+id.String(), nil, nil)
}
