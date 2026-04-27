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
