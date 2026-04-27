package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/service/calendar"
	"github.com/CreateFutureMWilkinson/cue/internal/service/planner"
	"github.com/google/uuid"
)

// ErrNotImplemented is returned by stub handlers that have not been implemented yet.
var ErrNotImplemented = errors.New("not implemented")

// ScheduleStore is the subset of ScheduleRepository needed by the schedule handlers.
type ScheduleStore interface {
	LoadByDate(ctx context.Context, date time.Time) (*repository.Schedule, error)
	Save(ctx context.Context, schedule *repository.Schedule) error
	Delete(ctx context.Context, date time.Time) error
}

// scheduleBlockItem is the JSON representation of a single time block.
type scheduleBlockItem struct {
	Start    string  `json:"start"`
	End      string  `json:"end"`
	Type     string  `json:"type"`
	TaskID   *string `json:"task_id,omitempty"`
	TaskName string  `json:"task_name,omitempty"`
}

// scheduleResponse is the JSON representation of a day schedule.
type scheduleResponse struct {
	Date      string              `json:"date"`
	Strategy  string              `json:"strategy"`
	Blocks    []scheduleBlockItem `json:"blocks"`
	CreatedAt string              `json:"created_at"`
}

// blockTypeString converts a ScheduleBlockType to its JSON string representation.
func blockTypeString(t repository.ScheduleBlockType) string {
	switch t {
	case repository.ScheduleBlockFocus:
		return "focus"
	case repository.ScheduleBlockShortBreak:
		return "short_break"
	case repository.ScheduleBlockLongBreak:
		return "long_break"
	case repository.ScheduleBlockMeeting:
		return "meeting"
	default:
		return "unknown"
	}
}

// putScheduleRequest is the JSON body for PUT /api/v1/planner/{date}.
type putScheduleRequest struct {
	Strategy string              `json:"strategy"`
	Blocks   []scheduleBlockItem `json:"blocks"`
}

// parseBlockType converts a JSON block type string to its ScheduleBlockType constant.
// Returns the type and true if valid, or 0 and false if the string is unrecognized.
func parseBlockType(s string) (repository.ScheduleBlockType, bool) {
	switch s {
	case "focus":
		return repository.ScheduleBlockFocus, true
	case "short_break":
		return repository.ScheduleBlockShortBreak, true
	case "long_break":
		return repository.ScheduleBlockLongBreak, true
	case "meeting":
		return repository.ScheduleBlockMeeting, true
	default:
		return 0, false
	}
}

// scheduleToResponse converts a repository.Schedule to its JSON response form.
func scheduleToResponse(s *repository.Schedule) scheduleResponse {
	blocks := make([]scheduleBlockItem, len(s.Blocks))
	for i, b := range s.Blocks {
		item := scheduleBlockItem{
			Start:    b.Start.Format("15:04"),
			End:      b.End.Format("15:04"),
			Type:     blockTypeString(b.Type),
			TaskName: b.TaskName,
		}
		if b.TaskID != nil {
			str := b.TaskID.String()
			item.TaskID = &str
		}
		blocks[i] = item
	}
	return scheduleResponse{
		Date:      s.Date.Format("2006-01-02"),
		Strategy:  s.Strategy,
		Blocks:    blocks,
		CreatedAt: s.CreatedAt.Format(time.RFC3339),
	}
}

// parseDateParam extracts and validates a {date} path parameter.
func parseDateParam(r *http.Request) (time.Time, error) {
	return time.Parse("2006-01-02", r.PathValue("date"))
}

// GetScheduleHandler returns an http.HandlerFunc for GET /api/v1/planner/{date}.
func GetScheduleHandler(repo ScheduleStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		date, err := parseDateParam(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid date format")
			return
		}

		schedule, err := repo.LoadByDate(r.Context(), date)
		if err != nil {
			writeNotFoundOrError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, scheduleToResponse(schedule))
	}
}

// PutScheduleHandler returns an http.HandlerFunc for PUT /api/v1/planner/{date}.
func PutScheduleHandler(store ScheduleStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		date, err := parseDateParam(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid date format")
			return
		}

		var req putScheduleRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		blocks := make([]repository.ScheduleBlock, len(req.Blocks))
		for i, b := range req.Blocks {
			bt, ok := parseBlockType(b.Type)
			if !ok {
				writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid block type: %s", b.Type))
				return
			}
			start, err := time.Parse("15:04", b.Start)
			if err != nil {
				writeJSONError(w, http.StatusBadRequest, "invalid start time")
				return
			}
			end, err := time.Parse("15:04", b.End)
			if err != nil {
				writeJSONError(w, http.StatusBadRequest, "invalid end time")
				return
			}
			blocks[i] = repository.ScheduleBlock{
				Start:    time.Date(date.Year(), date.Month(), date.Day(), start.Hour(), start.Minute(), 0, 0, time.UTC),
				End:      time.Date(date.Year(), date.Month(), date.Day(), end.Hour(), end.Minute(), 0, 0, time.UTC),
				Type:     bt,
				TaskName: b.TaskName,
			}
		}

		schedule := &repository.Schedule{
			ID:        uuid.New(),
			Date:      date,
			Strategy:  req.Strategy,
			Blocks:    blocks,
			CreatedAt: time.Now(),
		}

		if err := store.Save(r.Context(), schedule); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to save schedule")
			return
		}

		writeJSON(w, http.StatusOK, scheduleToResponse(schedule))
	}
}

// ScheduleGenerator is the subset of planner.Planner needed to generate schedule options.
type ScheduleGenerator interface {
	GenerateSchedules(ctx context.Context, tasks []planner.TaskEstimate, events []calendar.CalendarEvent, targetDate time.Time) (*planner.DaySchedule, *planner.DaySchedule, error)
	TargetDate(now time.Time) time.Time
}

// CalendarFetcher is the subset of CalendarProvider needed to fetch events for a date.
type CalendarFetcher interface {
	FetchEvents(ctx context.Context, date time.Time) ([]calendar.CalendarEvent, error)
}

// generateRequest is the JSON body for POST /api/v1/planner/generate.
type generateRequest struct {
	Date *string `json:"date"`
}

// generateOptionItem is the JSON representation of a single schedule option.
type generateOptionItem struct {
	Strategy          string              `json:"strategy"`
	TotalFocusMinutes int                 `json:"total_focus_minutes"`
	BreakCount        int                 `json:"break_count"`
	Blocks            []scheduleBlockItem `json:"blocks"`
}

// generateResponse is the JSON response for POST /api/v1/planner/generate.
type generateResponse struct {
	Date    string               `json:"date"`
	Options []generateOptionItem `json:"options"`
}

// plannerBlockTypeString converts a planner.BlockType to its JSON string representation.
func plannerBlockTypeString(t planner.BlockType) string {
	switch t {
	case planner.BlockFocus:
		return "focus"
	case planner.BlockShortBreak:
		return "short_break"
	case planner.BlockLongBreak:
		return "long_break"
	case planner.BlockMeeting:
		return "meeting"
	default:
		return "unknown"
	}
}

// dayScheduleToOption converts a planner.DaySchedule to its generateOptionItem form.
func dayScheduleToOption(ds *planner.DaySchedule) generateOptionItem {
	blocks := make([]scheduleBlockItem, len(ds.Blocks))
	var totalFocus int
	var breakCount int
	for i, b := range ds.Blocks {
		blocks[i] = scheduleBlockItem{
			Start:    b.Start.Format("15:04"),
			End:      b.End.Format("15:04"),
			Type:     plannerBlockTypeString(b.Type),
			TaskName: b.TaskName,
		}
		if b.TaskID != nil {
			s := b.TaskID.String()
			blocks[i].TaskID = &s
		}
		switch b.Type {
		case planner.BlockFocus:
			totalFocus += int(b.End.Sub(b.Start).Minutes())
		case planner.BlockShortBreak, planner.BlockLongBreak:
			breakCount++
		}
	}
	return generateOptionItem{
		Strategy:          ds.Strategy,
		TotalFocusMinutes: totalFocus,
		BreakCount:        breakCount,
		Blocks:            blocks,
	}
}

// GenerateSchedulesHandler returns an http.HandlerFunc for POST /api/v1/planner/generate.
func GenerateSchedulesHandler(gen ScheduleGenerator, cal CalendarFetcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req generateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		var targetDate time.Time
		if req.Date != nil {
			var err error
			targetDate, err = time.Parse("2006-01-02", *req.Date)
			if err != nil {
				writeJSONError(w, http.StatusBadRequest, "invalid date format")
				return
			}
		} else {
			targetDate = gen.TargetDate(time.Now())
		}

		events, err := cal.FetchEvents(r.Context(), targetDate)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to fetch calendar events")
			return
		}

		focus, recovery, err := gen.GenerateSchedules(r.Context(), []planner.TaskEstimate{}, events, targetDate)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to generate schedules")
			return
		}

		resp := generateResponse{
			Date: targetDate.Format("2006-01-02"),
			Options: []generateOptionItem{
				dayScheduleToOption(focus),
				dayScheduleToOption(recovery),
			},
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

// DeleteScheduleHandler returns an http.HandlerFunc for DELETE /api/v1/planner/{date}.
func DeleteScheduleHandler(store ScheduleStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		date, err := parseDateParam(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid date format")
			return
		}

		if _, err := store.LoadByDate(r.Context(), date); err != nil {
			writeNotFoundOrError(w, err)
			return
		}

		if err := store.Delete(r.Context(), date); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to delete schedule")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
