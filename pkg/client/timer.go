package client

import (
	"context"
	"net/http"
)

const timerPath = "/api/v1/timer"

// TimerState mirrors the server's timerResponse emitted by GET /api/v1/timer.
//
// When Running is false the server sends only {"running":false}; all other
// fields use omitempty so callers can distinguish "no active block" from a
// block whose metadata happens to be zero-valued.
type TimerState struct {
	Running          bool    `json:"running"`
	BlockType        string  `json:"block_type,omitempty"`
	TaskName         string  `json:"task_name,omitempty"`
	DurationSeconds  int     `json:"duration_seconds,omitempty"`
	ElapsedSeconds   int     `json:"elapsed_seconds,omitempty"`
	RemainingSeconds int     `json:"remaining_seconds,omitempty"`
	DisplayTime      string  `json:"display_time,omitempty"`
	ElapsedFraction  float64 `json:"elapsed_fraction,omitempty"`
}

// TimerClient wraps /api/v1/timer.
type TimerClient interface {
	GetTimerState(ctx context.Context) (*TimerState, error)
}

// timerAdapter is the concrete TimerClient backed by an *APIClient.
type timerAdapter struct {
	client *APIClient
}

// NewTimerClient returns a TimerClient backed by the given APIClient.
func NewTimerClient(c *APIClient) TimerClient {
	return &timerAdapter{client: c}
}

// GetTimerState issues GET /api/v1/timer and returns the decoded TimerState.
func (t *timerAdapter) GetTimerState(ctx context.Context) (*TimerState, error) {
	var state TimerState
	if err := t.client.doJSON(ctx, http.MethodGet, timerPath, nil, &state); err != nil {
		return nil, err
	}
	return &state, nil
}
