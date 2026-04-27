package server

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrUnknownSubscriber is returned when Unsubscribe is called with an
// ID that does not correspond to an active subscriber.
var ErrUnknownSubscriber = errors.New("hub: unknown subscriber")

// subscriberBufferSize bounds each subscriber's in-flight broadcast
// queue. Slow consumers drop messages rather than block the hub.
const subscriberBufferSize = 16

// ringCapacity is the fixed size of the internal ring buffer that
// retains recent ActivityEnvelopes for history replay.
const ringCapacity = 500

// Subscriber represents a connected WebSocket client that receives
// broadcast events.
type Subscriber struct {
	ID     string
	Events chan []byte
}

// Hub is a central event broadcaster. It manages subscriber connections
// and fans out events to all connected clients.
type Hub struct {
	mu          sync.RWMutex
	subscribers map[string]*Subscriber
	seq         uint64             // monotonic sequence counter for ActivityEnvelopes
	ring        []ActivityEnvelope // circular buffer for history replay
	ringPos     int                // current write position in ring buffer
	ringUsed    int                // number of occupied slots in ring buffer
}

// NewHub creates a Hub ready to accept subscribers.
func NewHub() *Hub {
	return &Hub{
		subscribers: make(map[string]*Subscriber),
		ring:        make([]ActivityEnvelope, ringCapacity),
	}
}

// Run blocks until ctx is cancelled. It exists so callers can manage
// the hub lifecycle alongside other server goroutines; broadcasts are
// dispatched synchronously from Broadcast itself.
func (h *Hub) Run(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

// Subscribe registers a new subscriber and returns it. The caller
// should read from Subscriber.Events to receive broadcasts.
func (h *Hub) Subscribe(id string) (*Subscriber, error) {
	sub := &Subscriber{
		ID:     id,
		Events: make(chan []byte, subscriberBufferSize),
	}
	h.mu.Lock()
	h.subscribers[id] = sub
	h.mu.Unlock()
	return sub, nil
}

// Unsubscribe removes a subscriber by ID. The subscriber's Events
// channel is left open — it simply stops receiving broadcasts. Callers
// that want to detect disconnection should signal it through their own
// connection lifecycle, not by reading a closed channel.
func (h *Hub) Unsubscribe(id string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.subscribers[id]; !ok {
		return ErrUnknownSubscriber
	}
	delete(h.subscribers, id)
	return nil
}

// Broadcast sends a message to all connected subscribers. Subscribers
// whose buffers are full silently drop the message rather than block
// the hub.
func (h *Hub) Broadcast(data []byte) error {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, sub := range h.subscribers {
		select {
		case sub.Events <- data:
		default:
		}
	}
	return nil
}

// Publish creates an ActivityEnvelope for the given data, assigns a
// monotonically increasing sequence number, and stores it in the ring
// buffer for history replay. Does not broadcast to current subscribers.
func (h *Hub) Publish(data ActivityData) ActivityEnvelope {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.seq++
	env := ActivityEnvelope{
		Seq:       h.seq,
		Type:      "activity",
		Timestamp: time.Now().UTC(),
		Data:      data,
	}

	h.ring[h.ringPos] = env
	h.ringPos = (h.ringPos + 1) % ringCapacity
	if h.ringUsed < ringCapacity {
		h.ringUsed++
	}

	return env
}

// SubscriberCount returns the number of active subscribers.
func (h *Hub) SubscriberCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subscribers)
}
