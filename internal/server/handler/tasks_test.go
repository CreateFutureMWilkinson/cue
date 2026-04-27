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
	return m.updateResult, m.updateErr
}

func (m *mockTaskServicer) Delete(_ context.Context, id uuid.UUID) error {
	m.deleteID = id
	return m.deleteErr
}

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

// TaskHandlerSuite tests the task handler endpoints.
type TaskHandlerSuite struct {
	suite.Suite
}

func TestTaskHandler(t *testing.T) {
	suite.Run(t, new(TaskHandlerSuite))
}

func (s *TaskHandlerSuite) TestListTasksHandler() {
	now := time.Now().UTC().Truncate(time.Second)
	llmEst := 30

	task1 := &repository.Task{
		ID:                 uuid.New(),
		Title:              "Review PR #42",
		Description:        "Check the new auth middleware",
		Priority:           5,
		Categories:         []repository.Category{{NameKey: "code_review"}},
		LLMEstimateMinutes: &llmEst,
		CreatedAt:          now.Add(-1 * time.Hour),
	}

	mock := &mockTaskServicer{
		listTasks: []*repository.Task{task1},
		listTotal: 1,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks?limit=50&offset=0", nil)
	rec := httptest.NewRecorder()

	handler.ListTasksHandler(mock, stubEffectiveEstimate)(rec, req)

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
	s.Equal("Check the new auth middleware", task["description"])
	s.Equal(float64(5), task["priority"])
	s.NotNil(task["categories"])
	cats := task["categories"].([]any)
	s.Len(cats, 1)
	s.Equal("code-review", cats[0])
	s.Equal(float64(30), task["llm_estimate_minutes"])
	s.Equal(float64(30), task["effective_estimate_minutes"])
	s.Nil(task["estimate_minutes"])
}

func (s *TaskHandlerSuite) TestListTasksHandlerEmpty() {
	mock := &mockTaskServicer{
		listTasks: []*repository.Task{},
		listTotal: 0,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	rec := httptest.NewRecorder()

	handler.ListTasksHandler(mock, stubEffectiveEstimate)(rec, req)

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

	body := `{"title":"Write tests","description":"Unit tests for auth","priority":3}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.CreateTaskHandler(mock, stubEffectiveEstimate)(rec, req)

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

	body := `{"description":"no title here","priority":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.CreateTaskHandler(mock, stubEffectiveEstimate)(rec, req)

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

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+taskID.String(), nil)
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
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+unknownID.String(), nil)
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
		getResult:    updated, // for fetching existing
		updateResult: updated,
	}

	body := `{"title":"Updated title","priority":9,"estimate_minutes":60}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tasks/"+taskID.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", taskID.String())
	rec := httptest.NewRecorder()

	handler.UpdateTaskHandler(mock, stubEffectiveEstimate)(rec, req)

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
		updateErr: repository.ErrNotFound,
	}

	unknownID := uuid.New()
	body := `{"title":"Nope"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tasks/"+unknownID.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", unknownID.String())
	rec := httptest.NewRecorder()

	handler.UpdateTaskHandler(mock, stubEffectiveEstimate)(rec, req)

	s.Equal(http.StatusNotFound, rec.Code, "expected 404 Not Found")
}

func (s *TaskHandlerSuite) TestDeleteTaskHandler() {
	mock := &mockTaskServicer{}

	taskID := uuid.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/"+taskID.String(), nil)
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
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/"+unknownID.String(), nil)
	req.SetPathValue("id", unknownID.String())
	rec := httptest.NewRecorder()

	handler.DeleteTaskHandler(mock)(rec, req)

	s.Equal(http.StatusNotFound, rec.Code, "expected 404 Not Found")
}
