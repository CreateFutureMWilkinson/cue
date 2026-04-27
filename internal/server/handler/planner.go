package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
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

// PutScheduleHandler returns an http.HandlerFunc for PUT /api/v1/planner/{date}.
func PutScheduleHandler(store ScheduleStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dateStr := r.PathValue("date")
		date, err := time.Parse("2006-01-02", dateStr)
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

		respBlocks := make([]scheduleBlockItem, len(schedule.Blocks))
		for i, b := range schedule.Blocks {
			item := scheduleBlockItem{
				Start:    b.Start.Format("15:04"),
				End:      b.End.Format("15:04"),
				Type:     blockTypeString(b.Type),
				TaskName: b.TaskName,
			}
			if b.TaskID != nil {
				s := b.TaskID.String()
				item.TaskID = &s
			}
			respBlocks[i] = item
		}

		resp := scheduleResponse{
			Date:      schedule.Date.Format("2006-01-02"),
			Strategy:  schedule.Strategy,
			Blocks:    respBlocks,
			CreatedAt: schedule.CreatedAt.Format(time.RFC3339),
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

// DeleteScheduleHandler returns an http.HandlerFunc for DELETE /api/v1/planner/{date}.
func DeleteScheduleHandler(store ScheduleStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dateStr := r.PathValue("date")
		date, err := time.Parse("2006-01-02", dateStr)
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

// GetScheduleHandler returns an http.HandlerFunc for GET /api/v1/planner/{date}.
func GetScheduleHandler(repo ScheduleStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dateStr := r.PathValue("date")
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid date format")
			return
		}

		schedule, err := repo.LoadByDate(r.Context(), date)
		if err != nil {
			writeNotFoundOrError(w, err)
			return
		}

		blocks := make([]scheduleBlockItem, len(schedule.Blocks))
		for i, b := range schedule.Blocks {
			item := scheduleBlockItem{
				Start:    b.Start.Format("15:04"),
				End:      b.End.Format("15:04"),
				Type:     blockTypeString(b.Type),
				TaskName: b.TaskName,
			}
			if b.TaskID != nil {
				s := b.TaskID.String()
				item.TaskID = &s
			}
			blocks[i] = item
		}

		resp := scheduleResponse{
			Date:      schedule.Date.Format("2006-01-02"),
			Strategy:  schedule.Strategy,
			Blocks:    blocks,
			CreatedAt: schedule.CreatedAt.Format(time.RFC3339),
		}
		writeJSON(w, http.StatusOK, resp)
	}
}
