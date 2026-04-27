package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

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

// todoToTaskItem converts a repository.Todo to a taskItem JSON struct.
func todoToTaskItem(t *repository.Todo, effectiveFn EffectiveEstimateFunc) taskItem {
	item := taskItem{
		ID:                       t.ID.String(),
		Title:                    t.Title,
		Description:              t.Description,
		Priority:                 t.Priority,
		EstimateMinutes:          t.EstimateMinutes,
		LLMEstimateMinutes:       t.LLMEstimateMinutes,
		EffectiveEstimateMinutes: effectiveFn(t),
		CreatedAt:                t.CreatedAt.Format(time.RFC3339),
	}

	// Convert categories to string slice of names.
	cats := make([]string, len(t.Categories))
	for i, c := range t.Categories {
		cats[i] = c.Name
	}
	item.Categories = cats

	if t.DueDate != nil {
		s := t.DueDate.Format(time.RFC3339)
		item.DueDate = &s
	}
	if t.CompletedAt != nil {
		s := t.CompletedAt.Format(time.RFC3339)
		item.CompletedAt = &s
	}

	return item
}

// ListTasksHandler returns an http.HandlerFunc for GET /api/v1/tasks.
func ListTasksHandler(svc TodoServicer, effectiveFn EffectiveEstimateFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit, offset := parsePagination(r)

		filter := repository.TodoFilter{
			Status:   r.URL.Query().Get("status"),
			Category: r.URL.Query().Get("category"),
			Search:   r.URL.Query().Get("search"),
			Limit:    limit,
			Offset:   offset,
		}

		todos, total, err := svc.List(r.Context(), filter)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to list tasks")
			return
		}

		items := make([]taskItem, len(todos))
		for i, t := range todos {
			items[i] = todoToTaskItem(t, effectiveFn)
		}

		writeJSON(w, http.StatusOK, taskListResponse{
			Tasks: items,
			Total: total,
			Count: len(items),
		})
	}
}

// CreateTaskHandler returns an http.HandlerFunc for POST /api/v1/tasks.
func CreateTaskHandler(svc TodoServicer, effectiveFn EffectiveEstimateFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createTaskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if req.Title == "" {
			writeJSONError(w, http.StatusBadRequest, "title is required")
			return
		}

		todo := &repository.Todo{
			Title:           req.Title,
			Description:     req.Description,
			Priority:        req.Priority,
			EstimateMinutes: req.EstimateMinutes,
		}

		// Parse due_date if present.
		if req.DueDate != nil {
			t, err := parseDueDate(*req.DueDate)
			if err != nil {
				writeJSONError(w, http.StatusBadRequest, "invalid due_date format")
				return
			}
			todo.DueDate = &t
		}

		// Convert category names to repository.Category slice.
		if len(req.Categories) > 0 {
			cats := make([]repository.Category, len(req.Categories))
			for i, name := range req.Categories {
				cats[i] = repository.Category{Name: name}
			}
			todo.Categories = cats
		}

		created, err := svc.Create(r.Context(), todo)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to create task")
			return
		}

		writeJSON(w, http.StatusCreated, todoToTaskItem(created, effectiveFn))
	}
}

// GetTaskHandler returns an http.HandlerFunc for GET /api/v1/tasks/{id}.
func GetTaskHandler(svc TodoServicer, effectiveFn EffectiveEstimateFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseTaskID(r)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, "not found")
			return
		}

		todo, err := svc.Get(r.Context(), id)
		if err != nil {
			writeNotFoundOrError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, todoToTaskItem(todo, effectiveFn))
	}
}

// UpdateTaskHandler returns an http.HandlerFunc for PUT /api/v1/tasks/{id}.
func UpdateTaskHandler(svc TodoServicer, effectiveFn EffectiveEstimateFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseTaskID(r)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, "not found")
			return
		}

		// Fetch existing todo.
		existing, err := svc.Get(r.Context(), id)
		if err != nil {
			writeNotFoundOrError(w, err)
			return
		}
		if existing == nil {
			writeJSONError(w, http.StatusNotFound, "not found")
			return
		}

		// Decode update request.
		var req updateTaskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		// Apply non-nil fields.
		if req.Title != nil {
			existing.Title = *req.Title
		}
		if req.Description != nil {
			existing.Description = *req.Description
		}
		if req.Priority != nil {
			existing.Priority = *req.Priority
		}
		if req.EstimateMinutes != nil {
			existing.EstimateMinutes = req.EstimateMinutes
		}
		if req.DueDate != nil {
			t, err := parseDueDate(*req.DueDate)
			if err != nil {
				writeJSONError(w, http.StatusBadRequest, "invalid due_date format")
				return
			}
			existing.DueDate = &t
		}
		if req.Categories != nil {
			cats := make([]repository.Category, len(req.Categories))
			for i, name := range req.Categories {
				cats[i] = repository.Category{Name: name}
			}
			existing.Categories = cats
		}
		if req.CompletedAt != nil {
			t, err := time.Parse(time.RFC3339, *req.CompletedAt)
			if err != nil {
				writeJSONError(w, http.StatusBadRequest, "invalid completed_at format")
				return
			}
			existing.CompletedAt = &t
		}

		updated, err := svc.Update(r.Context(), existing)
		if err != nil {
			writeNotFoundOrError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, todoToTaskItem(updated, effectiveFn))
	}
}

// DeleteTaskHandler returns an http.HandlerFunc for DELETE /api/v1/tasks/{id}.
func DeleteTaskHandler(svc TodoServicer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseTaskID(r)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, "not found")
			return
		}

		if err := svc.Delete(r.Context(), id); err != nil {
			writeNotFoundOrError(w, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// parseTaskID extracts and parses the {id} path parameter as a UUID.
func parseTaskID(r *http.Request) (uuid.UUID, error) {
	return uuid.Parse(r.PathValue("id"))
}

// parseDueDate parses a date string, accepting both "2006-01-02" and RFC3339 formats.
func parseDueDate(s string) (time.Time, error) {
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, s)
}
