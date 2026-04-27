package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/coder/websocket"
)

// Subscription represents a connected subscriber that receives broadcast
// events on a byte-slice channel. It mirrors the shape of *server.Subscriber
// without importing the server package, breaking the import cycle.
type Subscription struct {
	ID     string
	Events <-chan []byte
}

// Publisher is the subset of *server.Hub needed by the WebSocket handler.
type Publisher interface {
	Subscribe(id string) (*Subscription, error)
	Unsubscribe(id string) error
}

// WebSocketHandler returns an http.Handler that upgrades the connection
// to WebSocket, subscribes to the hub, and forwards envelopes to the client.
func WebSocketHandler(hub Publisher) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return // Accept already wrote the HTTP error response.
		}

		id := fmt.Sprintf("ws-%d", time.Now().UnixNano())
		sub, err := hub.Subscribe(id)
		if err != nil {
			conn.Close(websocket.StatusInternalError, "subscribe failed")
			return
		}
		defer hub.Unsubscribe(id)

		// CloseRead starts a background reader that detects client-side
		// close frames and cancels the returned context.
		ctx := conn.CloseRead(r.Context())

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
				writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				err := conn.Write(writeCtx, websocket.MessageText, data)
				cancel()
				if err != nil {
					return
				}
			}
		}
	})
}
