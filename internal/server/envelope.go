package server

import "time"

// ActivityEnvelope is the JSON envelope wrapper for WebSocket activity event broadcasts.
// It provides sequencing, typing, and metadata for events streamed to connected clients.
type ActivityEnvelope struct {
	Seq              uint64    `json:"seq"`
	Type             string    `json:"type"`
	Timestamp        time.Time `json:"timestamp"`
	Data             any       `json:"data"`
	DroppedSinceLast int       `json:"dropped_since_last,omitempty"`
}

// ActivityData is the type-specific payload for activity events.
type ActivityData struct {
	Source  string `json:"source"`
	Message string `json:"message"`
	IsError bool   `json:"is_error"`
}

// AlertData is the type-specific payload for alert events.
// It represents different kinds of system alerts that can be broadcast
// to connected clients via the activity stream.
type AlertData struct {
	Kind string `json:"kind"`
}

// TimerTickData is the payload for timer_tick WebSocket events.
type TimerTickData struct {
	Running          bool    `json:"running"`
	BlockType        string  `json:"block_type"`
	TaskName         string  `json:"task_name"`
	ElapsedSeconds   int     `json:"elapsed_seconds"`
	RemainingSeconds int     `json:"remaining_seconds"`
	DisplayTime      string  `json:"display_time"`
	ElapsedFraction  float64 `json:"elapsed_fraction"`
}

// TimerBlockCompleteData is the payload for timer_block_complete WebSocket events.
type TimerBlockCompleteData struct {
	CompletedBlock string `json:"completed_block"`
	TaskName       string `json:"task_name"`
	NextBlock      string `json:"next_block"`
}
