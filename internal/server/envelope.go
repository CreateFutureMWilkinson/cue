package server

import "time"

// ActivityEnvelope is the JSON envelope for WebSocket event broadcasts.
type ActivityEnvelope struct {
	Seq              uint64    `json:"seq_stub"`
	Type             string    `json:"type_stub"`
	Timestamp        time.Time `json:"timestamp_stub"`
	Data             any       `json:"data_stub"`
	DroppedSinceLast int       `json:"dropped_since_last_stub,omitempty"`
}

// ActivityData is the type-specific payload for activity events.
type ActivityData struct {
	Source  string `json:"source_stub"`
	Message string `json:"message_stub"`
	IsError bool   `json:"is_error_stub"`
}
