package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
)

// ErrNotImplemented is returned by stub functions that have not yet been implemented.
var ErrNotImplemented = errors.New("not implemented")

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

// GetScheduleHandler returns an http.HandlerFunc for GET /api/v1/planner/{date}.
func GetScheduleHandler(_ ScheduleQuerier) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSONError(w, http.StatusNotImplemented, ErrNotImplemented.Error())
	}
}
