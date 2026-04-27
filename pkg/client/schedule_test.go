package client_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/pkg/client"
)

// ScheduleSuite covers the ScheduleClient adapter over /api/v1/planner/*.
type ScheduleSuite struct {
	suite.Suite
}

func TestSchedule(t *testing.T) {
	suite.Run(t, new(ScheduleSuite))
}

// testScheduleDate is the deterministic date used across suite tests so
// path interpolation (YYYY-MM-DD) can be asserted directly.
var testScheduleDate = time.Date(2026, 4, 25, 0, 0, 0, 0, time.UTC)

// TestActiveScheduleReturnsSchedule verifies that ActiveSchedule issues
// GET /api/v1/planner/active and decodes the full Schedule payload,
// including a block with a non-nil task_id pointer.
func (s *ScheduleSuite) TestActiveScheduleReturnsSchedule() {
	taskID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal("/api/v1/planner/active", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"date":     "2026-04-24",
			"strategy": "pomodoro",
			"blocks": []map[string]any{
				{
					"start":     "09:00",
					"end":       "09:25",
					"type":      "focus",
					"task_id":   taskID,
					"task_name": "Design review",
				},
				{
					"start": "09:25",
					"end":   "09:30",
					"type":  "short_break",
				},
			},
			"created_at": "2026-04-24T08:00:00Z",
		})
	}))
	defer ts.Close()

	sc := client.NewScheduleClient(client.New(ts.URL))
	schedule, err := sc.ActiveSchedule(context.Background())
	s.Require().NoError(err)
	s.Require().NotNil(schedule)
	s.Equal("2026-04-24", schedule.Date)
	s.Equal("pomodoro", schedule.Strategy)
	s.Equal("2026-04-24T08:00:00Z", schedule.CreatedAt)
	s.Require().Len(schedule.Blocks, 2)

	first := schedule.Blocks[0]
	s.Equal("09:00", first.Start)
	s.Equal("09:25", first.End)
	s.Equal("focus", first.Type)
	s.Require().NotNil(first.TaskID, "task_id must decode to non-nil pointer")
	s.Equal(taskID, *first.TaskID)
	s.Equal("Design review", first.TaskName)

	second := schedule.Blocks[1]
	s.Nil(second.TaskID, "task_id must decode to nil when omitted by server")
	s.Equal("", second.TaskName)
}

// TestDeleteActiveScheduleReturnsNil verifies that DeleteActiveSchedule
// issues DELETE /api/v1/planner/active, tolerates a 204 No Content body,
// and returns nil.
func (s *ScheduleSuite) TestDeleteActiveScheduleReturnsNil() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodDelete, r.Method)
		s.Equal("/api/v1/planner/active", r.URL.Path)

		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	sc := client.NewScheduleClient(client.New(ts.URL))
	err := sc.DeleteActiveSchedule(context.Background())
	s.Require().NoError(err)
}

// TestGetScheduleUsesDatePath verifies that GetSchedule formats the supplied
// date as YYYY-MM-DD and uses it as the {date} path parameter.
func (s *ScheduleSuite) TestGetScheduleUsesDatePath() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal("/api/v1/planner/2026-04-25", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"date":       "2026-04-25",
			"strategy":   "long-form",
			"blocks":     []map[string]any{},
			"created_at": "2026-04-24T10:00:00Z",
		})
	}))
	defer ts.Close()

	sc := client.NewScheduleClient(client.New(ts.URL))
	schedule, err := sc.GetSchedule(context.Background(), testScheduleDate)
	s.Require().NoError(err)
	s.Require().NotNil(schedule)
	s.Equal("2026-04-25", schedule.Date)
	s.Equal("long-form", schedule.Strategy)
}

// TestPutSchedulePutsBody verifies that PutSchedule issues PUT with the
// strategy+blocks body (and NO date field — the server takes date from the
// URL) and decodes the returned full Schedule.
func (s *ScheduleSuite) TestPutSchedulePutsBody() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPut, r.Method)
		s.Equal("/api/v1/planner/2026-04-25", r.URL.Path)

		raw, err := io.ReadAll(r.Body)
		s.Require().NoError(err)

		// The outgoing body must carry strategy + blocks. It must NOT carry
		// a top-level "date" field since the server derives date from URL.
		var body map[string]json.RawMessage
		s.Require().NoError(json.Unmarshal(raw, &body))
		s.Contains(body, "strategy")
		s.Contains(body, "blocks")
		s.NotContains(body, "date", "date must not be in PUT body; it lives in URL")
		s.NotContains(body, "created_at", "created_at must not be sent by the client")

		var strategy string
		s.Require().NoError(json.Unmarshal(body["strategy"], &strategy))
		s.Equal("pomodoro", strategy)

		var blocks []client.ScheduleBlock
		s.Require().NoError(json.Unmarshal(body["blocks"], &blocks))
		s.Require().Len(blocks, 1)
		s.Equal("focus", blocks[0].Type)
		s.Equal("09:00", blocks[0].Start)
		s.Equal("09:25", blocks[0].End)

		// Respond with the full stored Schedule.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"date":     "2026-04-25",
			"strategy": "pomodoro",
			"blocks": []map[string]any{
				{
					"start":     "09:00",
					"end":       "09:25",
					"type":      "focus",
					"task_name": "Design",
				},
			},
			"created_at": "2026-04-25T08:00:00Z",
		})
	}))
	defer ts.Close()

	sc := client.NewScheduleClient(client.New(ts.URL))
	schedule, err := sc.PutSchedule(context.Background(), testScheduleDate, client.PutScheduleRequest{
		Strategy: "pomodoro",
		Blocks: []client.ScheduleBlock{
			{Start: "09:00", End: "09:25", Type: "focus", TaskName: "Design"},
		},
	})
	s.Require().NoError(err)
	s.Require().NotNil(schedule)
	s.Equal("2026-04-25", schedule.Date)
	s.Equal("pomodoro", schedule.Strategy)
	s.Require().Len(schedule.Blocks, 1)
	s.Equal("Design", schedule.Blocks[0].TaskName)
}

// TestDeleteScheduleByDate verifies that DeleteSchedule issues DELETE against
// the date-formatted path and returns nil on 204 No Content.
func (s *ScheduleSuite) TestDeleteScheduleByDate() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodDelete, r.Method)
		s.Equal("/api/v1/planner/2026-04-25", r.URL.Path)

		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	sc := client.NewScheduleClient(client.New(ts.URL))
	err := sc.DeleteSchedule(context.Background(), testScheduleDate)
	s.Require().NoError(err)
}

// TestGenerateSchedulesWithDate verifies that GenerateSchedules POSTs a
// body containing {"date":"YYYY-MM-DD"} when Date is populated, and decodes
// the two-option response payload correctly.
func (s *ScheduleSuite) TestGenerateSchedulesWithDate() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPost, r.Method)
		s.Equal("/api/v1/planner/generate", r.URL.Path)

		var body struct {
			Date string `json:"date"`
		}
		s.Require().NoError(json.NewDecoder(r.Body).Decode(&body))
		s.Equal("2026-04-25", body.Date)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"date": "2026-04-25",
			"options": []map[string]any{
				{
					"strategy":            "pomodoro",
					"total_focus_minutes": 150,
					"break_count":         3,
					"blocks": []map[string]any{
						{"start": "09:00", "end": "09:25", "type": "focus"},
					},
				},
				{
					"strategy":            "long-form",
					"total_focus_minutes": 180,
					"break_count":         2,
					"blocks": []map[string]any{
						{"start": "09:00", "end": "10:30", "type": "focus"},
					},
				},
			},
		})
	}))
	defer ts.Close()

	sc := client.NewScheduleClient(client.New(ts.URL))
	resp, err := sc.GenerateSchedules(context.Background(), client.GenerateSchedulesRequest{
		Date: "2026-04-25",
	})
	s.Require().NoError(err)
	s.Require().NotNil(resp)
	s.Equal("2026-04-25", resp.Date)
	s.Require().Len(resp.Options, 2)
	s.Equal("pomodoro", resp.Options[0].Strategy)
	s.Equal(150, resp.Options[0].TotalFocusMinutes)
	s.Equal(3, resp.Options[0].BreakCount)
	s.Equal("long-form", resp.Options[1].Strategy)
	s.Equal(180, resp.Options[1].TotalFocusMinutes)
}

// TestGenerateSchedulesWithoutDateOmitsField verifies that when the caller
// leaves Date empty, the outgoing JSON body does NOT include a "date" field
// (via the omitempty tag), so the server falls back to its own target date.
func (s *ScheduleSuite) TestGenerateSchedulesWithoutDateOmitsField() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPost, r.Method)
		s.Equal("/api/v1/planner/generate", r.URL.Path)

		raw, err := io.ReadAll(r.Body)
		s.Require().NoError(err)
		var body map[string]json.RawMessage
		s.Require().NoError(json.Unmarshal(raw, &body))
		s.NotContains(body, "date", "date must be omitted when caller leaves it empty")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"date": "2026-04-24",
			"options": []map[string]any{
				{
					"strategy":            "pomodoro",
					"total_focus_minutes": 50,
					"break_count":         1,
					"blocks":              []map[string]any{},
				},
				{
					"strategy":            "long-form",
					"total_focus_minutes": 60,
					"break_count":         0,
					"blocks":              []map[string]any{},
				},
			},
		})
	}))
	defer ts.Close()

	sc := client.NewScheduleClient(client.New(ts.URL))
	resp, err := sc.GenerateSchedules(context.Background(), client.GenerateSchedulesRequest{})
	s.Require().NoError(err)
	s.Require().NotNil(resp)
	s.Equal("2026-04-24", resp.Date)
	s.Require().Len(resp.Options, 2)
}

// TestGetScheduleNotFoundReturnsAPIError verifies that a 404 on GET surfaces
// as an *APIError with ErrCodeNotFound.
func (s *ScheduleSuite) TestGetScheduleNotFoundReturnsAPIError() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal("/api/v1/planner/2026-04-25", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "schedule not found",
		})
	}))
	defer ts.Close()

	sc := client.NewScheduleClient(client.New(ts.URL))
	schedule, err := sc.GetSchedule(context.Background(), testScheduleDate)
	s.Require().Error(err)
	s.Nil(schedule)

	var apiErr *client.APIError
	s.Require().True(errors.As(err, &apiErr), "expected *APIError, got %T", err)
	s.Equal(client.ErrCodeNotFound, apiErr.Code)
	s.Equal(http.StatusNotFound, apiErr.StatusCode)
}
