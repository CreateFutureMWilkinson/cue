package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
)

// EventEnvelope is a WebSocket event envelope. `Data` is deferred unmarshaling
// so consumers can decode the type-specific payload based on `Type`.
type EventEnvelope struct {
	Seq              uint64          `json:"seq"`
	Type             string          `json:"type"`
	Timestamp        time.Time       `json:"timestamp"`
	Data             json.RawMessage `json:"data"`
	DroppedSinceLast int             `json:"dropped_since_last,omitempty"`
}

// ReplayResponse is the response from GET /api/v1/events?since=<seq>.
type ReplayResponse struct {
	Events    []EventEnvelope `json:"events"`
	Truncated bool            `json:"truncated"`
	OldestSeq uint64          `json:"oldest_seq"`
	LatestSeq uint64          `json:"latest_seq"`
}

// ActivityClient wraps the WebSocket event stream and replay endpoint.
type ActivityClient interface {
	// Connect opens a WebSocket connection to the server and begins
	// forwarding events to the channel returned by Events().
	// Subsequent calls while connected return an error.
	Connect(ctx context.Context) error

	// Events returns the channel on which received events are delivered.
	// The channel is never closed by the client — callers observe
	// disconnects by checking `Err()` or by timing out on the channel.
	Events() <-chan EventEnvelope

	// LastSeq returns the highest Seq number the client has received
	// over the WebSocket connection (not counting replay events).
	LastSeq() uint64

	// Replay fetches retained events with Seq > sinceSeq via
	// GET /api/v1/events?since=<sinceSeq>.
	Replay(ctx context.Context, sinceSeq uint64) (*ReplayResponse, error)

	// Close cleanly shuts down the WebSocket connection (if any).
	// Safe to call multiple times.
	Close() error
}

type activityAdapter struct {
	client  *APIClient
	events  chan EventEnvelope
	lastSeq atomic.Uint64
	conn    *websocket.Conn
	mu      sync.Mutex
	closed  atomic.Bool
}

// NewActivityClient returns an ActivityClient backed by the given APIClient.
// The events channel is buffered to prevent blocking the read loop under load.
func NewActivityClient(c *APIClient) ActivityClient {
	return &activityAdapter{
		client: c,
		events: make(chan EventEnvelope, 64),
	}
}

// ErrAlreadyConnected is returned when Connect is called while already connected.
var ErrAlreadyConnected = errors.New("activity client already connected")

func (a *activityAdapter) Connect(ctx context.Context) error {
	return ErrNotImplemented
}

func (a *activityAdapter) Events() <-chan EventEnvelope {
	return a.events
}

func (a *activityAdapter) LastSeq() uint64 {
	return a.lastSeq.Load()
}

func (a *activityAdapter) Replay(ctx context.Context, sinceSeq uint64) (*ReplayResponse, error) {
	return nil, ErrNotImplemented
}

func (a *activityAdapter) Close() error {
	return nil
}

// Ensure imports are referenced (so Go doesn't complain pre-GREEN).
var _ = http.NewRequestWithContext
