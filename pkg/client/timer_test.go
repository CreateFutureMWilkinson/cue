package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/pkg/client"
)

// TimerSuite covers the TimerClient adapter over /api/v1/timer.
type TimerSuite struct {
	suite.Suite
}

func TestTimer(t *testing.T) {
	suite.Run(t, new(TimerSuite))
}

// TestGetTimerStateRunning verifies that GetTimerState issues
// GET /api/v1/timer and decodes the full running TimerState payload,
// including elapsed_fraction as a float.
func (s *TimerSuite) TestGetTimerStateRunning() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal("/api/v1/timer", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"running":           true,
			"block_type":        "focus",
			"task_name":         "Design review",
			"duration_seconds":  1500,
			"elapsed_seconds":   300,
			"remaining_seconds": 1200,
			"display_time":      "20:00",
			"elapsed_fraction":  0.2,
		})
	}))
	defer ts.Close()

	tc := client.NewTimerClient(client.New(ts.URL))
	state, err := tc.GetTimerState(context.Background())
	s.Require().NoError(err)
	s.Require().NotNil(state)
	s.True(state.Running)
	s.Equal("focus", state.BlockType)
	s.Equal("Design review", state.TaskName)
	s.Equal(1500, state.DurationSeconds)
	s.Equal(300, state.ElapsedSeconds)
	s.Equal(1200, state.RemainingSeconds)
	s.Equal("20:00", state.DisplayTime)
	s.InDelta(0.2, state.ElapsedFraction, 0.0001)
}

// TestGetTimerStateStoppedOmitsFields verifies that when the server returns
// only {"running":false} (all other fields omitted via omitempty), the
// TimerState decodes without error and every non-Running field stays at its
// Go zero value.
func (s *TimerSuite) TestGetTimerStateStoppedOmitsFields() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal("/api/v1/timer", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"running": false,
		})
	}))
	defer ts.Close()

	tc := client.NewTimerClient(client.New(ts.URL))
	state, err := tc.GetTimerState(context.Background())
	s.Require().NoError(err)
	s.Require().NotNil(state)
	s.False(state.Running)
	s.Equal("", state.BlockType)
	s.Equal("", state.TaskName)
	s.Equal(0, state.DurationSeconds)
	s.Equal(0, state.ElapsedSeconds)
	s.Equal(0, state.RemainingSeconds)
	s.Equal("", state.DisplayTime)
	s.InDelta(0.0, state.ElapsedFraction, 0.0001)
}
