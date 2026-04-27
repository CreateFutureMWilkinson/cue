package handler

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
)

const (
	// writeTimeout defines how long we wait for a WebSocket message to be written
	// before timing out. This prevents slow clients from blocking the handler.
	writeTimeout = 5 * time.Second

	// MaxConnections is the hard-coded maximum number of concurrent WebSocket
	// connections. The (MaxConnections+1)th upgrade attempt receives HTTP 503.
	MaxConnections = 16

	// Production defaults for heartbeat intervals when using WebSocketHandler.
	defaultHeartbeatInterval = 30 * time.Second // How often to send ping messages
	defaultHeartbeatTimeout  = 10 * time.Second // How long to wait for pong response
)

// Subscription represents a connected subscriber that receives broadcast
// events on a byte-slice channel. It mirrors the shape of *server.Subscriber
// without importing the server package, breaking the import cycle.
//
// ID is a unique identifier for this subscription, and Events is the
// receive-only channel that delivers serialized event data to the subscriber.
type Subscription struct {
	ID     string
	Events <-chan []byte
}

// Publisher is the subset of *server.Hub needed by the WebSocket handler.
// It provides subscription management without requiring the full server
// package interface, avoiding import cycles.
type Publisher interface {
	Subscribe(id string) (*Subscription, error)
	Unsubscribe(id string) error
}

// tryReserveSlot attempts to atomically reserve a connection slot using
// compare-and-swap to prevent TOCTOU races. Returns true if successful.
func tryReserveSlot(active *atomic.Int32) bool {
	for {
		cur := active.Load()
		if cur >= int32(MaxConnections) {
			return false
		}
		if active.CompareAndSwap(cur, cur+1) {
			return true
		}
		// CAS failed due to concurrent update, retry
	}
}

// writeTooManyConnections writes a 503 Service Unavailable response with
// appropriate headers and JSON error body when connection capacity is exceeded.
func writeTooManyConnections(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", "5") // Suggest client retry after 5 seconds
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte(`{"error":"too many connections"}`))
}

// runHeartbeat sends periodic ping messages to detect half-open connections.
// It runs in a separate goroutine and closes the connection if no pong is
// received within the configured timeout. This helps detect clients that
// have silently disconnected without sending a proper close frame.
func runHeartbeat(ctx context.Context, conn *websocket.Conn, interval, timeout time.Duration, cancel context.CancelFunc) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pingCtx, pingCancel := context.WithTimeout(ctx, timeout)
			pingErr := conn.Ping(pingCtx)
			pingCancel()
			if pingErr != nil {
				// Pong not received within timeout or connection already dead.
				// Use CloseNow instead of Close because CloseRead (called in the
				// main handler) already holds the read lock. Using Close would block
				// on the close handshake trying to acquire that read lock (up to 5s).
				conn.CloseNow()
				cancel()
				return
			}
		}
	}
}

// WebSocketHandlerWithHeartbeat returns an http.Handler identical to
// WebSocketHandler but with configurable ping/pong heartbeat intervals.
// The server sends a WebSocket ping every `interval`; if no pong is
// received within `timeout` the connection is closed and the subscriber
// is removed from the hub.
//
// For typical production use, prefer WebSocketHandler which uses sensible
// defaults (30s ping interval, 10s pong timeout). This function is useful
// for testing or specialized deployments requiring different timings.
func WebSocketHandlerWithHeartbeat(hub Publisher, interval, timeout time.Duration) http.Handler {
	var active atomic.Int32
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Enforce per-handler connection cap using atomic operations
		if !tryReserveSlot(&active) {
			writeTooManyConnections(w)
			return
		}
		defer active.Add(-1)

		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return // Accept already wrote the HTTP error response.
		}
		defer conn.CloseNow() //nolint:errcheck

		// Generate unique subscription ID using nanosecond timestamp.
		// This ensures uniqueness across concurrent connections without
		// requiring additional synchronization or state tracking.
		id := fmt.Sprintf("ws-%d", time.Now().UnixNano())
		sub, err := hub.Subscribe(id)
		if err != nil {
			conn.Close(websocket.StatusInternalError, "subscribe failed")
			return
		}
		defer hub.Unsubscribe(id)

		// CloseRead starts a background goroutine that monitors for client-side
		// close frames and ping/pong messages. It returns a context that gets
		// cancelled when the client disconnects, allowing us to detect connection
		// loss even when we're not actively writing.
		ctx, cancel := context.WithCancel(conn.CloseRead(r.Context()))
		defer cancel()

		// Start heartbeat monitoring in background goroutine
		go runHeartbeat(ctx, conn, interval, timeout, cancel)

		for {
			select {
			case <-ctx.Done():
				conn.Close(websocket.StatusNormalClosure, "")
				return
			case data, ok := <-sub.Events:
				if !ok {
					conn.Close(websocket.StatusNormalClosure, "")
					return
				}
				// Apply write timeout to prevent slow clients from blocking
				// the event stream for other subscribers. If the write takes
				// longer than writeTimeout, we close the connection.
				writeCtx, writeCancel := context.WithTimeout(ctx, writeTimeout)
				writeErr := conn.Write(writeCtx, websocket.MessageText, data)
				writeCancel()
				if writeErr != nil {
					return
				}
			}
		}
	})
}

// WebSocketHandler returns an http.Handler that upgrades HTTP connections
// to WebSocket, subscribes to the event hub, and forwards broadcast events
// to the connected client as JSON messages.
//
// The handler automatically manages the subscription lifecycle, cleaning up
// when the client disconnects or an error occurs. Write timeouts prevent
// slow clients from blocking the event stream. Heartbeat ping/pong messages
// detect half-open connections.
//
// This function uses production defaults for heartbeat timings (30s ping
// interval, 10s pong timeout). For custom timings, use
// WebSocketHandlerWithHeartbeat directly.
func WebSocketHandler(hub Publisher) http.Handler {
	return WebSocketHandlerWithHeartbeat(hub, defaultHeartbeatInterval, defaultHeartbeatTimeout)
}
