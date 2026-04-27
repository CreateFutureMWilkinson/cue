package server_test

import (
	"context"
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
