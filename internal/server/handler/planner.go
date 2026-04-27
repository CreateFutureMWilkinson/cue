package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
)

// ScheduleQuerier is the subset of ScheduleRepository needed by the GET schedule handler.
type ScheduleQuerier interface {
	LoadByDate(ctx context.Context, date time.Time) (*repository.Schedule, error)
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

// GetScheduleHandler returns an http.HandlerFunc for GET /api/v1/planner/{date}.
func GetScheduleHandler(repo ScheduleQuerier) http.HandlerFunc {
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
