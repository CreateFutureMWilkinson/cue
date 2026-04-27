package handler_test

import (
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

// mockTimerClock implements handler.TimerClock for testing.
type mockTimerClock struct{ now time.Time }

func (m *mockTimerClock) Now() time.Time { return m.now }

// TimerHandlerSuite tests the GET /api/v1/timer endpoint.
type TimerHandlerSuite struct {
	suite.Suite
}

func TestTimerHandler(t *testing.T) {
	suite.Run(t, new(TimerHandlerSuite))
}

// Behavior 8: GET timer with no schedule returns {"running": false}.
func (s *TimerHandlerSuite) TestGetTimerNoScheduleReturnsNotRunning() {
	store := &mockScheduleStore{loadErr: repository.ErrNotFound}
	clock := &mockTimerClock{now: time.Date(2026, 4, 22, 9, 10, 0, 0, time.UTC)}

	h := handler.GetTimerHandler(store, clock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/timer", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)

	var resp map[string]any
	err := json.NewDecoder(rec.Body).Decode(&resp)
	s.Require().NoError(err)

	s.Equal(false, resp["running"])
	// Omitted fields should not be present (omitempty)
	s.Nil(resp["block_type"])
	s.Nil(resp["task_name"])
}

// Behavior 9: GET timer with active schedule returns full timer state.
func (s *TimerHandlerSuite) TestGetTimerActiveBlockReturnsFullState() {
	blockStart := time.Date(2026, 4, 22, 9, 0, 0, 0, time.UTC)
	blockEnd := time.Date(2026, 4, 22, 9, 25, 0, 0, time.UTC)

	schedule := &repository.Schedule{
		ID:       uuid.New(),
		Date:     time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC),
		Strategy: "focus-maximized",
		Blocks: []repository.ScheduleBlock{
			{
				Start:    blockStart,
				End:      blockEnd,
				Type:     repository.ScheduleBlockFocus,
				TaskName: "Deep work",
			},
		},
		CreatedAt: time.Now(),
	}

	store := &mockScheduleStore{loadResult: schedule}
	clock := &mockTimerClock{now: time.Date(2026, 4, 22, 9, 10, 0, 0, time.UTC)} // 10 min in

	h := handler.GetTimerHandler(store, clock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/timer", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)

	var resp map[string]any
	err := json.NewDecoder(rec.Body).Decode(&resp)
	s.Require().NoError(err)

	s.Equal(true, resp["running"])
	s.Equal("focus", resp["block_type"])
	s.Equal("Deep work", resp["task_name"])
	s.Equal(float64(600), resp["elapsed_seconds"])
	s.Equal(float64(900), resp["remaining_seconds"])
	s.Equal("15:00", resp["display_time"])
	s.Equal(float64(1500), resp["duration_seconds"])
	s.InDelta(0.4, resp["elapsed_fraction"], 0.01)
}

// Behavior 10: GET timer after all blocks are finished returns {"running": false}.
func (s *TimerHandlerSuite) TestGetTimerCompletedScheduleReturnsNotRunning() {
	blockStart := time.Date(2026, 4, 22, 9, 0, 0, 0, time.UTC)
	blockEnd := time.Date(2026, 4, 22, 9, 25, 0, 0, time.UTC)

	schedule := &repository.Schedule{
		ID:       uuid.New(),
		Date:     time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC),
		Strategy: "focus-maximized",
		Blocks: []repository.ScheduleBlock{
			{
				Start:    blockStart,
				End:      blockEnd,
				Type:     repository.ScheduleBlockFocus,
				TaskName: "Deep work",
			},
		},
		CreatedAt: time.Now(),
	}

	store := &mockScheduleStore{loadResult: schedule}
	clock := &mockTimerClock{now: time.Date(2026, 4, 22, 10, 0, 0, 0, time.UTC)} // after all blocks

	h := handler.GetTimerHandler(store, clock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/timer", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)

	var resp map[string]any
	err := json.NewDecoder(rec.Body).Decode(&resp)
	s.Require().NoError(err)

	s.Equal(false, resp["running"])
}
