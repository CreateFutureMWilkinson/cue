package server

import (
	"context"
	"errors"
)

// ErrHubNotImplemented is returned by hub stub methods.
var ErrHubNotImplemented = errors.New("hub: not implemented")

// Subscriber represents a connected WebSocket client that receives
// broadcast events.
type Subscriber struct {
	ID     string
	Events chan []byte
}

// Hub is a central event broadcaster. It manages subscriber connections
// and fans out events to all connected clients.
type Hub struct {
	subscribers map[string]*Subscriber
}

// NewHub creates a Hub ready to accept subscribers.
func NewHub() *Hub {
	return &Hub{
		subscribers: make(map[string]*Subscriber),
	}
}

// Run starts the hub's event loop. It blocks until ctx is cancelled.
func (h *Hub) Run(ctx context.Context) error {
	return ErrHubNotImplemented
}

// Subscribe registers a new subscriber and returns it. The caller
// should read from Subscriber.Events to receive broadcasts.
func (h *Hub) Subscribe(id string) (*Subscriber, error) {
	return nil, ErrHubNotImplemented
}

// Unsubscribe removes a subscriber by ID.
func (h *Hub) Unsubscribe(id string) error {
	return ErrHubNotImplemented
}

// Broadcast sends a message to all connected subscribers.
func (h *Hub) Broadcast(data []byte) error {
	return ErrHubNotImplemented
}

// SubscriberCount returns the number of active subscribers.
func (h *Hub) SubscriberCount() int {
	return 0
}
