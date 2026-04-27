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

// TaskServicer is the subset of todo.Service needed by task handlers.
type TaskServicer interface {
	Create(ctx context.Context, task *repository.Task) (*repository.Task, error)
	Get(ctx context.Context, id uuid.UUID) (*repository.Task, error)
	List(ctx context.Context, filter repository.TaskFilter) ([]*repository.Task, int, error)
	Update(ctx context.Context, task *repository.Task) (*repository.Task, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// EffectiveEstimateFunc computes the effective estimate for a task.
type EffectiveEstimateFunc func(t *repository.Task) *int

// taskCategoryEmbed is the inline {key, name} representation of a task's
// category in the task wire format. Per Feature 109 Decision 8 this carries
// only the canonical key and its presentation name — no colour, no
// task_count. Clients that need the full category record fetch
// /api/v1/todo/categories/{key}.
type taskCategoryEmbed struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

// taskItem is the JSON representation of a task in responses.
type taskItem struct {
	ID                       string             `json:"id"`
	Title                    string             `json:"title"`
	Description              string             `json:"description"`
	Priority                 int                `json:"priority"`
	DueDate                  *string            `json:"due_date,omitempty"`
	Category                 *taskCategoryEmbed `json:"category"`
	EstimateMinutes          *int               `json:"estimate_minutes"`
	LLMEstimateMinutes       *int               `json:"llm_estimate_minutes"`
	EffectiveEstimateMinutes *int               `json:"effective_estimate_minutes"`
	CreatedAt                string             `json:"created_at"`
	CompletedAt              *string            `json:"completed_at,omitempty"`
}

// taskListResponse is the JSON envelope for the task list endpoint.
type taskListResponse struct {
	Tasks []taskItem `json:"tasks"`
	Total int        `json:"total"`
	Count int        `json:"count"`
}

// createTaskRequest is the JSON body for POST /api/v1/todo/tasks.
//
// Category is decoded out-of-band via the raw map[string]json.RawMessage
// pattern so the handler can distinguish absent (leave nil) from explicit
// null (clear). Any present non-null value is treated as a raw display
// string and resolved through CategoryServicer.GetCategory.
type createTaskRequest struct {
	Title           string  `json:"title"`
	Description     string  `json:"description"`
	Priority        int     `json:"priority"`
	DueDate         *string `json:"due_date"`
	EstimateMinutes *int    `json:"estimate_minutes"`
}

// updateTaskRequest is the JSON body for PUT /api/v1/todo/tasks/{id}.
type updateTaskRequest struct {
	Title           *string `json:"title"`
	Description     *string `json:"description"`
	Priority        *int    `json:"priority"`
	DueDate         *string `json:"due_date"`
	EstimateMinutes *int    `json:"estimate_minutes"`
	CompletedAt     *string `json:"completed_at"`
}

// taskToItem converts a repository.Task to a taskItem JSON struct.
func taskToItem(t *repository.Task, effectiveFn EffectiveEstimateFunc) taskItem {
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

	if t.CategoryKey != nil {
		item.Category = &taskCategoryEmbed{
			Key:  *t.CategoryKey,
			Name: repository.PresentCategoryName(*t.CategoryKey),
		}
	}

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

// ListTasksHandler returns an http.HandlerFunc for GET /api/v1/todo/tasks.
//
// @Summary      List tasks
// @Description  Paginated list of task items, filterable by status, category,
// @Description  and free-text search across title/description.
// @Tags         tasks
// @Produce      json
// @Param        status    query     string  false  "open | completed"
// @Param        category  query     string  false  "Category name (any form)"
// @Param        search    query     string  false  "Substring match on title/description"
// @Param        limit     query     int     false  "Page size (default 50)"
// @Param        offset    query     int     false  "Page offset (default 0)"
// @Success      200       {object}  handler.taskListResponse
// @Failure      400       {object}  map[string]string
// @Failure      500       {object}  map[string]string
// @Router       /api/v1/todo/tasks [get]
func ListTasksHandler(svc TaskServicer, effectiveFn EffectiveEstimateFunc, cats CategoryServicer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit, offset := parsePagination(r)

		categoryKey := ""
		if rawCat := r.URL.Query().Get("category"); rawCat != "" {
			cat, err := cats.GetCategory(r.Context(), rawCat)
			if err != nil {
				if errors.Is(err, repository.ErrNotFound) {
					writeJSONError(w, http.StatusBadRequest, "category not found")
					return
				}
				writeJSONError(w, http.StatusInternalServerError, "failed to resolve category")
				return
			}
			categoryKey = cat.NameKey
		}

		filter := repository.TaskFilter{
			Status:      r.URL.Query().Get("status"),
			CategoryKey: categoryKey,
			Search:      r.URL.Query().Get("search"),
			Limit:       limit,
			Offset:      offset,
		}

		tasks, total, err := svc.List(r.Context(), filter)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to list tasks")
			return
		}

		items := make([]taskItem, len(tasks))
		for i, t := range tasks {
			items[i] = taskToItem(t, effectiveFn)
		}

		writeJSON(w, http.StatusOK, taskListResponse{
			Tasks: items,
			Total: total,
			Count: len(items),
		})
	}
}

// CreateTaskHandler returns an http.HandlerFunc for POST /api/v1/todo/tasks.
//
// @Summary      Create task
// @Description  Creates a new task. Title is required. Accepts YYYY-MM-DD or
// @Description  RFC3339 for due_date. The optional `category` field accepts
// @Description  the raw display name in any form and is normalized server-side.
// @Tags         tasks
// @Accept       json
// @Produce      json
// @Param        request  body      handler.createTaskRequest  true  "Task fields"
// @Success      201      {object}  handler.taskItem
// @Failure      400      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /api/v1/todo/tasks [post]
func CreateTaskHandler(svc TaskServicer, effectiveFn EffectiveEstimateFunc, cats CategoryServicer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Decode into a raw map so we can distinguish absent vs explicit
		// null on `category`, mirroring the pattern used by the category
		// PUT handler in Loop 6.
		var raw map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		var req createTaskRequest
		if err := decodeRawInto(raw, &req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if req.Title == "" {
			writeJSONError(w, http.StatusBadRequest, "title is required")
			return
		}

		task := &repository.Task{
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
			task.DueDate = &t
		}

		present, key, handled := resolveCategoryRaw(r.Context(), w, cats, raw)
		if handled {
			return
		}
		if present {
			task.CategoryKey = key
		}

		created, err := svc.Create(r.Context(), task)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to create task")
			return
		}

		writeJSON(w, http.StatusCreated, taskToItem(created, effectiveFn))
	}
}

// GetTaskHandler returns an http.HandlerFunc for GET /api/v1/todo/tasks/{id}.
//
// @Summary      Get task by ID
// @Tags         tasks
// @Produce      json
// @Param        id   path      string  true  "Task UUID"
// @Success      200  {object}  handler.taskItem
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/todo/tasks/{id} [get]
func GetTaskHandler(svc TaskServicer, effectiveFn EffectiveEstimateFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseTaskID(r)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, "not found")
			return
		}

		task, err := svc.Get(r.Context(), id)
		if err != nil {
			writeNotFoundOrError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, taskToItem(task, effectiveFn))
	}
}

// UpdateTaskHandler returns an http.HandlerFunc for PUT /api/v1/todo/tasks/{id}.
//
// @Summary      Update task
// @Description  Partial update: any non-nil field in the body is applied to
// @Description  the existing task. Unspecified fields retain their current
// @Description  value. The `category` field uses present/absent/null
// @Description  semantics: absent leaves the existing value intact, null
// @Description  clears the category, a string resolves through the category
// @Description  service.
// @Tags         tasks
// @Accept       json
// @Produce      json
// @Param        id       path      string                     true  "Task UUID"
// @Param        request  body      handler.updateTaskRequest  true  "Fields to update"
// @Success      200      {object}  handler.taskItem
// @Failure      400      {object}  map[string]string
// @Failure      404      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /api/v1/todo/tasks/{id} [put]
func UpdateTaskHandler(svc TaskServicer, effectiveFn EffectiveEstimateFunc, cats CategoryServicer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseTaskID(r)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, "not found")
			return
		}

		// Fetch existing task.
		existing, err := svc.Get(r.Context(), id)
		if err != nil {
			writeNotFoundOrError(w, err)
			return
		}
		if existing == nil {
			writeJSONError(w, http.StatusNotFound, "not found")
			return
		}

		// Decode into a raw map first to support absent-vs-null on
		// `category` (Decision 8 + Loop 6 PUT pattern).
		var raw map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		var req updateTaskRequest
		if err := decodeRawInto(raw, &req); err != nil {
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
		if req.CompletedAt != nil {
			t, err := time.Parse(time.RFC3339, *req.CompletedAt)
			if err != nil {
				writeJSONError(w, http.StatusBadRequest, "invalid completed_at format")
				return
			}
			existing.CompletedAt = &t
		}

		present, key, handled := resolveCategoryRaw(r.Context(), w, cats, raw)
		if handled {
			return
		}
		if present {
			existing.CategoryKey = key
		}

		updated, err := svc.Update(r.Context(), existing)
		if err != nil {
			writeNotFoundOrError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, taskToItem(updated, effectiveFn))
	}
}

// DeleteTaskHandler returns an http.HandlerFunc for DELETE /api/v1/todo/tasks/{id}.
//
// @Summary      Delete task
// @Tags         tasks
// @Param        id   path      string  true  "Task UUID"
// @Success      204  "No Content"
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/todo/tasks/{id} [delete]
func DeleteTaskHandler(svc TaskServicer) http.HandlerFunc {
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

// resolveCategoryRaw interprets a raw JSON value from the request body for
// the `category` field. The three states it returns are:
//
//   - present=false: the field was absent (caller should leave existing
//     value untouched).
//   - present=true, key=nil: the field was explicit JSON null (caller
//     should clear the category).
//   - present=true, key=&"...": the field was a non-null string and the
//     category service resolved it to a canonical key.
//
// On JSON decode error or service error the function writes the response
// directly and returns handled=true so callers can stop processing.
func resolveCategoryRaw(
	ctx context.Context,
	w http.ResponseWriter,
	cats CategoryServicer,
	raw map[string]json.RawMessage,
) (present bool, key *string, handled bool) {
	rawCat, ok := raw["category"]
	if !ok {
		return false, nil, false
	}
	if string(rawCat) == "null" {
		return true, nil, false
	}
	var s string
	if err := json.Unmarshal(rawCat, &s); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid category")
		return true, nil, true
	}
	cat, err := cats.GetCategory(ctx, s)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeJSONError(w, http.StatusBadRequest, "category not found")
			return true, nil, true
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to resolve category")
		return true, nil, true
	}
	k := cat.NameKey
	return true, &k, false
}

// decodeRawInto re-encodes a raw JSON map into a typed struct. Used by the
// task handlers to share the absent-vs-null detection on `category` with
// the regular field decoding logic.
func decodeRawInto(raw map[string]json.RawMessage, v any) error {
	b, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}
