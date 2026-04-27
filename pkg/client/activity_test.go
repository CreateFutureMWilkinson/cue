package client_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/pkg/client"
)

// ActivitySuite covers the ActivityClient adapter, which wraps the
// WebSocket event stream at /api/v1/websocket/events and the HTTP
// replay endpoint at /api/v1/events?since=<seq>.
type ActivitySuite struct {
	suite.Suite
}

func TestActivity(t *testing.T) {
	suite.Run(t, new(ActivitySuite))
}

// writeEnvelope encodes env as JSON and writes it as a WebSocket text
// message on conn. Fails the test on any error.
func (s *ActivitySuite) writeEnvelope(ctx context.Context, conn *websocket.Conn, env client.EventEnvelope) {
	data, err := json.Marshal(env)
	s.Require().NoError(err)
	s.Require().NoError(conn.Write(ctx, websocket.MessageText, data))
}

// mustRawMessage marshals v to a json.RawMessage suitable for populating
// the Data field of a fake EventEnvelope in test fixtures.
func (s *ActivitySuite) mustRawMessage(v any) json.RawMessage {
	raw, err := json.Marshal(v)
	s.Require().NoError(err)
	return raw
}

// receiveOrFail reads one envelope from ch with a 1s timeout; fails the
// test if nothing arrives within the window.
func (s *ActivitySuite) receiveOrFail(ch <-chan client.EventEnvelope) client.EventEnvelope {
	select {
	case env := <-ch:
		return env
	case <-time.After(1 * time.Second):
		s.FailNow("timed out waiting for event on Events() channel")
		return client.EventEnvelope{}
	}
}

// TestConnectEstablishesWebSocket verifies that Connect upgrades to
// a WebSocket, forwards the token as a query parameter, forwards one
// pushed envelope to the Events channel, and updates LastSeq.
func (s *ActivitySuite) TestConnectEstablishesWebSocket() {
	var sawToken atomic.Value
	sawToken.Store("")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("/api/v1/websocket/events", r.URL.Path)
		sawToken.Store(r.URL.Query().Get("token"))

		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "") //nolint:errcheck

		env := client.EventEnvelope{
			Seq:       1,
			Type:      "activity",
			Timestamp: time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC),
			Data:      s.mustRawMessage(map[string]string{"source": "slack", "message": "hi"}),
		}
		s.writeEnvelope(r.Context(), conn, env)
		// Keep connection open briefly so the client has time to read.
		time.Sleep(200 * time.Millisecond)
	}))
	defer ts.Close()

	c := client.New(ts.URL)
	c.SetToken("test-token")

	activity := client.NewActivityClient(c)
	defer activity.Close() //nolint:errcheck

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	s.Require().NoError(activity.Connect(ctx))

	env := s.receiveOrFail(activity.Events())
	s.Equal(uint64(1), env.Seq)
	s.Equal("activity", env.Type)
	s.Equal("test-token", sawToken.Load())

	s.Equal(uint64(1), activity.LastSeq())
	s.Require().NoError(activity.Close())
}

// TestConnectReceivesMultipleEventsInOrder verifies that a burst of
// envelopes is delivered in sequence order and LastSeq() reflects the
// highest Seq seen.
func (s *ActivitySuite) TestConnectReceivesMultipleEventsInOrder() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "") //nolint:errcheck

		for i := uint64(1); i <= 3; i++ {
			s.writeEnvelope(r.Context(), conn, client.EventEnvelope{
				Seq:       i,
				Type:      "activity",
				Timestamp: time.Date(2026, 4, 24, 10, 0, int(i), 0, time.UTC),
				Data:      s.mustRawMessage(map[string]any{"n": i}),
			})
		}
		time.Sleep(200 * time.Millisecond)
	}))
	defer ts.Close()

	c := client.New(ts.URL)
	c.SetToken("test-token")

	activity := client.NewActivityClient(c)
	defer activity.Close() //nolint:errcheck

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	s.Require().NoError(activity.Connect(ctx))

	for want := uint64(1); want <= 3; want++ {
		env := s.receiveOrFail(activity.Events())
		s.Equal(want, env.Seq, "events should arrive in sequence order")
	}

	s.Equal(uint64(3), activity.LastSeq())
	s.Require().NoError(activity.Close())
}

// TestConnectTwiceReturnsError verifies that calling Connect while
// already connected returns ErrAlreadyConnected.
func (s *ActivitySuite) TestConnectTwiceReturnsError() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "") //nolint:errcheck
		// Keep connection alive long enough for the second Connect attempt.
		time.Sleep(500 * time.Millisecond)
	}))
	defer ts.Close()

	c := client.New(ts.URL)
	c.SetToken("test-token")

	activity := client.NewActivityClient(c)
	defer activity.Close() //nolint:errcheck

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	s.Require().NoError(activity.Connect(ctx))

	err := activity.Connect(ctx)
	s.Require().Error(err)
	s.ErrorIs(err, client.ErrAlreadyConnected)
}

// TestCloseStopsReceiving verifies that Close terminates the WebSocket
// and that no further events are delivered afterwards, and that Close
// is idempotent (safe to call twice).
func (s *ActivitySuite) TestCloseStopsReceiving() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "") //nolint:errcheck

		s.writeEnvelope(r.Context(), conn, client.EventEnvelope{
			Seq:       1,
			Type:      "activity",
			Timestamp: time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC),
			Data:      s.mustRawMessage(map[string]string{"n": "first"}),
		})
		// Hold briefly; the client should Close before we push more.
		time.Sleep(500 * time.Millisecond)
	}))
	defer ts.Close()

	c := client.New(ts.URL)
	c.SetToken("test-token")

	activity := client.NewActivityClient(c)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	s.Require().NoError(activity.Connect(ctx))

	env := s.receiveOrFail(activity.Events())
	s.Equal(uint64(1), env.Seq)

	s.Require().NoError(activity.Close())

	// After Close, no further events should arrive in a short window.
	select {
	case extra := <-activity.Events():
		s.FailNow("received unexpected event after Close", "seq=%d", extra.Seq)
	case <-time.After(100 * time.Millisecond):
		// expected: no more events
	}

	// Idempotent: second Close returns nil.
	s.Require().NoError(activity.Close())
}

// TestReplayFetchesEvents verifies that Replay issues a GET against
// /api/v1/events with the correct since query parameter and decodes
// the ReplayResponse shape.
func (s *ActivitySuite) TestReplayFetchesEvents() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal("/api/v1/events", r.URL.Path)
		s.Equal("10", r.URL.Query().Get("since"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"events": []map[string]any{
				{
					"seq":       11,
					"type":      "activity",
					"timestamp": "2026-04-24T10:00:11Z",
					"data":      map[string]string{"source": "slack", "message": "a"},
				},
				{
					"seq":       15,
					"type":      "activity",
					"timestamp": "2026-04-24T10:00:15Z",
					"data":      map[string]string{"source": "email", "message": "b"},
				},
			},
			"truncated":  false,
			"oldest_seq": 1,
			"latest_seq": 15,
		})
	}))
	defer ts.Close()

	c := client.New(ts.URL)
	c.SetToken("test-token")
	activity := client.NewActivityClient(c)

	resp, err := activity.Replay(context.Background(), 10)
	s.Require().NoError(err)
	s.Require().NotNil(resp)
	s.False(resp.Truncated)
	s.Equal(uint64(1), resp.OldestSeq)
	s.Equal(uint64(15), resp.LatestSeq)
	s.Require().Len(resp.Events, 2)
	s.Equal(uint64(11), resp.Events[0].Seq)
	s.Equal("activity", resp.Events[0].Type)
	s.Equal(uint64(15), resp.Events[1].Seq)
}

// TestReplayWithTruncatedFlag verifies that the Truncated flag and
// OldestSeq are surfaced through ReplayResponse.
func (s *ActivitySuite) TestReplayWithTruncatedFlag() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("/api/v1/events", r.URL.Path)
		s.Equal("5", r.URL.Query().Get("since"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"events":     []any{},
			"truncated":  true,
			"oldest_seq": 100,
			"latest_seq": 200,
		})
	}))
	defer ts.Close()

	c := client.New(ts.URL)
	c.SetToken("test-token")
	activity := client.NewActivityClient(c)

	resp, err := activity.Replay(context.Background(), 5)
	s.Require().NoError(err)
	s.Require().NotNil(resp)
	s.True(resp.Truncated)
	s.Equal(uint64(100), resp.OldestSeq)
	s.Equal(uint64(200), resp.LatestSeq)
}

// TestReplayReturnsAPIErrorOnServerFailure verifies that a 500 response
// surfaces as *APIError with ErrCodeServerError.
func (s *ActivitySuite) TestReplayReturnsAPIErrorOnServerFailure() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("/api/v1/events", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "boom",
		})
	}))
	defer ts.Close()

	c := client.New(ts.URL)
	c.SetToken("test-token")
	activity := client.NewActivityClient(c)

	resp, err := activity.Replay(context.Background(), 0)
	s.Require().Error(err)
	s.Nil(resp)

	var apiErr *client.APIError
	s.Require().True(errors.As(err, &apiErr), "expected *APIError, got %T", err)
	s.Equal(client.ErrCodeServerError, apiErr.Code)
	s.Equal(http.StatusInternalServerError, apiErr.StatusCode)
}
