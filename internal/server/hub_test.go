package server_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/server"
)

type HubSuite struct {
	suite.Suite
}

func TestHub(t *testing.T) {
	suite.Run(t, new(HubSuite))
}

// ---------------------------------------------------------------------------
// Behavior 5: WebSocket hub — subscribe, broadcast, disconnect
// ---------------------------------------------------------------------------

func (s *HubSuite) TestNewHubReturnsNonNil() {
	hub := server.NewHub()
	s.NotNil(hub)
}

func (s *HubSuite) TestSubscribeReturnsSubscriber() {
	hub := server.NewHub()

	sub, err := hub.Subscribe("client-1")
	s.Require().NoError(err)
	s.NotNil(sub, "Subscribe must return a non-nil Subscriber")
	s.Equal("client-1", sub.ID)
	s.NotNil(sub.Events, "Subscriber.Events channel must be non-nil")
}

func (s *HubSuite) TestSubscribeIncrementsCount() {
	hub := server.NewHub()

	_, err := hub.Subscribe("client-1")
	s.Require().NoError(err)

	s.Equal(1, hub.SubscriberCount(), "SubscriberCount must be 1 after one subscribe")

	_, err = hub.Subscribe("client-2")
	s.Require().NoError(err)

	s.Equal(2, hub.SubscriberCount(), "SubscriberCount must be 2 after two subscribes")
}

func (s *HubSuite) TestUnsubscribeRemovesClient() {
	hub := server.NewHub()

	_, err := hub.Subscribe("client-1")
	s.Require().NoError(err)
	s.Equal(1, hub.SubscriberCount())

	err = hub.Unsubscribe("client-1")
	s.NoError(err)
	s.Equal(0, hub.SubscriberCount(), "SubscriberCount must be 0 after unsubscribe")
}

func (s *HubSuite) TestBroadcastDeliversToAllSubscribers() {
	hub := server.NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start hub event loop.
	go hub.Run(ctx)

	sub1, err := hub.Subscribe("client-1")
	s.Require().NoError(err)

	sub2, err := hub.Subscribe("client-2")
	s.Require().NoError(err)

	msg := []byte(`{"type":"test","data":"hello"}`)
	err = hub.Broadcast(msg)
	s.Require().NoError(err)

	// Both subscribers should receive the message.
	select {
	case received := <-sub1.Events:
		s.Equal(msg, received)
	case <-time.After(time.Second):
		s.Fail("subscriber 1 did not receive broadcast within timeout")
	}

	select {
	case received := <-sub2.Events:
		s.Equal(msg, received)
	case <-time.After(time.Second):
		s.Fail("subscriber 2 did not receive broadcast within timeout")
	}
}

func (s *HubSuite) TestBroadcastDoesNotDeliverToUnsubscribed() {
	hub := server.NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	sub, err := hub.Subscribe("client-1")
	s.Require().NoError(err)

	err = hub.Unsubscribe("client-1")
	s.Require().NoError(err)

	msg := []byte(`{"type":"test"}`)
	err = hub.Broadcast(msg)
	s.Require().NoError(err)

	// Unsubscribed client should NOT receive the message.
	select {
	case <-sub.Events:
		s.Fail("unsubscribed client should not receive broadcasts")
	case <-time.After(200 * time.Millisecond):
		// Expected: no message received.
	}
}

func (s *HubSuite) TestRunStopsOnContextCancel() {
	hub := server.NewHub()
	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- hub.Run(ctx)
	}()

	cancel()

	select {
	case err := <-errCh:
		// Run should return nil or context.Canceled — not hang.
		if err != nil {
			s.ErrorIs(err, context.Canceled)
		}
	case <-time.After(2 * time.Second):
		s.Fail("Hub.Run did not return after context cancellation")
	}
}

func (s *HubSuite) TestSubscriberCountZeroInitially() {
	hub := server.NewHub()
	s.Equal(0, hub.SubscriberCount(), "new hub should have 0 subscribers")
}

// ---------------------------------------------------------------------------
// Behavior 2: Publish assigns seq and stamps envelope
// ---------------------------------------------------------------------------

func (s *HubSuite) TestPublishAssignsSeqAndStamps() {
	hub := server.NewHub()

	data1 := server.ActivityData{
		Source:  "slack",
		Message: "new message in #general",
		IsError: false,
	}
	data2 := server.ActivityData{
		Source:  "email",
		Message: "urgent from boss",
		IsError: true,
	}

	env1 := hub.Publish(data1)
	env2 := hub.Publish(data2)

	// First publish gets seq=1, second gets seq=2.
	s.Equal(uint64(1), env1.Seq, "first Publish must return Seq==1")
	s.Equal(uint64(2), env2.Seq, "second Publish must return Seq==2")

	// Type is always "activity".
	s.Equal("activity", env1.Type, "envelope Type must be 'activity'")
	s.Equal("activity", env2.Type, "envelope Type must be 'activity'")

	// Timestamp is non-zero and in UTC.
	s.False(env1.Timestamp.IsZero(), "envelope Timestamp must be non-zero")
	s.Equal(time.UTC, env1.Timestamp.Location(), "envelope Timestamp must be UTC")
	s.False(env2.Timestamp.IsZero(), "envelope Timestamp must be non-zero")
	s.Equal(time.UTC, env2.Timestamp.Location(), "envelope Timestamp must be UTC")

	// Data round-trips the payload.
	got1, ok1 := env1.Data.(server.ActivityData)
	s.Require().True(ok1, "envelope Data must be ActivityData")
	s.Equal(data1, got1, "envelope Data must match published payload")

	got2, ok2 := env2.Data.(server.ActivityData)
	s.Require().True(ok2, "envelope Data must be ActivityData")
	s.Equal(data2, got2, "envelope Data must match published payload")
}

// ---------------------------------------------------------------------------
// Behavior 3: History returns replayable events from ring buffer
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Behavior 4: Publish broadcasts serialized envelope to all subscribers
// ---------------------------------------------------------------------------

func (s *HubSuite) TestPublishBroadcastsToSubscribers() {
	hub := server.NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	sub1, err := hub.Subscribe("client-1")
	s.Require().NoError(err)

	sub2, err := hub.Subscribe("client-2")
	s.Require().NoError(err)

	data := server.ActivityData{
		Source:  "slack",
		Message: "test",
		IsError: false,
	}
	env := hub.Publish(data)

	// Both subscribers must receive a JSON-serialized envelope.
	for _, sub := range []*server.Subscriber{sub1, sub2} {
		select {
		case raw := <-sub.Events:
			// Unmarshal into an envelope with typed Data.
			var got struct {
				Seq       uint64              `json:"seq"`
				Type      string              `json:"type"`
				Timestamp time.Time           `json:"timestamp"`
				Data      server.ActivityData `json:"data"`
			}
			err := json.Unmarshal(raw, &got)
			s.Require().NoError(err, "received bytes must be valid JSON envelope")

			s.Equal(env.Seq, got.Seq, "envelope Seq must match published Seq")
			s.Equal("activity", got.Type, "envelope Type must be 'activity'")
			s.Equal("slack", got.Data.Source, "envelope Data.Source must match")
		case <-time.After(time.Second):
			s.Fail("subscriber %s did not receive publish broadcast within timeout", sub.ID)
		}
	}
}

func (s *HubSuite) TestHistoryReturnsReplayableEvents() {
	hub := server.NewHub()

	// --- Empty buffer: zero-valued response ---
	resp := hub.History(0)
	s.Empty(resp.Events, "empty buffer must return no events")
	s.Equal(uint64(0), resp.OldestSeq, "empty buffer OldestSeq must be 0")
	s.Equal(uint64(0), resp.LatestSeq, "empty buffer LatestSeq must be 0")
	s.False(resp.Truncated, "empty buffer must not be truncated")

	// Publish 3 events.
	hub.Publish(server.ActivityData{Source: "slack", Message: "msg-1"})
	hub.Publish(server.ActivityData{Source: "email", Message: "msg-2"})
	hub.Publish(server.ActivityData{Source: "slack", Message: "msg-3"})

	// --- History(0): returns all 3 events ---
	resp = hub.History(0)
	s.Require().Len(resp.Events, 3, "History(0) must return all 3 events")
	s.Equal(uint64(1), resp.Events[0].Seq)
	s.Equal(uint64(2), resp.Events[1].Seq)
	s.Equal(uint64(3), resp.Events[2].Seq)
	s.Equal(uint64(1), resp.OldestSeq, "OldestSeq must be 1")
	s.Equal(uint64(3), resp.LatestSeq, "LatestSeq must be 3")
	s.False(resp.Truncated, "no eviction so Truncated must be false")

	// --- History(1): returns events with seq 2 and 3 ---
	resp = hub.History(1)
	s.Require().Len(resp.Events, 2, "History(1) must return 2 events")
	s.Equal(uint64(2), resp.Events[0].Seq)
	s.Equal(uint64(3), resp.Events[1].Seq)
	s.False(resp.Truncated)

	// --- History(3): future since, returns empty ---
	resp = hub.History(3)
	s.Empty(resp.Events, "History(latestSeq) must return no events")
	s.False(resp.Truncated, "future since must not be truncated")
	s.Equal(uint64(3), resp.LatestSeq, "LatestSeq must still be 3")
	s.Equal(uint64(1), resp.OldestSeq, "OldestSeq must still be 1")
}

func (s *HubSuite) TestHistoryTruncationAfterEviction() {
	hub := server.NewHub()

	// Publish 502 events — ring capacity is 500, so events 1 and 2 are evicted.
	for i := 0; i < 502; i++ {
		hub.Publish(server.ActivityData{Source: "test", Message: "evt"})
	}

	// History(0): since < oldest retained (3), so truncated.
	resp := hub.History(0)
	s.True(resp.Truncated, "must be truncated when since < oldest retained seq")
	s.Equal(uint64(3), resp.OldestSeq, "OldestSeq must be 3 after evicting 1 and 2")
	s.Equal(uint64(502), resp.LatestSeq, "LatestSeq must be 502")
	s.Require().Len(resp.Events, 500, "must return all 500 retained events")
	s.Equal(uint64(3), resp.Events[0].Seq, "first returned event must be seq 3")
	s.Equal(uint64(502), resp.Events[len(resp.Events)-1].Seq, "last returned event must be seq 502")

	// History(2): since=2 < oldest=3, still truncated.
	resp = hub.History(2)
	s.True(resp.Truncated, "since=2 < oldest=3 must be truncated")
	s.Len(resp.Events, 500, "must return all 500 retained events")

	// History(3): since=3 == oldest, not truncated — returns seq 4..502.
	resp = hub.History(3)
	s.False(resp.Truncated, "since==oldest must not be truncated")
	s.Len(resp.Events, 499, "must return events 4..502")
	s.Equal(uint64(4), resp.Events[0].Seq)
}
