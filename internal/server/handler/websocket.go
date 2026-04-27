package handler

import (
	"context"
	"fmt"
	"net/http"
	"sync"
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

// Manager owns a WebSocket handler and a registry of active connections.
// It allows the server to close all live WebSocket connections on shutdown,
// which is necessary because net/http.Shutdown does not close hijacked
// (upgraded) connections.
type Manager struct {
	mu       sync.Mutex
	conns    map[*websocket.Conn]struct{}
	active   atomic.Int32
	hub      Publisher
	interval time.Duration
	timeout  time.Duration
}

// NewManager creates a Manager with production-default heartbeat timings.
func NewManager(hub Publisher) *Manager {
	return NewManagerWithHeartbeat(hub, defaultHeartbeatInterval, defaultHeartbeatTimeout)
}

// NewManagerWithHeartbeat creates a Manager with configurable heartbeat timings.
func NewManagerWithHeartbeat(hub Publisher, interval, timeout time.Duration) *Manager {
	return &Manager{
		conns:    make(map[*websocket.Conn]struct{}),
		hub:      hub,
		interval: interval,
		timeout:  timeout,
	}
}

// Handler returns an http.Handler that upgrades HTTP connections to WebSocket,
// subscribes to the event hub, and forwards broadcast events to the connected
// client as JSON messages. Connections are tracked in the Manager's registry
// so they can be closed on server shutdown via CloseAll.
func (m *Manager) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !tryReserveSlot(&m.active) {
			writeTooManyConnections(w)
			return
		}
		defer m.active.Add(-1)

		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return // Accept already wrote the HTTP error response.
		}
		defer conn.CloseNow() //nolint:errcheck

		// Register connection for shutdown tracking.
		m.mu.Lock()
		m.conns[conn] = struct{}{}
		m.mu.Unlock()
		defer func() {
			m.mu.Lock()
			delete(m.conns, conn)
			m.mu.Unlock()
		}()

		// Generate unique subscription ID using nanosecond timestamp.
		id := fmt.Sprintf("ws-%d", time.Now().UnixNano())
		sub, err := m.hub.Subscribe(id)
		if err != nil {
			conn.Close(websocket.StatusInternalError, "subscribe failed")
			return
		}
		defer m.hub.Unsubscribe(id)

		ctx, cancel := context.WithCancel(conn.CloseRead(r.Context()))
		defer cancel()

		go runHeartbeat(ctx, conn, m.interval, m.timeout, cancel)

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

// CloseAll closes all tracked WebSocket connections with a GoingAway status.
// This is called by Server.Shutdown to ensure hijacked connections are
// properly terminated before the HTTP server shuts down.
func (m *Manager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for c := range m.conns {
		_ = c.Close(websocket.StatusGoingAway, "server shutting down")
	}
	m.conns = make(map[*websocket.Conn]struct{})
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
//
// Note: connections created via this standalone function are NOT tracked
// for server shutdown. Use NewManagerWithHeartbeat when shutdown-aware
// connection management is needed.
func WebSocketHandlerWithHeartbeat(hub Publisher, interval, timeout time.Duration) http.Handler {
	return NewManagerWithHeartbeat(hub, interval, timeout).Handler()
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
//
// Note: connections created via this standalone function are NOT tracked
// for server shutdown. Use NewManager when shutdown-aware connection
// management is needed.
func WebSocketHandler(hub Publisher) http.Handler {
	return WebSocketHandlerWithHeartbeat(hub, defaultHeartbeatInterval, defaultHeartbeatTimeout)
}
