package handler_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/server"
	"github.com/CreateFutureMWilkinson/cue/internal/server/handler"
	"github.com/coder/websocket"
	"github.com/stretchr/testify/suite"
)

// hubAdapter wraps *server.Hub to satisfy handler.Publisher, bridging
// the *server.Subscriber → *handler.Subscription type difference.
type hubAdapter struct {
	hub *server.Hub
}

func (a *hubAdapter) Subscribe(id string) (*handler.Subscription, error) {
	sub, err := a.hub.Subscribe(id)
	if err != nil {
		return nil, err
	}
	return &handler.Subscription{
		ID:     sub.ID,
		Events: sub.Events,
	}, nil
}

func (a *hubAdapter) Unsubscribe(id string) error {
	return a.hub.Unsubscribe(id)
}

// ---------- suite ----------

type WebSocketHandlerSuite struct {
	suite.Suite
}

func TestWebSocketHandler(t *testing.T) {
	suite.Run(t, new(WebSocketHandlerSuite))
}

// ---------- tests ----------

func (s *WebSocketHandlerSuite) TestWebSocketHandler_HappyPath() {
	hub := server.NewHub()
	pub := &hubAdapter{hub: hub}

	srv := httptest.NewServer(handler.WebSocketHandler(pub))
	defer srv.Close()

	// Convert http:// URL to ws://
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	s.Require().NoError(err, "WebSocket dial should succeed")
	defer conn.CloseNow() //nolint:errcheck

	// Publish an activity event through the hub.
	hub.Publish(server.ActivityData{
		Source:  "slack",
		Message: "hello",
		IsError: false,
	})

	// Read one message with a 2s timeout.
	readCtx, readCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer readCancel()

	msgType, payload, err := conn.Read(readCtx)
	s.Require().NoError(err, "should read a message from the WebSocket")
	s.Equal(websocket.MessageText, msgType, "message should be text")

	// Unmarshal and verify envelope fields.
	var env struct {
		Seq  uint64 `json:"seq"`
		Type string `json:"type"`
		Data struct {
			Source string `json:"source"`
		} `json:"data"`
	}
	s.Require().NoError(json.Unmarshal(payload, &env))
	s.Equal("activity", env.Type)
	s.Equal("slack", env.Data.Source)
	s.Greater(env.Seq, uint64(0), "seq should be positive")

	// Close cleanly.
	err = conn.Close(websocket.StatusNormalClosure, "done")
	s.Require().NoError(err, "clean close should succeed")

	// Allow short time for server-side cleanup.
	s.Eventually(func() bool {
		return hub.SubscriberCount() == 0
	}, 500*time.Millisecond, 10*time.Millisecond, "hub should have 0 subscribers after client disconnect")
}
