package server

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"
)

// ErrUnknownSubscriber is returned when Unsubscribe is called with an
// ID that does not correspond to an active subscriber.
var ErrUnknownSubscriber = errors.New("hub: unknown subscriber")

// subscriberBufferSize bounds each subscriber's in-flight broadcast
// queue. Slow consumers drop messages rather than block the hub.
const subscriberBufferSize = 64

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
	subs := make([]*Subscriber, 0, len(h.subscribers))
	for _, sub := range h.subscribers {
		subs = append(subs, sub)
	}
	h.mu.RUnlock()

	h.fanout(subs, data)
	return nil
}

// fanout sends data to all subscribers in the provided slice. Slow
// consumers drop the message rather than block the hub. This helper
// is used by both Broadcast and Publish to avoid code duplication.
func (h *Hub) fanout(subs []*Subscriber, data []byte) {
	for _, sub := range subs {
		select {
		case sub.Events <- data:
		default:
		}
	}
}

// Publish creates an ActivityEnvelope for the given data, assigns a
// monotonically increasing sequence number, stores it in the ring
// buffer for history replay, and broadcasts the JSON-serialized
// envelope to all current subscribers.
func (h *Hub) Publish(data ActivityData) ActivityEnvelope {
	h.mu.Lock()

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

	// Serialize envelope for broadcast. On marshal failure (defensive),
	// keep the ring entry but skip the fan-out to avoid corrupted data.
	raw, err := json.Marshal(env)
	if err != nil {
		h.mu.Unlock()
		return env
	}

	// Snapshot subscriber list under the write lock, then release the
	// lock before fan-out to avoid holding it during channel operations.
	subs := make([]*Subscriber, 0, len(h.subscribers))
	for _, sub := range h.subscribers {
		subs = append(subs, sub)
	}
	h.mu.Unlock()

	h.fanout(subs, raw)

	return env
}

// oldestSeq returns the sequence number of the oldest retained envelope.
// Must be called with at least a read lock held and ringUsed > 0.
func (h *Hub) oldestSeq() uint64 {
	return h.seq - uint64(h.ringUsed) + 1
}

// HistoryResponse is the return value of Hub.History, containing a slice
// of retained activity envelopes plus metadata for client-side replay.
type HistoryResponse struct {
	Events    []ActivityEnvelope `json:"events"`
	Truncated bool               `json:"truncated"`
	OldestSeq uint64             `json:"oldest_seq"`
	LatestSeq uint64             `json:"latest_seq"`
}

// History returns all retained envelopes with seq > sinceSeq. When events
// have been evicted from the ring buffer (sinceSeq < oldest retained seq),
// the response starts at the oldest retained event and Truncated is true.
func (h *Hub) History(sinceSeq uint64) HistoryResponse {
	h.mu.RLock()

	if h.ringUsed == 0 {
		h.mu.RUnlock()
		return HistoryResponse{}
	}

	latestSeq := h.seq
	oldestSeq := h.oldestSeq()

	if sinceSeq >= latestSeq {
		h.mu.RUnlock()
		return HistoryResponse{
			OldestSeq: oldestSeq,
			LatestSeq: latestSeq,
		}
	}

	// Truncation occurs when: (1) client requests sequence that's been evicted,
	// or (2) client requests from beginning (sinceSeq=0) but ring is full
	truncated := (sinceSeq > 0 && sinceSeq < oldestSeq) ||
		(sinceSeq == 0 && h.ringUsed == ringCapacity)

	// Start from the next sequence after sinceSeq, but clamp to oldest available
	startSeq := max(sinceSeq+1, oldestSeq)

	count := int(latestSeq - startSeq + 1)
	events := make([]ActivityEnvelope, count)

	// Calculate ring buffer positions for traversal:
	// - oldestIdx: ring position of the oldest retained envelope
	// - startIdx: ring position of the envelope with seq == startSeq
	oldestIdx := (h.ringPos - h.ringUsed + ringCapacity) % ringCapacity
	startIdx := (oldestIdx + int(startSeq-oldestSeq)) % ringCapacity

	// Traverse the ring buffer in chronological order, wrapping as needed
	for i := 0; i < count; i++ {
		events[i] = h.ring[(startIdx+i)%ringCapacity]
	}

	h.mu.RUnlock()

	return HistoryResponse{
		Events:    events,
		Truncated: truncated,
		OldestSeq: oldestSeq,
		LatestSeq: latestSeq,
	}
}

// SubscriberCount returns the number of active subscribers.
func (h *Hub) SubscriberCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subscribers)
}
