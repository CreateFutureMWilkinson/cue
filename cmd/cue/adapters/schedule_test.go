package adapters_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/cmd/cue/adapters"
	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/service/planner"
	"github.com/CreateFutureMWilkinson/cue/pkg/client"
)

// ScheduleAdapterSuite covers the ScheduleRepository CRUD + the
// ScheduleGenerator translation from /api/v1/planner endpoints.
type ScheduleAdapterSuite struct {
	suite.Suite
}

func TestScheduleAdapter(t *testing.T) {
	suite.Run(t, new(ScheduleAdapterSuite))
}

// AC: Save → PUT, LoadByDate → GET, Delete → DELETE round-trip with
// HH:MM ↔ time.Time and string ↔ ScheduleBlockType translation.
func (s *ScheduleAdapterSuite) TestRepositoryRoundTrip() {
	day := time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC)
	dateStr := "2026-04-27"
	var lastPut client.PutScheduleRequest
	var deleteCalls int

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/planner/"+dateStr, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			body := map[string]any{
				"date":     dateStr,
				"strategy": "focus-maximized",
				"blocks": []any{
					map[string]any{
						"start":     "09:00",
						"end":       "09:25",
						"type":      "focus",
						"task_name": "Write report",
					},
					map[string]any{
						"start": "09:25",
						"end":   "09:30",
						"type":  "short_break",
					},
				},
				"created_at": "2026-04-27T08:00:00Z",
			}
			s.Require().NoError(json.NewEncoder(w).Encode(body))
		case http.MethodPut:
			body, err := io.ReadAll(r.Body)
			s.Require().NoError(err)
			s.Require().NoError(json.Unmarshal(body, &lastPut))
			w.Header().Set("Content-Type", "application/json")
			s.Require().NoError(json.NewEncoder(w).Encode(map[string]any{
				"date":       dateStr,
				"strategy":   lastPut.Strategy,
				"blocks":     lastPut.Blocks,
				"created_at": "2026-04-27T08:30:00Z",
			}))
		case http.MethodDelete:
			deleteCalls++
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	api := client.New(ts.URL)
	api.SetToken("test-token")
	a := adapters.NewScheduleAdapter(client.NewScheduleClient(api))
	ctx := context.Background()

	// LoadByDate.
	got, err := a.LoadByDate(ctx, day)
	s.Require().NoError(err)
	s.Require().NotNil(got)
	s.Equal("focus-maximized", got.Strategy)
	s.Require().Len(got.Blocks, 2)
	s.Equal(repository.ScheduleBlockFocus, got.Blocks[0].Type)
	s.Equal(9, got.Blocks[0].Start.Hour())
	s.Equal(0, got.Blocks[0].Start.Minute())
	s.Equal("Write report", got.Blocks[0].TaskName)
	s.Equal(repository.ScheduleBlockShortBreak, got.Blocks[1].Type)
	s.False(got.CreatedAt.IsZero())

	// Save.
	out := &repository.Schedule{
		Date:     day,
		Strategy: "recovery-balanced",
		Blocks: []repository.ScheduleBlock{
			{
				Start:    time.Date(2026, 4, 27, 10, 0, 0, 0, time.UTC),
				End:      time.Date(2026, 4, 27, 10, 25, 0, 0, time.UTC),
				Type:     repository.ScheduleBlockFocus,
				TaskName: "Lunch prep",
			},
		},
	}
	s.Require().NoError(a.Save(ctx, out))
	s.Equal("recovery-balanced", lastPut.Strategy)
	s.Require().Len(lastPut.Blocks, 1)
	s.Equal("10:00", lastPut.Blocks[0].Start)
	s.Equal("10:25", lastPut.Blocks[0].End)
	s.Equal("focus", lastPut.Blocks[0].Type)
	s.False(out.CreatedAt.IsZero(), "Save must populate CreatedAt from the response")

	// Delete.
	s.Require().NoError(a.Delete(ctx, day))
	s.Equal(1, deleteCalls)
}

// AC: LoadByDate returns (nil, nil) when the server reports 404 so
// the presenter's nil-check works without inspecting error values.
func (s *ScheduleAdapterSuite) TestLoadByDateMissing() {
	day := time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/planner/2026-04-27", func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"error":{"code":"NOT_FOUND","message":"no schedule"}}`))
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	api := client.New(ts.URL)
	api.SetToken("test-token")
	a := adapters.NewScheduleAdapter(client.NewScheduleClient(api))

	got, err := a.LoadByDate(context.Background(), day)
	s.Require().NoError(err)
	s.Nil(got, "missing schedule must surface as (nil, nil)")
}

// AC: GenerateSchedules splits the response options into focus and
// recovery DaySchedules with HH:MM strings parsed onto the request day.
func (s *ScheduleAdapterSuite) TestGenerateSchedules() {
	day := time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/planner/generate", func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPost, r.Method)
		w.Header().Set("Content-Type", "application/json")
		body := map[string]any{
			"date": "2026-04-27",
			"options": []any{
				map[string]any{
					"strategy":            "focus-maximized",
					"total_focus_minutes": 100,
					"break_count":         3,
					"blocks": []any{
						map[string]any{"start": "09:00", "end": "09:25", "type": "focus"},
					},
				},
				map[string]any{
					"strategy":            "recovery-balanced",
					"total_focus_minutes": 75,
					"break_count":         5,
					"blocks": []any{
						map[string]any{"start": "09:00", "end": "09:20", "type": "focus"},
						map[string]any{"start": "09:20", "end": "09:30", "type": "short_break"},
					},
				},
			},
		}
		s.Require().NoError(json.NewEncoder(w).Encode(body))
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	api := client.New(ts.URL)
	api.SetToken("test-token")
	a := adapters.NewScheduleAdapter(client.NewScheduleClient(api))

	focus, recovery, err := a.GenerateSchedules(context.Background(), day)
	s.Require().NoError(err)
	s.Require().NotNil(focus)
	s.Require().NotNil(recovery)
	s.Equal("focus-maximized", focus.Strategy)
	s.Equal("recovery-balanced", recovery.Strategy)
	s.Require().Len(focus.Blocks, 1)
	s.Equal(planner.BlockFocus, focus.Blocks[0].Type)
	s.Equal(9, focus.Blocks[0].Start.Hour())
	s.Require().Len(recovery.Blocks, 2)
	s.Equal(planner.BlockShortBreak, recovery.Blocks[1].Type)
}
