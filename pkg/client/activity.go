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
	client         *APIClient
	events         chan EventEnvelope
	lastSeq        atomic.Uint64
	conn           *websocket.Conn
	mu             sync.Mutex
	closed         atomic.Bool
	backoffInitial time.Duration
	backoffMax     time.Duration
	connectCtx     context.Context
	connectCancel  context.CancelFunc
}

// ActivityOption configures an ActivityClient.
type ActivityOption func(*activityAdapter)

// WithBackoff sets the exponential-backoff parameters used during
// reconnection. Intervals are clamped: initial must be > 0; max must
// be >= initial. Default: 1s initial, 30s max.
func WithBackoff(initial, max time.Duration) ActivityOption {
	return func(a *activityAdapter) {
		a.backoffInitial = initial
		a.backoffMax = max
	}
}

// NewActivityClient returns an ActivityClient backed by the given APIClient.
// The events channel is buffered to prevent blocking the read loop under load.
// Optional ActivityOption arguments customize reconnection behavior.
func NewActivityClient(c *APIClient, opts ...ActivityOption) ActivityClient {
	a := &activityAdapter{
		client:         c,
		events:         make(chan EventEnvelope, 64),
		backoffInitial: 1 * time.Second,
		backoffMax:     30 * time.Second,
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// ErrAlreadyConnected is returned when Connect is called while already connected.
var ErrAlreadyConnected = errors.New("activity client already connected")

// ErrClosed is returned when Connect is called on a client that has been closed.
var ErrClosed = errors.New("activity client is closed")

// Connect opens a WebSocket to /api/v1/websocket/events, forwarding the bearer
// token via query string. A background manager goroutine reads envelopes,
// forwards them to the Events() channel, and transparently reconnects with
// exponential backoff when the connection drops. Returns ErrAlreadyConnected
// if a connection is already active, or ErrClosed if Close has been called.
// Only the first dial error is returned; subsequent reconnect failures happen
// in the background.
func (a *activityAdapter) Connect(ctx context.Context) error {
	a.mu.Lock()

	if a.conn != nil {
		a.mu.Unlock()
		return ErrAlreadyConnected
	}
	if a.closed.Load() {
		a.mu.Unlock()
		return ErrClosed
	}

	wsURL, err := buildWebSocketURL(a.client.baseURL)
	if err != nil {
		a.mu.Unlock()
		return fmt.Errorf("build websocket url: %w", err)
	}

	dialOpts := dialOptionsForToken(a.client.token)
	conn, _, err := websocket.Dial(ctx, wsURL, dialOpts)
	if err != nil {
		a.mu.Unlock()
		return fmt.Errorf("websocket dial: %w", err)
	}

	managerCtx, managerCancel := context.WithCancel(ctx)
	a.conn = conn
	a.connectCtx = managerCtx
	a.connectCancel = managerCancel
	a.mu.Unlock()

	go a.manageConnection(managerCtx, wsURL, conn)
	return nil
}

// manageConnection runs the read loop for each successive WebSocket
// connection, reconnecting with exponential backoff when the read loop
// returns (meaning the connection dropped). Backoff doubles on each failed
// dial (capped at backoffMax) and resets to backoffInitial after a
// successful reconnect. Exits when the context is cancelled or the adapter
// has been Closed.
func (a *activityAdapter) manageConnection(ctx context.Context, wsURL string, initial *websocket.Conn) {
	currentConn := initial
	backoff := a.backoffInitial

	for {
		// Block until the current connection ends (error, close, etc).
		a.readLoop(ctx, currentConn)

		// Either context cancellation or Close will stop the loop.
		if ctx.Err() != nil || a.closed.Load() {
			return
		}

		// Wait for backoff, but bail out early if the context is cancelled.
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}

		if a.closed.Load() {
			return
		}

		newConn, _, err := websocket.Dial(ctx, wsURL, dialOptionsForToken(a.client.token))
		if err != nil {
			// Double the backoff, capped at backoffMax.
			backoff *= 2
			if backoff > a.backoffMax {
				backoff = a.backoffMax
			}
			continue
		}

		// Successful reconnect: reset backoff and swap the active conn.
		backoff = a.backoffInitial
		a.mu.Lock()
		a.conn = newConn
		a.mu.Unlock()
		currentConn = newConn
	}
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

// Close shuts down the WebSocket connection and stops the reconnection
// manager. Safe to call multiple times; subsequent calls are a no-op.
// Uses CloseNow rather than the close handshake to avoid blocking when
// the server is not actively reading (the handshake's readMu acquisition
// would otherwise deadlock against an in-flight readLoop).
func (a *activityAdapter) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.closed.Load() {
		return nil
	}
	a.closed.Store(true)

	if a.connectCancel != nil {
		a.connectCancel()
	}

	if a.conn != nil {
		err := a.conn.CloseNow()
		a.conn = nil
		return err
	}
	return nil
}

// readLoop consumes frames from conn, unmarshals EventEnvelopes, updates
// lastSeq atomically, and forwards envelopes to the events channel. If the
// channel is full, envelopes are dropped (no reconnection logic yet — that
// arrives in Loop 12). Exits silently when the connection returns an error
// or when the provided context is cancelled.
func (a *activityAdapter) readLoop(ctx context.Context, conn *websocket.Conn) {
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}

		var env EventEnvelope
		if err := json.Unmarshal(data, &env); err != nil {
			continue // skip malformed frames
		}

		a.lastSeq.Store(env.Seq)
		select {
		case a.events <- env:
		default:
			// buffer full — drop (no reconnect logic yet)
		}
	}
}

// buildWebSocketURL converts an http(s) base URL into a ws(s) URL for
// the event stream endpoint. Authentication is carried in the
// Authorization header on the upgrade request (see dialOptionsForToken),
// matching the rest of the HTTP API rather than the legacy ?token= query
// param.
func buildWebSocketURL(baseURL string) (string, error) {
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
	u.RawQuery = ""
	return u.String(), nil
}

// dialOptionsForToken returns websocket.DialOptions carrying the Bearer
// token on the upgrade request's Authorization header. Returns nil when
// the token is empty so unauthenticated dials still work in tests.
func dialOptionsForToken(token string) *websocket.DialOptions {
	if token == "" {
		return nil
	}
	return &websocket.DialOptions{
		HTTPHeader: http.Header{"Authorization": []string{"Bearer " + token}},
	}
}
