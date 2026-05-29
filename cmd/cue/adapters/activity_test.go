package adapters_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/cmd/cue/adapters"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
	"github.com/CreateFutureMWilkinson/cue/pkg/client"
)

// ActivityAdapterSuite covers DTO decoding, drop-on-slow fan-out, the
// synthetic system event surfaced for EventEnvelope.DroppedSinceLast,
// and Close propagation. A single round-trip test against an
// httptest.NewServer + real *websocket.Conn pins the wire shape.
type ActivityAdapterSuite struct {
	suite.Suite
}

func TestActivityAdapter(t *testing.T) {
	suite.Run(t, new(ActivityAdapterSuite))
}

// fakeActivityClient is a stub client.ActivityClient that lets tests
// drive the adapter's input channel directly without booting a real
// WebSocket. Tests pump envelopes into Events() and assert what comes
// out of the adapter's subscribers.
type fakeActivityClient struct {
	events     chan client.EventEnvelope
	closeCalls atomic.Int32
}

func newFakeActivityClient(buf int) *fakeActivityClient {
	return &fakeActivityClient{events: make(chan client.EventEnvelope, buf)}
}

func (f *fakeActivityClient) Connect(ctx context.Context) error   { return nil }
func (f *fakeActivityClient) Events() <-chan client.EventEnvelope { return f.events }
func (f *fakeActivityClient) LastSeq() uint64                     { return 0 }
func (f *fakeActivityClient) Replay(ctx context.Context, since uint64) (*client.ReplayResponse, error) {
	return &client.ReplayResponse{}, nil
}
func (f *fakeActivityClient) Close() error {
	f.closeCalls.Add(1)
	close(f.events)
	return nil
}

// rawActivity returns a json.RawMessage matching the server's
// ActivityData shape so the adapter can decode it.
func (s *ActivityAdapterSuite) rawActivity(source, message string, isError bool) json.RawMessage {
	b, err := json.Marshal(map[string]any{
		"source":   source,
		"message":  message,
		"is_error": isError,
	})
	s.Require().NoError(err)
	return b
}

func (s *ActivityAdapterSuite) drain(ch <-chan presenter.ActivityEvent) presenter.ActivityEvent {
	select {
	case ev, ok := <-ch:
		s.Require().True(ok, "subscriber channel closed unexpectedly")
		return ev
	case <-time.After(time.Second):
		s.FailNow("timed out waiting for activity event")
		return presenter.ActivityEvent{}
	}
}

// AC: an activity envelope is decoded and delivered to every
// subscriber identically.
func (s *ActivityAdapterSuite) TestDispatchFansOutToBothConsumers() {
	fc := newFakeActivityClient(4)
	a := adapters.NewActivityAdapter(fc)
	srcA := a.Subscribe()
	srcB := a.Subscribe()

	a.Start(context.Background())
	defer a.Close() //nolint:errcheck

	fc.events <- client.EventEnvelope{
		Seq:  1,
		Type: "activity",
		Data: s.rawActivity("slack", "hi", false),
	}

	got1 := s.drain(srcA.Events())
	got2 := s.drain(srcB.Events())
	want := presenter.ActivityEvent{Source: "slack", Message: "hi"}
	s.Equal(want, got1)
	s.Equal(want, got2)
}

// AC: a slow consumer drops events, but the read loop and other
// subscribers continue to receive in real time.
func (s *ActivityAdapterSuite) TestDropOnSlowConsumer() {
	fc := newFakeActivityClient(64)
	a := adapters.NewActivityAdapter(fc)
	slow := a.Subscribe() // never read
	fast := a.Subscribe()

	a.Start(context.Background())
	defer a.Close() //nolint:errcheck

	// Push enough events to overflow the slow subscriber's buffer.
	const n = 64
	for i := 0; i < n; i++ {
		fc.events <- client.EventEnvelope{
			Seq:  uint64(i + 1),
			Type: "activity",
			Data: s.rawActivity("slack", "msg", false),
		}
	}

	// The fast subscriber reads everything it can; we just need to see
	// that more than one event arrived (i.e. the read loop did not
	// block on the slow channel).
	deadline := time.After(2 * time.Second)
	received := 0
loop:
	for {
		select {
		case <-fast.Events():
			received++
			if received >= 5 {
				break loop
			}
		case <-deadline:
			break loop
		}
	}
	s.GreaterOrEqual(received, 5,
		"fast consumer should have received events even while the slow consumer never reads")

	// The slow subscriber's buffer (32) plus already-delivered events
	// should not have stalled the loop. Drain it and confirm we did
	// not block forever.
	slowReceived := 0
drainSlow:
	for {
		select {
		case <-slow.Events():
			slowReceived++
		case <-time.After(50 * time.Millisecond):
			break drainSlow
		}
	}
	s.LessOrEqual(slowReceived, n,
		"slow consumer can drop events but cannot magically receive more than were sent")
}

// AC: DroppedSinceLast > 0 surfaces as a synthetic system event
// alongside the original envelope's payload.
func (s *ActivityAdapterSuite) TestDroppedSinceLastEmitsSyntheticSystemEvent() {
	fc := newFakeActivityClient(4)
	a := adapters.NewActivityAdapter(fc)
	src := a.Subscribe()

	a.Start(context.Background())
	defer a.Close() //nolint:errcheck

	fc.events <- client.EventEnvelope{
		Seq:              7,
		Type:             "activity",
		DroppedSinceLast: 3,
		Data:             s.rawActivity("slack", "after gap", false),
	}

	first := s.drain(src.Events())
	s.Equal("system", first.Source)
	s.Contains(first.Message, "3")

	second := s.drain(src.Events())
	s.Equal(presenter.ActivityEvent{Source: "slack", Message: "after gap"}, second)
}

// AC: non-activity envelope types are ignored (no panic, no synthetic
// event, no fan-out).
func (s *ActivityAdapterSuite) TestNonActivityTypesAreIgnored() {
	fc := newFakeActivityClient(4)
	a := adapters.NewActivityAdapter(fc)
	src := a.Subscribe()

	a.Start(context.Background())
	defer a.Close() //nolint:errcheck

	fc.events <- client.EventEnvelope{
		Seq:  1,
		Type: "timer_tick",
		Data: json.RawMessage(`{"running":true}`),
	}

	select {
	case ev := <-src.Events():
		s.Failf("unexpected fan-out", "received %#v", ev)
	case <-time.After(50 * time.Millisecond):
		// expected
	}
}

// AC: Close calls the underlying client's Close exactly once and
// causes subscriber channels to close.
func (s *ActivityAdapterSuite) TestCloseClosesSubscribers() {
	fc := newFakeActivityClient(4)
	a := adapters.NewActivityAdapter(fc)
	src := a.Subscribe()

	a.Start(context.Background())

	s.Require().NoError(a.Close())
	s.Require().NoError(a.Close()) // idempotent

	select {
	case _, ok := <-src.Events():
		s.False(ok, "subscriber channel must close after adapter Close()")
	case <-time.After(time.Second):
		s.FailNow("subscriber channel was not closed after adapter shutdown")
	}

	s.Equal(int32(1), fc.closeCalls.Load(),
		"underlying ActivityClient.Close must be invoked exactly once")
}

// AC: integration round-trip — adapter consumes envelopes from a real
// httptest WebSocket through the SDK, no raw HTTP in the test.
func (s *ActivityAdapterSuite) TestIntegrationRoundTrip() {
	var connOnce sync.Once
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("/api/v1/websocket/events", r.URL.Path)
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "") //nolint:errcheck

		connOnce.Do(func() {
			env := client.EventEnvelope{
				Seq:       1,
				Type:      "activity",
				Timestamp: time.Date(2026, 4, 27, 9, 0, 0, 0, time.UTC),
				Data:      s.rawActivity("email", "deadline soon", true),
			}
			payload, mErr := json.Marshal(env)
			s.Require().NoError(mErr)
			s.Require().NoError(conn.Write(r.Context(), websocket.MessageText, payload))
			// Hold the connection open briefly so the SDK reads the frame.
			time.Sleep(150 * time.Millisecond)
		})
	}))
	defer ts.Close()

	api := client.New(ts.URL)
	api.SetToken("test-token")
	sdk := client.NewActivityClient(api)

	a := adapters.NewActivityAdapter(sdk)
	src := a.Subscribe()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	s.Require().NoError(sdk.Connect(ctx))
	a.Start(ctx)
	defer a.Close() //nolint:errcheck

	got := s.drain(src.Events())
	s.Equal(presenter.ActivityEvent{Source: "email", Message: "deadline soon", IsError: true}, got)
}
