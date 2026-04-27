package client_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/pkg/client"
)

// TaskSuite covers the TaskClient adapter over /api/v1/todo/tasks.
type TaskSuite struct {
	suite.Suite
}

func TestTask(t *testing.T) {
	suite.Run(t, new(TaskSuite))
}

// testTaskID is a deterministic UUID used across suite tests so path
// interpolation can be asserted directly.
var testTaskID = uuid.MustParse("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee")

// intPtr returns a pointer to i. Used to construct optional *int fields in
// task request and response payloads. (stringPtr lives in feedback_test.go,
// same _test package — reused here.)
func intPtr(i int) *int { return &i }

// TestListTasksSendsAllFiltersAsQueryParams verifies that every populated
// field of TaskFilter (status, category, search, limit, offset) is emitted
// as a query parameter on GET /api/v1/todo/tasks. Per Feature 109, Category
// accepts any form (display, mixed case, key) — the server normalizes.
func (s *TaskSuite) TestListTasksSendsAllFiltersAsQueryParams() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal("/api/v1/todo/tasks", r.URL.Path)

		q := r.URL.Query()
		s.Equal("active", q.Get("status"))
		s.Equal("Foo Bar", q.Get("category"))
		s.Equal("design", q.Get("search"))
		s.Equal("20", q.Get("limit"))
		s.Equal("40", q.Get("offset"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tasks": []any{},
			"total": 0,
			"count": 0,
		})
	}))
	defer ts.Close()

	tc := client.NewTaskClient(client.New(ts.URL))
	tasks, total, err := tc.ListTasks(context.Background(), client.TaskFilter{
		Status:   "active",
		Category: "Foo Bar",
		Search:   "design",
		Limit:    20,
		Offset:   40,
	})
	s.Require().NoError(err)
	s.Empty(tasks)
	s.Equal(0, total)
}

// TestListTasksDecodesResponse verifies that snake_case JSON fields on the
// task list payload — including the embedded `category: {key,name}` object
// (or null) and the nullable pointer fields — decode into the typed Task
// struct.
func (s *TaskSuite) TestListTasksDecodesResponse() {
	secondID := uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("/api/v1/todo/tasks", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tasks": []map[string]any{
				{
					"id":                         testTaskID.String(),
					"title":                      "Design review",
					"description":                "review v2 spec",
					"priority":                   2,
					"due_date":                   "2026-05-01T10:00:00Z",
					"category":                   map[string]any{"key": "foo_bar", "name": "Foo Bar"},
					"estimate_minutes":           45,
					"llm_estimate_minutes":       30,
					"effective_estimate_minutes": 45,
					"created_at":                 "2026-04-20T09:00:00Z",
					// completed_at omitted intentionally (task not done).
				},
				{
					"id":                         secondID.String(),
					"title":                      "Submit timesheet",
					"description":                "",
					"priority":                   1,
					"category":                   nil,
					"estimate_minutes":           nil,
					"llm_estimate_minutes":       nil,
					"effective_estimate_minutes": nil,
					"created_at":                 "2026-04-18T08:00:00Z",
					"completed_at":               "2026-04-19T17:30:00Z",
				},
			},
			"total": 2,
			"count": 2,
		})
	}))
	defer ts.Close()

	tc := client.NewTaskClient(client.New(ts.URL))
	tasks, total, err := tc.ListTasks(context.Background(), client.TaskFilter{})
	s.Require().NoError(err)
	s.Equal(2, total)
	s.Require().Len(tasks, 2)

	first := tasks[0]
	s.Equal(testTaskID, first.ID)
	s.Equal("Design review", first.Title)
	s.Equal("review v2 spec", first.Description)
	s.Equal(2, first.Priority)
	s.Require().NotNil(first.DueDate)
	s.Equal("2026-05-01T10:00:00Z", *first.DueDate)
	s.Require().NotNil(first.Category, "category embed must decode for tagged tasks")
	s.Equal("foo_bar", first.Category.Key)
	s.Equal("Foo Bar", first.Category.Name)
	s.Require().NotNil(first.EstimateMinutes)
	s.Equal(45, *first.EstimateMinutes)
	s.Require().NotNil(first.LLMEstimateMinutes)
	s.Equal(30, *first.LLMEstimateMinutes)
	s.Require().NotNil(first.EffectiveEstimateMinutes)
	s.Equal(45, *first.EffectiveEstimateMinutes)
	s.Equal("2026-04-20T09:00:00Z", first.CreatedAt)
	s.Nil(first.CompletedAt, "completed_at must decode to nil when omitted")

	second := tasks[1]
	s.Equal(secondID, second.ID)
	s.Equal("Submit timesheet", second.Title)
	s.Nil(second.Category, "category null on the wire must decode to nil pointer")
	s.Nil(second.EstimateMinutes, "estimate_minutes must decode to nil when server sends null")
	s.Nil(second.LLMEstimateMinutes)
	s.Nil(second.EffectiveEstimateMinutes)
	s.Require().NotNil(second.CompletedAt)
	s.Equal("2026-04-19T17:30:00Z", *second.CompletedAt)
}

// TestCreateTaskPostsBody verifies that CreateTask POSTs a JSON body with
// title, priority, and category (raw string), and decodes the returned
// Task with its embedded category object.
func (s *TaskSuite) TestCreateTaskPostsBody() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPost, r.Method)
		s.Equal("/api/v1/todo/tasks", r.URL.Path)

		var body struct {
			Title           string  `json:"title"`
			Description     string  `json:"description"`
			Priority        int     `json:"priority"`
			Category        *string `json:"category"`
			EstimateMinutes *int    `json:"estimate_minutes"`
		}
		s.Require().NoError(json.NewDecoder(r.Body).Decode(&body))
		s.Equal("New task", body.Title)
		s.Equal(3, body.Priority)
		s.Require().NotNil(body.Category)
		s.Equal("home", *body.Category)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":                         testTaskID.String(),
			"title":                      "New task",
			"description":                "",
			"priority":                   3,
			"category":                   map[string]any{"key": "home", "name": "Home"},
			"estimate_minutes":           nil,
			"llm_estimate_minutes":       nil,
			"effective_estimate_minutes": 60,
			"created_at":                 "2026-04-24T12:00:00Z",
		})
	}))
	defer ts.Close()

	tc := client.NewTaskClient(client.New(ts.URL))
	homeRaw := "home"
	task, err := tc.CreateTask(context.Background(), client.CreateTaskRequest{
		Title:    "New task",
		Priority: 3,
		Category: &homeRaw,
	})
	s.Require().NoError(err)
	s.Require().NotNil(task)
	s.Equal(testTaskID, task.ID)
	s.Equal("New task", task.Title)
	s.Equal(3, task.Priority)
	s.Require().NotNil(task.Category)
	s.Equal("home", task.Category.Key)
	s.Equal("Home", task.Category.Name)
	s.Require().NotNil(task.EffectiveEstimateMinutes)
	s.Equal(60, *task.EffectiveEstimateMinutes)
	s.Equal("2026-04-24T12:00:00Z", task.CreatedAt)
}

// TestCreateTaskWithoutCategoryOmitsField verifies that a CreateTaskRequest
// with Category == nil omits the "category" key from the outgoing body
// entirely, so the server can distinguish "no category" from "explicit null".
func (s *TaskSuite) TestCreateTaskWithoutCategoryOmitsField() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		s.Require().NoError(err)
		var body map[string]json.RawMessage
		s.Require().NoError(json.Unmarshal(raw, &body))
		s.NotContains(body, "category", "nil Category must omit the JSON key entirely")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":                         testTaskID.String(),
			"title":                      "New task",
			"description":                "",
			"priority":                   0,
			"category":                   nil,
			"estimate_minutes":           nil,
			"llm_estimate_minutes":       nil,
			"effective_estimate_minutes": 60,
			"created_at":                 "2026-04-24T12:00:00Z",
		})
	}))
	defer ts.Close()

	tc := client.NewTaskClient(client.New(ts.URL))
	_, err := tc.CreateTask(context.Background(), client.CreateTaskRequest{
		Title: "New task",
	})
	s.Require().NoError(err)
}

// TestGetTaskReturnsTask verifies that GetTask issues
// GET /api/v1/todo/tasks/{id} and decodes the Task payload, including the
// embedded category {key,name}.
func (s *TaskSuite) TestGetTaskReturnsTask() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal("/api/v1/todo/tasks/"+testTaskID.String(), r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":                         testTaskID.String(),
			"title":                      "Existing task",
			"description":                "details",
			"priority":                   1,
			"category":                   map[string]any{"key": "work", "name": "Work"},
			"estimate_minutes":           20,
			"llm_estimate_minutes":       nil,
			"effective_estimate_minutes": 20,
			"created_at":                 "2026-04-10T08:00:00Z",
		})
	}))
	defer ts.Close()

	tc := client.NewTaskClient(client.New(ts.URL))
	task, err := tc.GetTask(context.Background(), testTaskID)
	s.Require().NoError(err)
	s.Require().NotNil(task)
	s.Equal(testTaskID, task.ID)
	s.Equal("Existing task", task.Title)
	s.Equal("details", task.Description)
	s.Equal(1, task.Priority)
	s.Require().NotNil(task.Category)
	s.Equal("work", task.Category.Key)
	s.Equal("Work", task.Category.Name)
	s.Require().NotNil(task.EstimateMinutes)
	s.Equal(20, *task.EstimateMinutes)
	s.Nil(task.LLMEstimateMinutes)
	s.Require().NotNil(task.EffectiveEstimateMinutes)
	s.Equal(20, *task.EffectiveEstimateMinutes)
	s.Equal("2026-04-10T08:00:00Z", task.CreatedAt)
}

// TestUpdateTaskSendsPartialBody verifies that UpdateTask sends only the
// fields the caller populated; nil pointer fields must be omitted from the
// outgoing JSON body. The "category" key MUST also be absent in the
// "no change" case (Category == nil and ClearCategory == false).
func (s *TaskSuite) TestUpdateTaskSendsPartialBody() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPut, r.Method)
		s.Equal("/api/v1/todo/tasks/"+testTaskID.String(), r.URL.Path)

		// Decode the raw body as a map to assert omitted keys are absent.
		raw, err := io.ReadAll(r.Body)
		s.Require().NoError(err)
		var body map[string]json.RawMessage
		s.Require().NoError(json.Unmarshal(raw, &body))

		// Title and priority MUST be present.
		s.Contains(body, "title")
		s.Contains(body, "priority")

		// All other update fields MUST be absent due to omitempty / no-change.
		s.NotContains(body, "description")
		s.NotContains(body, "due_date")
		s.NotContains(body, "category", "no-change category must omit the key entirely")
		s.NotContains(body, "estimate_minutes")
		s.NotContains(body, "completed_at")

		// Decode values of present fields.
		var title string
		s.Require().NoError(json.Unmarshal(body["title"], &title))
		s.Equal("new title", title)

		var priority int
		s.Require().NoError(json.Unmarshal(body["priority"], &priority))
		s.Equal(3, priority)

		// Respond with the full updated Task.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":                         testTaskID.String(),
			"title":                      "new title",
			"description":                "old description",
			"priority":                   3,
			"category":                   map[string]any{"key": "work", "name": "Work"},
			"estimate_minutes":           15,
			"llm_estimate_minutes":       nil,
			"effective_estimate_minutes": 15,
			"created_at":                 "2026-04-01T10:00:00Z",
		})
	}))
	defer ts.Close()

	tc := client.NewTaskClient(client.New(ts.URL))
	task, err := tc.UpdateTask(context.Background(), testTaskID, client.UpdateTaskRequest{
		Title:    stringPtr("new title"),
		Priority: intPtr(3),
	})
	s.Require().NoError(err)
	s.Require().NotNil(task)
	s.Equal(testTaskID, task.ID)
	s.Equal("new title", task.Title)
	s.Equal(3, task.Priority)
}

// TestUpdateTaskSetCategoryEmitsString verifies that an UpdateTaskRequest
// with Category set to a non-nil string emits `"category":"<value>"` on
// the wire — distinct from "absent" and "explicit null".
func (s *TaskSuite) TestUpdateTaskSetCategoryEmitsString() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		s.Require().NoError(err)
		var body map[string]json.RawMessage
		s.Require().NoError(json.Unmarshal(raw, &body))

		s.Require().Contains(body, "category", "set Category must include the key")
		var got string
		s.Require().NoError(json.Unmarshal(body["category"], &got))
		s.Equal("work", got)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":                         testTaskID.String(),
			"title":                      "x",
			"description":                "",
			"priority":                   0,
			"category":                   map[string]any{"key": "work", "name": "Work"},
			"estimate_minutes":           nil,
			"llm_estimate_minutes":       nil,
			"effective_estimate_minutes": 0,
			"created_at":                 "2026-04-01T10:00:00Z",
		})
	}))
	defer ts.Close()

	tc := client.NewTaskClient(client.New(ts.URL))
	work := "work"
	_, err := tc.UpdateTask(context.Background(), testTaskID, client.UpdateTaskRequest{
		Category: &work,
	})
	s.Require().NoError(err)
}

// TestUpdateTaskClearCategoryEmitsNull verifies that an UpdateTaskRequest
// with ClearCategory == true (and Category == nil) emits `"category":null`
// on the wire so the server clears the FK on the task row.
func (s *TaskSuite) TestUpdateTaskClearCategoryEmitsNull() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		s.Require().NoError(err)
		var body map[string]json.RawMessage
		s.Require().NoError(json.Unmarshal(raw, &body))

		s.Require().Contains(body, "category", "ClearCategory must include the key")
		s.Equal("null", string(body["category"]), "ClearCategory must serialise as JSON null")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":                         testTaskID.String(),
			"title":                      "x",
			"description":                "",
			"priority":                   0,
			"category":                   nil,
			"estimate_minutes":           nil,
			"llm_estimate_minutes":       nil,
			"effective_estimate_minutes": 0,
			"created_at":                 "2026-04-01T10:00:00Z",
		})
	}))
	defer ts.Close()

	tc := client.NewTaskClient(client.New(ts.URL))
	_, err := tc.UpdateTask(context.Background(), testTaskID, client.UpdateTaskRequest{
		ClearCategory: true,
	})
	s.Require().NoError(err)
}

// TestDeleteTaskReturns204 verifies that DeleteTask issues
// DELETE /api/v1/todo/tasks/{id} and tolerates a 204 No Content response
// with an empty body.
func (s *TaskSuite) TestDeleteTaskReturns204() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodDelete, r.Method)
		s.Equal("/api/v1/todo/tasks/"+testTaskID.String(), r.URL.Path)

		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	tc := client.NewTaskClient(client.New(ts.URL))
	err := tc.DeleteTask(context.Background(), testTaskID)
	s.Require().NoError(err)
}

// TestGetTaskNotFoundReturnsAPIError verifies that a 404 from the server
// surfaces as an *APIError with ErrCodeNotFound.
func (s *TaskSuite) TestGetTaskNotFoundReturnsAPIError() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal("/api/v1/todo/tasks/"+testTaskID.String(), r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "task not found",
		})
	}))
	defer ts.Close()

	tc := client.NewTaskClient(client.New(ts.URL))
	task, err := tc.GetTask(context.Background(), testTaskID)
	s.Require().Error(err)
	s.Nil(task)

	var apiErr *client.APIError
	s.Require().True(errors.As(err, &apiErr), "expected *APIError, got %T", err)
	s.Equal(client.ErrCodeNotFound, apiErr.Code)
	s.Equal(http.StatusNotFound, apiErr.StatusCode)
}
