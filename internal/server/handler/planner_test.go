package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/server/handler"
	"github.com/CreateFutureMWilkinson/cue/internal/service/calendar"
	"github.com/CreateFutureMWilkinson/cue/internal/service/planner"
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

// mockScheduleGenerator implements handler.ScheduleGenerator for testing.
type mockScheduleGenerator struct {
	focusSchedule    *planner.DaySchedule
	recoverySchedule *planner.DaySchedule
	genErr           error
	targetDate       time.Time
}

func (m *mockScheduleGenerator) GenerateSchedules(_ context.Context, _ []planner.TaskEstimate, _ []calendar.CalendarEvent, _ time.Time) (*planner.DaySchedule, *planner.DaySchedule, error) {
	return m.focusSchedule, m.recoverySchedule, m.genErr
}

func (m *mockScheduleGenerator) TargetDate(_ time.Time) time.Time {
	return m.targetDate
}

// mockCalendarFetcher implements handler.CalendarFetcher for testing.
type mockCalendarFetcher struct {
	events   []calendar.CalendarEvent
	fetchErr error
}

func (m *mockCalendarFetcher) FetchEvents(_ context.Context, _ time.Time) ([]calendar.CalendarEvent, error) {
	return m.events, m.fetchErr
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

// --- POST /api/v1/planner/generate ---

func (s *PlannerHandlerSuite) TestGenerateSchedulesWithDate() {
	targetDate := time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC)
	taskID := uuid.New()

	calMock := &mockCalendarFetcher{
		events: []calendar.CalendarEvent{
			{
				ID:    "meeting-1",
				Title: "Stand-up",
				Start: time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC),
				End:   time.Date(2026, 4, 20, 10, 30, 0, 0, time.UTC),
			},
		},
	}

	genMock := &mockScheduleGenerator{
		focusSchedule: &planner.DaySchedule{
			ID:       uuid.New(),
			Date:     targetDate,
			Strategy: "focus-maximized",
			Blocks: []planner.TimeBlock{
				{Start: time.Date(2026, 4, 20, 9, 0, 0, 0, time.UTC), End: time.Date(2026, 4, 20, 9, 25, 0, 0, time.UTC), Type: planner.BlockFocus, TaskID: &taskID, TaskName: "Write tests"},
				{Start: time.Date(2026, 4, 20, 9, 25, 0, 0, time.UTC), End: time.Date(2026, 4, 20, 9, 30, 0, 0, time.UTC), Type: planner.BlockShortBreak},
				{Start: time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC), End: time.Date(2026, 4, 20, 10, 30, 0, 0, time.UTC), Type: planner.BlockMeeting, TaskName: "Stand-up"},
			},
			CreatedAt: time.Now(),
		},
		recoverySchedule: &planner.DaySchedule{
			ID:       uuid.New(),
			Date:     targetDate,
			Strategy: "recovery-balanced",
			Blocks: []planner.TimeBlock{
				{Start: time.Date(2026, 4, 20, 9, 0, 0, 0, time.UTC), End: time.Date(2026, 4, 20, 9, 25, 0, 0, time.UTC), Type: planner.BlockFocus},
				{Start: time.Date(2026, 4, 20, 9, 25, 0, 0, time.UTC), End: time.Date(2026, 4, 20, 9, 30, 0, 0, time.UTC), Type: planner.BlockShortBreak},
				{Start: time.Date(2026, 4, 20, 9, 30, 0, 0, time.UTC), End: time.Date(2026, 4, 20, 9, 55, 0, 0, time.UTC), Type: planner.BlockFocus},
				{Start: time.Date(2026, 4, 20, 9, 55, 0, 0, time.UTC), End: time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC), Type: planner.BlockShortBreak},
				{Start: time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC), End: time.Date(2026, 4, 20, 10, 30, 0, 0, time.UTC), Type: planner.BlockMeeting, TaskName: "Stand-up"},
			},
			CreatedAt: time.Now(),
		},
	}

	h := handler.GenerateSchedulesHandler(genMock, calMock)

	body := `{"date": "2026-04-20"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/planner/generate", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)

	var resp map[string]any
	err := json.NewDecoder(rec.Body).Decode(&resp)
	s.Require().NoError(err)

	s.Equal("2026-04-20", resp["date"])

	options, ok := resp["options"].([]any)
	s.Require().True(ok)
	s.Require().Len(options, 2)

	// First option: focus-maximized
	opt0 := options[0].(map[string]any)
	s.Equal("focus-maximized", opt0["strategy"])
	// 1 focus block * 25 min = 25 total focus minutes
	s.Equal(float64(25), opt0["total_focus_minutes"])
	// 1 short break
	s.Equal(float64(1), opt0["break_count"])

	blocks0, ok := opt0["blocks"].([]any)
	s.Require().True(ok)
	s.Require().Len(blocks0, 3)

	b0 := blocks0[0].(map[string]any)
	s.Equal("09:00", b0["start"])
	s.Equal("09:25", b0["end"])
	s.Equal("focus", b0["type"])

	b1 := blocks0[1].(map[string]any)
	s.Equal("09:25", b1["start"])
	s.Equal("09:30", b1["end"])
	s.Equal("short_break", b1["type"])

	b2 := blocks0[2].(map[string]any)
	s.Equal("10:00", b2["start"])
	s.Equal("10:30", b2["end"])
	s.Equal("meeting", b2["type"])

	// Second option: recovery-balanced
	opt1 := options[1].(map[string]any)
	s.Equal("recovery-balanced", opt1["strategy"])
	// 2 focus blocks * 25 min = 50 total focus minutes
	s.Equal(float64(50), opt1["total_focus_minutes"])
	// 2 short breaks
	s.Equal(float64(2), opt1["break_count"])

	blocks1, ok := opt1["blocks"].([]any)
	s.Require().True(ok)
	s.Require().Len(blocks1, 5)
}

func (s *PlannerHandlerSuite) TestGenerateSchedulesDefaultDate() {
	defaultDate := time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC)

	calMock := &mockCalendarFetcher{
		events: nil,
	}

	genMock := &mockScheduleGenerator{
		targetDate: defaultDate,
		focusSchedule: &planner.DaySchedule{
			ID:       uuid.New(),
			Date:     defaultDate,
			Strategy: "focus-maximized",
			Blocks: []planner.TimeBlock{
				{Start: time.Date(2026, 4, 21, 9, 0, 0, 0, time.UTC), End: time.Date(2026, 4, 21, 9, 25, 0, 0, time.UTC), Type: planner.BlockFocus},
			},
			CreatedAt: time.Now(),
		},
		recoverySchedule: &planner.DaySchedule{
			ID:       uuid.New(),
			Date:     defaultDate,
			Strategy: "recovery-balanced",
			Blocks: []planner.TimeBlock{
				{Start: time.Date(2026, 4, 21, 9, 0, 0, 0, time.UTC), End: time.Date(2026, 4, 21, 9, 25, 0, 0, time.UTC), Type: planner.BlockFocus},
			},
			CreatedAt: time.Now(),
		},
	}

	h := handler.GenerateSchedulesHandler(genMock, calMock)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/planner/generate", strings.NewReader("{}"))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)

	var resp map[string]any
	err := json.NewDecoder(rec.Body).Decode(&resp)
	s.Require().NoError(err)

	// Should use the default date from TargetDate
	s.Equal("2026-04-21", resp["date"])

	options, ok := resp["options"].([]any)
	s.Require().True(ok)
	s.Len(options, 2)
}

// --- GET /api/v1/planner/active ---

func (s *PlannerHandlerSuite) TestGetActiveSchedule() {
	today := time.Now().UTC().Format("2006-01-02")
	todayDate, err := time.Parse("2006-01-02", today)
	s.Require().NoError(err)

	schedule := &repository.Schedule{
		ID:       uuid.New(),
		Date:     todayDate,
		Strategy: "focus-maximized",
		Blocks: []repository.ScheduleBlock{
			{
				Start:    time.Date(todayDate.Year(), todayDate.Month(), todayDate.Day(), 9, 0, 0, 0, time.UTC),
				End:      time.Date(todayDate.Year(), todayDate.Month(), todayDate.Day(), 9, 25, 0, 0, time.UTC),
				Type:     repository.ScheduleBlockFocus,
				TaskName: "Active task",
			},
		},
		CreatedAt: time.Now(),
	}

	mock := &mockScheduleStore{loadResult: schedule}
	h := handler.ActiveDateHandler(handler.GetScheduleHandler(mock))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/planner/active", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)

	var resp map[string]any
	err = json.NewDecoder(rec.Body).Decode(&resp)
	s.Require().NoError(err)

	s.Equal(today, resp["date"])
	s.Equal("focus-maximized", resp["strategy"])
}

// --- DELETE /api/v1/planner/active ---

func (s *PlannerHandlerSuite) TestDeleteActiveSchedule() {
	today := time.Now().UTC().Format("2006-01-02")
	todayDate, err := time.Parse("2006-01-02", today)
	s.Require().NoError(err)

	mock := &mockScheduleStore{
		loadResult: &repository.Schedule{
			ID:        uuid.New(),
			Date:      todayDate,
			Strategy:  "focus-maximized",
			CreatedAt: time.Now(),
		},
	}
	h := handler.ActiveDateHandler(handler.DeleteScheduleHandler(mock))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/planner/active", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	s.Equal(http.StatusNoContent, rec.Code)
}

func (s *PlannerHandlerSuite) TestGenerateSchedulesCalendarError() {
	calMock := &mockCalendarFetcher{
		fetchErr: errors.New("calendar unavailable"),
	}

	genMock := &mockScheduleGenerator{
		targetDate: time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
	}

	h := handler.GenerateSchedulesHandler(genMock, calMock)

	body := `{"date": "2026-04-20"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/planner/generate", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	s.Equal(http.StatusInternalServerError, rec.Code)
}
