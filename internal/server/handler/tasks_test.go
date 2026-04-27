package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/server/handler"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// mockTaskServicer implements handler.TaskServicer for testing.
type mockTaskServicer struct {
	// List
	listTasks []*repository.Task
	listTotal int
	listErr   error

	// Create
	createResult *repository.Task
	createErr    error
	createInput  *repository.Task

	// Get
	getResult *repository.Task
	getErr    error

	// Update
	updateResult *repository.Task
	updateErr    error
	updateInput  *repository.Task

	// Delete
	deleteErr error
	deleteID  uuid.UUID

	// captured filter from List call
	capturedFilter repository.TaskFilter
}

func (m *mockTaskServicer) Create(_ context.Context, task *repository.Task) (*repository.Task, error) {
	m.createInput = task
	if m.createResult == nil && m.createErr == nil {
		// Echo the input so handlers that round-trip the created task
		// have something non-nil to serialize.
		echo := *task
		if echo.ID == uuid.Nil {
			echo.ID = uuid.New()
		}
		if echo.CreatedAt.IsZero() {
			echo.CreatedAt = time.Now().UTC()
		}
		return &echo, nil
	}
	return m.createResult, m.createErr
}

func (m *mockTaskServicer) Get(_ context.Context, _ uuid.UUID) (*repository.Task, error) {
	return m.getResult, m.getErr
}

func (m *mockTaskServicer) List(_ context.Context, filter repository.TaskFilter) ([]*repository.Task, int, error) {
	m.capturedFilter = filter
	return m.listTasks, m.listTotal, m.listErr
}

func (m *mockTaskServicer) Update(_ context.Context, task *repository.Task) (*repository.Task, error) {
	m.updateInput = task
	if m.updateResult == nil && m.updateErr == nil {
		echo := *task
		return &echo, nil
	}
	return m.updateResult, m.updateErr
}

func (m *mockTaskServicer) Delete(_ context.Context, id uuid.UUID) error {
	m.deleteID = id
	return m.deleteErr
}

// (mockCategoryServicer is defined in categories_test.go; we reuse it here.)

// stringPtr returns a pointer to the given string. Used for optional
// fields like CategoryKey in test fixtures.
func stringPtr(s string) *string { return &s }

// stubEffectiveEstimate returns EstimateMinutes if set, else LLMEstimateMinutes.
func stubEffectiveEstimate(t *repository.Task) *int {
	if t.EstimateMinutes != nil && *t.EstimateMinutes > 0 {
		return t.EstimateMinutes
	}
	if t.LLMEstimateMinutes != nil {
		return t.LLMEstimateMinutes
	}
	return nil
}

// TaskHandlerSuite tests the task handler endpoints under the new
// /api/v1/todo/tasks route prefix introduced in Feature 109 Loop 7.
type TaskHandlerSuite struct {
	suite.Suite
}

func TestTaskHandler(t *testing.T) {
	suite.Run(t, new(TaskHandlerSuite))
}

// TestRouteMoved confirms the legacy /api/v1/tasks path is gone and the
// new /api/v1/todo/tasks path is mounted on the server's mux. We use a
// minimal test mux that mirrors what server.registerRoutes does for the
// task handlers — just enough to assert the path-not-found behaviour
// without spinning up the full server.
func (s *TaskHandlerSuite) TestRouteMoved() {
	mock := &mockTaskServicer{
		listTasks: []*repository.Task{},
		listTotal: 0,
	}
	cats := &mockCategoryServicer{}

	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/todo/tasks", handler.ListTasksHandler(mock, stubEffectiveEstimate, cats))

	// Old path should 404.
	oldReq := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	oldRec := httptest.NewRecorder()
	mux.ServeHTTP(oldRec, oldReq)
	s.Equal(http.StatusNotFound, oldRec.Code, "legacy /api/v1/tasks path must 404")

	// New path should 200 with empty list.
	newReq := httptest.NewRequest(http.MethodGet, "/api/v1/todo/tasks", nil)
	newRec := httptest.NewRecorder()
	mux.ServeHTTP(newRec, newReq)
	s.Equal(http.StatusOK, newRec.Code, "new /api/v1/todo/tasks path must serve")
}

func (s *TaskHandlerSuite) TestListTasksHandler() {
	now := time.Now().UTC().Truncate(time.Second)
	llmEst := 30

	task1 := &repository.Task{
		ID:                 uuid.New(),
		Title:              "Review PR #42",
		Description:        "Check the new auth middleware",
		Priority:           5,
		LLMEstimateMinutes: &llmEst,
		CreatedAt:          now.Add(-1 * time.Hour),
	}

	mock := &mockTaskServicer{
		listTasks: []*repository.Task{task1},
		listTotal: 1,
	}
	cats := &mockCategoryServicer{}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/todo/tasks?limit=50&offset=0", nil)
	rec := httptest.NewRecorder()

	handler.ListTasksHandler(mock, stubEffectiveEstimate, cats)(rec, req)

	s.Equal(http.StatusOK, rec.Code, "expected 200 OK")

	var body map[string]any
	err := json.NewDecoder(rec.Body).Decode(&body)
	s.Require().NoError(err, "response body should be valid JSON")

	tasks, ok := body["tasks"].([]any)
	s.Require().True(ok, "response should have a 'tasks' array")
	s.Len(tasks, 1, "should return 1 task")

	total, ok := body["total"].(float64)
	s.Require().True(ok, "response should have a 'total' field")
	s.Equal(float64(1), total)

	count, ok := body["count"].(float64)
	s.Require().True(ok, "response should have a 'count' field")
	s.Equal(float64(1), count)

	// Verify task shape.
	task := tasks[0].(map[string]any)
	s.Equal(task1.ID.String(), task["id"])
	s.Equal("Review PR #42", task["title"])
	s.Equal(float64(30), task["llm_estimate_minutes"])
	s.Equal(float64(30), task["effective_estimate_minutes"])
	s.Nil(task["estimate_minutes"])

	// Legacy "categories" field must not be present.
	_, hasLegacy := task["categories"]
	s.False(hasLegacy, "legacy 'categories' field must be removed from the DTO")
}

// TestListIncludesCategoryEmbed asserts that when a task carries a
// CategoryKey, the response JSON includes a {key, name} object on the
// `category` field — and excludes any `categories` array.
func (s *TaskHandlerSuite) TestListIncludesCategoryEmbed() {
	task := &repository.Task{
		ID:          uuid.New(),
		Title:       "Document the API",
		CategoryKey: stringPtr("foo_bar"),
		CreatedAt:   time.Now().UTC(),
	}
	mock := &mockTaskServicer{
		listTasks: []*repository.Task{task},
		listTotal: 1,
	}
	cats := &mockCategoryServicer{}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/todo/tasks", nil)
	rec := httptest.NewRecorder()

	handler.ListTasksHandler(mock, stubEffectiveEstimate, cats)(rec, req)

	s.Equal(http.StatusOK, rec.Code)

	var body map[string]any
	s.Require().NoError(json.NewDecoder(rec.Body).Decode(&body))

	tasks := body["tasks"].([]any)
	s.Require().Len(tasks, 1)
	t := tasks[0].(map[string]any)

	cat, ok := t["category"].(map[string]any)
	s.Require().True(ok, "category must be a {key, name} object")
	s.Equal("foo_bar", cat["key"])
	s.Equal("Foo Bar", cat["name"])
	s.Nil(cat["colour"], "colour must not be present on the embed")
	s.Nil(cat["task_count"], "task_count must not be present on the embed")

	_, hasLegacy := t["categories"]
	s.False(hasLegacy, "legacy 'categories' array must be removed")
}

// TestListNoCategory: a task with CategoryKey == nil yields category: null.
func (s *TaskHandlerSuite) TestListNoCategory() {
	task := &repository.Task{
		ID:        uuid.New(),
		Title:     "Untagged",
		CreatedAt: time.Now().UTC(),
	}
	mock := &mockTaskServicer{
		listTasks: []*repository.Task{task},
		listTotal: 1,
	}
	cats := &mockCategoryServicer{}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/todo/tasks", nil)
	rec := httptest.NewRecorder()

	handler.ListTasksHandler(mock, stubEffectiveEstimate, cats)(rec, req)

	s.Equal(http.StatusOK, rec.Code)

	// Use a typed decode to distinguish present-and-null from absent.
	var typed struct {
		Tasks []map[string]json.RawMessage `json:"tasks"`
	}
	s.Require().NoError(json.NewDecoder(rec.Body).Decode(&typed))
	s.Require().Len(typed.Tasks, 1)

	raw, ok := typed.Tasks[0]["category"]
	s.Require().True(ok, "category field must be present")
	s.Equal("null", string(raw), "category must serialize as JSON null when absent")
}

func (s *TaskHandlerSuite) TestListTasksHandlerEmpty() {
	mock := &mockTaskServicer{
		listTasks: []*repository.Task{},
		listTotal: 0,
	}
	cats := &mockCategoryServicer{}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/todo/tasks", nil)
	rec := httptest.NewRecorder()

	handler.ListTasksHandler(mock, stubEffectiveEstimate, cats)(rec, req)

	s.Equal(http.StatusOK, rec.Code, "expected 200 OK")

	var body map[string]any
	err := json.NewDecoder(rec.Body).Decode(&body)
	s.Require().NoError(err, "response body should be valid JSON")

	tasks, ok := body["tasks"].([]any)
	s.Require().True(ok, "response should have a 'tasks' array")
	s.Empty(tasks, "tasks should be empty")

	s.Equal(float64(0), body["total"])
	s.Equal(float64(0), body["count"])
}

// TestCreateAcceptsRawCategory: POST with raw "foo BAR" → handler resolves
// via category service to "foo_bar" and assigns it to the task.
func (s *TaskHandlerSuite) TestCreateAcceptsRawCategory() {
	mock := &mockTaskServicer{}
	cats := &mockCategoryServicer{
		getReturn: &repository.Category{NameKey: "foo_bar"},
	}

	body := `{"title":"x","category":"foo BAR"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/todo/tasks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.CreateTaskHandler(mock, stubEffectiveEstimate, cats)(rec, req)

	s.Equal(http.StatusCreated, rec.Code, "expected 201 Created")
	s.True(cats.getCalled, "category service must be consulted")
	s.Equal("foo BAR", cats.getInput, "raw input must be passed to GetCategory")
	s.Require().NotNil(mock.createInput, "Create must be called")
	s.Require().NotNil(mock.createInput.CategoryKey, "task must carry resolved category key")
	s.Equal("foo_bar", *mock.createInput.CategoryKey)
}

// TestCreateUnknownCategoryReturns400: unknown category → 400.
func (s *TaskHandlerSuite) TestCreateUnknownCategoryReturns400() {
	mock := &mockTaskServicer{}
	cats := &mockCategoryServicer{
		getErr: repository.ErrNotFound,
	}

	body := `{"title":"x","category":"nope"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/todo/tasks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.CreateTaskHandler(mock, stubEffectiveEstimate, cats)(rec, req)

	s.Equal(http.StatusBadRequest, rec.Code, "unknown category must yield 400")
	s.Nil(mock.createInput, "Create must not be called when category is unknown")
}

// TestCreateNullCategory: POST with `category: null` → CategoryKey nil,
// category service NOT consulted.
func (s *TaskHandlerSuite) TestCreateNullCategory() {
	mock := &mockTaskServicer{}
	cats := &mockCategoryServicer{}

	body := `{"title":"x","category":null}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/todo/tasks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.CreateTaskHandler(mock, stubEffectiveEstimate, cats)(rec, req)

	s.Equal(http.StatusCreated, rec.Code)
	s.False(cats.getCalled, "category service must NOT be consulted for null")
	s.Require().NotNil(mock.createInput)
	s.Nil(mock.createInput.CategoryKey, "CategoryKey must be nil for null input")
}

// TestCreateMissingCategory: POST without `category` field → CategoryKey nil,
// category service NOT consulted.
func (s *TaskHandlerSuite) TestCreateMissingCategory() {
	mock := &mockTaskServicer{}
	cats := &mockCategoryServicer{}

	body := `{"title":"x"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/todo/tasks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.CreateTaskHandler(mock, stubEffectiveEstimate, cats)(rec, req)

	s.Equal(http.StatusCreated, rec.Code)
	s.False(cats.getCalled, "category service must NOT be consulted when field omitted")
	s.Require().NotNil(mock.createInput)
	s.Nil(mock.createInput.CategoryKey, "CategoryKey must be nil when field omitted")
}

func (s *TaskHandlerSuite) TestCreateTaskHandler() {
	now := time.Now().UTC().Truncate(time.Second)
	createdID := uuid.New()
	llmEst := 25

	created := &repository.Task{
		ID:                 createdID,
		Title:              "Write tests",
		Description:        "Unit tests for auth",
		Priority:           3,
		LLMEstimateMinutes: &llmEst,
		CreatedAt:          now,
	}

	mock := &mockTaskServicer{
		createResult: created,
	}
	cats := &mockCategoryServicer{}

	body := `{"title":"Write tests","description":"Unit tests for auth","priority":3}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/todo/tasks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.CreateTaskHandler(mock, stubEffectiveEstimate, cats)(rec, req)

	s.Equal(http.StatusCreated, rec.Code, "expected 201 Created")

	var respBody map[string]any
	err := json.NewDecoder(rec.Body).Decode(&respBody)
	s.Require().NoError(err, "response body should be valid JSON")

	s.Equal(createdID.String(), respBody["id"])
	s.Equal("Write tests", respBody["title"])
	s.Equal("Unit tests for auth", respBody["description"])
	s.Equal(float64(3), respBody["priority"])
}

func (s *TaskHandlerSuite) TestCreateTaskHandlerMissingTitle() {
	mock := &mockTaskServicer{}
	cats := &mockCategoryServicer{}

	body := `{"description":"no title here","priority":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/todo/tasks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.CreateTaskHandler(mock, stubEffectiveEstimate, cats)(rec, req)

	s.Equal(http.StatusBadRequest, rec.Code, "expected 400 for missing title")
}

func (s *TaskHandlerSuite) TestGetTaskHandler() {
	now := time.Now().UTC().Truncate(time.Second)
	taskID := uuid.New()
	est := 45

	task := &repository.Task{
		ID:              taskID,
		Title:           "Deploy v2",
		Description:     "Production deployment",
		Priority:        8,
		EstimateMinutes: &est,
		CreatedAt:       now.Add(-2 * time.Hour),
	}

	mock := &mockTaskServicer{
		getResult: task,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/todo/tasks/"+taskID.String(), nil)
	req.SetPathValue("id", taskID.String())
	rec := httptest.NewRecorder()

	handler.GetTaskHandler(mock, stubEffectiveEstimate)(rec, req)

	s.Equal(http.StatusOK, rec.Code, "expected 200 OK")

	var respBody map[string]any
	err := json.NewDecoder(rec.Body).Decode(&respBody)
	s.Require().NoError(err, "response body should be valid JSON")

	s.Equal(taskID.String(), respBody["id"])
	s.Equal("Deploy v2", respBody["title"])
	s.Equal("Production deployment", respBody["description"])
	s.Equal(float64(8), respBody["priority"])
	s.Equal(float64(45), respBody["estimate_minutes"])
	s.Equal(float64(45), respBody["effective_estimate_minutes"])
}

func (s *TaskHandlerSuite) TestGetTaskHandlerNotFound() {
	mock := &mockTaskServicer{
		getErr: repository.ErrNotFound,
	}

	unknownID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/todo/tasks/"+unknownID.String(), nil)
	req.SetPathValue("id", unknownID.String())
	rec := httptest.NewRecorder()

	handler.GetTaskHandler(mock, stubEffectiveEstimate)(rec, req)

	s.Equal(http.StatusNotFound, rec.Code, "expected 404 Not Found")
}

func (s *TaskHandlerSuite) TestUpdateTaskHandler() {
	now := time.Now().UTC().Truncate(time.Second)
	taskID := uuid.New()
	newEst := 60

	updated := &repository.Task{
		ID:              taskID,
		Title:           "Updated title",
		Description:     "Updated desc",
		Priority:        9,
		EstimateMinutes: &newEst,
		CreatedAt:       now.Add(-2 * time.Hour),
	}

	mock := &mockTaskServicer{
		getResult:    updated,
		updateResult: updated,
	}
	cats := &mockCategoryServicer{}

	body := `{"title":"Updated title","priority":9,"estimate_minutes":60}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/todo/tasks/"+taskID.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", taskID.String())
	rec := httptest.NewRecorder()

	handler.UpdateTaskHandler(mock, stubEffectiveEstimate, cats)(rec, req)

	s.Equal(http.StatusOK, rec.Code, "expected 200 OK")

	var respBody map[string]any
	err := json.NewDecoder(rec.Body).Decode(&respBody)
	s.Require().NoError(err, "response body should be valid JSON")

	s.Equal(taskID.String(), respBody["id"])
	s.Equal("Updated title", respBody["title"])
	s.Equal(float64(9), respBody["priority"])
	s.Equal(float64(60), respBody["estimate_minutes"])
	s.Equal(float64(60), respBody["effective_estimate_minutes"])
}

func (s *TaskHandlerSuite) TestUpdateTaskHandlerNotFound() {
	mock := &mockTaskServicer{
		getErr: repository.ErrNotFound,
	}
	cats := &mockCategoryServicer{}

	unknownID := uuid.New()
	body := `{"title":"Nope"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/todo/tasks/"+unknownID.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", unknownID.String())
	rec := httptest.NewRecorder()

	handler.UpdateTaskHandler(mock, stubEffectiveEstimate, cats)(rec, req)

	s.Equal(http.StatusNotFound, rec.Code, "expected 404 Not Found")
}

// TestUpdateChangesCategory: PUT body with `category: "work"` → handler
// resolves and sets CategoryKey on the task.
func (s *TaskHandlerSuite) TestUpdateChangesCategory() {
	taskID := uuid.New()
	existing := &repository.Task{
		ID:        taskID,
		Title:     "Existing",
		CreatedAt: time.Now().UTC(),
	}
	mock := &mockTaskServicer{
		getResult: existing,
	}
	cats := &mockCategoryServicer{
		getReturn: &repository.Category{NameKey: "work"},
	}

	body := `{"category":"work"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/todo/tasks/"+taskID.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", taskID.String())
	rec := httptest.NewRecorder()

	handler.UpdateTaskHandler(mock, stubEffectiveEstimate, cats)(rec, req)

	s.Equal(http.StatusOK, rec.Code)
	s.True(cats.getCalled)
	s.Require().NotNil(mock.updateInput)
	s.Require().NotNil(mock.updateInput.CategoryKey)
	s.Equal("work", *mock.updateInput.CategoryKey)
}

// TestUpdateClearsCategory: PUT with `category: null` → CategoryKey nil.
func (s *TaskHandlerSuite) TestUpdateClearsCategory() {
	taskID := uuid.New()
	existing := &repository.Task{
		ID:          taskID,
		Title:       "Existing",
		CategoryKey: stringPtr("work"),
		CreatedAt:   time.Now().UTC(),
	}
	mock := &mockTaskServicer{
		getResult: existing,
	}
	cats := &mockCategoryServicer{}

	body := `{"category":null}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/todo/tasks/"+taskID.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", taskID.String())
	rec := httptest.NewRecorder()

	handler.UpdateTaskHandler(mock, stubEffectiveEstimate, cats)(rec, req)

	s.Equal(http.StatusOK, rec.Code)
	s.False(cats.getCalled, "category service must NOT be consulted for null")
	s.Require().NotNil(mock.updateInput)
	s.Nil(mock.updateInput.CategoryKey, "CategoryKey must be cleared")
}

// TestUpdateOmittedCategory: PUT without `category` field → CategoryKey
// unchanged. Distinguishes absent vs null using map[string]json.RawMessage.
func (s *TaskHandlerSuite) TestUpdateOmittedCategory() {
	taskID := uuid.New()
	existing := &repository.Task{
		ID:          taskID,
		Title:       "Existing",
		CategoryKey: stringPtr("work"),
		CreatedAt:   time.Now().UTC(),
	}
	mock := &mockTaskServicer{
		getResult: existing,
	}
	cats := &mockCategoryServicer{}

	body := `{"title":"Renamed"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/todo/tasks/"+taskID.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", taskID.String())
	rec := httptest.NewRecorder()

	handler.UpdateTaskHandler(mock, stubEffectiveEstimate, cats)(rec, req)

	s.Equal(http.StatusOK, rec.Code)
	s.False(cats.getCalled, "category service must NOT be consulted when field omitted")
	s.Require().NotNil(mock.updateInput)
	s.Require().NotNil(mock.updateInput.CategoryKey, "CategoryKey must be preserved")
	s.Equal("work", *mock.updateInput.CategoryKey)
}

// TestFilterByCategoryAcceptsAnyForm: ?category=Foo%20Bar normalizes to
// foo_bar before reaching the repo filter.
func (s *TaskHandlerSuite) TestFilterByCategoryAcceptsAnyForm() {
	mock := &mockTaskServicer{
		listTasks: []*repository.Task{},
		listTotal: 0,
	}
	cats := &mockCategoryServicer{
		getReturn: &repository.Category{NameKey: "foo_bar"},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/todo/tasks?category=Foo%20Bar", nil)
	rec := httptest.NewRecorder()

	handler.ListTasksHandler(mock, stubEffectiveEstimate, cats)(rec, req)

	s.Equal(http.StatusOK, rec.Code)
	s.True(cats.getCalled, "category service must normalize the filter input")
	s.Equal("foo_bar", mock.capturedFilter.CategoryKey, "filter must use canonical key")

	// And empty filter must NOT consult the service.
	mock2 := &mockTaskServicer{listTasks: []*repository.Task{}}
	cats2 := &mockCategoryServicer{}
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/todo/tasks", nil)
	rec2 := httptest.NewRecorder()
	handler.ListTasksHandler(mock2, stubEffectiveEstimate, cats2)(rec2, req2)
	s.Equal(http.StatusOK, rec2.Code)
	s.False(cats2.getCalled, "empty filter must skip the category service")
	s.Equal("", mock2.capturedFilter.CategoryKey)
}

// TestFilterUnknownCategoryReturns400: ?category=unknown where category
// service returns ErrNotFound → 400.
func (s *TaskHandlerSuite) TestFilterUnknownCategoryReturns400() {
	mock := &mockTaskServicer{}
	cats := &mockCategoryServicer{
		getErr: repository.ErrNotFound,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/todo/tasks?category=unknown", nil)
	rec := httptest.NewRecorder()

	handler.ListTasksHandler(mock, stubEffectiveEstimate, cats)(rec, req)

	s.Equal(http.StatusBadRequest, rec.Code, "unknown category in filter must yield 400")
}

func (s *TaskHandlerSuite) TestDeleteTaskHandler() {
	mock := &mockTaskServicer{}

	taskID := uuid.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/todo/tasks/"+taskID.String(), nil)
	req.SetPathValue("id", taskID.String())
	rec := httptest.NewRecorder()

	handler.DeleteTaskHandler(mock)(rec, req)

	s.Equal(http.StatusNoContent, rec.Code, "expected 204 No Content")
	s.Equal(taskID, mock.deleteID, "Delete should receive the correct task ID")
}

func (s *TaskHandlerSuite) TestDeleteTaskHandlerNotFound() {
	mock := &mockTaskServicer{
		deleteErr: repository.ErrNotFound,
	}

	unknownID := uuid.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/todo/tasks/"+unknownID.String(), nil)
	req.SetPathValue("id", unknownID.String())
	rec := httptest.NewRecorder()

	handler.DeleteTaskHandler(mock)(rec, req)

	s.Equal(http.StatusNotFound, rec.Code, "expected 404 Not Found")
}
