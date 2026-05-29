// Package adapters wraps pkg/client SDK clients in shims that satisfy
// the presenter and repository interfaces consumed by the Fyne client.
//
// Adapters are thin: they translate DTOs and route errors. They never
// perform their own transport. Reconnection, request retries, and HTTP
// concerns belong in pkg/client.
package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
	"github.com/CreateFutureMWilkinson/cue/pkg/client"
)

// activityEventBufferSize is the buffer applied to each subscriber
// channel. Small but not zero so a momentary stall on one consumer
// does not drop in lockstep with the other consumer.
const activityEventBufferSize = 32

// envelopeTypeActivity is the EventEnvelope.Type value the server
// uses for activity log events. Other envelope types (alerts, timer
// ticks, etc.) are ignored by the activity adapter.
const envelopeTypeActivity = "activity"

// systemEventSource labels synthetic events the adapter emits when the
// server reports it dropped envelopes for this client.
const systemEventSource = "system"

// ActivityAdapter consumes the SDK's WebSocket EventEnvelope stream,
// decodes activity-typed envelopes into presenter.ActivityEvent, and
// fans them out to one or more consumers with non-blocking sends.
//
// Reconnection, backoff, and the underlying read loop are owned by
// the wrapped client.ActivityClient — this adapter never adds its own
// retry logic.
//
// EventEnvelope.DroppedSinceLast > 0 surfaces as a synthetic
// presenter.ActivityEvent{Source: "system", Message: "N events
// dropped"} so the activity log records server-side drops.
type ActivityAdapter struct {
	src client.ActivityClient

	mu      sync.Mutex
	sinks   []chan presenter.ActivityEvent
	started bool
	stopped bool
}

// NewActivityAdapter constructs an adapter wrapping the given SDK
// activity client. Subscribers must be added (via Subscribe) before
// Start is called.
func NewActivityAdapter(src client.ActivityClient) *ActivityAdapter {
	return &ActivityAdapter{src: src}
}

// Subscribe returns a fresh ActivitySource. Each subscriber receives
// every event the adapter decodes; slow subscribers cause events to
// be dropped for that subscriber only — the read loop and other
// subscribers are never blocked.
//
// Subscribe must be called before Start. Subscribing afterwards
// returns a source that will receive only events from that point on.
func (a *ActivityAdapter) Subscribe() presenter.ActivitySource {
	a.mu.Lock()
	defer a.mu.Unlock()
	ch := make(chan presenter.ActivityEvent, activityEventBufferSize)
	a.sinks = append(a.sinks, ch)
	return chanActivitySource{ch: ch}
}

// Start launches the adapter's read goroutine. It must be called
// exactly once. The goroutine exits cleanly when ctx is cancelled or
// when the SDK closes its events channel; in either case all
// subscriber channels are closed.
func (a *ActivityAdapter) Start(ctx context.Context) {
	a.mu.Lock()
	if a.started {
		a.mu.Unlock()
		return
	}
	a.started = true
	a.mu.Unlock()

	go a.run(ctx)
}

// Close stops the underlying SDK activity client. The read goroutine
// observes the closed Events() channel (or the cancelled context) and
// exits, closing every subscriber channel on its way out. Safe to call
// multiple times.
func (a *ActivityAdapter) Close() error {
	a.mu.Lock()
	if a.stopped {
		a.mu.Unlock()
		return nil
	}
	a.stopped = true
	a.mu.Unlock()
	return a.src.Close()
}

func (a *ActivityAdapter) run(ctx context.Context) {
	events := a.src.Events()
	defer a.closeSinks()

	for {
		select {
		case <-ctx.Done():
			return
		case env, ok := <-events:
			if !ok {
				return
			}
			a.dispatch(env)
		}
	}
}

func (a *ActivityAdapter) dispatch(env client.EventEnvelope) {
	if env.DroppedSinceLast > 0 {
		a.fanOut(presenter.ActivityEvent{
			Source:  systemEventSource,
			Message: fmt.Sprintf("%d events dropped", env.DroppedSinceLast),
		})
	}
	if ev, ok := decodeActivityEvent(env); ok {
		a.fanOut(ev)
	}
}

func (a *ActivityAdapter) fanOut(ev presenter.ActivityEvent) {
	a.mu.Lock()
	sinks := append([]chan presenter.ActivityEvent(nil), a.sinks...)
	a.mu.Unlock()
	for _, s := range sinks {
		select {
		case s <- ev:
		default:
			// Drop on slow consumer.
		}
	}
}

func (a *ActivityAdapter) closeSinks() {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, s := range a.sinks {
		close(s)
	}
	a.sinks = nil
}

// activityPayload mirrors the server's ActivityData JSON shape.
type activityPayload struct {
	Source  string `json:"source"`
	Message string `json:"message"`
	IsError bool   `json:"is_error"`
}

func decodeActivityEvent(env client.EventEnvelope) (presenter.ActivityEvent, bool) {
	if env.Type != envelopeTypeActivity {
		return presenter.ActivityEvent{}, false
	}
	var d activityPayload
	if err := json.Unmarshal(env.Data, &d); err != nil {
		return presenter.ActivityEvent{}, false
	}
	return presenter.ActivityEvent{
		Source:  d.Source,
		Message: d.Message,
		IsError: d.IsError,
	}, true
}

// chanActivitySource adapts a receive-only channel into a
// presenter.ActivitySource.
type chanActivitySource struct {
	ch <-chan presenter.ActivityEvent
}

func (s chanActivitySource) Events() <-chan presenter.ActivityEvent { return s.ch }
