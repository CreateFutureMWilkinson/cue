package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
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

// ErrClosed is returned when Connect is called on a client that has been closed.
var ErrClosed = errors.New("activity client is closed")

// Connect opens a WebSocket to /api/v1/websocket/events, forwarding the bearer
// token via query string. A background goroutine reads envelopes and forwards
// them to the Events() channel. Returns ErrAlreadyConnected if a connection is
// already active, or ErrClosed if Close has been called.
func (a *activityAdapter) Connect(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.conn != nil {
		return ErrAlreadyConnected
	}
	if a.closed.Load() {
		return ErrClosed
	}

	wsURL, err := buildWebSocketURL(a.client.baseURL, a.client.token)
	if err != nil {
		return fmt.Errorf("build websocket url: %w", err)
	}

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		return fmt.Errorf("websocket dial: %w", err)
	}

	a.conn = conn
	go a.readLoop(conn)
	return nil
}

// Events returns the buffered channel of received event envelopes.
func (a *activityAdapter) Events() <-chan EventEnvelope {
	return a.events
}

// LastSeq returns the highest Seq observed on the WebSocket connection.
func (a *activityAdapter) LastSeq() uint64 {
	return a.lastSeq.Load()
}

// Replay fetches retained events with Seq > sinceSeq via the shared
// doJSON helper. Errors from the server surface as *APIError.
func (a *activityAdapter) Replay(ctx context.Context, sinceSeq uint64) (*ReplayResponse, error) {
	path := fmt.Sprintf("/api/v1/events?since=%d", sinceSeq)
	var resp ReplayResponse
	if err := a.client.doJSON(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Close shuts down the WebSocket connection cleanly. Safe to call multiple
// times; subsequent calls are a no-op.
func (a *activityAdapter) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.closed.Store(true)
	if a.conn != nil {
		err := a.conn.Close(websocket.StatusNormalClosure, "")
		a.conn = nil
		return err
	}
	return nil
}

// readLoop consumes frames from conn, unmarshals EventEnvelopes, updates
// lastSeq atomically, and forwards envelopes to the events channel. If the
// channel is full, envelopes are dropped (no reconnection logic yet — that
// arrives in Loop 12). Exits silently when the connection returns an error.
func (a *activityAdapter) readLoop(conn *websocket.Conn) {
	ctx := context.Background()
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		var env EventEnvelope
		if err := json.Unmarshal(data, &env); err != nil {
			continue
		}
		a.lastSeq.Store(env.Seq)
		select {
		case a.events <- env:
		default:
			// buffer full — drop (no reconnect logic yet)
		}
	}
}

// buildWebSocketURL converts an http(s) base URL into a ws(s) URL for the
// event stream endpoint, appending the token as a query parameter when set.
func buildWebSocketURL(baseURL, token string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	case "ws", "wss":
		// already websocket
	default:
		return "", fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}
	u.Path = "/api/v1/websocket/events"
	q := u.Query()
	if token != "" {
		q.Set("token", token)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}
