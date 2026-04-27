package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/google/uuid"
)

// tasksPath is the base URL path for the tasks REST resource. Per Feature 109
// Decision 5, tasks now live under the /api/v1/todo/ bounded-context prefix
// alongside categories (was /api/v1/tasks pre-109).
const tasksPath = "/api/v1/todo/tasks"

// CategoryEmbed is the abbreviated category shape returned inline on Task
// reads. Only the canonical key and its derived display name are carried;
// callers fetch /api/v1/todo/categories/{key} for full detail (colour,
// task_count, created_at).
type CategoryEmbed struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

// Task mirrors the server's taskItem DTO returned by /api/v1/todo/tasks
// routes. Per Feature 109 Decision 8, each task carries at most one
// category as an embedded {key,name} object, or nil when uncategorized.
//
// Nullable fields are represented as pointers: DueDate and CompletedAt use
// *string (RFC3339) with omitempty so the server's nullable timestamps round-
// trip correctly. EstimateMinutes, LLMEstimateMinutes, and
// EffectiveEstimateMinutes are *int to distinguish unset (nil) from zero.
type Task struct {
	ID                       uuid.UUID      `json:"id"`
	Title                    string         `json:"title"`
	Description              string         `json:"description"`
	Priority                 int            `json:"priority"`
	DueDate                  *string        `json:"due_date,omitempty"`
	Category                 *CategoryEmbed `json:"category"`
	EstimateMinutes          *int           `json:"estimate_minutes"`
	LLMEstimateMinutes       *int           `json:"llm_estimate_minutes"`
	EffectiveEstimateMinutes *int           `json:"effective_estimate_minutes"`
	CreatedAt                string         `json:"created_at"`
	CompletedAt              *string        `json:"completed_at,omitempty"`
}

// CreateTaskRequest is the POST body for creating a task via
// POST /api/v1/todo/tasks. Only Title is required server-side; other fields
// use omitempty so zero-valued defaults are not transmitted. Category is
// raw input — any case/spacing form — and the server normalizes via
// NormalizeCategoryKey before lookup. Pass nil for an uncategorized task.
type CreateTaskRequest struct {
	Title           string  `json:"title"`
	Description     string  `json:"description,omitempty"`
	Priority        int     `json:"priority,omitempty"`
	DueDate         *string `json:"due_date,omitempty"`
	Category        *string `json:"category,omitempty"`
	EstimateMinutes *int    `json:"estimate_minutes,omitempty"`
}

// UpdateTaskRequest is the PUT body for partial updates via
// PUT /api/v1/todo/tasks/{id}. All fields are optional; nil pointers are
// omitted from the outgoing JSON via the custom MarshalJSON below so the
// server only applies fields the caller explicitly set.
//
// Category is tri-state on the wire:
//
//   - Category == nil and ClearCategory == false: the field is OMITTED
//     from the outgoing JSON. Server-side: leave existing category alone.
//   - Category != nil: emits `"category": "<value>"`. Server normalizes
//     and links to the canonical key. ClearCategory is ignored.
//   - Category == nil and ClearCategory == true: emits `"category": null`.
//     Server-side: clear the FK on the task (leave it uncategorized).
//
// The two distinct "no value" states (omitted vs explicit null) are why a
// plain `*string` with omitempty is insufficient — JSON omitempty cannot
// emit `null`, and a non-pointer cannot omit. The custom marshaller bridges
// that gap.
type UpdateTaskRequest struct {
	Title           *string `json:"title,omitempty"`
	Description     *string `json:"description,omitempty"`
	Priority        *int    `json:"priority,omitempty"`
	DueDate         *string `json:"due_date,omitempty"`
	Category        *string `json:"-"`
	ClearCategory   bool    `json:"-"`
	EstimateMinutes *int    `json:"estimate_minutes,omitempty"`
	CompletedAt     *string `json:"completed_at,omitempty"`
}

// MarshalJSON emits the tri-state category encoding documented on
// UpdateTaskRequest while preserving omitempty semantics on the other
// fields. It works by marshalling a private alias type (with Category and
// ClearCategory hidden via json:"-") to a map[string]json.RawMessage, then
// conditionally inserting the "category" key.
func (r UpdateTaskRequest) MarshalJSON() ([]byte, error) {
	type alias UpdateTaskRequest
	raw, err := json.Marshal(alias(r))
	if err != nil {
		return nil, fmt.Errorf("marshal task update: %w", err)
	}

	// Decode into an ordered-insensitive map so we can splice category in.
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, fmt.Errorf("re-decode task update: %w", err)
	}

	switch {
	case r.Category != nil:
		// Explicit value: emit string. ClearCategory ignored.
		b, err := json.Marshal(*r.Category)
		if err != nil {
			return nil, fmt.Errorf("marshal category value: %w", err)
		}
		fields["category"] = b
	case r.ClearCategory:
		// Explicit null: clear the FK server-side.
		fields["category"] = json.RawMessage("null")
		// default (Category == nil, ClearCategory == false): omit the key.
	}

	// Re-marshal. Using json.Marshal on the map produces deterministic output
	// only by key (Go sorts map keys), which is fine for the wire format.
	out, err := json.Marshal(fields)
	if err != nil {
		return nil, fmt.Errorf("re-encode task update: %w", err)
	}
	// Preserve a literal empty-object encoding rather than a stray buffer.
	if bytes.Equal(out, []byte("null")) {
		return []byte("{}"), nil
	}
	return out, nil
}

// TaskFilter captures the optional query parameters accepted by
// GET /api/v1/todo/tasks. Empty strings and zero ints are omitted from the
// outgoing query string. Category accepts any form (raw display, mixed
// case, or the canonical key) — the server normalizes before lookup.
type TaskFilter struct {
	Status   string
	Category string
	Search   string
	Limit    int
	Offset   int
}

// TaskClient wraps /api/v1/todo/tasks routes: listing, creating, fetching,
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

// ListTasks issues GET /api/v1/todo/tasks with the provided filter encoded
// as query parameters. Returns the tasks slice, total count, or an error.
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

// CreateTask issues POST /api/v1/todo/tasks with the provided request body
// and returns the created Task decoded from the 201 response.
func (a *taskAdapter) CreateTask(ctx context.Context, req CreateTaskRequest) (*Task, error) {
	var task Task
	if err := a.client.doJSON(ctx, http.MethodPost, tasksPath, req, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

// GetTask issues GET /api/v1/todo/tasks/{id} and returns the decoded Task.
func (a *taskAdapter) GetTask(ctx context.Context, id uuid.UUID) (*Task, error) {
	var task Task
	if err := a.client.doJSON(ctx, http.MethodGet, tasksPath+"/"+id.String(), nil, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

// UpdateTask issues PUT /api/v1/todo/tasks/{id} with the provided partial
// update body. See UpdateTaskRequest for the tri-state category encoding.
func (a *taskAdapter) UpdateTask(ctx context.Context, id uuid.UUID, req UpdateTaskRequest) (*Task, error) {
	var task Task
	if err := a.client.doJSON(ctx, http.MethodPut, tasksPath+"/"+id.String(), req, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

// DeleteTask issues DELETE /api/v1/todo/tasks/{id}. The server responds
// with 204 No Content on success; doJSON's nil-out path skips body decoding.
func (a *taskAdapter) DeleteTask(ctx context.Context, id uuid.UUID) error {
	return a.client.doJSON(ctx, http.MethodDelete, tasksPath+"/"+id.String(), nil, nil)
}
