package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/server/handler"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// mockScheduleQuerier implements handler.ScheduleQuerier for testing.
type mockScheduleQuerier struct {
	loadResult *repository.Schedule
	loadErr    error
}

func (m *mockScheduleQuerier) LoadByDate(_ context.Context, _ time.Time) (*repository.Schedule, error) {
	return m.loadResult, m.loadErr
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

	mock := &mockScheduleQuerier{loadResult: schedule}
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
	mock := &mockScheduleQuerier{loadErr: repository.ErrNotFound}
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
	mock := &mockScheduleQuerier{}
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
