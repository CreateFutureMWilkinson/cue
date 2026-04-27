package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/google/uuid"
)

// ErrNotImplemented is returned by stub handlers pending implementation.
var ErrNotImplemented = errors.New("not implemented")

// TodoServicer is the subset of todo.Service needed by todo handlers.
type TodoServicer interface {
	Create(ctx context.Context, todo *repository.Todo) (*repository.Todo, error)
	Get(ctx context.Context, id uuid.UUID) (*repository.Todo, error)
	List(ctx context.Context, filter repository.TodoFilter) ([]*repository.Todo, int, error)
	Update(ctx context.Context, todo *repository.Todo) (*repository.Todo, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// EffectiveEstimateFunc computes the effective estimate for a todo.
type EffectiveEstimateFunc func(t *repository.Todo) *int

// taskItem is the JSON representation of a task in responses.
type taskItem struct {
	ID                       string   `json:"id"`
	Title                    string   `json:"title"`
	Description              string   `json:"description"`
	Priority                 int      `json:"priority"`
	DueDate                  *string  `json:"due_date,omitempty"`
	Categories               []string `json:"categories"`
	EstimateMinutes          *int     `json:"estimate_minutes"`
	LLMEstimateMinutes       *int     `json:"llm_estimate_minutes"`
	EffectiveEstimateMinutes *int     `json:"effective_estimate_minutes"`
	CreatedAt                string   `json:"created_at"`
	CompletedAt              *string  `json:"completed_at,omitempty"`
}

// taskListResponse is the JSON envelope for the task list endpoint.
type taskListResponse struct {
	Tasks []taskItem `json:"tasks"`
	Total int        `json:"total"`
	Count int        `json:"count"`
}

// createTaskRequest is the JSON body for POST /api/v1/tasks.
type createTaskRequest struct {
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	Priority        int      `json:"priority"`
	DueDate         *string  `json:"due_date"`
	Categories      []string `json:"categories"`
	EstimateMinutes *int     `json:"estimate_minutes"`
}

// updateTaskRequest is the JSON body for PUT /api/v1/tasks/{id}.
type updateTaskRequest struct {
	Title           *string  `json:"title"`
	Description     *string  `json:"description"`
	Priority        *int     `json:"priority"`
	DueDate         *string  `json:"due_date"`
	Categories      []string `json:"categories"`
	EstimateMinutes *int     `json:"estimate_minutes"`
	CompletedAt     *string  `json:"completed_at"`
}

// ListTasksHandler returns an http.HandlerFunc for GET /api/v1/tasks.
func ListTasksHandler(_ TodoServicer, _ EffectiveEstimateFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSONError(w, http.StatusNotImplemented, ErrNotImplemented.Error())
	}
}

// CreateTaskHandler returns an http.HandlerFunc for POST /api/v1/tasks.
func CreateTaskHandler(_ TodoServicer, _ EffectiveEstimateFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSONError(w, http.StatusNotImplemented, ErrNotImplemented.Error())
	}
}

// GetTaskHandler returns an http.HandlerFunc for GET /api/v1/tasks/{id}.
func GetTaskHandler(_ TodoServicer, _ EffectiveEstimateFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSONError(w, http.StatusNotImplemented, ErrNotImplemented.Error())
	}
}

// UpdateTaskHandler returns an http.HandlerFunc for PUT /api/v1/tasks/{id}.
func UpdateTaskHandler(_ TodoServicer, _ EffectiveEstimateFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSONError(w, http.StatusNotImplemented, ErrNotImplemented.Error())
	}
}

// DeleteTaskHandler returns an http.HandlerFunc for DELETE /api/v1/tasks/{id}.
func DeleteTaskHandler(_ TodoServicer) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSONError(w, http.StatusNotImplemented, ErrNotImplemented.Error())
	}
}
