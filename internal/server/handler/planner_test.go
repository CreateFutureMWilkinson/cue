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

// mockScheduleStore implements handler.ScheduleStore for testing.
type mockScheduleStore struct {
	loadResult *repository.Schedule
	loadErr    error
	saveErr    error
	deleteErr  error
}

func (m *mockScheduleStore) LoadByDate(_ context.Context, _ time.Time) (*repository.Schedule, error) {
	return m.loadResult, m.loadErr
}

func (m *mockScheduleStore) Save(_ context.Context, _ *repository.Schedule) error {
	return m.saveErr
}

func (m *mockScheduleStore) Delete(_ context.Context, _ time.Time) error {
	return m.deleteErr
}

// PlannerHandlerSuite tests the planner handler endpoints.
type PlannerHandlerSuite struct {
	suite.Suite
}

func TestPlannerHandler(t *testing.T) {
	suite.Run(t, new(PlannerHandlerSuite))
}

func (s *PlannerHandlerSuite) TestGetScheduleReturnsSchedule() {
	taskID := uuid.New()
	createdAt := time.Date(2026, 4, 20, 8, 0, 0, 0, time.UTC)
	blockStart := time.Date(2026, 4, 20, 9, 0, 0, 0, time.UTC)
	blockEnd := time.Date(2026, 4, 20, 9, 25, 0, 0, time.UTC)
	breakStart := time.Date(2026, 4, 20, 9, 25, 0, 0, time.UTC)
	breakEnd := time.Date(2026, 4, 20, 9, 30, 0, 0, time.UTC)

	schedule := &repository.Schedule{
		ID:       uuid.New(),
		Date:     time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
		Strategy: "focus-maximized",
		Blocks: []repository.ScheduleBlock{
			{
				Start:    blockStart,
				End:      blockEnd,
				Type:     repository.ScheduleBlockFocus,
				TaskID:   &taskID,
				TaskName: "Review PR #42",
			},
			{
				Start:    breakStart,
				End:      breakEnd,
				Type:     repository.ScheduleBlockShortBreak,
				TaskID:   nil,
				TaskName: "",
			},
		},
		CreatedAt: createdAt,
	}

	mock := &mockScheduleStore{loadResult: schedule}
	h := handler.GetScheduleHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/planner/2026-04-20", nil)
	req.SetPathValue("date", "2026-04-20")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)

	var resp map[string]any
	err := json.NewDecoder(rec.Body).Decode(&resp)
	s.Require().NoError(err)

	s.Equal("2026-04-20", resp["date"])
	s.Equal("focus-maximized", resp["strategy"])
	s.Equal(createdAt.Format(time.RFC3339), resp["created_at"])

	blocks, ok := resp["blocks"].([]any)
	s.Require().True(ok)
	s.Require().Len(blocks, 2)

	b0 := blocks[0].(map[string]any)
	s.Equal("09:00", b0["start"])
	s.Equal("09:25", b0["end"])
	s.Equal("focus", b0["type"])
	s.Equal(taskID.String(), b0["task_id"])
	s.Equal("Review PR #42", b0["task_name"])

	b1 := blocks[1].(map[string]any)
	s.Equal("09:25", b1["start"])
	s.Equal("09:30", b1["end"])
	s.Equal("short_break", b1["type"])
	s.Nil(b1["task_id"])
}

func (s *PlannerHandlerSuite) TestGetScheduleNotFound() {
	mock := &mockScheduleStore{loadErr: repository.ErrNotFound}
	h := handler.GetScheduleHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/planner/2026-04-20", nil)
	req.SetPathValue("date", "2026-04-20")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	s.Equal(http.StatusNotFound, rec.Code)

	var resp map[string]string
	err := json.NewDecoder(rec.Body).Decode(&resp)
	s.Require().NoError(err)
	s.Equal("not found", resp["error"])
}

func (s *PlannerHandlerSuite) TestGetScheduleInvalidDate() {
	mock := &mockScheduleStore{}
	h := handler.GetScheduleHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/planner/not-a-date", nil)
	req.SetPathValue("date", "not-a-date")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	s.Equal(http.StatusBadRequest, rec.Code)

	var resp map[string]string
	err := json.NewDecoder(rec.Body).Decode(&resp)
	s.Require().NoError(err)
	s.Contains(resp["error"], "invalid date")
}

// --- PUT /api/v1/planner/{date} ---

func (s *PlannerHandlerSuite) TestPutScheduleCreatesNew() {
	mock := &mockScheduleStore{}
	h := handler.PutScheduleHandler(mock)

	body := `{
		"strategy": "focus-maximized",
		"blocks": [
			{"start": "09:00", "end": "09:25", "type": "focus", "task_name": "Write tests"},
			{"start": "09:25", "end": "09:30", "type": "short_break"}
		]
	}`

	req := httptest.NewRequest(http.MethodPut, "/api/v1/planner/2026-04-20", strings.NewReader(body))
	req.SetPathValue("date", "2026-04-20")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)

	var resp map[string]any
	err := json.NewDecoder(rec.Body).Decode(&resp)
	s.Require().NoError(err)

	s.Equal("2026-04-20", resp["date"])
	s.Equal("focus-maximized", resp["strategy"])

	blocks, ok := resp["blocks"].([]any)
	s.Require().True(ok)
	s.Len(blocks, 2)

	b0 := blocks[0].(map[string]any)
	s.Equal("09:00", b0["start"])
	s.Equal("09:25", b0["end"])
	s.Equal("focus", b0["type"])
	s.Equal("Write tests", b0["task_name"])
}

func (s *PlannerHandlerSuite) TestPutScheduleInvalidDate() {
	mock := &mockScheduleStore{}
	h := handler.PutScheduleHandler(mock)

	body := `{"strategy": "focus-maximized", "blocks": []}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/planner/not-a-date", strings.NewReader(body))
	req.SetPathValue("date", "not-a-date")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	s.Equal(http.StatusBadRequest, rec.Code)
}

func (s *PlannerHandlerSuite) TestPutScheduleInvalidBody() {
	mock := &mockScheduleStore{}
	h := handler.PutScheduleHandler(mock)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/planner/2026-04-20", strings.NewReader("{invalid"))
	req.SetPathValue("date", "2026-04-20")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	s.Equal(http.StatusBadRequest, rec.Code)
}

func (s *PlannerHandlerSuite) TestPutScheduleInvalidBlockType() {
	mock := &mockScheduleStore{}
	h := handler.PutScheduleHandler(mock)

	body := `{
		"strategy": "focus-maximized",
		"blocks": [
			{"start": "09:00", "end": "09:25", "type": "nap_time"}
		]
	}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/planner/2026-04-20", strings.NewReader(body))
	req.SetPathValue("date", "2026-04-20")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	s.Equal(http.StatusBadRequest, rec.Code)
}

// --- DELETE /api/v1/planner/{date} ---

func (s *PlannerHandlerSuite) TestDeleteScheduleSuccess() {
	mock := &mockScheduleStore{
		loadResult: &repository.Schedule{
			ID:        uuid.New(),
			Date:      time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
			Strategy:  "focus-maximized",
			CreatedAt: time.Now(),
		},
	}
	h := handler.DeleteScheduleHandler(mock)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/planner/2026-04-20", nil)
	req.SetPathValue("date", "2026-04-20")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	s.Equal(http.StatusNoContent, rec.Code)
}

func (s *PlannerHandlerSuite) TestDeleteScheduleNotFound() {
	mock := &mockScheduleStore{loadErr: repository.ErrNotFound}
	h := handler.DeleteScheduleHandler(mock)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/planner/2026-04-20", nil)
	req.SetPathValue("date", "2026-04-20")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	s.Equal(http.StatusNotFound, rec.Code)
}

func (s *PlannerHandlerSuite) TestDeleteScheduleInvalidDate() {
	mock := &mockScheduleStore{}
	h := handler.DeleteScheduleHandler(mock)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/planner/not-a-date", nil)
	req.SetPathValue("date", "not-a-date")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	s.Equal(http.StatusBadRequest, rec.Code)
}
