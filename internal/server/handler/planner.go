package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
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
func parseBlockType(_ string) (repository.ScheduleBlockType, bool) {
	return 0, false
}

// PutScheduleHandler returns an http.HandlerFunc for PUT /api/v1/planner/{date}.
func PutScheduleHandler(_ ScheduleStore) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
	}
}

// DeleteScheduleHandler returns an http.HandlerFunc for DELETE /api/v1/planner/{date}.
func DeleteScheduleHandler(_ ScheduleStore) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
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
